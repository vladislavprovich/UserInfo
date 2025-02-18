package server

import (
	"context"
	"github.com/vladislavprovich/UserInfo/internal/storage"
	pb "github.com/vladislavprovich/protobufContract/gen/go/userinfo"
)

type UserService struct {
	pb.UnimplementedUserInfoServiceServer
	storage storage.UserStorage
	// todo logger
}

func NewUserService(storage storage.UserStorage) *UserService {
	return &UserService{storage: storage}
}

// GetUserByID повертає користувача за ID
func (s *UserService) GetUserByID(ctx context.Context, req *pb.GetUserByIDRequest) (*pb.UserByIDResponse, error) {
	user, err := s.storage.GetUserByID(req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &pb.UserByIDResponse{
		UserId: user.UserID,
		Email:  user.Email,
	}, nil
}

// GetUserByEmail повертає користувача за Email
func (s *UserService) GetUserByEmail(ctx context.Context, req *pb.GetUserByEmailRequest) (*pb.UserByEmailResponse, error) {
	user, err := s.storage.GetUserByEmail(req.GetEmail())
	if err != nil {
		return nil, err
	}
	return &pb.UserByEmailResponse{
		UserId: user.UserID,
		Email:  user.Email,
	}, nil
}
