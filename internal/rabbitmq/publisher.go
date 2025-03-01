package rabbitmq

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/streadway/amqp"
)

// Publisher responsible for publishing messages to RabbitMQ.
type Publisher struct {
	Channel  *amqp.Channel // RabbitMQ channel for communication.
	Exchange string        // Exchange name to publish to.
	log      *slog.Logger
}

// NewPublisher creates a new publisher and configures delivery confirmation.
func NewPublisher(conn *amqp.Connection, exchange string) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Enable confirm mode to confirm message delivery.
	err = ch.Confirm(false)
	if err != nil {
		return nil, err
	}

	// Declare Exchange if it doesn't exist yet.
	err = ch.ExchangeDeclare(
		exchange, // Name Exchange.
		"direct", // Exchange type (direct – sends messages to specific queues).
		true,     // Durable (remains after reboot).
		false,    // Auto-deleted
		false,    // Internal
		false,    // No-wait
		nil,      // Arguments
	)
	if err != nil {
		return nil, err
	}

	return &Publisher{Channel: ch, Exchange: exchange}, nil
}

// PublishMessage publishes a delivery confirmation message.
func (p *Publisher) PublishMessage(routingKey string, message interface{}, messageID string) error {
	// We serialize messages in JSON.
	body, err := json.Marshal(message)
	if err != nil {
		p.log.Error("Error marshalling message:", slog.Any("error", err))
		return err
	}

	// Channel for receiving delivery confirmation.
	confirm := p.Channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	// We publish messages in the queue.
	err = p.Channel.Publish(
		p.Exchange, // Exchange
		routingKey, // Routing key
		false,      // Mandatory
		false,      // Immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // The message is saved after the broker is down.
			MessageId:    messageID,       // Unique ID for handling duplicates.
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		p.log.Error("Error publishing message:", slog.Any("error", err))
		return err
	}

	// We are waiting for delivery confirmation.
	if confirmed := <-confirm; !confirmed.Ack {
		p.log.Warn("Failed to confirm message delivery")
		return err
	}

	p.log.Info("Message published:", slog.String("body_msg", string(body)))
	return nil
}
