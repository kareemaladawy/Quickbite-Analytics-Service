package auth

import (
	"net/http"

	"quickbite/analytics-service/lib/appcontext"
	apperr "quickbite/analytics-service/lib/errors"
)

// Authenticate is the Go analogue of order-service's lib/auth/guard.ts
// authenticate middleware: token from cookie first, then Authorization:
// Bearer (the brief's explicit precedence; order-service's Node guard only
// checks the cookie, since its clients are browser-only — this service also
// serves server-to-server callers, so Bearer is the fallback).
func Authenticate(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return apperr.Wrap(func(w http.ResponseWriter, r *http.Request) error {
			token := bearerOrCookieToken(r)
			if token == "" {
				return ErrNotAuthenticated
			}

			claims, err := VerifyAccessToken(token, secret)
			if err != nil {
				return err
			}

			ctx := appcontext.WithClaims(r.Context(), appcontext.Claims{
				UserID:         claims.UserID,
				Role:           string(claims.Role),
				RestaurantID:   claims.RestaurantID,
				RestaurantRole: claims.RestaurantRole,
				BranchIDs:      claims.BranchIDs,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return nil
		})
	}
}

func bearerOrCookieToken(r *http.Request) string {
	if cookie, err := r.Cookie("access_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	const prefix = "Bearer "
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) > len(prefix) && authHeader[:len(prefix)] == prefix {
		return authHeader[len(prefix):]
	}
	return ""
}
