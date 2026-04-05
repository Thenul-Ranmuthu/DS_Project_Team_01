package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name  string         `json:"name"`
	Email string         `json:"email"`
	Files []UploadedFile `json:"files" gorm:"foreignKey:UserID"` // one-to-many
}
