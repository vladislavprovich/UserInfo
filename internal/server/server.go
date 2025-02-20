package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vladislavprovich/UserInfo/internal/models"

	"github.com/vladislavprovich/UserInfo/internal/storage"
	userinfov3 "github.com/vladislavprovich/protobufContract/gen/go/userinfo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	lookTypeID    = "user_id"
	lookTypeEmail = "email"
)

type Server interface {
	GetUserByID(ctx context.Context, req *userinfov3.GetUserByIDRequest) (*userinfov3.UserByIDResponse, error)
	GetUserByEmail(ctx context.Context, req *userinfov3.GetUserByEmailRequest) (*userinfov3.UserByEmailResponse, error)
}

type APIServer struct {
	userinfov3.UnimplementedUserInfoServiceServer
	Server  Server
	Storage storage.UserStorage
	Log     *slog.Logger
	Tracer  trace.Tracer
}

func Register(gRPC *grpc.Server, tracer trace.Tracer, log *slog.Logger, storage storage.UserStorage) {
	userinfov3.RegisterUserInfoServiceServer(gRPC, &APIServer{
		Storage: storage,
		Log:     log,
		Tracer:  tracer,
	})
}

func (s *APIServer) fetchUser(
	ctx context.Context,
	lookupValue string,
	lookupType string,
	fetchFunc func(context.Context, string) (*models.User, error),
) (*models.User, error) {
	s.Log.InfoContext(ctx, fmt.Sprintf("Call GetUserBy%s", lookupType), slog.String(lookupType, lookupValue))

	ctx, span := s.Tracer.Start(ctx, fmt.Sprintf("server.GetUserBy%s", lookupType))
	defer span.End()

	span.SetAttributes(attribute.String(lookupType, lookupValue))

	user, err := fetchFunc(ctx, lookupValue)
	if err != nil {
		s.Log.WarnContext(ctx, fmt.Sprintf("GetUserBy%s error", lookupType),
			slog.String(lookupType, lookupValue),
			slog.String("error", err.Error()),
		)
		span.RecordError(err)

		return nil, fmt.Errorf("get user by %s error: %w", lookupType, err)
	}

	return user, nil
}

func (s *APIServer) GetUserByID(
	ctx context.Context,
	req *userinfov3.GetUserByIDRequest,
) (*userinfov3.UserByIDResponse, error) {
	ctx, span := s.Tracer.Start(ctx, "server.GetUserByID")
	defer span.End()

	user, err := s.fetchUser(ctx, req.GetUserId(), lookTypeID, s.Storage.GetUserByID)
	if err != nil {
		s.Log.ErrorContext(ctx, "GetUserByID error",
			slog.String("user_id", req.GetUserId()),
			slog.String("error", err.Error()))
		span.RecordError(err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &userinfov3.UserByIDResponse{
		UserId: user.UserID,
		Email:  user.Email,
	}, nil
}

func (s *APIServer) GetUserByEmail(
	ctx context.Context,
	req *userinfov3.GetUserByEmailRequest,
) (*userinfov3.UserByEmailResponse, error) {
	ctx, span := s.Tracer.Start(ctx, "server.GetUserByEmail")
	defer span.End()

	user, err := s.fetchUser(ctx, req.GetEmail(), lookTypeEmail, s.Storage.GetUserByEmail)
	if err != nil {
		s.Log.ErrorContext(ctx, "GetUserByEmail error",
			slog.String("email", req.GetEmail()),
			slog.String("error", err.Error()))
		span.RecordError(err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &userinfov3.UserByEmailResponse{
		UserId: user.UserID,
		Email:  user.Email,
	}, nil
}
