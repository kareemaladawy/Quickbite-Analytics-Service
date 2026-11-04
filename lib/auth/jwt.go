package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

type accessTokenClaims struct {
	UserID         int64      `json:"userId"`
	Role           SystemRole `json:"role"`
	Email          string     `json:"email"`
	RestaurantID   *int64     `json:"restaurantId,omitempty"`
	RestaurantRole *string    `json:"restaurantRole,omitempty"`
	BranchIDs      []int64    `json:"branchIds,omitempty"`
	jwt.RegisteredClaims
}

// VerifyAccessToken is the Go analogue of order-service's lib/auth/jwt.ts
// verifyAccessToken — same secret, same HS256, same claim shape.
func VerifyAccessToken(token, secret string) (Claims, error) {
	parsed := &accessTokenClaims{}
	_, err := jwt.ParseWithClaims(token, parsed, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil {
		return Claims{}, ErrNotAuthenticated
	}

	return Claims{
		UserID:         parsed.UserID,
		Role:           parsed.Role,
		Email:          parsed.Email,
		RestaurantID:   parsed.RestaurantID,
		RestaurantRole: parsed.RestaurantRole,
		BranchIDs:      parsed.BranchIDs,
	}, nil
}
