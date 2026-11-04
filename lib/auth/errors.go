package auth

import (
	"net/http"

	apperr "quickbite/analytics-service/lib/errors"
)

// Cross-cutting auth errors — not analytics-module-specific, so they live
// here rather than in app/analytics/errors.go (lib/ may not import app/).
var (
	ErrNotAuthenticated = apperr.New("UNAUTHENTICATED", "User not authenticated", http.StatusUnauthorized)
	ErrForbidden        = apperr.New("FORBIDDEN", "Permission denied", http.StatusForbidden)
)
