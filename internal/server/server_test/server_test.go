package server_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/stretchr/testify/require"
	"github.com/vladislavprovich/UserInfo/internal/server"

	"log/slog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vladislavprovich/UserInfo/internal/models"
	userinfov3 "github.com/vladislavprovich/protobufContract/gen/go/userinfo"
)

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) SaveUser(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockStorage) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if user, ok := args.Get(0).(*models.User); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStorage) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if user, ok := args.Get(0).(*models.User); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}

func TestGetUserByID(t *testing.T) {
	mockStorage := new(MockStorage)
	logger := slog.Default()
	tracer := noop.NewTracerProvider().Tracer("test_get_user_by_id")

	server := &server.APIServer{
		Storage: mockStorage,
		Log:     logger,
		Tracer:  tracer,
	}

	tests := []struct {
		name          string
		userID        string
		mockReturn    *models.User
		mockError     error
		expectedResp  *userinfov3.UserByIDResponse
		expectedError string
	}{
		{
			name:   "User found",
			userID: "123",
			mockReturn: &models.User{
				UserID:    "123",
				Email:     "test@example.com",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mockError: nil,
			expectedResp: &userinfov3.UserByIDResponse{
				UserId: "123",
				Email:  "test@example.com",
			},
			expectedError: "",
		},
		{
			name:          "User not found",
			userID:        "999",
			mockReturn:    nil,
			mockError:     errors.New("user not found"),
			expectedResp:  nil,
			expectedError: "rpc error: code = Internal desc = get user by ID error: user not found",
		},
		{
			name:          "Database error",
			userID:        "500",
			mockReturn:    nil,
			mockError:     errors.New("database connection error"),
			expectedResp:  nil,
			expectedError: "rpc error: code = Internal desc = get user by ID error: database connection error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage.On("GetUserByID", mock.Anything, tt.userID).Return(tt.mockReturn, tt.mockError)

			req := &userinfov3.GetUserByIDRequest{UserId: tt.userID}
			resp, err := server.GetUserByID(context.Background(), req)

			assert.Equal(t, tt.expectedResp, resp)

			if tt.expectedError != "" {
				require.EqualError(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
			}

			mockStorage.AssertExpectations(t)
		})
	}
}

func TestGetUserByEmail(t *testing.T) {
	mockStorage := new(MockStorage)
	logger := slog.Default()
	tracer := noop.NewTracerProvider().Tracer("test_get_user_by_email")

	server := &server.APIServer{
		Storage: mockStorage,
		Log:     logger,
		Tracer:  tracer,
	}

	tests := []struct {
		name          string
		email         string
		mockReturn    *models.User
		mockError     error
		expectedResp  *userinfov3.UserByEmailResponse
		expectedError string
	}{
		{
			name:  "User found",
			email: "test@example.com",
			mockReturn: &models.User{
				UserID:    "123",
				Email:     "test@example.com",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			mockError: nil,
			expectedResp: &userinfov3.UserByEmailResponse{
				UserId: "123",
				Email:  "test@example.com",
			},
			expectedError: "",
		},
		{
			name:          "User not found",
			email:         "notfound@example.com",
			mockReturn:    nil,
			mockError:     errors.New("user not found"),
			expectedResp:  nil,
			expectedError: "rpc error: code = Internal desc = get user by Email error: user not found",
		},
		{
			name:          "Database error",
			email:         "error@example.com",
			mockReturn:    nil,
			mockError:     errors.New("database error"),
			expectedResp:  nil,
			expectedError: "rpc error: code = Internal desc = get user by Email error: database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage.On("GetUserByEmail", mock.Anything, tt.email).Return(tt.mockReturn, tt.mockError)

			req := &userinfov3.GetUserByEmailRequest{Email: tt.email}
			resp, err := server.GetUserByEmail(context.Background(), req)

			assert.Equal(t, tt.expectedResp, resp)

			if tt.expectedError != "" {
				require.EqualError(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
			}

			mockStorage.AssertExpectations(t)
		})
	}
}
