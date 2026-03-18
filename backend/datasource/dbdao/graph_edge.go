package dbdao

import (
	"gorm.io/gorm"
)

type EdgeType string

const (
	EdgeQuestions EdgeType = "questions"
	EdgeExplains  EdgeType = "explains"
	EdgeSupports  EdgeType = "supports"
	EdgeWeakens   EdgeType = "weakens"
	EdgeLeadsTo   EdgeType = "leads_to"
)

// GraphEdge represents an edge in the exploration graph
type GraphEdge struct {
	gorm.Model
	WorkspaceID uint     `json:"workspace_id" gorm:"index"`
	SessionID   string   `json:"session_id" gorm:"index"`
	FromID      string   `json:"from_node_id"`
	ToID        string   `json:"to_node_id"`
	Type        EdgeType `json:"edge_type"`
}

func (d *DB) CreateEdge(edge *GraphEdge) error {
	return d.DB().Create(edge).Error
}
