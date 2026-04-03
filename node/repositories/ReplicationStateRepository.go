package repositories

import (
	initializers "github.com/DS_node/Initializers"
	"github.com/DS_node/models"
)

const replicationStateSingletonID uint = 1

// GetLastAppliedLeaderWALID returns the highest leader log_id successfully applied locally (0 if none).
func GetLastAppliedLeaderWALID() uint64 {
	var row models.ReplicationState
	if err := initializers.DB.First(&row, replicationStateSingletonID).Error; err != nil {
		return 0
	}
	return row.LastLeaderWALLogID
}

// SetLastAppliedLeaderWALID persists the replication watermark after a successful sync batch.
func SetLastAppliedLeaderWALID(logID uint64) error {
	var row models.ReplicationState
	res := initializers.DB.First(&row, replicationStateSingletonID)
	if res.Error != nil {
		return initializers.DB.Create(&models.ReplicationState{
			ID:                 replicationStateSingletonID,
			LastLeaderWALLogID: logID,
		}).Error
	}
	row.LastLeaderWALLogID = logID
	return initializers.DB.Save(&row).Error
}
