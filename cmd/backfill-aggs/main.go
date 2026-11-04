// backfill-aggs re-derives agg_restaurant_day/agg_branch_day/
// agg_product_day/agg_platform_day from a dump of historical order.placed
// events (docs/implementation-plan.md Phase 11). It reuses
// AnalyticsService.OnOrderPlaced unchanged — the upsert is the same
// associative $inc regardless of whether it is driven by a live RabbitMQ
// delivery or a one-time backfill line — but it deliberately does NOT go
// through lib/coreevents.Consumer: there is no RabbitMQ message to ack/nack
// here, and event_ids dedupe must not apply to a backfill, since every line
// in the dump is by definition being seen for the first time by this
// service (the live consumer dedupes against *future* redeliveries of
// events that already ran through this exact path once; a backfill isn't
// that).
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/joho/godotenv"

	analytics "quickbite/analytics-service/app/analytics"
	"quickbite/analytics-service/app/analytics/repository"
	"quickbite/analytics-service/app/analytics/service"
	pkgmongo "quickbite/analytics-service/pkg/mongo"
)

func main() {
	filePath := flag.String("file", "", "path to a JSON-lines file of order.placed payloads (one analytics.OnOrderPlacedInput per line)")
	flag.Parse()

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "backfill-aggs: -file is required")
		os.Exit(1)
	}

	_ = godotenv.Load()
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		dbName = "quickbite_analytics"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mongoClient, err := pkgmongo.Connect(ctx, uri, dbName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "backfill-aggs: mongo connect failed:", err)
		os.Exit(1)
	}
	defer mongoClient.Disconnect(context.Background())

	// 7 days is this service's default dedupe-ledger TTL, unused by this
	// command — EnsureIndexes creates every collection's indexes
	// idempotently regardless, which the backfill depends on (the unique
	// index on (restaurant_id, date) etc. must exist before the first
	// upsert, same as the live service).
	if err := repository.EnsureIndexes(ctx, mongoClient.Database(), 7*24*time.Hour); err != nil {
		fmt.Fprintln(os.Stderr, "backfill-aggs: ensure indexes failed:", err)
		os.Exit(1)
	}

	svc := service.NewAnalyticsService(
		repository.NewRestaurantDayRepo(mongoClient.Database()),
		repository.NewBranchDayRepo(mongoClient.Database()),
		repository.NewProductDayRepo(mongoClient.Database()),
		repository.NewPlatformDayRepo(mongoClient.Database()),
	)

	f, err := os.Open(*filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "backfill-aggs: open file failed:", err)
		os.Exit(1)
	}
	defer f.Close()

	var applied, failed int
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for lineNum := 1; scanner.Scan(); lineNum++ {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var input analytics.OnOrderPlacedInput
		if err := json.Unmarshal(line, &input); err != nil {
			fmt.Fprintf(os.Stderr, "backfill-aggs: line %d: decode failed: %v\n", lineNum, err)
			failed++
			continue
		}

		if err := svc.OnOrderPlaced(input); err != nil {
			fmt.Fprintf(os.Stderr, "backfill-aggs: line %d (orderId=%d): apply failed: %v\n", lineNum, input.OrderID, err)
			failed++
			continue
		}
		applied++
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintln(os.Stderr, "backfill-aggs: scan failed:", err)
		os.Exit(1)
	}

	fmt.Printf("backfill-aggs: applied=%d failed=%d\n", applied, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
