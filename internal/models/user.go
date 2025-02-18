package models

type User struct {
	UserID   string `bson:"user_id"`
	Email    string `bson:"email"`
	Password string `bson:"password"`
}
