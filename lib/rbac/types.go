package rbac

// PermissionsFetcher is the Go analogue of coreclient's RBAC call — an
// interface, not a concrete import, so Cache stays decoupled from
// lib/coreclient's HTTP details (same duck-typing inversion the brief
// describes for EventDeduper).
type PermissionsFetcher interface {
	FetchPermissions(role string) ([]string, error)
}
