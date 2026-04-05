package models

import "gorm.io/gorm"

type UploadedFile struct {
	gorm.Model
	OriginalName string `json:"original_name"`
	FilePath     string `json:"file_path"`
	MimeType     string `json:"mime_type"`
	FileSize     int64  `json:"file_size"`
	UserID       uint   `json:"user_id"`                       // foreign key
	User         User   `json:"user" gorm:"foreignKey:UserID"` // belongs-to
}
