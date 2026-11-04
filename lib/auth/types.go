package auth

// SystemRole mirrors core/order's fixed system roles carried in the JWT's
// `role` claim.
type SystemRole string

const (
	SystemRoleSystemAdmin    SystemRole = "system_admin"
	SystemRoleRestaurantUser SystemRole = "restaurant_user"
	SystemRoleDeliveryAgent  SystemRole = "delivery_agent"
	SystemRoleCustomer       SystemRole = "customer"
)

// Claims is the decoded access-token payload, same contract as core/order's
// JWTPayload: userId, role, restaurantId?, restaurantRole?, branchIds?.
type Claims struct {
	UserID         int64      `json:"userId"`
	Role           SystemRole `json:"role"`
	Email          string     `json:"email"`
	RestaurantID   *int64     `json:"restaurantId,omitempty"`
	RestaurantRole *string    `json:"restaurantRole,omitempty"`
	BranchIDs      []int64    `json:"branchIds,omitempty"`
}
