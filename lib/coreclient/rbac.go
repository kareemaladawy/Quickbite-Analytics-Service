package coreclient

import (
	"context"
	"net/url"
)

// FetchPermissions satisfies lib/rbac.PermissionsFetcher — the sync HTTP
// call behind the read-through cache, mirroring order-service's
// permission-cache.service.ts hitting
// GET /api/v1/internal/rbac/permissions?role=.
func (c *Client) FetchPermissions(role string) ([]string, error) {
	var envelope CoreEnvelope[RolePermissionsResponse]
	path := "/api/v1/internal/rbac/permissions?role=" + url.QueryEscape(role)
	if err := c.get(context.Background(), path, &envelope); err != nil {
		return nil, err
	}
	return envelope.Data.Permissions, nil
}
