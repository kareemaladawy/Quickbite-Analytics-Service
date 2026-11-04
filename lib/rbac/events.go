// rbac.permissions_changed consumer wiring (docs/implementation-plan.md
// Phase 10). This lives in lib/, not app/analytics/eventhandlers, because
// cache invalidation is cross-cutting infra (every module's RBAC check
// shares one Cache), not an analytics concept — the Go analogue of
// order-service's lib/auth/rbac.events.ts registering against its own
// lib/core-events/consumer.ts rather than any one app module's handler map.
package rbac

import (
	"encoding/json"
	"fmt"

	"quickbite/analytics-service/lib/coreevents"
)

// EventTypePermissionsChanged is the routing key core-service publishes to
// its `core.events` exchange — see core-service's CoreEventType enum
// (lib/events/event-types.ts).
const EventTypePermissionsChanged = "rbac.permissions_changed"

// permissionsChangedPayload mirrors core-service's migration-published
// payload shape ({ roles: string[] }, see
// 20261017090300_publish_rbac_permissions_changed.ts). An empty/absent
// roles list means "assume everything may have changed" — the same
// conservative stance order-service's own handler takes (it calls
// invalidateAll() unconditionally, ignoring the payload).
type permissionsChangedPayload struct {
	Roles []string `json:"roles"`
}

// RegisterInvalidationHandler wires the one handler this consumer needs
// onto a *separate* coreevents.Consumer bound to core-service's
// `core.events` exchange — not the order-events consumer, since
// rbac.permissions_changed is published on a different exchange entirely
// (confirmed against core-service/order-service's own
// RABBITMQ_CORE_EVENTS_EXCHANGE, distinct from RABBITMQ_ORDER_EVENTS_EXCHANGE).
// A dedicated queue also keeps failure domains separate: a malformed RBAC
// payload nacks onto analytics-service.core-events.dlq, never contending
// with the order-events queue's own DLQ.
func RegisterInvalidationHandler(consumer *coreevents.Consumer, cache *Cache) {
	consumer.RegisterHandler(EventTypePermissionsChanged, func(payload json.RawMessage) error {
		var decoded permissionsChangedPayload
		if err := json.Unmarshal(payload, &decoded); err != nil {
			return fmt.Errorf("rbac: decode rbac.permissions_changed payload: %w", err)
		}

		if len(decoded.Roles) == 0 {
			cache.InvalidateAll()
			return nil
		}
		for _, role := range decoded.Roles {
			cache.Invalidate(role)
		}
		return nil
	})
}
