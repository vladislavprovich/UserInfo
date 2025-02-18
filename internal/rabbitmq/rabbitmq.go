package rabbitmq

import (
	"github.com/streadway/amqp"
	"log"
)

func NewRabbitMQ() (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Помилка підключення до RabbitMQ: %v", err)
		return nil, nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Помилка створення каналу: %v", err)
		return nil, nil, err
	}

	return conn, ch, nil
}
