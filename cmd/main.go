package main

import (
	"context"
	"github.com/vladislavprovich/UserInfo/config"
	"github.com/vladislavprovich/UserInfo/internal/app/grpc"
	"github.com/vladislavprovich/UserInfo/internal/rabbitmq"
	"github.com/vladislavprovich/UserInfo/internal/storage"
	"github.com/vladislavprovich/UserInfo/lib/logger/slogpretty"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	ctx := context.Background()
	// Ініціалізація MongoDB
	db, err := storage.NewMongoDB("mongodb://localhost:27017", "userinfo_db")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err = db.DB.Client().Disconnect(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Ініціалізація RabbitMQ
	rabbitConn, rabbitCh, err := rabbitmq.NewRabbitMQ()
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitConn.Close()
	defer rabbitCh.Close()

	// Запускаємо споживача RabbitMQ
	rabbitmq.StartConsumer(rabbitCh, db)

	// Запускаємо gRPC сервер
	grpc.StartServer(db)
}

func setupLogger(cfg *config.Config) *slog.Logger {
	var log *slog.Logger
	logFilePath := filepath.Join(cfg.Logging.LogDir, "app.log")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Error("failed to open log file",
			slog.String("logFilePath", logFilePath),
			slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Use io.MultiWriter to write logs to both stdout and file.
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	switch cfg.Logger.Level {
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
