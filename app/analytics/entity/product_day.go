package entity

import "time"

// ProductDay is one document in agg_product_day: a day-grained,
// per-product rollup of line items across every order.placed event — one
// upsert per (product_id, date), applied via BulkWrite since a single order
// can carry multiple line items for different products.
type ProductDay struct {
	ProductID   int64     `bson:"product_id"`
	Date        string    `bson:"date"`
	Currency    string    `bson:"currency"`
	OrdersCount int64     `bson:"orders_count"` // number of line items referencing this product
	QuantitySum int64     `bson:"quantity_sum"`
	RevenueSum  int64     `bson:"revenue_sum"` // sum of lineTotal across those line items
	UpdatedAt   time.Time `bson:"updated_at"`
}
