package entity

import "time"

// BranchDay is one document in agg_branch_day — the branch-grained analogue
// of RestaurantDay. Fed by the same three event handlers (order.placed,
// order.rejected, order.delivered) as agg_restaurant_day, keyed on
// (branch_id, date) instead of (restaurant_id, date).
type BranchDay struct {
	BranchID            int64     `bson:"branch_id"`
	Date                string    `bson:"date"`
	Currency            string    `bson:"currency"`
	OrdersCount         int64     `bson:"orders_count"`
	RevenueSum          int64     `bson:"revenue_sum"`
	OrdersRejectedCount int64     `bson:"orders_rejected_count"`
	DeliveryMsSum       int64     `bson:"delivery_ms_sum"`
	DeliveryMsCount     int64     `bson:"delivery_ms_count"`
	UpdatedAt           time.Time `bson:"updated_at"`
}
