// Package rbac is the Go analogue of order-service's
// lib/auth/permission-cache.service.ts: an in-process, TTL'd read-through
// cache in front of core-service's permission catalog. There is no Redis
// here (this service has no cache dependency elsewhere, so a sync.Map with
// per-entry expiry is the simplest thing that works) — invalidation by the
// rbac.permissions_changed event is homework (see docs/implementation-plan.md).
package rbac

import (
	"sync"
	"time"
)

type cacheEntry struct {
	permissions []string
	expiresAt   time.Time
}

type Cache struct {
	fetcher PermissionsFetcher
	ttl     time.Duration

	mu      sync.RWMutex
	entries map[string]cacheEntry
}

func NewCache(fetcher PermissionsFetcher, ttl time.Duration) *Cache {
	return &Cache{
		fetcher: fetcher,
		ttl:     ttl,
		entries: make(map[string]cacheEntry),
	}
}

func (c *Cache) HasPermission(role, permission string) (bool, error) {
	permissions, err := c.getPermissions(role)
	if err != nil {
		return false, err
	}
	for _, p := range permissions {
		if p == permission {
			return true, nil
		}
	}
	return false, nil
}

func (c *Cache) getPermissions(role string) ([]string, error) {
	c.mu.RLock()
	entry, ok := c.entries[role]
	c.mu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		return entry.permissions, nil
	}

	permissions, err := c.fetcher.FetchPermissions(role)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.entries[role] = cacheEntry{permissions: permissions, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()

	return permissions, nil
}

// Invalidate drops one role's cached entry — called per role named in a
// rbac.permissions_changed event's payload (see events.go).
func (c *Cache) Invalidate(role string) {
	c.mu.Lock()
	delete(c.entries, role)
	c.mu.Unlock()
}

// InvalidateAll drops every cached entry — the fallback events.go takes
// when a rbac.permissions_changed event names no specific roles (mirrors
// order-service's PermissionCacheService.invalidateAll, which the Node
// RbacEventsController always calls regardless of payload).
func (c *Cache) InvalidateAll() {
	c.mu.Lock()
	c.entries = make(map[string]cacheEntry)
	c.mu.Unlock()
}
