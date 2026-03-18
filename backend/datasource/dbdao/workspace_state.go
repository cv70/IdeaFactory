package dbdao

import (
	"time"

	"gorm.io/gorm"
)

type WorkspaceState struct {
	gorm.Model
	Topic               string     `json:"topic"`
	OutputGoal          string     `json:"output_goal"`
	Constraints         string     `json:"constraints"`
	Strategy            string     `json:"strategy" gorm:"type:text"`
	Favorites           string     `json:"favorites" gorm:"type:text"`
	RunNotes            string     `json:"run_notes" gorm:"type:text"`
	ActiveOpportunityID string     `json:"active_opportunity_id"`
	LastRunRound        int        `json:"last_run_round"`
	LastCompactedAt     time.Time  `json:"last_compacted_at"`
	ArchivedAt          *time.Time `json:"archived_at" gorm:"index"`
	PausedAt            *time.Time `json:"paused_at" gorm:"index"`
}

func (d *DB) UpsertWorkspaceState(state *WorkspaceState) error {
	if state.ID == 0 {
		return d.DB().Create(state).Error
	}
	return d.DB().Save(state).Error
}

func (d *DB) GetWorkspaceState(workspaceID uint) (*WorkspaceState, error) {
	var state WorkspaceState
	err := d.DB().First(&state, workspaceID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &state, nil
}

func (d *DB) ListWorkspaceStates(limit int, includeArchived bool) ([]WorkspaceState, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := d.DB().Order("updated_at desc").Limit(limit)
	if !includeArchived {
		query = query.Where("archived_at IS NULL")
	}
	var states []WorkspaceState
	if err := query.Find(&states).Error; err != nil {
		return nil, err
	}
	return states, nil
}

func (d *DB) ArchiveWorkspaceState(workspaceID uint) error {
	now := time.Now()
	return d.DB().
		Model(&WorkspaceState{}).
		Where("id = ?", workspaceID).
		Update("archived_at", &now).Error
}

func (d *DB) PauseWorkspaceState(workspaceID uint) error {
	now := time.Now()
	return d.DB().
		Model(&WorkspaceState{}).
		Where("id = ?", workspaceID).
		Update("paused_at", &now).Error
}

func (d *DB) ResumeWorkspaceState(workspaceID uint) error {
	return d.DB().
		Model(&WorkspaceState{}).
		Where("id = ?", workspaceID).
		Update("paused_at", gorm.Expr("NULL")).Error
}

func (d *DB) ListActiveWorkspaceStates(limit int) ([]WorkspaceState, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var states []WorkspaceState
	err := d.DB().
		Where("archived_at IS NULL AND paused_at IS NULL").
		Order("updated_at desc").
		Limit(limit).
		Find(&states).Error
	return states, err
}
