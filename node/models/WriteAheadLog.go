package models

import "time"

// WALStatus represents the lifecycle state of a WAL entry.
type WALStatus string

const (
	WALStatusPending   WALStatus = "PENDING"
	WALStatusCompleted WALStatus = "COMPLETED"
	WALStatusFailed    WALStatus = "FAILED"
)

// WALOperation represents the type of write operation being logged.
type WALOperation string

const (
	WALOpUpload     WALOperation = "UPLOAD"
	WALOpDelete     WALOperation = "DELETE"
	WALOpCreateUser WALOperation = "CREATE_USER"
)

// WriteAheadLog is a durable operation record written before any state change.
// Leaders write entries here; followers replay COMPLETED entries to sync metadata.
type WriteAheadLog struct {
	LogID     uint64       `gorm:"primaryKey;autoIncrement" json:"log_id"`
	Operation WALOperation `gorm:"type:varchar(50);not null" json:"operation"`
	// Payload holds JSON-encoded data specific to the operation (e.g. file record, user record).
	Payload   string    `gorm:"type:text;not null" json:"payload"`
	Status    WALStatus `gorm:"type:varchar(20);not null;default:'PENDING'" json:"status"`
	NodeID    string    `gorm:"type:varchar(100);not null" json:"node_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
