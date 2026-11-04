// Package appcontext holds the context.Context keys this service attaches
// request-scoped values under — the Go analogue of order-service's
// lib/types/express.d.ts augmenting req.user / req.correlationId.
package appcontext

import "context"

type ctxKey int

const (
	claimsKey ctxKey = iota
	correlationIDKey
)

// Claims is the decoded JWT payload, same shape as core/order's JWTPayload.
type Claims struct {
	UserID         int64
	Role           string
	RestaurantID   *int64
	RestaurantRole *string
	BranchIDs      []int64
}

func WithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(Claims)
	return claims, ok
}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

func CorrelationID(ctx context.Context) string {
	id, _ := ctx.Value(correlationIDKey).(string)
	return id
}
