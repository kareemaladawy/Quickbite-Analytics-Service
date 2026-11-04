package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	analytics "quickbite/analytics-service/app/analytics"
	"quickbite/analytics-service/app/analytics/entity"
)

type ProductDayRepo struct {
	collection *mongo.Collection
}

func NewProductDayRepo(db *mongo.Database) *ProductDayRepo {
	return &ProductDayRepo{collection: db.Collection(ProductDayCollection)}
}

// BulkApplyOrderPlaced upserts one row per (product_id, date) line item via
// a single BulkWrite call — a single order.placed event can carry multiple
// line items for different products, and CLAUDE.md §7/§10 forbid a per-row
// Mongo call inside a loop where BulkWrite applies.
func (r *ProductDayRepo) BulkApplyOrderPlaced(items []analytics.OrderPlacedItemInput, date, currency string) error {
	if len(items) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	models := make([]mongo.WriteModel, 0, len(items))
	for _, item := range items {
		filter := bson.D{{Key: "product_id", Value: item.ProductID}, {Key: "date", Value: date}}
		update := bson.D{
			{Key: "$inc", Value: bson.D{
				{Key: "orders_count", Value: 1},
				{Key: "quantity_sum", Value: item.Quantity},
				{Key: "revenue_sum", Value: item.LineTotal},
			}},
			{Key: "$set", Value: bson.D{
				{Key: "currency", Value: currency},
				{Key: "updated_at", Value: now},
			}},
			{Key: "$setOnInsert", Value: bson.D{
				{Key: "product_id", Value: item.ProductID},
				{Key: "date", Value: date},
			}},
		}
		models = append(models, mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true))
	}

	_, err := r.collection.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
	return err
}

// FindByProductAndDateRange returns rows with date in [from, to], inclusive,
// ordered by date ascending — supports GET /analytics/products/:id/days.
func (r *ProductDayRepo) FindByProductAndDateRange(productID int64, from, to string) ([]entity.ProductDay, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{
		{Key: "product_id", Value: productID},
		{Key: "date", Value: bson.D{{Key: "$gte", Value: from}, {Key: "$lte", Value: to}}},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "date", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	rows := make([]entity.ProductDay, 0)
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}
