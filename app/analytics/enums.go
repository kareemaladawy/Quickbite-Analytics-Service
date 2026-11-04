package analytics

// PermAnalyticsRead is the permission namespace this module checks via
// rbac.Require — mirrors order-service's `orders:*`/`payments:*` namespacing
// convention, seeded in core-service's permission catalog (homework to add
// there; out of scope for this service).
const PermAnalyticsRead = "analytics:read"

// OrderEventType* mirror order-service's OrderEventType enum
// (lib/events/event-types.ts) for the routing keys this service consumes.
// All of these are already bound on analytics-service.order-events (see
// lib/boot) — adding a handler here needs no topology change, only a
// RegisterHandler call.
const (
	OrderEventTypePlaced    = "order.placed"
	OrderEventTypeRejected  = "order.rejected"
	OrderEventTypeDelivered = "order.delivered"
)

// PaymentEventTypeCompleted mirrors order-service's
// OrderEventType.PAYMENT_COMPLETED — also bound on analytics-service's
// order-events queue (bindings include `payment.#`).
const PaymentEventTypeCompleted = "payment.completed"
