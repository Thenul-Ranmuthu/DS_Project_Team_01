package models

import "gorm.io/gorm"

type ReplicationType string

const (
	ReplicateFileUpload ReplicationType = "FILE_UPLOAD"
	ReplicateFileDelete ReplicationType = "FILE_DELETE"
	ReplicateUserCreate ReplicationType = "USER_CREATE"
)

type PendingReplication struct {
	gorm.Model
	Type          ReplicationType `json:"type"`
	TargetPeer    string          `json:"target_peer"`
	Payload       string          `json:"payload"`   // JSON encoded metadata or filename
	FilePath      string          `json:"file_path"` // For file uploads
	Attempts      int             `json:"attempts"`
	LastAttemptAt int64           `json:"last_attempt_at"`
	Status        string          `json:"status"` // "pending", "failed"
}
