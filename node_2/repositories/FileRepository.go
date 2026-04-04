package repositories

import (
	initializers "github.com/DS_node/Initializers"
	"github.com/DS_node/models"
)

// Save a new file record
func CreateFile(file *models.UploadedFile) error {
	result := initializers.DB.Create(file)
	return result.Error
}

// Get all files belonging to a user
func GetFilesByUser(userID uint) ([]models.UploadedFile, error) {
	var files []models.UploadedFile
	result := initializers.DB.Where("user_id = ?", userID).Find(&files)
	return files, result.Error
}

// Get a single file by ID
func GetFileByID(fileID uint) (models.UploadedFile, error) {
	var file models.UploadedFile
	result := initializers.DB.First(&file, fileID)
	return file, result.Error
}

// Delete a file record by ID
func DeleteFile(fileID uint) error {
	result := initializers.DB.Unscoped().Delete(&models.UploadedFile{}, fileID)
	return result.Error
}
