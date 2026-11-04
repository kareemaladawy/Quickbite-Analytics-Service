package entity

import "time"

// PlatformDay is one document in agg_platform_day: a day-grained, global
// rollup with no restaurant/branch/product dimension — the only aggregate
// payment.completed can feed, since PaymentOutboxPayload carries no
// restaurantId.
type PlatformDay struct {
	Date                   string    `bson:"date"`
	Currency               string    `bson:"currency"`
	OrdersCount            int64     `bson:"orders_count"`
	RevenueSum             int64     `bson:"revenue_sum"`
	OrdersRejectedCount    int64     `bson:"orders_rejected_count"`
	DeliveryMsSum          int64     `bson:"delivery_ms_sum"`
	DeliveryMsCount        int64     `bson:"delivery_ms_count"`
	PaymentsCompletedCount int64     `bson:"payments_completed_count"`
	PaymentsCompletedSum   int64     `bson:"payments_completed_sum"`
	UpdatedAt              time.Time `bson:"updated_at"`
}
