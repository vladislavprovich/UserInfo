package main

import (
	"context"
	"io"
	log2 "log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/vladislavprovich/user-info/config"
	"github.com/vladislavprovich/user-info/internal/app/consumer"
	"github.com/vladislavprovich/user-info/pkg/logger/slogpretty"
	"github.com/vladislavprovich/user-info/pkg/telemetry"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := setupLogger(ctx, cfg)

	// Init logs directory.
	err := telemetry.EnsureLogDir(cfg.Logging.LogDir)
	if err != nil {
		log.ErrorContext(ctx, "failed to ensure log dir",
			slog.String("dir", cfg.Logging.LogDir),
			slog.String("error ", err.Error()))
	}

	log.InfoContext(ctx, "starting consumer application",
		slog.String("env", cfg.Logger.Env),
	)

	_, err = telemetry.InitMetrics(ctx, log, cfg)
	if err != nil {
		log2.Fatalf("failed to init metrics: %v", err)
	}

	tracerProvider, err := telemetry.InitTracing(ctx, cfg.Otel.Endpoint, log)
	if err != nil {
		log2.Fatalf("failed to init tracing: %v", err)
	}
	defer func() {
		if err = tracerProvider.Shutdown(context.Background()); err != nil {
			log.ErrorContext(ctx, "failed to shutdown tracer", slog.String("error", err.Error()))
		}
	}()

	consumerApp := consumer.New(ctx, log, cfg)

	// Run consumer.
	go consumerApp.Run(ctx, consumerApp.Storage)

	// Graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	sign := <-stop
	log.InfoContext(ctx, "stopping consumer application", slog.String("signal", sign.String()))
	consumerApp.Stop(ctx)
	log.InfoContext(ctx, "consumer application stopped")
}

func setupLogger(ctx context.Context, cfg *config.Config) *slog.Logger {
	var log *slog.Logger
	logFilePath := filepath.Join(cfg.Logging.LogDir, "consumer.log")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.ErrorContext(ctx, "failed to open log file",
			slog.String("logFilePath", logFilePath),
			slog.String("error", err.Error()))
		os.Exit(1)
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	switch cfg.Logger.Env {
	case envLocal:
		prettyHandler := slogpretty.PrettyHandlerOptions{
			SlogOpts: &slog.HandlerOptions{Level: slog.LevelDebug},
		}.NewPrettyHandler(multiWriter)
		log = slog.New(prettyHandler)
	case envDev, envProd:
		log = slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{Level: slog.LevelDebug}))
	default:
		log = slog.New(slog.NewTextHandler(multiWriter, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	return log
}
