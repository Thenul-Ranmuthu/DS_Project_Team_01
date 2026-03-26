package models

import "gorm.io/gorm"

type ElectionEvent struct {
	gorm.Model
	NodeID    string `json:"node_id"`
	EventType string `json:"event_type"` // "became_leader" | "lost_leadership"
	Term      int    `json:"term"`
}
