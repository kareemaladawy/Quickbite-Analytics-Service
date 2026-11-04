// Package dto holds request/response shapes — the Go analogue of
// order-service's dto/*.request.dto.ts and dto/*.response.dto.ts.
// Validated with go-playground/validator struct tags instead of
// class-validator decorators.
package dto

// GetRestaurantDaysQuery is the query-string shape for
// GET /analytics/restaurants/:restaurantId/days?from=&to=. Both fields are
// required — a missing `from` is VALIDATION_ERROR, not treated as "no
// lower bound", matching the brief's acceptance check.
type GetRestaurantDaysQuery struct {
	From string `validate:"required,datetime=2006-01-02"`
	To   string `validate:"required,datetime=2006-01-02"`
}

// DateRangeQuery is the same {from, to} shape reused by every other
// day-range endpoint added in the homework slice (branches/:id/days,
// products/:id/days, platform/days, the failure-rate endpoints,
// delivery-avg, active-restaurants) — one query shape, one validation rule,
// since none of them need a field GetRestaurantDaysQuery doesn't already
// have.
type DateRangeQuery struct {
	From string `validate:"required,datetime=2006-01-02"`
	To   string `validate:"required,datetime=2006-01-02"`
}
