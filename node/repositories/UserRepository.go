package repositories

import (
	"errors"

	initializers "github.com/DS_node/Initializers"
	"github.com/DS_node/models"
	"gorm.io/gorm"
)

func GetUserByID(userID uint) (models.User, error) {
	var user models.User
	result := initializers.DB.First(&user, userID)
	return user, result.Error
}

func GetUserByEmail(email string) (models.User, error) {
	var user models.User
	result := initializers.DB.Where("email = ?", email).First(&user)
	return user, result.Error
}

func CreateUser(user *models.User) error {
	result := initializers.DB.Create(user)
	return result.Error
}

// CreateUserFromReplication applies CREATE_USER from leader WAL; no-op if email already registered.
func CreateUserFromReplication(user *models.User) error {
	_, err := GetUserByEmail(user.Email)
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return CreateUser(user)
}
