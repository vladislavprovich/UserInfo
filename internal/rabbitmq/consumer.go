package rabbitmq

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/streadway/amqp"
	"github.com/vladislavprovich/user-info/internal/models"
	"github.com/vladislavprovich/user-info/internal/repository/mongo_models"
	"github.com/vladislavprovich/user-info/internal/repository/storage"
)

const recordAttemptsDatabase = 5

// Consumer is responsible for processing messages from the queue.
type Consumer struct {
	Channel  *amqp.Channel // Channel RabbitMQ.
	Queue    string        // Queue name.
	cache    sync.Map      // Local cache for message uniqueness.
	cacheTTL time.Duration // The lifetime of a cache entry.
	log      *slog.Logger
}

// NewConsumer creates a new consumer and initializes the cache.
func NewConsumer(conn *amqp.Connection, queue string, cacheTTL time.Duration) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Declare the queue if it doesn't exist yet.
	_, err = ch.QueueDeclare(
		queue, // Queue name.
		true,  // Durable (remains after restart).
		false, // Auto-deleted
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		Channel:  ch,
		Queue:    queue,
		cacheTTL: cacheTTL,
	}, nil
}

// isDuplicate checks whether the message has already been processed through the cache.
func (c *Consumer) isDuplicate(messageID string) bool {
	_, exists := c.cache.Load(messageID)
	return exists
}

// addToCache adds the ID of the processed message to the cache for a specified time.
func (c *Consumer) addToCache(messageID string) {
	c.cache.Store(messageID, struct{}{})
	go func() {
		time.Sleep(c.cacheTTL)
		c.cache.Delete(messageID)
	}()
}

// StartConsumer starts message processing.
func (c *Consumer) StartConsumer(ctx context.Context, storage storage.UserStorage) {
	// We consume messages from the queue.
	msgs, err := c.Channel.Consume(
		c.Queue, // The name of the queue from which we consume messages.
		"",      // Consumer Tag (if empty, RabbitMQ generates it automatically).
		false,   // Auto-Ack: false (turn on manual confirmation of message processing).
		false,   // Exclusive: false (allows several consumers to listen to one queue).
		false,   // No-Local: false (the consumer can receive its own messages, if true - no).
		false,   // No-Wait: false (we wait for a response from the broker before starting consumption).
		nil,     // Arguments: nil (additional parameters can be passed, they are not needed here).
	)
	if err != nil {
		c.log.ErrorContext(ctx, "Error consuming messages")
	}

	go func() {
		for d := range msgs {
			messageID := d.MessageId
			if messageID == "" {
				c.log.WarnContext(ctx, "Received message without MessageId")
				_ = d.Nack(false, false) // We reject messages without an ID.
				continue
			}

			// Check if it's not a duplicate via cache.
			if c.isDuplicate(messageID) {
				c.log.WarnContext(ctx, "Duplicate message detected in cache", slog.String("message_id", messageID))
				_ = d.Ack(false) // Confirm that the message has been processed.
				continue
			}

			var userPayload models.User
			err = json.Unmarshal(d.Body, &userPayload)
			if err != nil {
				c.log.ErrorContext(ctx, "Error parsing message", slog.Any("error", err))
				_ = d.Nack(false, false) // We reject the message without redelivery.
				continue
			}

			// Convert to a DB model.
			user := &mongo_models.User{
				UserID:    userPayload.UserID,
				Email:     userPayload.Email,
				Password:  userPayload.Password,
				CreatedAt: userPayload.CreatedAt,
				UpdatedAt: userPayload.UpdatedAt,
			}

			// We make several attempts to write to the database (up to 5 times).
			for attempt := 0; attempt < recordAttemptsDatabase; attempt++ {
				err = storage.SaveUser(ctx, user)
				if err == nil {
					c.log.InfoContext(ctx, "User successfully saved", slog.String("user_id", user.UserID))
					c.addToCache(messageID) // Add the ID to the cache.
					_ = d.Ack(false)
					break
				}
				c.log.ErrorContext(ctx, "error saving user",
					slog.Int("attempt+1:", attempt+1),
					slog.Any("error", err))
			}

			if err != nil {
				c.log.ErrorContext(ctx, "failed to save user after attempts:",
					slog.String("user_id", user.UserID),
					slog.Any("error", err))

				_ = d.Nack(false, true) // We send a message for redelivery
			}
		}
	}()

	c.log.InfoContext(ctx, "Consumer started for queue:", slog.Any("S", c.Queue))
}
