package storage

import (
	"context"
	"log/slog"
	"time"

	"github.com/vladislavprovich/user-info/internal/repository/mongomodels"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserStorage interface {
	GetUserByID(ctx context.Context, id int64) (*mongomodels.User, error)
	GetUserByEmail(ctx context.Context, email string) (*mongomodels.User, error)
	SaveUser(ctx context.Context, user *mongomodels.User) error
}

type MongoDBStorage struct {
	coll *mongo.Collection
	log  *slog.Logger
}

func NewMongoStorage(db *mongo.Database, log *slog.Logger) *MongoDBStorage {
	return &MongoDBStorage{
		coll: db.Collection("user"),
		log:  log,
	}
}

func (s *MongoDBStorage) SaveUser(ctx context.Context, user *mongomodels.User) error {
	s.log.InfoContext(ctx, "Saving user in db")

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	doc := bson.M{
		"_id":        user.UserID,
		"user_id":    user.UserID,
		"email":      user.Email,
		"password":   user.Password,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}

	_, err := s.coll.InsertOne(ctx, doc)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to save user in db", slog.String("error", err.Error()))
	}

	return nil
}

func (s *MongoDBStorage) GetUserByID(ctx context.Context, id int64) (*mongomodels.User, error) {
	s.log.InfoContext(ctx, "Get user by ID")

	var user mongomodels.User
	err := s.coll.FindOne(ctx, bson.M{"user_id": id}).Decode(&user)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to find user in db", slog.String("error", err.Error()))
	}

	return &user, nil
}

func (s *MongoDBStorage) GetUserByEmail(ctx context.Context, email string) (*mongomodels.User, error) {
	s.log.InfoContext(ctx, "Get user by email")

	var user mongomodels.User
	err := s.coll.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to find user in db", slog.String("error", err.Error()))
	}

	return &user, nil
}
