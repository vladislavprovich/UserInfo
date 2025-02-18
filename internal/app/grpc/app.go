package grpc

import (
	"fmt"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	servergrpc "github.com/vladislavprovich/UserInfo/internal/server"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"net"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int
	tracer     trace.Tracer
}

func New(
	log *slog.Logger,
	authService servergrpc.UserService,
	port int,
	tracer trace.Tracer,
) *App {
	options := []recovery.Option{
		recovery.WithRecoveryHandler(func(p interface{}) error {
			log.Error("recovery from panic", slog.Any("panic", p))

			return status.Errorf(codes.Internal, "internal error")
		}),
	}

	loggingOptions := []logging.Option{
		logging.WithLogOnEvents(
			logging.PayloadReceived,
			logging.PayloadSent,
		),
	}

	grpc_prometheus.EnableHandlingTimeHistogram()

	interceptors := grpc.ChainUnaryInterceptor(
		grpc_middleware.ChainUnaryServer(
			logging.UnaryServerInterceptor(logInterceptor(log), loggingOptions...),
			recovery.UnaryServerInterceptor(options...),
			grpc_prometheus.UnaryServerInterceptor,
		),
	)

	gRPCServer := grpc.NewServer(
		interceptors,
	)

	servergrpc.NewUserService(gRPCServer, authService, tracer)

	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		port:       port,
		tracer:     tracer,
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

// MustRun runs gRPC server and panics if any error occurs.
func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	const op = "grpcapp.Run"

	log := a.log.With(
		slog.String("op", op),
		slog.Int("port", a.port),
	)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("grpc server is running", slog.String("addr", l.Addr().String()))

	if err = a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.With(slog.String("op", op)).
		Info("grpc server is stopping", slog.Int("port", a.port))

	a.gRPCServer.GracefulStop()
}
