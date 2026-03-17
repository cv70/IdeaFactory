package dbdao

import (
	"gorm.io/gorm"
)

type WorkspaceRuntimeState struct {
	gorm.Model
	WorkspaceID string `json:"workspace_id"`
	Snapshot    string `json:"snapshot" gorm:"type:text"`
}

func (d *DB) UpsertWorkspaceRuntimeState(state *WorkspaceRuntimeState) error {
	return d.DB().Save(state).Error
}

func (d *DB) GetWorkspaceRuntimeState(workspaceID string) (*WorkspaceRuntimeState, error) {
	var state WorkspaceRuntimeState
	err := d.DB().Where("workspace_id = ?", workspaceID).First(&state).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &state, nil
}
