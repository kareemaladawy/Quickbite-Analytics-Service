// Package coreevents is the Go analogue of order-service's
// lib/core-events/consumer.ts, generalized to this service's naming
// (it consumes order-service's events, not core's, but the shape of the
// problem — dedupe, dispatch, DLQ-on-failure — is identical). Generic only:
// no knowledge of order.placed or any other concrete event type lives here.
package coreevents

import (
	"encoding/json"

	"quickbite/analytics-service/lib/logger"
	"quickbite/analytics-service/pkg/messaging"
)

type Consumer struct {
	broker   messaging.Broker
	deduper  EventDeduper
	topology messaging.Topology
	handlers map[string]EventHandler
}

func NewConsumer(broker messaging.Broker, deduper EventDeduper, topology messaging.Topology) *Consumer {
	return &Consumer{
		broker:   broker,
		deduper:  deduper,
		topology: topology,
		handlers: make(map[string]EventHandler),
	}
}

// RegisterHandler is the Go analogue of registerHandler — panics on a
// duplicate registration (a wiring bug caught at boot, same as the Node
// side's thrown Error).
func (c *Consumer) RegisterHandler(eventType string, handler EventHandler) {
	if _, exists := c.handlers[eventType]; exists {
		panic("coreevents: handler already registered for " + eventType)
	}
	c.handlers[eventType] = handler
}

// Start declares topology and begins consuming — the Go analogue of
// startCoreEventsConsumer.
func (c *Consumer) Start() error {
	if err := c.broker.DeclareTopology(c.topology); err != nil {
		return err
	}
	return c.broker.Consume(c.topology, c.handleMessage)
}

func (c *Consumer) handleMessage(msg messaging.Message) error {
	var envelope Envelope
	if err := json.Unmarshal(msg.Body, &envelope); err != nil || envelope.EventID == "" || envelope.EventType == "" {
		logger.Base().Warn("coreevents: malformed envelope, dropping", "routing_key", msg.RoutingKey)
		msg.Nack(false)
		return nil
	}

	fresh, err := c.deduper.MarkSeen(envelope.EventID)
	if err != nil {
		return err
	}
	if !fresh {
		msg.Ack()
		return nil
	}

	handler, ok := c.handlers[envelope.EventType]
	if !ok {
		logger.Base().Warn("coreevents: no handler, acking", "event_type", envelope.EventType, "event_id", envelope.EventID)
		msg.Ack()
		return nil
	}

	if err := handler(envelope.Payload); err != nil {
		logger.Base().Error("coreevents: handler failed, sending to DLQ",
			"event_type", envelope.EventType, "event_id", envelope.EventID, "error", err.Error())
		msg.Nack(false)
		return nil
	}

	msg.Ack()
	return nil
}
