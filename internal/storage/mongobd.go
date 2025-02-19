package storage

import (
	"context"
	"github.com/vladislavprovich/UserInfo/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log/slog"
	"time"
)

type UserStorage interface {
	GetUserByID(id string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	SaveUser(user *models.User) error
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

func (s *MongoDBStorage) SaveUser(user *models.User) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := s.DB.Collection("users").InsertOne(context.Background(), user)
	return err
}

func (s *MongoDBStorage) GetUserByID(id string) (*models.User, error) {
	var user models.User
	err := s.DB.Collection("users").FindOne(context.Background(), bson.M{"user_id": id}).Decode(&user)
	return &user, err
}

func (s *MongoDBStorage) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := s.DB.Collection("users").FindOne(context.Background(), bson.M{"email": email}).Decode(&user)
	return &user, err
}
