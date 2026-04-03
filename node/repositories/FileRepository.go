package repositories

import (
	"errors"

	initializers "github.com/DS_node/Initializers"
	"github.com/DS_node/models"
	"gorm.io/gorm"
)

// Save a new file record
func CreateFile(file *models.UploadedFile) error {
	result := initializers.DB.Create(file)
	return result.Error
}

// CreateFileFromReplication inserts metadata from leader WAL replay; skips if storage_key already exists (idempotent).
func CreateFileFromReplication(file *models.UploadedFile) error {
	var existing models.UploadedFile
	err := initializers.DB.Where("storage_key = ?", file.StorageKey).First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return initializers.DB.Create(file).Error
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
