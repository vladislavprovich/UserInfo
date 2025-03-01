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
	}, nil
}

func (m *MongoFactory) connect(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, m.cfg.ConnectionTimeout)
	defer cancel()

	clientOptions := options.Client().ApplyURI(m.cfg.URL)

	// Auth login and password.
	if m.cfg.User != "" && m.cfg.Password != "" {
		clientOptions.SetAuth(options.Credential{
			Username:   m.cfg.User,
			Password:   m.cfg.Password,
			AuthSource: m.cfg.AuthSource,
		})
	}

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("mongo connect error: %w", err)
	}

	// Check connect.
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("mongo ping error: %w", err)
	}

	m.client = client
	m.DB = client.Database(m.cfg.Database)

	return nil
}

func (m *MongoFactory) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
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
