package app

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"

	"github.com/streadway/amqp"
	"github.com/vladislavprovich/UserInfo/internal/rabbitmq"

	"github.com/vladislavprovich/UserInfo/config"
	grpcapp "github.com/vladislavprovich/UserInfo/internal/app/grpc"
	"github.com/vladislavprovich/UserInfo/internal/storage"
	"go.opentelemetry.io/otel/trace"
)

type App struct {
	GRPCSrv *grpcapp.App
	RMQConn *amqp.Connection
}

func New(
	log *slog.Logger,
	cfg *config.Config,
	trace trace.TracerProvider,
) *App {
	hostAndPort := net.JoinHostPort(cfg.MongoDB.Host, strconv.Itoa(cfg.MongoDB.Port))
	mongoURL := fmt.Sprintf("mongodb://%s:%s@%s/%s?authSource=%s",
		cfg.MongoDB.User,
		cfg.MongoDB.Password,
		hostAndPort,
		cfg.MongoDB.Database,
		cfg.MongoDB.AuthSource,
	)

	db, err := storage.NewMongoDB(mongoURL, cfg.MongoDB.Database)
	if err != nil {
		log.Error("Error creating MongoDB storage", "error", err)
		panic(err)
	}

	conn, ch, err := rabbitmq.NewRabbitMQ()
	if err != nil {
		log.Error("Error creating RabbitMQ connection", "error", err)
		panic(err)
	}

	go rabbitmq.StartConsumer(ch, db)

	grpcApp := grpcapp.New(log, cfg.GRPC.PortGRPC, trace.Tracer(cfg.Tracing.NameSpase), db)

	return &App{
		GRPCSrv: grpcApp,
		RMQConn: conn,
	}
}

func (a *App) Shutdown() {
	if a.RMQConn != nil {
		defer func() {
			err := a.RMQConn.Close()
			if err != nil {
				panic(err)
			}
		}()
	}
}
