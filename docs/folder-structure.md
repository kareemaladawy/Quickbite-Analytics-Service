# Folder structure

```
analytics-service/
├── cmd/
│   └── api/
│       └── main.go              # ~10 lines: calls boot.Run(), exits on error.
├── pkg/                          # framework-free, app-agnostic, no env, no globals
│   ├── mongo/
│   │   └── client.go             # Connect/Disconnect ONLY. Knows nothing about collections.
│   ├── messaging/
│   │   ├── types.go              # Broker interface, Topology, Message, Handler.
│   │   └── amqp.go                # amqp091-go implementation of Broker.
│   └── httpclient/
│       └── client.go             # net/http wrapper: timeout, JSON, retry-on-5xx.
├── lib/                           # app-aware glue: env, middleware, cross-cutting infra
│   ├── boot/
│   │   └── boot.go                # Wires every singleton. Adding a module = edit one function here.
│   ├── config/
│   │   └── env.go                  # Env struct + Load(). The Go analogue of the zod schema.
│   ├── logger/
│   │   └── logger.go                # slog wrapper + FromContext(ctx) for correlation-scoped logging.
│   ├── appcontext/
│   │   └── context.go               # ctx keys: Claims, CorrelationID.
│   ├── errors/
│   │   ├── apperror.go               # AppError{Code, Message, StatusCode, Details}.
│   │   └── handler.go                 # Wrap(handlerFunc) -> http.HandlerFunc, renders the envelope.
│   ├── http/
│   │   └── response.go                # SendSuccess, SendPaginated.
│   ├── middleware/
│   │   └── correlation.go              # Correlation (X-Correlation-Id) + AccessLog.
│   ├── auth/
│   │   ├── types.go                     # SystemRole, Claims.
│   │   ├── errors.go                     # ErrNotAuthenticated, ErrForbidden.
│   │   ├── jwt.go                         # VerifyAccessToken (HS256, same secret as core/order).
│   │   ├── middleware.go                  # Authenticate (cookie, then Bearer).
│   │   └── apikey.go                       # RequireInternalAPIKey (unused in the video slice, present for parity).
│   ├── rbac/
│   │   ├── types.go                        # PermissionsFetcher interface.
│   │   ├── cache.go                         # In-process, TTL'd read-through Cache.
│   │   └── middleware.go                    # Require(cache, permission).
│   ├── coreclient/
│   │   ├── types.go                        # CoreEnvelope[T], RolePermissionsResponse.
│   │   ├── client.go                        # HTTP client to core-service.
│   │   └── rbac.go                          # FetchPermissions — satisfies rbac.PermissionsFetcher.
│   └── coreevents/
│       ├── payloads.go                     # Envelope, EventHandler, EventDeduper interface.
│       └── consumer.go                      # Generic dedupe -> dispatch -> ack/nack-to-DLQ loop.
├── app/
│   └── analytics/                          # PACKAGE analytics — shared module types live here
│       ├── types.go                         # OnOrderPlacedInput, RestaurantDayRow, GetRestaurantDaysInput.
│       ├── errors.go                        # ErrInvalidDateRange, ErrInvalidRestaurantID.
│       ├── enums.go                          # PermAnalyticsRead, OrderEventTypePlaced.
│       ├── entity/
│       │   ├── restaurant_day.go              # bson-tagged RestaurantDay.
│       │   └── event_id.go                     # bson-tagged EventID.
│       ├── repository/                        # ONLY place mongo-driver appears
│       │   ├── indexes.go                       # EnsureIndexes for agg_restaurant_day + event_ids.
│       │   ├── restaurant_day.repo.go            # ApplyOrderPlaced (upsert), FindByRestaurantAndDateRange.
│       │   └── event_id.repo.go                   # MarkSeen — satisfies coreevents.EventDeduper.
│       ├── service/
│       │   └── analytics.service.go               # OnOrderPlaced, GetRestaurantDays (derives avgOrderMinor).
│       ├── controller/
│       │   ├── analytics.controller.go              # GetRestaurantDays handler.
│       │   └── routes.go                              # Mount(router, controller, secret, rbacCache).
│       ├── dto/
│       │   ├── days.request.go                        # GetRestaurantDaysQuery (validator tags).
│       │   └── days.response.go                        # RestaurantDayResponse + factory.
│       └── eventhandlers/
│           └── handlers.go                              # order.placed -> AnalyticsService.OnOrderPlaced.
├── play/                          # GITIGNORED — dev aids, not part of the service
│   ├── mock-core/main.go
│   ├── mint-jwt/main.go
│   ├── publish-test/main.go
│   └── check-mongo/main.go
├── go.mod / go.sum
├── .env.example
├── .gitignore
├── CLAUDE.md
├── README.md
├── plan.md
└── docs/
    ├── folder-structure.md        # this file
    ├── system-design.md
    ├── api-contracts.md
    ├── node-to-go-mapping.md
    ├── ai-prompts.md
    └── implementation-plan.md
```

## Layer rules

```
app/  -> may import lib, pkg
lib/  -> may import pkg, env; may NOT import app/<module>/*
pkg/  -> no imports from lib or app, no env, no globals, NO app-specific knowledge
```

Every arrow in the dependency graph points **down**. `app/analytics` is the only package that knows `agg_restaurant_day` exists; `lib/coreevents` only knows "some string key might already have been seen" (via the `EventDeduper` interface); `pkg/mongo` only knows how to open a connection.
