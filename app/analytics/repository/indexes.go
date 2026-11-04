// Package repository is the only place the mongo-driver appears — the Go
// analogue of order-service's repository/<module>.repo.ts, exported
// functions (not classes) each taking the collection to work against.
// Indexes for this module's collections are declared here (not in
// pkg/mongo), since pkg/ must not know about app-specific collection names.
package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	RestaurantDayCollection = "agg_restaurant_day"
	BranchDayCollection     = "agg_branch_day"
	ProductDayCollection    = "agg_product_day"
	PlatformDayCollection   = "agg_platform_day"
	EventIDCollection       = "event_ids"
)

// EnsureIndexes creates every index this module's collections need,
// idempotently — called once on boot (lib/boot), no DDL migrations (Mongo is
// schemaless).
func EnsureIndexes(ctx context.Context, db *mongo.Database, eventIDsTTL time.Duration) error {
	restaurantDayIndexes := []mongo.IndexModel{
		{
			// Upsert target for every order.placed/payment.completed/etc.
			// handler — unique so a duplicate upsert key is a programming
			// error, not a silent second row.
			Keys:    bson.D{{Key: "restaurant_id", Value: 1}, {Key: "date", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uq_restaurant_day_restaurant_date"),
		},
		{
			// Supports GET /analytics/restaurants/:id/days?from=&to= — range
			// scan on date, restaurant_id as the leading equality column.
			Keys:    bson.D{{Key: "date", Value: 1}, {Key: "restaurant_id", Value: 1}},
			Options: options.Index().SetName("idx_restaurant_day_date_restaurant"),
		},
	}
	if _, err := db.Collection(RestaurantDayCollection).Indexes().CreateMany(ctx, restaurantDayIndexes); err != nil {
		return err
	}

	branchDayIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "branch_id", Value: 1}, {Key: "date", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uq_branch_day_branch_date"),
		},
		{
			Keys:    bson.D{{Key: "date", Value: 1}, {Key: "branch_id", Value: 1}},
			Options: options.Index().SetName("idx_branch_day_date_branch"),
		},
	}
	if _, err := db.Collection(BranchDayCollection).Indexes().CreateMany(ctx, branchDayIndexes); err != nil {
		return err
	}

	productDayIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "product_id", Value: 1}, {Key: "date", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uq_product_day_product_date"),
		},
		{
			Keys:    bson.D{{Key: "date", Value: 1}, {Key: "product_id", Value: 1}},
			Options: options.Index().SetName("idx_product_day_date_product"),
		},
	}
	if _, err := db.Collection(ProductDayCollection).Indexes().CreateMany(ctx, productDayIndexes); err != nil {
		return err
	}

	platformDayIndexes := []mongo.IndexModel{
		{
			// date alone is the whole key — there is no second dimension —
			// so this single ascending unique index also serves the
			// from/to range scan GetPlatformDays needs.
			Keys:    bson.D{{Key: "date", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uq_platform_day_date"),
		},
	}
	if _, err := db.Collection(PlatformDayCollection).Indexes().CreateMany(ctx, platformDayIndexes); err != nil {
		return err
	}

	eventIDIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "event_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uq_event_ids_event_id"),
		},
		{
			// TTL index — Mongo's background reaper drops documents once
			// received_at is older than eventIDsTTL. This is the dedupe
			// safety window, not a correctness requirement: RabbitMQ
			// redelivery happens within seconds to minutes, never days.
			Keys:    bson.D{{Key: "received_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(int32(eventIDsTTL.Seconds())).SetName("ttl_event_ids_received_at"),
		},
	}
	if _, err := db.Collection(EventIDCollection).Indexes().CreateMany(ctx, eventIDIndexes); err != nil {
		return err
	}

	return nil
}
