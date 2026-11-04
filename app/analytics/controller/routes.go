package controller

import (
	"github.com/go-chi/chi/v5"

	analytics "quickbite/analytics-service/app/analytics"
	"quickbite/analytics-service/lib/auth"
	apperr "quickbite/analytics-service/lib/errors"
	"quickbite/analytics-service/lib/rbac"
)

// Mount wires authenticate + rbac.Require to controller methods — the Go
// analogue of order-service's module routes.ts.
func Mount(r chi.Router, c *AnalyticsController, accessSecret string, rbacCache *rbac.Cache) {
	r.Route("/api/v1/analytics", func(sub chi.Router) {
		sub.Use(auth.Authenticate(accessSecret))
		sub.Use(rbac.Require(rbacCache, analytics.PermAnalyticsRead))

		sub.Get("/restaurants/{restaurantId}/days", apperr.Wrap(c.GetRestaurantDays))
		sub.Get("/restaurants/{restaurantId}/failure-rate", apperr.Wrap(c.GetRestaurantFailureRate))
		sub.Get("/restaurants/{restaurantId}/delivery-avg", apperr.Wrap(c.GetRestaurantDeliveryAvg))
		// Registered ahead of the :restaurantId routes above only in intent,
		// not in chi's matching order — chi dispatches "active" as a static
		// segment ahead of the {restaurantId} wildcard regardless of
		// registration order, so there's no literal "/active" vs
		// "/42" ambiguity to worry about.
		sub.Get("/restaurants/active", apperr.Wrap(c.GetActiveRestaurants))

		sub.Get("/branches/{branchId}/days", apperr.Wrap(c.GetBranchDays))
		sub.Get("/branches/{branchId}/failure-rate", apperr.Wrap(c.GetBranchFailureRate))

		sub.Get("/products/{productId}/days", apperr.Wrap(c.GetProductDays))

		sub.Get("/platform/days", apperr.Wrap(c.GetPlatformDays))
	})
}
