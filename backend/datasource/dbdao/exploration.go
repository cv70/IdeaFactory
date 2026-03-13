package dbdao


import "time"

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

type GraphNode struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	SessionID   string    `json:"session_id" gorm:"index"`
	Type        NodeType  `json:"node_type"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	Body        string    `json:"body"`
	Status      Status    `json:"status"`
	Metadata    string    `json:"metadata"` // JSON string
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type EdgeType string

const (
	EdgeQuestions EdgeType = "questions"
	EdgeExplains  EdgeType = "explains"
	EdgeSupports  EdgeType = "supports"
	EdgeWeakens   EdgeType = "weakens"
	EdgeLeadsTo   EdgeType = "leads_to"
)

type GraphEdge struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	SessionID string    `json:"session_id" gorm:"index"`
	FromID    string    `json:"from_node_id"`
	ToID      string    `json:"to_node_id"`
	Type      EdgeType  `json:"edge_type"`
	CreatedAt time.Time `json:"created_at"`
}

// ExplorationSession represents a single exploration context
type ExplorationSession struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	Topic       string    `json:"topic"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (d *DB) CreateSession(session *ExplorationSession) error {
	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()
	return d.DB().Create(session).Error
}

func (d *DB) CreateNode(node *GraphNode) error {
	node.CreatedAt = time.Now()
	node.UpdatedAt = time.Now()
	return d.DB().Create(node).Error
}

func (d *DB) CreateEdge(edge *GraphEdge) error {
	edge.CreatedAt = time.Now()
	return d.DB().Create(edge).Error
}

func (d *DB) GetSessionGraph(sessionID string) ([]GraphNode, []GraphEdge, error) {
	var nodes []GraphNode
	if err := d.DB().Where("session_id = ?", sessionID).Find(&nodes).Error; err != nil {
		return nil, nil, err
	}

	var edges []GraphEdge
	if err := d.DB().Where("session_id = ?", sessionID).Find(&edges).Error; err != nil {
		return nil, nil, err
	}

	return nodes, edges, nil
}
