package rbac

import (
	"net/http"

	"quickbite/analytics-service/lib/appcontext"
	"quickbite/analytics-service/lib/auth"
	apperr "quickbite/analytics-service/lib/errors"
)

// Require is the Go analogue of order-service's lib/auth/rbac.ts rbac()
// middleware factory: system_admin always passes; a restaurant_user with a
// restaurantRole is checked against the cache; everyone else is forbidden.
func Require(cache *Cache, permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return apperr.Wrap(func(w http.ResponseWriter, r *http.Request) error {
			claims, ok := appcontext.ClaimsFromContext(r.Context())
			if !ok {
				return auth.ErrNotAuthenticated
			}

			if claims.Role == string(auth.SystemRoleSystemAdmin) {
				next.ServeHTTP(w, r)
				return nil
			}

			if claims.RestaurantRole != nil {
				allowed, err := cache.HasPermission(*claims.RestaurantRole, permission)
				if err != nil {
					return err
				}
				if allowed {
					next.ServeHTTP(w, r)
					return nil
				}
			}

			return auth.ErrForbidden
		})
	}
}
