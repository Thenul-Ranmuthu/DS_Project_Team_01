package models

import "time"

type IdempotencyRecord struct {
	Key        string    `gorm:"primaryKey;type:varchar(255)"`
	StatusCode int       `json:"status_code"`
	Body       string    `json:"body"`
	CreatedAt  time.Time
}
