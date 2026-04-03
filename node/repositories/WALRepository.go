package repositories

import (
	"encoding/json"
	"fmt"

	initializers "github.com/DS_node/Initializers"
	"github.com/DS_node/models"
)

// CreateWALEntry writes a PENDING record to the local WAL before performing any state change.
func CreateWALEntry(op models.WALOperation, payload any, nodeID string) (*models.WriteAheadLog, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("WAL marshal failed: %w", err)
	}
	entry := &models.WriteAheadLog{
		Operation: op,
		Payload:   string(data),
		Status:    models.WALStatusPending,
		NodeID:    nodeID,
	}
	if result := initializers.DB.Create(entry); result.Error != nil {
		return nil, fmt.Errorf("WAL insert failed: %w", result.Error)
	}
	return entry, nil
}

// MarkWALCompleted updates a WAL entry to COMPLETED after the write succeeds.
func MarkWALCompleted(logID uint64) {
	initializers.DB.Model(&models.WriteAheadLog{}).
		Where("log_id = ?", logID).
		Updates(map[string]any{"status": models.WALStatusCompleted})
}

// MarkWALFailed updates a WAL entry to FAILED if the write operation could not complete.
func MarkWALFailed(logID uint64) {
	initializers.DB.Model(&models.WriteAheadLog{}).
		Where("log_id = ?", logID).
		Updates(map[string]any{"status": models.WALStatusFailed})
}

// GetCompletedWALAfter returns all COMPLETED WAL entries with log_id > afterLogID,
// ordered ascending so followers can replay in the exact original order.
func GetCompletedWALAfter(afterLogID uint64, limit int) ([]models.WriteAheadLog, error) {
	var entries []models.WriteAheadLog
	result := initializers.DB.
		Where("log_id > ? AND status = ?", afterLogID, models.WALStatusCompleted).
		Order("log_id ASC").
		Limit(limit).
		Find(&entries)
	return entries, result.Error
}
