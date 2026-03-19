package dbdao

import (
	"gorm.io/gorm"
)

func (d *DB) GetWorkspaceGraph(workspaceID uint) ([]GraphNode, []GraphEdge, error) {
	var nodes []GraphNode
	if err := d.DB().Where("workspace_id = ?", workspaceID).Find(&nodes).Error; err != nil {
		return nil, nil, err
	}

	var edges []GraphEdge
	if err := d.DB().Where("workspace_id = ?", workspaceID).Find(&edges).Error; err != nil {
		return nil, nil, err
	}

	return nodes, edges, nil
}

func (d *DB) ReplaceWorkspaceGraph(workspaceID uint, nodes []GraphNode, edges []GraphEdge) error {
	tx := d.DB().Begin()
	if tx.Error != nil {
		return tx.Error
	}
	rollback := func(err error) error {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Where("workspace_id = ?", workspaceID).Delete(&GraphNode{}).Error; err != nil {
		return rollback(err)
	}
	if err := tx.Where("workspace_id = ?", workspaceID).Delete(&GraphEdge{}).Error; err != nil {
		return rollback(err)
	}
	if len(nodes) > 0 {
		if err := tx.Create(&nodes).Error; err != nil {
			return rollback(err)
		}
	}
	if len(edges) > 0 {
		if err := tx.Create(&edges).Error; err != nil {
			return rollback(err)
		}
	}
	return tx.Commit().Error
}

type RuntimeBalanceRecord struct {
	gorm.Model
	WorkspaceID        uint    `json:"workspace_id" gorm:"index"`
	RunID              string  `json:"run_id" gorm:"index"`
	Divergence         float64 `json:"divergence"`
	Research           float64 `json:"research"`
	Aggression         float64 `json:"aggression"`
	Reason             string  `json:"reason"`
	UpdatedAtMs        int64   `json:"updated_at"`
	LatestReplanReason string  `json:"latest_replan_reason"`
}
