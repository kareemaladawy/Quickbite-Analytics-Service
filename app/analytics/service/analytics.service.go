// Package service holds just the service struct and its methods — no
// types/errors/enums here (those live in the parent app/analytics package
// per this service's folder-structure rule).
package service

import (
	"strings"
	"time"

	analytics "quickbite/analytics-service/app/analytics"
	"quickbite/analytics-service/app/analytics/repository"
)

type AnalyticsService struct {
	restaurantDayRepo *repository.RestaurantDayRepo
	branchDayRepo     *repository.BranchDayRepo
	productDayRepo    *repository.ProductDayRepo
	platformDayRepo   *repository.PlatformDayRepo
}

func NewAnalyticsService(
	restaurantDayRepo *repository.RestaurantDayRepo,
	branchDayRepo *repository.BranchDayRepo,
	productDayRepo *repository.ProductDayRepo,
	platformDayRepo *repository.PlatformDayRepo,
) *AnalyticsService {
	return &AnalyticsService{
		restaurantDayRepo: restaurantDayRepo,
		branchDayRepo:     branchDayRepo,
		productDayRepo:    productDayRepo,
		platformDayRepo:   platformDayRepo,
	}
}

// OnOrderPlaced is the event handler's entry point into business logic —
// derives the UTC calendar date from placedAt and upserts the row on every
// aggregate this event feeds: agg_restaurant_day, agg_branch_day (one
// upsert each, same key shape), and agg_product_day (one BulkWrite for
// every line item). One event in, four upserts out — all independent and
// associative, so a partial failure (e.g. the product upsert errors) is
// safe to retry: the earlier upserts in this call are idempotent to
// re-apply alongside it once dedupe is bypassed by a nack/redelivery.
func (s *AnalyticsService) OnOrderPlaced(input analytics.OnOrderPlacedInput) error {
	date := dateOnly(input.PlacedAt)

	if err := s.restaurantDayRepo.ApplyOrderPlaced(input.RestaurantID, date, input.Currency, input.Total); err != nil {
		return err
	}
	if err := s.branchDayRepo.ApplyOrderPlaced(input.BranchID, date, input.Currency, input.Total); err != nil {
		return err
	}
	if err := s.productDayRepo.BulkApplyOrderPlaced(input.Items, date, input.Currency); err != nil {
		return err
	}
	return s.platformDayRepo.ApplyOrderPlaced(date, input.Currency, input.Total)
}

// OnOrderRejected fans the orders_rejected_count increment out to every
// aggregate agg_restaurant_day/agg_branch_day/agg_platform_day — see
// plan.md's out-of-order-delivery gotcha: this can be processed before
// order.placed for the same order, and each repo's $setOnInsert makes that
// safe.
func (s *AnalyticsService) OnOrderRejected(input analytics.OnOrderRejectedInput) error {
	date := dateOnly(input.CreatedAt)

	if err := s.restaurantDayRepo.ApplyOrderRejected(input.RestaurantID, date); err != nil {
		return err
	}
	if err := s.branchDayRepo.ApplyOrderRejected(input.BranchID, date); err != nil {
		return err
	}
	return s.platformDayRepo.ApplyOrderRejected(date)
}

// OnOrderDelivered derives the delivery duration (deliveredAt - createdAt,
// in milliseconds) and fans the delivery_ms_sum/delivery_ms_count increment
// out the same way OnOrderRejected does. Malformed timestamps are not
// expected from order-service's own ISO-8601 output, but a parse failure
// here is returned as an error (nacked -> DLQ) rather than silently
// skipped, since silently dropping a delivery sample would corrupt the
// avgDeliveryMs read-time derivation.
func (s *AnalyticsService) OnOrderDelivered(input analytics.OnOrderDeliveredInput) error {
	date := dateOnly(input.CreatedAt)

	deliveryMs, err := deliveryDurationMs(input.CreatedAt, input.DeliveredAt)
	if err != nil {
		return err
	}

	if err := s.restaurantDayRepo.ApplyOrderDelivered(input.RestaurantID, date, deliveryMs); err != nil {
		return err
	}
	if err := s.branchDayRepo.ApplyOrderDelivered(input.BranchID, date, deliveryMs); err != nil {
		return err
	}
	return s.platformDayRepo.ApplyOrderDelivered(date, deliveryMs)
}

// OnPaymentCompleted is the one handler that can only ever touch
// agg_platform_day — PaymentOutboxPayload carries no restaurantId.
func (s *AnalyticsService) OnPaymentCompleted(input analytics.OnPaymentCompletedInput) error {
	date := dateOnly(input.CreatedAt)
	return s.platformDayRepo.ApplyPaymentCompleted(date, input.Currency, input.Amount)
}

// GetRestaurantDays is where avgOrderMinor is derived — never stored,
// always revenue/count at read time, so a future replay never leaves a
// stale average sitting in Mongo.
func (s *AnalyticsService) GetRestaurantDays(input analytics.GetRestaurantDaysInput) ([]analytics.RestaurantDayRow, error) {
	if input.From > input.To {
		return nil, analytics.ErrInvalidDateRange
	}

	days, err := s.restaurantDayRepo.FindByRestaurantAndDateRange(input.RestaurantID, input.From, input.To)
	if err != nil {
		return nil, err
	}

	rows := make([]analytics.RestaurantDayRow, 0, len(days))
	for _, d := range days {
		rows = append(rows, analytics.RestaurantDayRow{
			Date:          d.Date,
			OrdersCount:   d.OrdersCount,
			RevenueMinor:  d.RevenueSum,
			Currency:      d.Currency,
			AvgOrderMinor: floorDiv(d.RevenueSum, d.OrdersCount),
		})
	}
	return rows, nil
}

// GetBranchDays mirrors GetRestaurantDays exactly, one dimension over.
func (s *AnalyticsService) GetBranchDays(input analytics.GetBranchDaysInput) ([]analytics.BranchDayRow, error) {
	if input.From > input.To {
		return nil, analytics.ErrInvalidDateRange
	}

	days, err := s.branchDayRepo.FindByBranchAndDateRange(input.BranchID, input.From, input.To)
	if err != nil {
		return nil, err
	}

	rows := make([]analytics.BranchDayRow, 0, len(days))
	for _, d := range days {
		rows = append(rows, analytics.BranchDayRow{
			Date:          d.Date,
			OrdersCount:   d.OrdersCount,
			RevenueMinor:  d.RevenueSum,
			Currency:      d.Currency,
			AvgOrderMinor: floorDiv(d.RevenueSum, d.OrdersCount),
		})
	}
	return rows, nil
}

// GetProductDays mirrors GetRestaurantDays; avgUnitPriceMinor is
// revenueMinor/quantitySum rather than revenueMinor/ordersCount, since
// ordersCount here counts line items, not units.
func (s *AnalyticsService) GetProductDays(input analytics.GetProductDaysInput) ([]analytics.ProductDayRow, error) {
	if input.From > input.To {
		return nil, analytics.ErrInvalidDateRange
	}

	days, err := s.productDayRepo.FindByProductAndDateRange(input.ProductID, input.From, input.To)
	if err != nil {
		return nil, err
	}

	rows := make([]analytics.ProductDayRow, 0, len(days))
	for _, d := range days {
		rows = append(rows, analytics.ProductDayRow{
			Date:              d.Date,
			OrdersCount:       d.OrdersCount,
			QuantitySum:       d.QuantitySum,
			RevenueMinor:      d.RevenueSum,
			Currency:          d.Currency,
			AvgUnitPriceMinor: floorDiv(d.RevenueSum, d.QuantitySum),
		})
	}
	return rows, nil
}

// GetPlatformDays derives every rate/average field at read time from the
// stored sum+count columns.
func (s *AnalyticsService) GetPlatformDays(input analytics.GetPlatformDaysInput) ([]analytics.PlatformDayRow, error) {
	if input.From > input.To {
		return nil, analytics.ErrInvalidDateRange
	}

	days, err := s.platformDayRepo.FindByDateRange(input.From, input.To)
	if err != nil {
		return nil, err
	}

	rows := make([]analytics.PlatformDayRow, 0, len(days))
	for _, d := range days {
		rows = append(rows, analytics.PlatformDayRow{
			Date:                   d.Date,
			OrdersCount:            d.OrdersCount,
			RevenueMinor:           d.RevenueSum,
			Currency:               d.Currency,
			AvgOrderMinor:          floorDiv(d.RevenueSum, d.OrdersCount),
			OrdersRejectedCount:    d.OrdersRejectedCount,
			FailureRate:            floorDivFloat(d.OrdersRejectedCount, d.OrdersCount),
			DeliveredCount:         d.DeliveryMsCount,
			AvgDeliveryMs:          floorDiv(d.DeliveryMsSum, d.DeliveryMsCount),
			PaymentsCompletedCount: d.PaymentsCompletedCount,
			PaymentsCompletedSum:   d.PaymentsCompletedSum,
		})
	}
	return rows, nil
}

// GetRestaurantFailureRate is derived from fields already on
// agg_restaurant_day — no new collection, per docs/implementation-plan.md
// Phase 9.
func (s *AnalyticsService) GetRestaurantFailureRate(input analytics.GetRestaurantFailureRateInput) ([]analytics.FailureRateRow, error) {
	if input.From > input.To {
		return nil, analytics.ErrInvalidDateRange
	}

	days, err := s.restaurantDayRepo.FindByRestaurantAndDateRange(input.RestaurantID, input.From, input.To)
	if err != nil {
		return nil, err
	}

	rows := make([]analytics.FailureRateRow, 0, len(days))
	for _, d := range days {
		rows = append(rows, analytics.FailureRateRow{
			Date:                d.Date,
			OrdersCount:         d.OrdersCount,
			OrdersRejectedCount: d.OrdersRejectedCount,
			FailureRate:         floorDivFloat(d.OrdersRejectedCount, d.OrdersCount),
		})
	}
	return rows, nil
}

// GetBranchFailureRate mirrors GetRestaurantFailureRate, one dimension
// over.
func (s *AnalyticsService) GetBranchFailureRate(input analytics.GetBranchFailureRateInput) ([]analytics.FailureRateRow, error) {
	if input.From > input.To {
		return nil, analytics.ErrInvalidDateRange
	}

	days, err := s.branchDayRepo.FindByBranchAndDateRange(input.BranchID, input.From, input.To)
	if err != nil {
		return nil, err
	}

	rows := make([]analytics.FailureRateRow, 0, len(days))
	for _, d := range days {
		rows = append(rows, analytics.FailureRateRow{
			Date:                d.Date,
			OrdersCount:         d.OrdersCount,
			OrdersRejectedCount: d.OrdersRejectedCount,
			FailureRate:         floorDivFloat(d.OrdersRejectedCount, d.OrdersCount),
		})
	}
	return rows, nil
}

// GetRestaurantDeliveryAvg is derived from agg_restaurant_day's
// delivery_ms_sum/delivery_ms_count — no new collection.
func (s *AnalyticsService) GetRestaurantDeliveryAvg(input analytics.GetRestaurantDeliveryAvgInput) ([]analytics.DeliveryAvgRow, error) {
	if input.From > input.To {
		return nil, analytics.ErrInvalidDateRange
	}

	days, err := s.restaurantDayRepo.FindByRestaurantAndDateRange(input.RestaurantID, input.From, input.To)
	if err != nil {
		return nil, err
	}

	rows := make([]analytics.DeliveryAvgRow, 0, len(days))
	for _, d := range days {
		rows = append(rows, analytics.DeliveryAvgRow{
			Date:           d.Date,
			DeliveredCount: d.DeliveryMsCount,
			AvgDeliveryMs:  floorDiv(d.DeliveryMsSum, d.DeliveryMsCount),
		})
	}
	return rows, nil
}

// GetActiveRestaurants lists every restaurant with at least one
// order.placed in [from, to] — a $group aggregation over agg_restaurant_day,
// no new collection.
func (s *AnalyticsService) GetActiveRestaurants(input analytics.GetActiveRestaurantsInput) ([]analytics.ActiveRestaurantRow, error) {
	if input.From > input.To {
		return nil, analytics.ErrInvalidDateRange
	}

	active, err := s.restaurantDayRepo.FindActiveInRange(input.From, input.To)
	if err != nil {
		return nil, err
	}

	rows := make([]analytics.ActiveRestaurantRow, 0, len(active))
	for _, a := range active {
		rows = append(rows, analytics.ActiveRestaurantRow{
			RestaurantID: a.RestaurantID,
			OrdersCount:  a.OrdersCount,
		})
	}
	return rows, nil
}

// dateOnly extracts the YYYY-MM-DD prefix of an ISO-8601 UTC timestamp —
// the aggregate's grain. placedAt/createdAt are always UTC (order-service's
// outbox payloads are *.toISOString()), so no timezone conversion is needed
// here.
func dateOnly(isoTimestamp string) string {
	if idx := strings.IndexByte(isoTimestamp, 'T'); idx >= 0 {
		return isoTimestamp[:idx]
	}
	return isoTimestamp
}

// deliveryDurationMs is deliveredAt - createdAt, in milliseconds — never
// negative in practice (delivery always follows creation), but a negative
// result is passed through rather than clamped: a negative delivery_ms_sum
// contribution would be a data bug upstream worth surfacing, not masking.
func deliveryDurationMs(createdAt, deliveredAt string) (int64, error) {
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return 0, err
	}
	delivered, err := time.Parse(time.RFC3339, deliveredAt)
	if err != nil {
		return 0, err
	}
	return delivered.Sub(created).Milliseconds(), nil
}

// floorDiv is the one place every avg*Minor/avgDeliveryMs field is derived
// — integer division, floor-divided (no rounding), 0 when the denominator
// is 0 (an empty day, not a divide-by-zero panic).
func floorDiv(numerator, denominator int64) int64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

// floorDivFloat is floorDiv's float64 analogue for rate fields
// (failureRate) where integer division would always round to 0 or 1.
func floorDivFloat(numerator, denominator int64) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}
