package coreevents

import "encoding/json"

// Envelope mirrors order-service's lib/events/types.ts OutboxEventEnvelope —
// the wire shape order-service's outbox drain publishes:
// {eventId, eventType, occurredAt, payload}.
type Envelope struct {
	EventID    string          `json:"eventId"`
	EventType  string          `json:"eventType"`
	OccurredAt string          `json:"occurredAt"`
	Payload    json.RawMessage `json:"payload"`
}

// EventHandler processes one decoded event payload. Returning an error nacks
// the message (routes to the DLQ) rather than acking — mirrors
// order-service's core-events/types.ts CoreEventHandler contract.
type EventHandler func(payload json.RawMessage) error

// EventDeduper is the "have I seen this event before?" capability the
// consumer needs. Defined here (lib/) rather than depending on
// app/analytics/repository directly — repository.EventIDsRepo satisfies this
// implicitly via Go duck typing, inverting the dependency the brief
// describes (lib/ may not import app/).
type EventDeduper interface {
	// MarkSeen returns true if this is the first time eventID has been
	// seen (and records it), false if it's a replay.
	MarkSeen(eventID string) (bool, error)
}
