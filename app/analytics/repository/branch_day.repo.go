package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"quickbite/analytics-service/app/analytics/entity"
)

// BranchDayRepo mirrors RestaurantDayRepo exactly, keyed on (branch_id,
// date) instead of (restaurant_id, date) — see that file for the upsert
// rationale (associative $inc, $setOnInsert for out-of-order delivery).
type BranchDayRepo struct {
	collection *mongo.Collection
}

func NewBranchDayRepo(db *mongo.Database) *BranchDayRepo {
	return &BranchDayRepo{collection: db.Collection(BranchDayCollection)}
}

func (r *BranchDayRepo) ApplyOrderPlaced(branchID int64, date, currency string, totalMinor int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{Key: "branch_id", Value: branchID}, {Key: "date", Value: date}}
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
			{Key: "branch_id", Value: branchID},
			{Key: "date", Value: date},
		}},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

func (r *BranchDayRepo) ApplyOrderRejected(branchID int64, date string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{Key: "branch_id", Value: branchID}, {Key: "date", Value: date}}
	update := bson.D{
		{Key: "$inc", Value: bson.D{{Key: "orders_rejected_count", Value: 1}}},
		{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now().UTC()}}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "branch_id", Value: branchID},
			{Key: "date", Value: date},
		}},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

func (r *BranchDayRepo) ApplyOrderDelivered(branchID int64, date string, deliveryMs int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{Key: "branch_id", Value: branchID}, {Key: "date", Value: date}}
	update := bson.D{
		{Key: "$inc", Value: bson.D{
			{Key: "delivery_ms_sum", Value: deliveryMs},
			{Key: "delivery_ms_count", Value: 1},
		}},
		{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now().UTC()}}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "branch_id", Value: branchID},
			{Key: "date", Value: date},
		}},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

// FindByBranchAndDateRange returns rows with date in [from, to], inclusive,
// ordered by date ascending — supports GET /analytics/branches/:id/days and
// the branch failure-rate endpoint (both read off the same rows).
func (r *BranchDayRepo) FindByBranchAndDateRange(branchID int64, from, to string) ([]entity.BranchDay, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{
		{Key: "branch_id", Value: branchID},
		{Key: "date", Value: bson.D{{Key: "$gte", Value: from}, {Key: "$lte", Value: to}}},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "date", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	rows := make([]entity.BranchDay, 0)
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}
