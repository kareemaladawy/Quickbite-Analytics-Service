package auth

import (
	"crypto/subtle"
	"net/http"

	apperr "quickbite/analytics-service/lib/errors"
)

// RequireInternalAPIKey is the Go analogue of order-service's
// lib/auth/api-key.ts requireInternalApiKey: plain equality (constant-time,
// to avoid a timing side-channel) against the shared-secret `api-key`
// header. This service has no internal routes in the video slice, but the
// guard is cross-cutting infra and lands with the rest of lib/auth.
func RequireInternalAPIKey(expected string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return apperr.Wrap(func(w http.ResponseWriter, r *http.Request) error {
			provided := r.Header.Get("api-key")
			if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
				return ErrNotAuthenticated
			}
			next.ServeHTTP(w, r)
			return nil
		})
	}
}
