package storage

import (
	"context"
	"log/slog"
	"time"

	"github.com/vladislavprovich/UserInfo/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserStorage interface {
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	SaveUser(ctx context.Context, user *models.User) error
}

type MongoDBStorage struct {
	Client *mongo.Client
	DB     *mongo.Database
	log    *slog.Logger
}

func NewMongoDB(uri, dbName string) (*MongoDBStorage, error) {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, err
	}

	db := client.Database(dbName)
	return &MongoDBStorage{Client: client, DB: db}, nil
}

func (s *MongoDBStorage) SaveUser(ctx context.Context, user *models.User) error {
	s.log.InfoContext(ctx, "Saving user in db")

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := s.DB.Collection("users").InsertOne(context.Background(), user)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to save user in db", slog.String("error", err.Error()))
	}

	return nil
}

func (s *MongoDBStorage) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	s.log.InfoContext(ctx, "Get user by ID")

	var user models.User
	err := s.DB.Collection("users").FindOne(context.Background(), bson.M{"user_id": id}).Decode(&user)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to find user in db", slog.String("error", err.Error()))
	}

	return &user, nil
}

func (s *MongoDBStorage) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	s.log.InfoContext(ctx, "Get user by email")

	var user models.User
	err := s.DB.Collection("users").FindOne(context.Background(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to find user in db", slog.String("error", err.Error()))
	}

	return &user, nil
}
