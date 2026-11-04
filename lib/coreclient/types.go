package coreclient

// CoreEnvelope mirrors core-service's generic {success, data} response
// envelope — the Go analogue of order-service's lib/core-client/types.ts
// CoreEnvelope<T>.
type CoreEnvelope[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

// RolePermissionsResponse mirrors core's GET
// /api/v1/internal/rbac/permissions?role= response shape.
type RolePermissionsResponse struct {
	Permissions []string `json:"permissions"`
}
