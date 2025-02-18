package app

import (
	"fmt"
	"github.com/vladislavprovich/UserInfo/config"
	grpcapp "github.com/vladislavprovich/UserInfo/internal/app/grpc"
	"github.com/vladislavprovich/UserInfo/internal/storage"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
	"net"
	"strconv"
)

type App struct {
	GRPCSrv *grpcapp.App
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

	// Підключаємося до MongoDB
	storage, err := storage.NewMongoDB(mongoURL, cfg.MongoDB.Database)
	if err != nil {
		log.Error("Error creating MongoDB storage", "error", err)
		panic(err)
	}

	// Піднімаємо gRPC сервер
	grpcApp := grpcapp.New(log, cfg.GRPC.PortGRPC, trace.Tracer(cfg.Tracing.NameSpase), storage)

	return &App{
		GRPCSrv: grpcApp,
	}
}
