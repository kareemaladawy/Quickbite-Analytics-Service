package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"quickbite/analytics-service/app/analytics/entity"
)

type RestaurantDayRepo struct {
	collection *mongo.Collection
}

func NewRestaurantDayRepo(db *mongo.Database) *RestaurantDayRepo {
	return &RestaurantDayRepo{collection: db.Collection(RestaurantDayCollection)}
}

// ApplyOrderPlaced upserts the (restaurant_id, date) row: $inc on
// orders_count/revenue_sum so replays of a *different* event compound
// correctly, while the unique index + event_ids dedupe (lib/coreevents)
// guarantees this is never called twice for the *same* event.
func (r *RestaurantDayRepo) ApplyOrderPlaced(restaurantID int64, date, currency string, totalMinor int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{Key: "restaurant_id", Value: restaurantID}, {Key: "date", Value: date}}
	update := bson.D{
		{Key: "$inc", Value: bson.D{
			{Key: "orders_count", Value: 1},
			{Key: "revenue_sum", Value: totalMinor},
		}},
		{Key: "$set", Value: bson.D{
			{Key: "currency", Value: currency},
			{Key: "updated_at", Value: time.Now().UTC()},
		}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "restaurant_id", Value: restaurantID},
			{Key: "date", Value: date},
		}},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

// ApplyOrderRejected increments orders_rejected_count for the
// (restaurant_id, date) row. $setOnInsert means this is safe even if
// order.rejected is processed before order.placed for the same order (the
// plan.md out-of-order-delivery gotcha) — the row is created with zero
// orders_count/revenue_sum and order.placed's own upsert fills those in
// later, associatively.
func (r *RestaurantDayRepo) ApplyOrderRejected(restaurantID int64, date string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{Key: "restaurant_id", Value: restaurantID}, {Key: "date", Value: date}}
	update := bson.D{
		{Key: "$inc", Value: bson.D{{Key: "orders_rejected_count", Value: 1}}},
		{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now().UTC()}}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "restaurant_id", Value: restaurantID},
			{Key: "date", Value: date},
		}},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

// ApplyOrderDelivered increments delivery_ms_sum/delivery_ms_count for the
// (restaurant_id, date) row — an average is never stored directly; deriving
// delivery_ms_sum/delivery_ms_count lets the service compute avgDeliveryMs
// at read time.
func (r *RestaurantDayRepo) ApplyOrderDelivered(restaurantID int64, date string, deliveryMs int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{Key: "restaurant_id", Value: restaurantID}, {Key: "date", Value: date}}
	update := bson.D{
		{Key: "$inc", Value: bson.D{
			{Key: "delivery_ms_sum", Value: deliveryMs},
			{Key: "delivery_ms_count", Value: 1},
		}},
		{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now().UTC()}}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "restaurant_id", Value: restaurantID},
			{Key: "date", Value: date},
		}},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

// FindActiveInRange groups by restaurant_id across [from, to], summing
// orders_count — backs GET /analytics/restaurants/active.
func (r *RestaurantDayRepo) FindActiveInRange(from, to string) ([]entity.ActiveRestaurant, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "date", Value: bson.D{{Key: "$gte", Value: from}, {Key: "$lte", Value: to}}},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$restaurant_id"},
			{Key: "orders_count", Value: bson.D{{Key: "$sum", Value: "$orders_count"}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	rows := make([]entity.ActiveRestaurant, 0)
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// FindByRestaurantAndDateRange returns rows with date in [from, to],
// inclusive, ordered by date ascending — supports
// GET /analytics/restaurants/:id/days?from=&to=.
func (r *RestaurantDayRepo) FindByRestaurantAndDateRange(restaurantID int64, from, to string) ([]entity.RestaurantDay, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{
		{Key: "restaurant_id", Value: restaurantID},
		{Key: "date", Value: bson.D{{Key: "$gte", Value: from}, {Key: "$lte", Value: to}}},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "date", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	rows := make([]entity.RestaurantDay, 0)
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}
