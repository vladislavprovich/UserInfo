package app

import (
	"github.com/vladislavprovich/UserInfo/config"
	grpcapp "github.com/vladislavprovich/UserInfo/internal/app/grpc"
	"github.com/vladislavprovich/UserInfo/internal/storage"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(
	log *slog.Logger,
	cfg *config.Config,
	trace trace.TracerProvider,
) *App {
	// Підключаємося до MongoDB
	storage, err := storage.NewMongoDB(cfg.MongoDB.URI, cfg.MongoDB.Database)
	if err != nil {
		log.Error("Error creating MongoDB storage", "error", err)
		panic(err)
	}

	// Створюємо сервіс користувачів
	userService := user.New(log, storage)

	// Піднімаємо gRPC сервер
	grpcApp := grpcapp.New(log, userService, cfg.GRPC.Port, trace.Tracer(cfg.Tracing.Namespace))

	return &App{
		GRPCSrv: grpcApp,
	}
}
