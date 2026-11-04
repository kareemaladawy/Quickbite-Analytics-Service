package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"quickbite/analytics-service/app/analytics/entity"
)

// PlatformDayRepo is keyed on date alone — there is no restaurant/branch/
// product dimension, so every method takes just the date (plus whatever
// the specific increment needs).
type PlatformDayRepo struct {
	collection *mongo.Collection
}

func NewPlatformDayRepo(db *mongo.Database) *PlatformDayRepo {
	return &PlatformDayRepo{collection: db.Collection(PlatformDayCollection)}
}

func (r *PlatformDayRepo) ApplyOrderPlaced(date, currency string, totalMinor int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{Key: "date", Value: date}}
	update := bson.D{
		{Key: "$inc", Value: bson.D{
			{Key: "orders_count", Value: 1},
			{Key: "revenue_sum", Value: totalMinor},
		}},
		{Key: "$set", Value: bson.D{
			{Key: "currency", Value: currency},
			{Key: "updated_at", Value: time.Now().UTC()},
		}},
		{Key: "$setOnInsert", Value: bson.D{{Key: "date", Value: date}}},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

func (r *PlatformDayRepo) ApplyOrderRejected(date string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{Key: "date", Value: date}}
	update := bson.D{
		{Key: "$inc", Value: bson.D{{Key: "orders_rejected_count", Value: 1}}},
		{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now().UTC()}}},
		{Key: "$setOnInsert", Value: bson.D{{Key: "date", Value: date}}},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

func (r *PlatformDayRepo) ApplyOrderDelivered(date string, deliveryMs int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{Key: "date", Value: date}}
	update := bson.D{
		{Key: "$inc", Value: bson.D{
			{Key: "delivery_ms_sum", Value: deliveryMs},
			{Key: "delivery_ms_count", Value: 1},
		}},
		{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now().UTC()}}},
		{Key: "$setOnInsert", Value: bson.D{{Key: "date", Value: date}}},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

// ApplyPaymentCompleted is the one mutation payment.completed drives — there
// is no restaurantId on PaymentOutboxPayload, so this is the only aggregate
// that event can update.
func (r *PlatformDayRepo) ApplyPaymentCompleted(date, currency string, amountMinor int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{Key: "date", Value: date}}
	update := bson.D{
		{Key: "$inc", Value: bson.D{
			{Key: "payments_completed_count", Value: 1},
			{Key: "payments_completed_sum", Value: amountMinor},
		}},
		{Key: "$set", Value: bson.D{
			{Key: "currency", Value: currency},
			{Key: "updated_at", Value: time.Now().UTC()},
		}},
		{Key: "$setOnInsert", Value: bson.D{{Key: "date", Value: date}}},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

// FindByDateRange returns rows with date in [from, to], inclusive, ordered
// by date ascending — supports GET /analytics/platform/days.
func (r *PlatformDayRepo) FindByDateRange(from, to string) ([]entity.PlatformDay, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{
		{Key: "date", Value: bson.D{{Key: "$gte", Value: from}, {Key: "$lte", Value: to}}},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "date", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	rows := make([]entity.PlatformDay, 0)
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}
