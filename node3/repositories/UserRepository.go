package repositories

import (
	initializers "github.com/DS_node/Initializers"
	"github.com/DS_node/models"
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
