package rabbitmq

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/vladislavprovich/user-info/config"

	"github.com/streadway/amqp"
	"github.com/vladislavprovich/user-info/internal/models"
	"github.com/vladislavprovich/user-info/internal/repository/mongomodels"
	"github.com/vladislavprovich/user-info/internal/repository/storage"
)

const recordAttemptsDatabase = 5

// Consumer is responsible for processing messages from the queue.
type Consumer struct {
	Channel   *amqp.Channel // Channel RabbitMQ.
	QueueName string        // Settings in the config.
	cache     sync.Map      // Local cache for message uniqueness.
	log       *slog.Logger
	CacheTTL  time.Duration
}

// NewConsumer creates a new consumer and initializes the cache.
func NewConsumer(conn *amqp.Connection, cfg *config.Config, log *slog.Logger) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	// Declare the queue if it doesn't exist yet.
	_, err = ch.QueueDeclare(
		cfg.Rabbit.QueueName,  // Queue name.
		cfg.Rabbit.Durable,    // Durable (remains after restart).
		cfg.Rabbit.AutoDelete, // Auto-deleted
		cfg.Rabbit.Exclusive,  // Exclusive
		cfg.Rabbit.NoWait,     // No-wait
		nil,                   // Arguments
	)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		Channel:   ch,
		QueueName: cfg.Rabbit.QueueName,
		CacheTTL:  cfg.Rabbit.CacheTTL,
		log:       log,
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
		time.Sleep(c.CacheTTL)
		c.cache.Delete(messageID)
	}()
}

// StartConsumer starts message processing.
func (c *Consumer) StartConsumer(ctx context.Context, cfg *config.Config, storage storage.UserStorage) {
	// Consume messages from the queue.
	msgs, err := c.Channel.Consume(
		cfg.Rabbit.QueueName,   // Queue name.
		cfg.Rabbit.ConsumerTag, // Consumer Tag (empty means RabbitMQ generates it automatically).
		cfg.Rabbit.AutoAck,     // Auto-Ack: false (manual message acknowledgment).
		cfg.Rabbit.Exclusive,   // Exclusive: false (allows multiple consumers on the same queue).
		cfg.Rabbit.NoLocal,     // No-Local: false (consumer can receive its own messages).
		cfg.Rabbit.NoWait,      // No-Wait: false (wait for broker response before starting consumption).
		nil,                    // Arguments: nil (no additional parameters needed).
	)
	if err != nil {
		c.log.ErrorContext(ctx, "Error consuming messages", slog.Any("error", err))
		return
	}

	go func() {
		for d := range msgs {
			c.processMessage(ctx, d, storage)
		}
	}()

	c.log.InfoContext(ctx, "Consumer started", slog.String("queue", c.QueueName))
}

// processMessage handles the processing of a single message.
func (c *Consumer) processMessage(ctx context.Context, d amqp.Delivery, storage storage.UserStorage) {
	messageID := d.MessageId
	if messageID == "" {
		c.log.WarnContext(ctx, "Received message without MessageId")
		_ = d.Nack(false, false) // Reject messages without an ID.
		return
	}

	// Check for duplicates using cache.
	if c.isDuplicate(messageID) {
		c.log.WarnContext(ctx, "Duplicate message detected in cache", slog.String("message_id", messageID))
		_ = d.Ack(false) // Confirm message as processed.
		return
	}

	userPayload, err := c.parseMessage(ctx, d.Body)
	if err != nil {
		_ = d.Nack(false, false) // Reject message without redelivery.
		return
	}

	user := c.convertToDBModel(userPayload)

	// Attempt to save the user to the database.
	if c.saveUserWithRetries(ctx, user, storage) {
		c.addToCache(messageID) // Add message ID to cache.
		_ = d.Ack(false)
	} else {
		_ = d.Nack(false, true) // Send message for redelivery.
	}
}

// parseMessage unmarshals the message body into a User struct.
func (c *Consumer) parseMessage(ctx context.Context, body []byte) (models.User, error) {
	var userPayload models.User
	err := json.Unmarshal(body, &userPayload)
	if err != nil {
		c.log.ErrorContext(ctx, "Error parsing message", slog.Any("error", err))
	}
	return userPayload, err
}

// convertToDBModel converts the incoming user model to the database model.
func (c *Consumer) convertToDBModel(userPayload models.User) *mongomodels.User {
	return &mongomodels.User{
		UserID:    userPayload.UserID,
		Email:     userPayload.Email,
		Password:  userPayload.Password,
		CreatedAt: userPayload.CreatedAt,
		UpdatedAt: userPayload.UpdatedAt,
	}
}

// saveUserWithRetries attempts to save the user to the database with retries.
func (c *Consumer) saveUserWithRetries(ctx context.Context, user *mongomodels.User, storage storage.UserStorage) bool {
	for attempt := range recordAttemptsDatabase {
		err := storage.SaveUser(ctx, user)
		if err == nil {
			c.log.InfoContext(ctx, "User successfully saved", slog.String("user_id", user.UserID))
			return true
		}

		c.log.ErrorContext(ctx, "Error saving user",
			slog.Int("attempt", attempt+1),
			slog.Any("error", err))
	}
	c.log.ErrorContext(ctx, "Failed to save user after multiple attempts",
		slog.String("user_id", user.UserID))
	return false
}
