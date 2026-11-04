// Package messaging is the Go analogue of order-service's
// pkg/messaging/message-broker.interface.ts — a broker-agnostic interface
// plus one amqp091-go implementation. No app types, no env.
package messaging

// Message is the broker-agnostic envelope handed to a Handler. It mirrors
// order-service's ConsumeMessage: callers must explicitly Ack or Nack.
type Message struct {
	RoutingKey string
	Body       []byte
	Ack        func()
	Nack       func(requeue bool)
}

// Handler processes one Message. Returning an error nacks it (no requeue by
// default) so it is routed to the dead-letter exchange, if one is configured.
type Handler func(msg Message) error

// Topology is the Go analogue of ConsumerOptions — the exchange/queue/bindings
// a consumer needs declared before it can consume.
type Topology struct {
	Exchange           string
	Queue              string
	BindingKeys        []string
	DeadLetterExchange string
	DeadLetterQueue    string
	Prefetch           int
}

// Broker is the Go analogue of IMessageBroker.
type Broker interface {
	Connect() error
	Close() error
	DeclareTopology(topology Topology) error
	DeclareExchange(exchange string) error
	Consume(topology Topology, handler Handler) error
	Publish(exchange, routingKey string, body []byte) error
}
