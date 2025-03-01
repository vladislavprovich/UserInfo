package server

import (
	"github.com/vladislavprovich/user-info/internal/models"
	"github.com/vladislavprovich/user-info/internal/repository/mongo_models"
)

type ConvertToStorage struct {
}

func NewConvertToStorage() *ConvertToStorage {
	return &ConvertToStorage{}
}

func (c *ConvertToStorage) ConvectorMongoModelsToUserModels(user *mongo_models.User) *models.User {
	return &models.User{
		UserID:    user.UserID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
