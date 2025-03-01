package app

import (
	"context"
	"github.com/streadway/amqp"
	"github.com/vladislavprovich/user-info/internal/rabbitmq"
	"github.com/vladislavprovich/user-info/internal/repository"
	"github.com/vladislavprovich/user-info/internal/repository/storage"
	"log/slog"

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
		log.Error("Failed to connect to MongoDB", "error", err)
		panic(err)
	}

	db := storage.NewMongoStorage(mongoFactory.DB, log)

	// Connect RabbitMQ.
	conn, err := rabbitmq.NewRabbitMQ(ctx, cfg)
	if err != nil {
		log.Error("Error creating RabbitMQ connection", "error", err)
		panic(err)
	}

	// Connect Publisher.
	publisher, err := rabbitmq.NewPublisher(conn, cfg.Rabbit.ExchangeName)
	if err != nil {
		log.Error("Error creating RabbitMQ Publisher", "error", err)
		panic(err)
	}

	// Connect Consumer.
	consumer, err := rabbitmq.NewConsumer(conn, cfg.Rabbit.QueueName)
	if err != nil {
		log.Error("Error creating RabbitMQ Consumer", "error", err)
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
