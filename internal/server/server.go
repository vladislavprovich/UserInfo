package server

import (
	"context"
	"fmt"
	"github.com/vladislavprovich/user-info/internal/models"
	"github.com/vladislavprovich/user-info/internal/repository/mongo_models"
	"github.com/vladislavprovich/user-info/internal/repository/storage"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log/slog"

	userinfo "github.com/vladislavprovich/protobuf-contract/gen/go/userinfo"
	"go.opentelemetry.io/otel/attribute"
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
	lookupValue string,
	lookupType string,
) (*models.User, error) {
	s.Log.InfoContext(ctx, fmt.Sprintf("Call GetUserBy%s", lookupType), slog.String(lookupType, lookupValue))

	ctx, span := s.Tracer.Start(ctx, fmt.Sprintf("server.GetUserBy%s", lookupType))
	defer span.End()

	span.SetAttributes(attribute.String(lookupType, lookupValue))

	var (
		userMongoModels *mongo_models.User
		err             error
	)
	switch lookupType {
	case lookTypeID:
		userMongoModels, err = s.Storage.GetUserByID(ctx, lookupValue)
		s.Log.InfoContext(ctx, "Search by id + mongoModels", slog.Any("userMongo", userMongoModels))
	case lookTypeEmail:
		userMongoModels, err = s.Storage.GetUserByEmail(ctx, lookupValue)
		s.Log.InfoContext(ctx, "Search by email + mongoModels", slog.Any("userMongo", userMongoModels))
	default:
		return nil, fmt.Errorf("invalid lookup type: %s", lookupType)
	}

	if err != nil {
		s.Log.WarnContext(ctx, fmt.Sprintf("GetUserBy%s error", lookupType),
			slog.String(lookupType, lookupValue),
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
			slog.String("user_id", req.GetUserId()),
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
