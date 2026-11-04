// Package boot wires every singleton this service needs — the Go analogue
// of order-service's lib/di/container.ts, but explicit constructor wiring
// instead of tsyringe. Adding a module means editing Run() once; main.go
// never changes.
package boot

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"quickbite/analytics-service/app/analytics/controller"
	"quickbite/analytics-service/app/analytics/eventhandlers"
	"quickbite/analytics-service/app/analytics/repository"
	"quickbite/analytics-service/app/analytics/service"
	"quickbite/analytics-service/lib/config"
	"quickbite/analytics-service/lib/coreclient"
	"quickbite/analytics-service/lib/coreevents"
	"quickbite/analytics-service/lib/logger"
	"quickbite/analytics-service/lib/middleware"
	"quickbite/analytics-service/lib/rbac"
	"quickbite/analytics-service/pkg/messaging"
	pkgmongo "quickbite/analytics-service/pkg/mongo"
)

// Run is the entire process entry point's logic. cmd/api/main.go just calls
// this and exits with whatever it returns.
func Run() error {
	log := logger.Base()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mongoClient, err := pkgmongo.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		return err
	}
	defer mongoClient.Disconnect(context.Background())
	log.Info("mongo connected")

	if err := repository.EnsureIndexes(ctx, mongoClient.Database(), cfg.EventIDsTTL()); err != nil {
		return err
	}

	broker := messaging.NewAMQPBroker(messaging.AMQPConfig{URL: cfg.RabbitMQURL})
	if err := broker.Connect(); err != nil {
		return err
	}
	defer broker.Close()
	log.Info("rabbit connected")

	// --- repositories ---
	restaurantDayRepo := repository.NewRestaurantDayRepo(mongoClient.Database())
	branchDayRepo := repository.NewBranchDayRepo(mongoClient.Database())
	productDayRepo := repository.NewProductDayRepo(mongoClient.Database())
	platformDayRepo := repository.NewPlatformDayRepo(mongoClient.Database())
	eventIDsRepo := repository.NewEventIDsRepo(mongoClient.Database())

	// --- services ---
	analyticsService := service.NewAnalyticsService(restaurantDayRepo, branchDayRepo, productDayRepo, platformDayRepo)

	// --- RBAC read-through cache (core-service over HTTP) ---
	coreClient := coreclient.New(cfg.CoreServiceBaseURL, cfg.CoreInternalAPIKey)
	rbacCache := rbac.NewCache(coreClient, cfg.RBACCacheTTL())

	// --- inbound event consumer: order.events (order.placed/rejected/
	// delivered, payment.completed) ---
	topology := messaging.Topology{
		Exchange:           cfg.RabbitMQOrderEventsExchange,
		Queue:              cfg.RabbitMQQueue,
		BindingKeys:        cfg.RabbitMQBindings,
		DeadLetterExchange: cfg.RabbitMQDLX,
		DeadLetterQueue:    cfg.RabbitMQDLQ,
		Prefetch:           cfg.RabbitMQPrefetch,
	}
	consumer := coreevents.NewConsumer(broker, eventIDsRepo, topology)
	eventhandlers.Register(consumer, analyticsService)
	if err := consumer.Start(); err != nil {
		return err
	}
	log.Info("event consumer started", "queue", cfg.RabbitMQQueue, "bindings", cfg.RabbitMQBindings)

	// --- second inbound event consumer: core.events (rbac.permissions_changed) ---
	coreEventsTopology := messaging.Topology{
		Exchange:           cfg.RabbitMQCoreEventsExchange,
		Queue:              cfg.RabbitMQCoreEventsQueue,
		BindingKeys:        cfg.RabbitMQCoreEventsBindings,
		DeadLetterExchange: cfg.RabbitMQCoreEventsDLX,
		DeadLetterQueue:    cfg.RabbitMQCoreEventsDLQ,
		Prefetch:           cfg.RabbitMQPrefetch,
	}
	coreEventsConsumer := coreevents.NewConsumer(broker, eventIDsRepo, coreEventsTopology)
	rbac.RegisterInvalidationHandler(coreEventsConsumer, rbacCache)
	if err := coreEventsConsumer.Start(); err != nil {
		return err
	}
	log.Info("core-events consumer started", "queue", cfg.RabbitMQCoreEventsQueue, "bindings", cfg.RabbitMQCoreEventsBindings)

	// --- HTTP ---
	analyticsController := controller.NewAnalyticsController(analyticsService)

	router := chi.NewRouter()
	router.Use(middleware.Correlation)
	router.Use(middleware.AccessLog)
	router.Get("/health", healthCheck)
	controller.Mount(router, analyticsController, cfg.AccessSecret, rbacCache)

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.Port),
		Handler: router,
	}

	go func() {
		log.Info("http listening", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server failed", "error", err.Error())
		}
	}()

	waitForShutdownSignal()
	log.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	return server.Shutdown(shutdownCtx)
}

func waitForShutdownSignal() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success":true,"data":{"status":"ok"}}`))
}
