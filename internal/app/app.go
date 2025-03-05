package app

import (
	"context"
	"log/slog"

	"github.com/vladislavprovich/user-info/internal/repository/storage"

	"github.com/vladislavprovich/user-info/config"
	grpcapp "github.com/vladislavprovich/user-info/internal/app/grpc"
	"go.opentelemetry.io/otel/trace"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(
	ctx context.Context,
	log *slog.Logger,
	cfg *config.Config,
	trace trace.TracerProvider,
) *App {
	// Connect MongoDB.
	mongoFactory, err := storage.NewMongo(cfg.MongoDB)
	if err != nil {
		log.ErrorContext(ctx, "Failed to connect to MongoDB", slog.Any("error", err))
		panic(err)
	}

	db := storage.NewMongoStorage(mongoFactory.DB, log)

	// Init gRPC-server.
	grpcApp := grpcapp.New(log, cfg.GRPC.PortGRPC, trace.Tracer(cfg.Tracing.NameSpase), db)

	return &App{
		GRPCSrv: grpcApp,
	}
}
