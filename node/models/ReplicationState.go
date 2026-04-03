package models

// ReplicationState stores how far this node has applied the leader's WAL via replication.
// Survives process restarts so followers do not replay from zero (duplicate metadata / heavy load).
type ReplicationState struct {
	ID                 uint   `gorm:"primaryKey"`
	LastLeaderWALLogID uint64 `gorm:"column:last_leader_wal_log_id"`
}
