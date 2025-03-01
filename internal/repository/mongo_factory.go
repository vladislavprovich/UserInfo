package repository

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoFactory struct {
	cfg    ConfigMongo
	client *mongo.Client
	DB     *mongo.Database
}

func NewMongo(cfg ConfigMongo) (*MongoFactory, error) {
	mongoURI := createMongoURI(cfg)

	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	return &MongoFactory{
		client: client,
		DB:     client.Database(cfg.Database),
		cfg:    cfg,
	}, nil
}

func (m *MongoFactory) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	return m.client.Ping(ctx, readpref.Primary())
}

func (m *MongoFactory) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

func createMongoURI(cfg ConfigMongo) string {
	hostAndPort := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	mongoURI := fmt.Sprintf("mongodb://%s:%s@%s/%s?authSource=%s",
		cfg.User,
		cfg.Password,
		hostAndPort,
		cfg.Database,
		cfg.AuthSource,
	)

	return mongoURI
}
