package dbdao

import (
	"gorm.io/gorm"
)

type NodeType string

const (
	NodeTopic       NodeType = "Topic"
	NodeQuestion    NodeType = "Question"
	NodeTension     NodeType = "Tension"
	NodeHypothesis  NodeType = "Hypothesis"
	NodeOpportunity NodeType = "Opportunity"
	NodeIdea        NodeType = "Idea"
	NodeEvidence    NodeType = "Evidence"
	NodeClaim       NodeType = "Claim"
	NodeDecision    NodeType = "Decision"
	NodeUnknown     NodeType = "Unknown"
)

type Status string

const (
	StatusActive Status = "active"
)

// GraphNode represents a node in the exploration graph
type GraphNode struct {
	gorm.Model
	WorkspaceID uint     `json:"workspace_id" gorm:"index"`
	SessionID   string   `json:"session_id" gorm:"index"`
	NodeID      string   `json:"node_id" gorm:"index"`
	Type        NodeType `json:"node_type"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	Body        string   `json:"body"`
	Status      Status   `json:"status"`
	Score       float64  `json:"score"`
	Depth       int      `json:"depth"`
	Metadata    string   `json:"metadata"` // JSON string
	Evidence    string   `json:"evidence" gorm:"type:text"`
	Decision    string   `json:"decision" gorm:"type:text"`
}

func (d *DB) CreateNode(node *GraphNode) error {
	return d.DB().Create(node).Error
}
