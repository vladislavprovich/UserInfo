package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vladislavprovich/user-info/internal/models"
	"github.com/vladislavprovich/user-info/internal/repository/mongomodels"
	"github.com/vladislavprovich/user-info/internal/repository/storage"
	"google.golang.org/protobuf/types/known/timestamppb"

	userinfo "github.com/vladislavprovich/protobuf-contract/gen/go/userinfo"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

const (
	lookTypeID    = "user_id"
	lookTypeEmail = "email"
)

type Server interface {
	GetUserByID(ctx context.Context, req *userinfo.GetUserByIDRequest) (*userinfo.UserByIDResponse, error)
	GetUserByEmail(ctx context.Context, req *userinfo.GetUserByEmailRequest) (*userinfo.UserByEmailResponse, error)
}

type APIServer struct {
	userinfo.UnimplementedUserInfoServiceServer
	Server    Server
	Storage   storage.UserStorage
	Log       *slog.Logger
	Tracer    trace.Tracer
	convector *ConvertToStorage
}

func Register(gRPC *grpc.Server, tracer trace.Tracer, log *slog.Logger, storage storage.UserStorage) {
	userinfo.RegisterUserInfoServiceServer(gRPC, &APIServer{
		Storage:   storage,
		Log:       log,
		Tracer:    tracer,
		convector: NewConvertToStorage(),
	})
}

func (s *APIServer) definitionReqForUser(
	ctx context.Context,
	lookupValue interface{},
	lookupType string,
) (*models.User, error) {
	s.Log.InfoContext(ctx, fmt.Sprintf("Call GetUserBy%s", lookupType), slog.Any(lookupType, lookupValue))

	ctx, span := s.Tracer.Start(ctx, fmt.Sprintf("server.GetUserBy%s", lookupType))
	defer span.End()

	var (
		userMongoModels *mongomodels.User
		err             error
	)

	switch lookupType {
	case lookTypeID:
		userID, ok := lookupValue.(int64)
		if !ok {
			return nil, fmt.Errorf("invalid type for userID: expected int64, got %T", lookupValue)
		}
		userMongoModels, err = s.Storage.GetUserByID(ctx, userID)
		s.Log.InfoContext(ctx, "Search by id", slog.Int64("userID", userID))

	case lookTypeEmail:
		email, ok := lookupValue.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type for email: expected string, got %T", lookupValue)
		}
		userMongoModels, err = s.Storage.GetUserByEmail(ctx, email)
		s.Log.InfoContext(ctx, "Search by email", slog.String("email", email))

	default:
		return nil, fmt.Errorf("invalid lookup type: %s", lookupType)
	}

	if err != nil {
		s.Log.WarnContext(ctx, fmt.Sprintf("GetUserBy%s error", lookupType),
			slog.Any(lookupType, lookupValue),
			slog.String("error", err.Error()),
		)
		span.RecordError(err)
		return nil, fmt.Errorf("get user by %s error: %w", lookupType, err)
	}

	user := s.convector.ConvectorMongoModelsToUserModels(userMongoModels)

	return user, nil
}

func (s *APIServer) GetUserByID(
	ctx context.Context,
	req *userinfo.GetUserByIDRequest,
) (*userinfo.UserByIDResponse, error) {
	ctx, span := s.Tracer.Start(ctx, "server.GetUserByID")
	defer span.End()

	user, err := s.definitionReqForUser(ctx, req.GetUserId(), lookTypeID)
	if err != nil {
		s.Log.ErrorContext(ctx, "GetUserByID error",
			slog.Int64("user_id", req.GetUserId()),
			slog.String("error", err.Error()))
		span.RecordError(err)
		return nil, err
	}

	return &userinfo.UserByIDResponse{
		UserId:    user.UserID,
		Email:     user.Email,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}, nil
}

func (s *APIServer) GetUserByEmail(
	ctx context.Context,
	req *userinfo.GetUserByEmailRequest,
) (*userinfo.UserByEmailResponse, error) {
	ctx, span := s.Tracer.Start(ctx, "server.GetUserByEmail")
	defer span.End()

	user, err := s.definitionReqForUser(ctx, req.GetEmail(), lookTypeEmail)
	if err != nil {
		s.Log.ErrorContext(ctx, "GetUserByEmail error",
			slog.String("email", req.GetEmail()),
			slog.String("error", err.Error()))
		span.RecordError(err)
		return nil, err
	}

	return &userinfo.UserByEmailResponse{
		UserId:    user.UserID,
		Email:     user.Email,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}, nil
}
