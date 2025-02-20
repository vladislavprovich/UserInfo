package rabbitmq

import (
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/vladislavprovich/UserInfo/config"

	"github.com/streadway/amqp"
)

func NewRabbitMQ(cfg *config.Config) (*amqp.Connection, *amqp.Channel, error) {
	hostAndPort := net.JoinHostPort(cfg.Rabbit.Host, strconv.Itoa(cfg.Rabbit.Port))
	rabbitURL := fmt.Sprintf("amqp://%s:%s@%s/",
		cfg.Rabbit.User,
		cfg.Rabbit.Password,
		hostAndPort,
	)

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Error connect to RabbitMQ: %v", err)
		return nil, nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Error creating chan: %v", err)
		return nil, nil, err
	}

	return conn, ch, nil
}
