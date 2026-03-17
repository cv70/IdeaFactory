package dbdao

import (
	"time"

	"gorm.io/gorm"
)

type WorkspaceState struct {
	gorm.Model
	WorkspaceID         string     `json:"workspace_id"`
	Topic               string     `json:"topic"`
	OutputGoal          string     `json:"output_goal"`
	Constraints         string     `json:"constraints"`
	ActiveOpportunityID string     `json:"active_opportunity_id"`
	LastRunRound        int        `json:"last_run_round"`
	LastCompactedAt     time.Time  `json:"last_compacted_at"`
	ArchivedAt          *time.Time `json:"archived_at" gorm:"index"`
	Snapshot            string     `json:"snapshot" gorm:"type:text"`
}

func (d *DB) UpsertWorkspaceState(state *WorkspaceState) error {
	var existing WorkspaceState
	err := d.DB().Where("workspace_id = ?", state.WorkspaceID).First(&existing).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return d.DB().Create(state).Error
		}
		return err
	}
	state.ID = existing.ID
	return d.DB().Save(state).Error
}

func (d *DB) GetWorkspaceState(workspaceID string) (*WorkspaceState, error) {
	var state WorkspaceState
	err := d.DB().Where("workspace_id = ?", workspaceID).First(&state).Error
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

func (d *DB) ArchiveWorkspaceState(workspaceID string) error {
	now := time.Now()
	return d.DB().
		Model(&WorkspaceState{}).
		Where("workspace_id = ?", workspaceID).
		Update("archived_at", &now).Error
}
