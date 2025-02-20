package rabbitmq

import (
	"context"
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
	"github.com/vladislavprovich/UserInfo/internal/models"
	"github.com/vladislavprovich/UserInfo/internal/storage"
)

func StartConsumer(ch *amqp.Channel, storage storage.UserStorage) {
	ctx := context.Background()
	q, err := ch.QueueDeclare(
		"user_registration",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("error creating queue: %v", err)
	}

	msg, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("error get message: %v", err)
	}

	go func() {
		for d := range msg {
			var userPayload *models.UserPayload
			err = json.Unmarshal(d.Body, &userPayload)
			if err != nil {
				log.Printf("error unparsing message: %v", err)
				continue
			}

			user := &models.User{
				UserID:    userPayload.UserID,
				Email:     userPayload.Email,
				Password:  userPayload.Password,
				CreatedAt: userPayload.CreatedAt,
				UpdatedAt: userPayload.UpdatedAt,
			}

			err = storage.SaveUser(ctx, user)
			if err != nil {
				log.Printf("error save user in db: %v", err)
			} else {
				log.Printf("user %s sucssesful saved", user.Email)
			}
		}
	}()
}
