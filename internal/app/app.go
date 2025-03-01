package app

import (
	"context"
	"log/slog"

	"github.com/streadway/amqp"
	"github.com/vladislavprovich/user-info/internal/rabbitmq"
	"github.com/vladislavprovich/user-info/internal/repository"
	"github.com/vladislavprovich/user-info/internal/repository/storage"

	"github.com/vladislavprovich/user-info/config"
	grpcapp "github.com/vladislavprovich/user-info/internal/app/grpc"
	"go.opentelemetry.io/otel/trace"
)

type App struct {
	GRPCSrv   *grpcapp.App
	RMQConn   *amqp.Connection
	Publisher *rabbitmq.Publisher
	Consumer  *rabbitmq.Consumer
}

func New(
	ctx context.Context,
	log *slog.Logger,
	cfg *config.Config,
	trace trace.TracerProvider,
) *App {
	// Connect MongoDB.
	mongoFactory, err := repository.NewMongo(cfg.MongoDB)
	if err != nil {
		log.ErrorContext(ctx, "Failed to connect to MongoDB", slog.Any("error", err))
		panic(err)
	}

	db := storage.NewMongoStorage(mongoFactory.DB, log)

	// Connect RabbitMQ.
	conn, err := rabbitmq.NewRabbitMQ(ctx, cfg)
	if err != nil {
		log.ErrorContext(ctx, "Error creating RabbitMQ connection", slog.Any("error", err))
		panic(err)
	}

	// Connect Publisher.
	publisher, err := rabbitmq.NewPublisher(conn, cfg.Rabbit.ExchangeName)
	if err != nil {
		log.ErrorContext(ctx, "Error creating RabbitMQ Publisher", slog.Any("error", err))
		panic(err)
	}

	// Connect Consumer.
	consumer, err := rabbitmq.NewConsumer(conn, cfg.Rabbit.QueueName, cfg.Rabbit.CacheTTL)
	if err != nil {
		log.ErrorContext(ctx, "Error creating RabbitMQ Consumer", slog.Any("error", err))
		panic(err)
	}

	// Start Consumer on gorutine.
	go consumer.StartConsumer(ctx, db)

	// Init gRPC-server.
	grpcApp := grpcapp.New(log, cfg.GRPC.PortGRPC, trace.Tracer(cfg.Tracing.NameSpase), db)

	return &App{
		GRPCSrv:   grpcApp,
		RMQConn:   conn,
		Publisher: publisher,
		Consumer:  consumer,
	}
}
