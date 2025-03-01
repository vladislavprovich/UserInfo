package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/cenkalti/backoff/v4"
	"github.com/streadway/amqp"
	"github.com/vladislavprovich/user-info/config"
)

// NewRabbitMQ connect to RabbitMQ with repetitions.
func NewRabbitMQ(ctx context.Context, cfg *config.Config) (*amqp.Connection, error) {
	hostAndPort := net.JoinHostPort(cfg.Rabbit.Host, strconv.Itoa(cfg.Rabbit.Port))
	rabbitURL := fmt.Sprintf(
		"amqp://%s:%s@%s/",
		cfg.Rabbit.User,
		cfg.Rabbit.Password,
		hostAndPort)

	var conn *amqp.Connection
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = cfg.Rabbit.MaxElapsedTime // Time to retries.
	maxRetries := cfg.Rabbit.MaxRetries

	err := backoff.Retry(func() error {
		var err error
		conn, err = amqp.Dial(rabbitURL)
		if err != nil {
			log.Printf("Failed to connect to RabbitMQ: %v. Retrying...", err)
			return err
		}
		return nil
	}, backoff.WithMaxRetries(bo, maxRetries))

	if err != nil {
		return nil, fmt.Errorf("could not establish RabbitMQ connection: %w", err)
	}

	log.Println("Connected to RabbitMQ")

	go func() {
		<-ctx.Done()
		if err = conn.Close(); err != nil {
			log.Println("Error closing RabbitMQ connection:", err)
		}
		log.Println("RabbitMQ connection closed")
	}()

	return conn, nil
}
