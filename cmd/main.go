package main

import (
	"context"
	"github.com/vladislavprovich/UserInfo/config"
	app "github.com/vladislavprovich/UserInfo/internal/app"

	"github.com/vladislavprovich/UserInfo/lib/telemetry"
	log2 "log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()
	ctx := context.Background()
	log := setupLogger(cfg)

	// Init logs directory.
	err := telemetry.EnsureLogDir(cfg.Logging.LogDir)
	if err != nil {
		log.Error("failed to ensure log dir",
			slog.String("dir", cfg.Logging.LogDir),
			slog.String("error ", err.Error()))
		os.Exit(1)
	}

	log.Info("starting application",
		slog.String("env", cfg.Logger.Env),
		slog.Int("grpc_port", cfg.GRPC.PortGRPC),
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
			log.Error("failed to shutdown tracer", slog.String("error", err.Error()))
		}
	}()

	application := app.New(log, cfg, tracerProvider)

	go application.GRPCSrv.Run()

	// Graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	sign := <-stop
	log.Info("stopping application", slog.String("signal", sign.String()))
	application.GRPCSrv.Stop()

	log.Info("application stopped")
}

//func setupLogger(cfg *config.Config) *slog.Logger {
//	var log *slog.Logger
//	logFilePath := filepath.Join(cfg.Logging.LogDir, "app.log")
//	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
//	if err != nil {
//		log.Error("failed to open log file",
//			slog.String("logFilePath", logFilePath),
//			slog.String("error", err.Error()))
//		os.Exit(1)
//	}
//
//	// Use io.MultiWriter to write logs to both stdout and file.
//	multiWriter := io.MultiWriter(os.Stdout, logFile)
//
//	switch cfg.Logger.Env {
//	case envLocal:
//		prettyHandler := slogpretty.PrettyHandlerOptions{
//			SlogOpts: &slog.HandlerOptions{Level: slog.LevelDebug},
//		}.NewPrettyHandler(multiWriter)
//		log = slog.New(prettyHandler)
//	case envDev, envProd:
//		log = slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{Level: slog.LevelDebug}))
//	default:
//		log = slog.New(slog.NewTextHandler(multiWriter, &slog.HandlerOptions{Level: slog.LevelDebug}))
//	}
//
//	return log
//}

func setupLogger(cfg *config.Config) *slog.Logger {
	if cfg == nil {
		panic("Config is nil, cannot initialize logger")
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo, // Можеш змінити рівень логування через cfg
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	return logger
}
