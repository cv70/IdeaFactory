package dbdao

import "gorm.io/gorm"

type AgentRunRecord struct {
	gorm.Model
	WorkspaceID uint   `json:"workspace_id" gorm:"index"`
	RunID       string `json:"run_id" gorm:"index"`
	RootAgent   string `json:"root_agent"`
	EventType   string `json:"event_type" gorm:"index"`
	Actor       string `json:"actor"`
	Target      string `json:"target"`
	Summary     string `json:"summary"`
	Payload     string `json:"payload" gorm:"type:text"`
}

func (d *DB) AppendAgentRunRecords(records []AgentRunRecord) error {
	if len(records) == 0 {
		return nil
	}
	return d.DB().Create(&records).Error
}

func (d *DB) ListAgentRunRecords(workspaceID uint) ([]AgentRunRecord, error) {
	var records []AgentRunRecord
	if err := d.DB().
		Where("workspace_id = ?", workspaceID).
		Order("created_at asc, id asc").
		Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}
