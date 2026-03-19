package dbdao

import (
	"strings"

	"gorm.io/gorm"
)

type InterventionEvent struct {
	gorm.Model
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	Type        string    `json:"type"`
	TargetID    string    `json:"target_id"`
	Note        string    `json:"note"`
}

func (d *DB) CreateInterventionEvent(event *InterventionEvent) error {
	return d.DB().Create(event).Error
}

func (d *DB) UpsertInterventionEvent(event *InterventionEvent) error {
	return d.DB().Save(event).Error
}

func (d *DB) GetInterventionEvent(workspaceID string, id string) (*InterventionEvent, error) {
	var event InterventionEvent
	err := d.DB().
		Where("workspace_id = ? AND id = ?", workspaceID, id).
		First(&event).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &event, nil
}

func (d *DB) GetLatestInterventionEventByTarget(workspaceID string, targetID string, eventType string) (*InterventionEvent, error) {
	var event InterventionEvent
	err := d.DB().
		Where("workspace_id = ? AND target_id = ?", workspaceID, targetID).
		Where("type = ?", eventType).
		Order("created_at desc, id desc").
		First(&event).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &event, nil
}

func (d *DB) ListInterventionEventsByPrefix(workspaceID string, idPrefix string, eventType string, limit int) ([]InterventionEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := d.DB().Where("workspace_id = ?", workspaceID)
	if idPrefix != "" {
		idPrefix = strings.TrimSuffix(idPrefix, "#")
		query = query.Where("target_id LIKE ?", idPrefix+"%")
	}
	if eventType != "" {
		query = query.Where("type = ?", eventType)
	}
	var events []InterventionEvent
	if err := query.Order("created_at asc, id asc").Limit(limit).Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}
