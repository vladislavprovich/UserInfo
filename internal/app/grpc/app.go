package grpcapp

import (
	"context"
	"fmt"
	"github.com/vladislavprovich/UserInfo/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"net"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"github.com/vladislavprovich/UserInfo/internal/server"
)

type App struct {
	log        slog.Logger
	gRPCServer *grpc.Server
	port       int
	tracer     trace.Tracer
	storage    storage.UserStorage
}

func New(
	log *slog.Logger,
	port int,
	tracer trace.Tracer,
	storage storage.UserStorage,
) *App {
	options := []recovery.Option{
		recovery.WithRecoveryHandler(func(p interface{}) error {
			log.Error("recovered from panic", slog.Any("panic", p))
			return status.Errorf(codes.Internal, "internal error")
		}),
	}

	loggingOptions := []logging.Option{
		logging.WithLogOnEvents(logging.PayloadReceived, logging.PayloadSent),
	}

	grpc_prometheus.EnableHandlingTimeHistogram()

	interceptors := grpc.ChainUnaryInterceptor(
		grpc_middleware.ChainUnaryServer(
			logging.UnaryServerInterceptor(logInterceptor(log), loggingOptions...),
			recovery.UnaryServerInterceptor(options...),
			grpc_prometheus.UnaryServerInterceptor,
		),
	)

	gRPCServer := grpc.NewServer(interceptors)

	server.Register(
		gRPCServer,
		tracer,
		log,
		storage,
	)

	return &App{
		log:        *log,
		gRPCServer: gRPCServer,
		port:       port,
		tracer:     tracer,
		storage:    storage,
	}
}

func logInterceptor(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(_ context.Context, level logging.Level, msg string, fields ...any) {
		switch level {
		case logging.LevelDebug:
			l.Debug(msg, fields...)
		case logging.LevelInfo:
			l.Info(msg, fields...)
		case logging.LevelWarn:
			l.Warn(msg, fields...)
		case logging.LevelError:
			l.Error(msg, fields...)
		default:
			l.Info(msg, fields...)
		}
	})
}

func (a *App) Run() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		panic(err)
	}

	a.log.Info("gRPC server is running", slog.String("address", lis.Addr().String()))

	if err = a.gRPCServer.Serve(lis); err != nil {
		panic(err)
	}

}

func (a *App) Stop() {
	a.log.Info("gRPC server is stopping", slog.Int("port", a.port))
	a.gRPCServer.GracefulStop()
}
