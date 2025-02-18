package server

import (
	"context"
	"fmt"
	"github.com/vladislavprovich/UserInfo/internal/storage"
	userinfov3 "github.com/vladislavprovich/protobufContract/gen/go/userinfo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
)

type Server interface {
	GetUserByID(ctx context.Context, req *userinfov3.GetUserByIDRequest) (*userinfov3.UserByIDResponse, error)
	GetUserByEmail(ctx context.Context, req *userinfov3.GetUserByEmailRequest) (*userinfov3.UserByEmailResponse, error)
}

type ApiServer struct {
	userinfov3.UnimplementedUserInfoServiceServer
	Server  Server
	Storage storage.UserStorage
	log     *slog.Logger
	Tracer  trace.Tracer
}

//todo params

func Register(gRPC *grpc.Server, tracer trace.Tracer, log *slog.Logger, storage storage.UserStorage) {
	userinfov3.RegisterUserInfoServiceServer(gRPC, &ApiServer{
		Storage: storage,
		log:     log,
		Tracer:  tracer,
	})
}

// GetUserByID return user by id(without password).
func (s *ApiServer) GetUserByID(
	ctx context.Context,
	req *userinfov3.GetUserByIDRequest,
) (*userinfov3.UserByIDResponse, error) {
	s.log.Info("Call GetUserByID", slog.String("user_id", req.GetUserId()))

	ctx, span := s.Tracer.Start(ctx, "server.GetUserByID")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", req.GetUserId()))

	user, err := s.Storage.GetUserByID(req.GetUserId())
	if err != nil {
		s.log.Error("GetUserByID error",
			slog.String("user_id", req.GetUserId()),
			slog.String("error", err.Error()),
		)
		span.RecordError(err)

		return nil, status.Error(codes.Internal, fmt.Sprintf("get user by id error: %s", err))
	}

	return &userinfov3.UserByIDResponse{
		UserId: user.UserID,
		Email:  user.Email,
	}, nil
}

// GetUserByEmail return user by email(without password).
func (s *ApiServer) GetUserByEmail(
	ctx context.Context,
	req *userinfov3.GetUserByEmailRequest,
) (*userinfov3.UserByEmailResponse, error) {
	s.log.Info("Call GetUserByEmail", slog.String("email", req.GetEmail()))

	ctx, span := s.Tracer.Start(ctx, "server.GetUserByEmail")
	defer span.End()

	span.SetAttributes(attribute.String("email", req.GetEmail()))

	user, err := s.Storage.GetUserByEmail(req.GetEmail())
	if err != nil {
		s.log.Error("GetUserByEmail error",
			slog.String("email", req.GetEmail()),
			slog.String("error", err.Error()))
		span.RecordError(err)

		return nil, status.Error(codes.Internal, fmt.Sprintf("get user by email error: %s", err))
	}

	return &userinfov3.UserByEmailResponse{
		UserId: user.UserID,
		Email:  user.Email,
	}, nil
}
