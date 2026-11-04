// Package analytics holds types, errors, and enums shared across this
// module's subpackages (service, controller, eventhandlers) — the Go
// analogue of order-service's app/<module>/types.ts, but at the module's
// parent level per this service's folder-structure rule, so every
// subpackage imports them by the same name.
package analytics

// OrderPlacedItemInput is one line item of an order.placed event — trimmed
// to what agg_product_day needs (mirrors order-service's
// OrderOutboxItemPayload, app/order/types.ts).
type OrderPlacedItemInput struct {
	ProductID int64 `json:"productId"`
	Quantity  int64 `json:"quantity"`
	LineTotal int64 `json:"lineTotal"`
}

// OnOrderPlacedInput is what eventhandlers decodes an order.placed payload
// into before handing it to the service — the Go analogue of a Node event
// handler's typed payload, mirroring order-service's
// OrderPlacedOutboxPayload (app/order/types.ts) field-for-field, trimmed to
// what agg_restaurant_day/agg_branch_day/agg_product_day/agg_platform_day
// need.
type OnOrderPlacedInput struct {
	OrderID      int64                  `json:"orderId"`
	RestaurantID int64                  `json:"restaurantId"`
	BranchID     int64                  `json:"branchId"`
	Total        int64                  `json:"total"`
	Currency     string                 `json:"currency"`
	Items        []OrderPlacedItemInput `json:"items"`
	PlacedAt     string                 `json:"placedAt"`
}

// OnOrderRejectedInput is what eventhandlers decodes an order.rejected
// payload into — trimmed from order-service's OrderOutboxPayload
// (app/order/types.ts) to what the rejected-count increment needs.
// CreatedAt (not rejectedAt) supplies the date key, so the increment lands
// on the same (restaurant, date)/(branch, date) row order.placed created —
// a rollup is keyed by the day the order was placed, not the day a later
// lifecycle event happened to land on.
type OnOrderRejectedInput struct {
	OrderID      int64  `json:"orderId"`
	RestaurantID int64  `json:"restaurantId"`
	BranchID     int64  `json:"branchId"`
	CreatedAt    string `json:"createdAt"`
}

// OnOrderDeliveredInput mirrors OnOrderRejectedInput's shape; DeliveredAt is
// used only to derive the delivery duration, never as the date key.
type OnOrderDeliveredInput struct {
	OrderID      int64  `json:"orderId"`
	RestaurantID int64  `json:"restaurantId"`
	BranchID     int64  `json:"branchId"`
	CreatedAt    string `json:"createdAt"`
	DeliveredAt  string `json:"deliveredAt"`
}

// OnPaymentCompletedInput mirrors order-service's PaymentOutboxPayload
// (app/payment/types.ts), trimmed to what agg_platform_day needs. There is
// deliberately no restaurantId field — PaymentOutboxPayload doesn't carry
// one, so this event can only ever feed the restaurant-less platform
// aggregate.
type OnPaymentCompletedInput struct {
	TransactionID int64  `json:"transactionId"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	CreatedAt     string `json:"createdAt"`
}

// RestaurantDayRow is the service-layer, already-derived shape returned by
// GetRestaurantDays — avgOrderMinor is computed here, not stored (the
// repository's entity.RestaurantDay only has the sum+count columns).
type RestaurantDayRow struct {
	Date          string
	OrdersCount   int64
	RevenueMinor  int64
	Currency      string
	AvgOrderMinor int64
}

// GetRestaurantDaysInput is the service method's parameter object — the Go
// analogue of a Node service method's destructured options argument.
type GetRestaurantDaysInput struct {
	RestaurantID int64
	From         string // YYYY-MM-DD, inclusive
	To           string // YYYY-MM-DD, inclusive
}

// BranchDayRow is GetBranchDays' derived row — same shape as
// RestaurantDayRow, one dimension over.
type BranchDayRow struct {
	Date          string
	OrdersCount   int64
	RevenueMinor  int64
	Currency      string
	AvgOrderMinor int64
}

type GetBranchDaysInput struct {
	BranchID int64
	From     string
	To       string
}

// ProductDayRow is GetProductDays' derived row — avgUnitPriceMinor is
// revenueMinor/quantitySum, floor-divided, same convention as
// avgOrderMinor.
type ProductDayRow struct {
	Date              string
	OrdersCount       int64
	QuantitySum       int64
	RevenueMinor      int64
	Currency          string
	AvgUnitPriceMinor int64
}

type GetProductDaysInput struct {
	ProductID int64
	From      string
	To        string
}

// PlatformDayRow is GetPlatformDays' derived row.
type PlatformDayRow struct {
	Date                   string
	OrdersCount            int64
	RevenueMinor           int64
	Currency               string
	AvgOrderMinor          int64
	OrdersRejectedCount    int64
	FailureRate            float64
	DeliveredCount         int64
	AvgDeliveryMs          int64
	PaymentsCompletedCount int64
	PaymentsCompletedSum   int64
}

type GetPlatformDaysInput struct {
	From string
	To   string
}

// FailureRateRow is the derived shape for both the restaurant- and
// branch-scoped failure-rate endpoints — failureRate is
// ordersRejectedCount/ordersCount, never stored.
type FailureRateRow struct {
	Date                string
	OrdersCount         int64
	OrdersRejectedCount int64
	FailureRate         float64
}

type GetRestaurantFailureRateInput struct {
	RestaurantID int64
	From         string
	To           string
}

type GetBranchFailureRateInput struct {
	BranchID int64
	From     string
	To       string
}

// DeliveryAvgRow is the derived shape for the restaurant-scoped
// delivery-avg endpoint — avgDeliveryMs is deliveryMsSum/deliveryMsCount,
// never stored.
type DeliveryAvgRow struct {
	Date           string
	DeliveredCount int64
	AvgDeliveryMs  int64
}

type GetRestaurantDeliveryAvgInput struct {
	RestaurantID int64
	From         string
	To           string
}

// ActiveRestaurantRow is one row of GET /analytics/restaurants/active — a
// restaurant with at least one order.placed in the window, with its order
// count across that window.
type ActiveRestaurantRow struct {
	RestaurantID int64
	OrdersCount  int64
}

type GetActiveRestaurantsInput struct {
	From string
	To   string
}
