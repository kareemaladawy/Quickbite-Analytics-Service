package analytics

import (
	"net/http"

	apperr "quickbite/analytics-service/lib/errors"
)

// Module-level, stable-named errors — the Go analogue of a Node
// service/errors.go's `export const Err… = apperr.New(...)`. No inline
// apperr.New(...) at call sites for cases named here.
var (
	ErrInvalidDateRange = apperr.New(
		"ANALYTICS_INVALID_DATE_RANGE",
		"'from' must not be after 'to'",
		http.StatusBadRequest,
	)
	ErrInvalidRestaurantID = apperr.New(
		"VALIDATION_ERROR",
		"restaurantId must be a positive integer",
		http.StatusBadRequest,
	)
	ErrInvalidBranchID = apperr.New(
		"VALIDATION_ERROR",
		"branchId must be a positive integer",
		http.StatusBadRequest,
	)
	ErrInvalidProductID = apperr.New(
		"VALIDATION_ERROR",
		"productId must be a positive integer",
		http.StatusBadRequest,
	)
)
