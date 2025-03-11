package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vladislavprovich/user-info/internal/models"
	"github.com/vladislavprovich/user-info/internal/repository/mongomodels"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/protobuf/types/known/timestamppb"

	userinfo "github.com/vladislavprovich/protobuf-contract/gen/go/userinfo"
	"github.com/vladislavprovich/user-info/internal/repository/storage"
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
	Server             Server
	Storage            storage.UserStorage
	Log                *slog.Logger
	Tracer             trace.Tracer
	convectorToStorage *ConvertToStorage
}

func Register(gRPC *grpc.Server, tracer trace.Tracer, log *slog.Logger, storage storage.UserStorage) {
	userinfo.RegisterUserInfoServiceServer(gRPC, &APIServer{
		Storage:            storage,
		Log:                log,
		Tracer:             tracer,
		convectorToStorage: NewConvertToStorage(),
	})
}

func (s *APIServer) GetUserByID(
	ctx context.Context,
	req *userinfo.GetUserByIDRequest,
) (*userinfo.UserByIDResponse, error) {
	ctx, span := s.Tracer.Start(ctx, "server.GetUserByID")
	defer span.End()

	user, err := s.definitionReqForUserByID(ctx, req.GetUserId())
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

	user, err := s.definitionReqForUserByEmail(ctx, req.GetEmail())
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

func (s *APIServer) definitionReqForUserByID(
	ctx context.Context,
	userID int64,
) (*models.User, error) {
	return definitionReqForUser(
		ctx,
		userID,
		s.Storage.GetUserByID,
		lookTypeID,
		s.convectorToStorage.ConvectorMongoModelsToUserModels,
		s.Log,
		s.Tracer,
	)
}

func (s *APIServer) definitionReqForUserByEmail(
	ctx context.Context,
	email string,
) (*models.User, error) {
	return definitionReqForUser(
		ctx,
		email,
		s.Storage.GetUserByEmail,
		lookTypeEmail,
		s.convectorToStorage.ConvectorMongoModelsToUserModels,
		s.Log,
		s.Tracer,
	)
}

func definitionReqForUser[T comparable](
	ctx context.Context,
	lookupValue T,
	getUserFunc func(context.Context, T) (*mongomodels.User, error),
	logKey string,
	converter func(*mongomodels.User) *models.User,
	log *slog.Logger,
	tracer trace.Tracer,
) (*models.User, error) {
	log.InfoContext(ctx, fmt.Sprintf("Call GetUserBy%s", logKey), slog.Any(logKey, lookupValue))

	ctx, span := tracer.Start(ctx, fmt.Sprintf("server.GetUserBy%s", logKey))
	defer span.End()

	span.SetAttributes(attribute.String(logKey, fmt.Sprintf("%v", lookupValue)))

	userMongoModels, err := getUserFunc(ctx, lookupValue)
	if err != nil {
		log.WarnContext(ctx, fmt.Sprintf("GetUserBy%s error", logKey),
			slog.Any(logKey, lookupValue),
			slog.String("error", err.Error()),
		)
		span.RecordError(err)
		return nil, fmt.Errorf("get user by %s error: %w", logKey, err)
	}

	return converter(userMongoModels), nil
}
