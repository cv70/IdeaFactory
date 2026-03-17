package dbdao

import (
	"time"

	"gorm.io/gorm"
)

type MutationLog struct {
	gorm.Model
	WorkspaceID string `json:"workspace_id" gorm:"index"`
	Kind        string `json:"kind"`
	Source      string `json:"source"`
	Payload     string `json:"payload" gorm:"type:text"`
}

func (d *DB) CreateMutationLogs(logs []MutationLog) error {
	if len(logs) == 0 {
		return nil
	}
	return d.DB().Create(&logs).Error
}

func (d *DB) ListMutationLogs(workspaceID string, sinceUnixMs int64, limit int) ([]MutationLog, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	query := d.DB().Where("workspace_id = ?", workspaceID).Order("created_at asc").Limit(limit)
	if sinceUnixMs > 0 {
		sinceTime := time.UnixMilli(sinceUnixMs)
		query = query.Where("created_at > ?", sinceTime)
	}
	var logs []MutationLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func (d *DB) ListMutationLogsByCursor(workspaceID string, cursorTime time.Time, cursorID string, limit int) ([]MutationLog, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	query := d.DB().Where("workspace_id = ?", workspaceID).Order("created_at asc, id asc").Limit(limit)
	if !cursorTime.IsZero() {
		if cursorID != "" {
			query = query.Where("(created_at > ?) OR (created_at = ? AND id > ?)", cursorTime, cursorTime, cursorID)
		} else {
			query = query.Where("created_at > ?", cursorTime)
		}
	}
	var logs []MutationLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func (d *DB) CountMutationLogs(workspaceID string) (int64, error) {
	var count int64
	if err := d.DB().Model(&MutationLog{}).Where("workspace_id = ?", workspaceID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (d *DB) DeleteMutationLogsBefore(workspaceID string, cutoff time.Time) error {
	return d.DB().Where("workspace_id = ? AND created_at < ?", workspaceID, cutoff).Delete(&MutationLog{}).Error
}

func (d *DB) GetMutationCutoffForRecent(workspaceID string, keepRecent int) (*MutationLog, error) {
	if keepRecent <= 0 {
		keepRecent = 1
	}
	var log MutationLog
	err := d.DB().
		Where("workspace_id = ?", workspaceID).
		Order("created_at desc, id desc").
		Offset(keepRecent - 1).
		Limit(1).
		Find(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}
