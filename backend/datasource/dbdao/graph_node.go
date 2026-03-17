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
	WorkspaceID string   `json:"workspace_id" gorm:"index"`
	SessionID   string   `json:"session_id" gorm:"index"`
	Type        NodeType `json:"node_type"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	Body        string   `json:"body"`
	Status      Status   `json:"status"`
	Metadata    string   `json:"metadata"` // JSON string
}

func (d *DB) CreateNode(node *GraphNode) error {
	return d.DB().Create(node).Error
}
