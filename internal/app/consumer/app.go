package consumer

import (
	"context"
	"log/slog"
	"os"

	"github.com/streadway/amqp"
	"github.com/vladislavprovich/user-info/config"
	"github.com/vladislavprovich/user-info/internal/rabbitmq"
	"github.com/vladislavprovich/user-info/internal/repository/storage"
)

type AppConsumer struct {
	Consumer *rabbitmq.Consumer
	RMQConn  *amqp.Connection
	Storage  storage.UserStorage
	Log      *slog.Logger
	cfg      *config.Config
}

func New(ctx context.Context, log *slog.Logger, cfg *config.Config) *AppConsumer {
	// Connect MongoDB.
	mongoFactory, err := storage.NewMongo(cfg.MongoDB)
	if err != nil {
		log.ErrorContext(ctx, "Failed to connect to MongoDB", slog.Any("error", err))
		panic(err)
	}

	db := storage.NewMongoStorage(mongoFactory.DB, log)

	// Connect RabbitMQ.
	conn, err := rabbitmq.NewRabbitMQ(ctx, cfg)
	if err != nil {
		log.ErrorContext(ctx, "Failed to connect to RabbitMQ", slog.Any("error", err))
		panic(err)
	}

	// Connect consumer.
	consumer, err := rabbitmq.NewConsumer(conn, cfg, log)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create RabbitMQ Consumer", slog.Any("error", err))
		panic(err)
	}

	return &AppConsumer{
		Consumer: consumer,
		RMQConn:  conn,
		Log:      log,
		Storage:  db,
	}
}

func (c *AppConsumer) Run(ctx context.Context, storage storage.UserStorage) {
	c.Log.InfoContext(ctx, "Starting RabbitMQ consumer...")
	if c.Consumer == nil {
		c.Log.ErrorContext(ctx, "Consumer is nil")
		os.Exit(1)
	}

	c.Consumer.StartConsumer(ctx, c.cfg, storage)
}

func (c *AppConsumer) Stop(ctx context.Context) {
	c.Log.InfoContext(ctx, "Stopping RabbitMQ consumer...")

	if err := c.RMQConn.Close(); err != nil {
		c.Log.ErrorContext(ctx, "Failed to close RabbitMQ connection", slog.Any("error", err))
	}
}
