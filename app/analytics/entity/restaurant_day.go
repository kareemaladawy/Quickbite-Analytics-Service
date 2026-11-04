// Package entity holds plain structs with bson tags — no methods beyond
// simple invariants, no mongo-driver imports. The Go analogue of
// order-service's entity/<module>.entity.ts.
package entity

import "time"

// RestaurantDay is one document in agg_restaurant_day: a day-grained,
// per-restaurant rollup. Averages/rates are stored as sum+count (or
// numerator+denominator) so replaying events re-merges associatively — never
// store a derived average/rate directly.
type RestaurantDay struct {
	RestaurantID        int64     `bson:"restaurant_id"`
	Date                string    `bson:"date"` // YYYY-MM-DD, UTC
	Currency            string    `bson:"currency"`
	OrdersCount         int64     `bson:"orders_count"`
	RevenueSum          int64     `bson:"revenue_sum"`
	OrdersRejectedCount int64     `bson:"orders_rejected_count"`
	DeliveryMsSum       int64     `bson:"delivery_ms_sum"`
	DeliveryMsCount     int64     `bson:"delivery_ms_count"`
	UpdatedAt           time.Time `bson:"updated_at"`
}
