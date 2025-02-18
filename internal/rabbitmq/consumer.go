package rabbitmq

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"github.com/vladislavprovich/UserInfo/internal/models"
	"github.com/vladislavprovich/UserInfo/internal/storage"
	"log"
)

// RegisterUserPayload – структура повідомлення
type RegisterUserPayload struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// StartConsumer запускає споживача RabbitMQ
func StartConsumer(ch *amqp.Channel, storage storage.UserStorage) {
	q, err := ch.QueueDeclare(
		"user_registration", // Назва черги
		true,                // Durable
		false,               // Auto-delete
		false,               // Exclusive
		false,               // No-wait
		nil,                 // Arguments
	)
	if err != nil {
		log.Fatalf("Помилка створення черги: %v", err)
	}

	msg, err := ch.Consume(
		q.Name, // Queue
		"",     // Consumer
		true,   // Auto-Ack
		false,  // Exclusive
		false,  // No-local
		false,  // No-wait
		nil,    // Args
	)
	if err != nil {
		log.Fatalf("Помилка отримання повідомлень: %v", err)
	}

	go func() {
		for d := range msg {
			var userPayload RegisterUserPayload
			err = json.Unmarshal(d.Body, &userPayload)
			if err != nil {
				log.Printf("Помилка розпарсення повідомлення: %v", err)
				continue
			}

			// Створюємо нового користувача в БД
			user := models.User{
				UserID:   userPayload.UserID, // UUID з SSO
				Email:    userPayload.Email,
				Password: userPayload.Password, // Пароль зберігається, але не повертається при запитах
			}

			err = storage.SaveUser(&user)
			if err != nil {
				log.Printf("Помилка збереження користувача в БД: %v", err)
			} else {
				log.Printf("Користувач %s успішно збережений", user.Email)
			}
		}
	}()
}
