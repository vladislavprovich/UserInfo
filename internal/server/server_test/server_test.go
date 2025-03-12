package server_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"log/slog"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	userinfo "github.com/vladislavprovich/protobuf-contract/gen/go/userinfo"
	"github.com/vladislavprovich/user-info/internal/repository/mongomodels"
	"github.com/vladislavprovich/user-info/internal/server"
)

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) SaveUser(ctx context.Context, user *mongomodels.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockStorage) GetUserByID(ctx context.Context, id int64) (*mongomodels.User, error) {
	args := m.Called(ctx, id)
	if user, ok := args.Get(0).(*mongomodels.User); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStorage) GetUserByEmail(ctx context.Context, email string) (*mongomodels.User, error) {
	args := m.Called(ctx, email)
	if user, ok := args.Get(0).(*mongomodels.User); ok {
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

	now := time.Now()

	tests := []struct {
		name          string
		userID        int64
		mockReturn    *mongomodels.User
		mockError     error
		expectedResp  *userinfo.UserByIDResponse
		expectedCode  codes.Code
		expectedError string
	}{
		{
			name:   "User found",
			userID: 123,
			mockReturn: &mongomodels.User{
				UserID:    123,
				Email:     "test@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			mockError: nil,
			expectedResp: &userinfo.UserByIDResponse{
				UserId:    123,
				Email:     "test@example.com",
				CreatedAt: timestamppb.New(now),
				UpdatedAt: timestamppb.New(now),
			},
			expectedCode:  codes.OK,
			expectedError: "",
		},
		{
			name:          "User not found",
			userID:        999,
			mockReturn:    nil,
			mockError:     errors.New("user not found"),
			expectedResp:  nil,
			expectedCode:  codes.Unknown,
			expectedError: "user not found",
		},
		{
			name:          "Database error",
			userID:        500,
			mockReturn:    nil,
			mockError:     errors.New("database connection error"),
			expectedResp:  nil,
			expectedCode:  codes.Unknown,
			expectedError: "database connection error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage.On("GetUserByID", mock.Anything, tt.userID).Return(tt.mockReturn, tt.mockError)

			req := &userinfo.GetUserByIDRequest{UserId: tt.userID}
			resp, err := server.GetUserByID(context.Background(), req)

			assert.Equal(t, tt.expectedResp, resp)

			if tt.expectedError != "" {
				st, _ := status.FromError(err)
				assert.Equal(t, tt.expectedCode, st.Code())
				assert.Contains(t, st.Message(), tt.expectedError)
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

	now := time.Now()

	tests := []struct {
		name          string
		email         string
		mockReturn    *mongomodels.User
		mockError     error
		expectedResp  *userinfo.UserByEmailResponse
		expectedCode  codes.Code
		expectedError string
	}{
		{
			name:  "User found",
			email: "test@example.com",
			mockReturn: &mongomodels.User{
				UserID:    123,
				Email:     "test@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			mockError: nil,
			expectedResp: &userinfo.UserByEmailResponse{
				UserId:    123,
				Email:     "test@example.com",
				CreatedAt: timestamppb.New(now),
				UpdatedAt: timestamppb.New(now),
			},
			expectedCode:  codes.OK,
			expectedError: "",
		},
		{
			name:          "User not found",
			email:         "notfound@example.com",
			mockReturn:    nil,
			mockError:     errors.New("user not found"),
			expectedResp:  nil,
			expectedCode:  codes.Unknown,
			expectedError: "user not found",
		},
		{
			name:          "Database error",
			email:         "error@example.com",
			mockReturn:    nil,
			mockError:     errors.New("database error"),
			expectedResp:  nil,
			expectedCode:  codes.Unknown,
			expectedError: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage.On("GetUserByEmail", mock.Anything, tt.email).Return(tt.mockReturn, tt.mockError)

			req := &userinfo.GetUserByEmailRequest{Email: tt.email}
			resp, err := server.GetUserByEmail(context.Background(), req)

			assert.Equal(t, tt.expectedResp, resp)

			if tt.expectedError != "" {
				st, _ := status.FromError(err)
				assert.Equal(t, tt.expectedCode, st.Code())
				assert.Contains(t, st.Message(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}

			mockStorage.AssertExpectations(t)
		})
	}
}
