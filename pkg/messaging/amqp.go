package messaging

import (
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// AMQPConfig is the Go analogue of RabbitMQConfig in both Node services.
type AMQPConfig struct {
	URL string
}

// AMQPBroker implements Broker over github.com/rabbitmq/amqp091-go. Mirrors
// core-service's rabbitmq.client.ts: one connection, one long-lived publish
// channel, throwaway channels for topology declaration, a dedicated channel
// per consumer. Unlike order-service's amqp-connection-manager-based client,
// there is no automatic reconnect loop here — the Go ecosystem has no
// equivalent off-the-shelf wrapper, and connection.NotifyClose is logged by
// the caller (lib/boot) so a crash-and-restart (the process supervisor's job)
// is the recovery path, same tradeoff core-service accepts.
type AMQPBroker struct {
	config AMQPConfig

	mu    sync.Mutex
	conn  *amqp.Connection
	pubCh *amqp.Channel
}

func NewAMQPBroker(config AMQPConfig) *AMQPBroker {
	return &AMQPBroker{config: config}
}

func (b *AMQPBroker) Connect() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.connectLocked()
}

func (b *AMQPBroker) connectLocked() error {
	if b.conn != nil && !b.conn.IsClosed() {
		return nil
	}

	conn, err := amqp.Dial(b.config.URL)
	if err != nil {
		return fmt.Errorf("messaging: dial failed: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("messaging: open publish channel failed: %w", err)
	}
	if err := ch.Confirm(false); err != nil {
		conn.Close()
		return fmt.Errorf("messaging: enable publisher confirms failed: %w", err)
	}

	b.conn = conn
	b.pubCh = ch
	return nil
}

// NotifyClose exposes the underlying connection's close notification so
// lib/boot can log an unrecoverable broker disconnect.
func (b *AMQPBroker) NotifyClose() <-chan *amqp.Error {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan *amqp.Error, 1)
	if b.conn != nil {
		b.conn.NotifyClose(ch)
	}
	return ch
}

func (b *AMQPBroker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.pubCh != nil {
		_ = b.pubCh.Close()
	}
	if b.conn != nil {
		return b.conn.Close()
	}
	return nil
}

func (b *AMQPBroker) DeclareExchange(exchange string) error {
	if err := b.Connect(); err != nil {
		return err
	}
	ch, err := b.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	return ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil)
}

func (b *AMQPBroker) DeclareTopology(topology Topology) error {
	if err := b.Connect(); err != nil {
		return err
	}
	ch, err := b.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	return assertTopology(ch, topology)
}

func (b *AMQPBroker) Publish(exchange, routingKey string, body []byte) error {
	if err := b.Connect(); err != nil {
		return err
	}
	confirm, pubErr := b.pubCh.PublishWithDeferredConfirm(
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if pubErr != nil {
		return fmt.Errorf("messaging: publish failed: %w", pubErr)
	}
	ok := confirm.Wait()
	if !ok {
		return fmt.Errorf("messaging: broker nacked publish to %s/%s", exchange, routingKey)
	}
	return nil
}

// Consume opens a dedicated channel, asserts topology, sets prefetch, and
// dispatches each delivery to handler. Ack on nil error, Nack (no requeue —
// routes to the DLQ when one is configured) otherwise. Blocks for the
// lifetime of the deliveries channel; callers run it in a goroutine.
func (b *AMQPBroker) Consume(topology Topology, handler Handler) error {
	if err := b.Connect(); err != nil {
		return err
	}
	ch, err := b.conn.Channel()
	if err != nil {
		return err
	}
	if err := assertTopology(ch, topology); err != nil {
		ch.Close()
		return err
	}
	if err := ch.Qos(topology.Prefetch, 0, false); err != nil {
		ch.Close()
		return err
	}

	deliveries, err := ch.Consume(topology.Queue, "", false, false, false, false, nil)
	if err != nil {
		ch.Close()
		return err
	}

	go func() {
		for d := range deliveries {
			delivery := d
			msg := Message{
				RoutingKey: delivery.RoutingKey,
				Body:       delivery.Body,
				Ack:        func() { _ = delivery.Ack(false) },
				Nack:       func(requeue bool) { _ = delivery.Nack(false, requeue) },
			}
			if err := handler(msg); err != nil {
				msg.Nack(false)
				continue
			}
		}
	}()

	return nil
}

func assertTopology(ch *amqp.Channel, topology Topology) error {
	if err := ch.ExchangeDeclare(topology.Exchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}

	args := amqp.Table{}
	if topology.DeadLetterExchange != "" && topology.DeadLetterQueue != "" {
		if err := ch.ExchangeDeclare(topology.DeadLetterExchange, "topic", true, false, false, false, nil); err != nil {
			return err
		}
		if _, err := ch.QueueDeclare(topology.DeadLetterQueue, true, false, false, false, nil); err != nil {
			return err
		}
		if err := ch.QueueBind(topology.DeadLetterQueue, "#", topology.DeadLetterExchange, false, nil); err != nil {
			return err
		}
		args["x-dead-letter-exchange"] = topology.DeadLetterExchange
	}

	if _, err := ch.QueueDeclare(topology.Queue, true, false, false, false, args); err != nil {
		return err
	}
	for _, key := range topology.BindingKeys {
		if err := ch.QueueBind(topology.Queue, key, topology.Exchange, false, nil); err != nil {
			return err
		}
	}
	return nil
}
