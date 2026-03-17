package dbdao

import (
	"time"

	"gorm.io/gorm"
)

// ExplorationSession represents a single exploration context
type ExplorationSession struct {
	gorm.Model
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	Topic       string    `json:"topic"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (d *DB) CreateSession(session *ExplorationSession) error {
	return d.DB().Create(session).Error
}
