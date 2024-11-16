package repositories

import (
	"user-service/database"
	"user-service/models"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	CreateUser(user *models.User) error
	FindUserByName(name string) (*models.User, error)
}

type GormUserRepository struct{}

func (repo *GormUserRepository) CreateUser(user *models.User) error {
	return database.DB.Create(user).Error
}

func (repo *GormUserRepository) FindUserByName(name string) (*models.User, error) {
	var user models.User
	err := database.DB.Where("name = ?", name).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
