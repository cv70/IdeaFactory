package exploration

type NodeType string

const (
	NodeTopic       NodeType = "topic"
	NodeQuestion    NodeType = "question"
	NodeTension     NodeType = "tension"
	NodeHypothesis  NodeType = "hypothesis"
	NodeOpportunity NodeType = "opportunity"
	NodeIdea        NodeType = "idea"
	NodeEvidence    NodeType = "evidence"
	NodeResearch    NodeType = "research"
	NodeClaim       NodeType = "claim"
	NodeDecision    NodeType = "decision"
	NodeUnknown     NodeType = "unknown"
)

type EdgeType string

const (
	EdgeSupports    EdgeType = "supports"
	EdgeRefines     EdgeType = "refines"
	EdgeLeadsTo     EdgeType = "leads_to"
	EdgeExpands     EdgeType = "expands"
	EdgeContradicts EdgeType = "contradicts"
	EdgeQuestions   EdgeType = "questions"
	EdgeExplains    EdgeType = "explains"
	EdgeWeakens     EdgeType = "weakens"
)

type NodeStatus string

const (
	NodeActive   NodeStatus = "active"
	NodeArchived NodeStatus = "archived"
)

type Evidence struct {
	ID        string `json:"id"`
	Source    string `json:"source"` // e.g., "web_search", "user_document"
	URL       string `json:"url,omitempty"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

type Decision struct {
	ID        string   `json:"id"`
	Reason    string   `json:"reason"`
	Evidence  []string `json:"evidence_ids"` // References to Evidence IDs
	Timestamp int64    `json:"timestamp"`
}

type NodeMetadata struct {
	BranchID string `json:"branchId,omitempty"`
	Slot     string `json:"slot,omitempty"`
	Cluster  string `json:"cluster,omitempty"` // For direction map grouping
}

type Node struct {
	ID              string       `json:"id" gorm:"primaryKey"`
	WorkspaceID     string       `json:"workspace_id" gorm:"index"`
	SessionID       string       `json:"sessionId"` // Internal/Runtime ID
	Type            NodeType     `json:"type"`
	Title           string       `json:"title"`
	Summary         string       `json:"summary"`
	Status          NodeStatus   `json:"status"`
	Score           float64      `json:"score"`
	Depth           int          `json:"depth"`
	ParentContext   string       `json:"parentContext,omitempty"`
	Metadata        NodeMetadata `json:"metadata" gorm:"type:text"` // GORM should handle JSON as text or similar
	EvidenceSummary string       `json:"evidenceSummary"`
	Decision        *Decision    `json:"decision,omitempty" gorm:"type:text"`
}

type Edge struct {
	ID          string   `json:"id" gorm:"primaryKey"`
	WorkspaceID string   `json:"workspace_id" gorm:"index"`
	From        string   `json:"from"`
	To          string   `json:"to"`
	Type        EdgeType `json:"type"`
}

type GenerationRun struct {
	ID      string `json:"id" gorm:"primaryKey"`
	Round   int    `json:"round"`
	Focus   string `json:"focus"`
	Summary string `json:"summary"`
}

type RuntimeStrategy struct {
	IntervalMs        int    `json:"interval_ms"`
	MaxRuns           int    `json:"max_runs"`
	ExpansionMode     string `json:"expansion_mode"`
	PreferredBranchID string `json:"preferred_branch_id,omitempty"`
}

type ExplorationSession struct {
	ID                  string          `json:"id" gorm:"primaryKey"`
	Topic               string          `json:"topic"`
	OutputGoal          string          `json:"outputGoal"`
	Constraints         string          `json:"constraints"`
	Strategy            RuntimeStrategy `json:"strategy" gorm:"type:text"`
	ActiveOpportunityID string          `json:"activeOpportunityId"`
	Nodes               []Node          `json:"nodes" gorm:"-"` // Not directly persisted in session table
	Edges               []Edge          `json:"edges" gorm:"-"` // Not directly persisted in session table
	Favorites           []string        `json:"favorites" gorm:"type:text"`
	Runs                []GenerationRun `json:"runs" gorm:"-"`
}

// DirectionMapProjection is the graph-first projection for the frontend.
type DirectionMapProjection struct {
	WorkspaceID string `json:"workspaceId"`
	Nodes       []Node `json:"nodes"`
	Edges       []Edge `json:"edges"`
}

// WorkbenchProjection represents the specific workbench view.
type WorkbenchProjection struct {
	Opportunities     []Node          `json:"opportunities"`
	ActiveOpportunity Node            `json:"activeOpportunity"`
	QuestionTrail     []Node          `json:"questionTrail"`
	HypothesisTrail   []Node          `json:"hypothesisTrail"`
	IdeaCards         []Node          `json:"ideaCards"`
	SavedIdeas        []Node          `json:"savedIdeas"`
	RunNotes          []GenerationRun `json:"runNotes"`
}

type WorkspaceSnapshot struct {
	Exploration  ExplorationSession     `json:"exploration"`
	DirectionMap DirectionMapProjection `json:"directionMap"`
	Workbench    WorkbenchProjection    `json:"workbench"`
}

type MutationEvent struct {
	ID                  string        `json:"id"`
	WorkspaceID         string        `json:"workspace_id"`
	Kind                string        `json:"kind"`
	Source              string        `json:"source"`
	Node                *Node          `json:"node,omitempty"`
	Edge                *Edge          `json:"edge,omitempty"`
	Run                 *GenerationRun `json:"run,omitempty"`
	Favorites           []string      `json:"favorites,omitempty"`
	ActiveOpportunityID string        `json:"active_opportunity_id,omitempty"`
	Strategy            *RuntimeStrategy `json:"strategy,omitempty"`
	CreatedAt           int64         `json:"created_at"`
}

type CreateWorkspaceReq struct {
	Topic       string           `json:"topic" binding:"required"`
	OutputGoal  string           `json:"output_goal"`
	Constraints string `json:"constraints"`
	Strategy    *RuntimeStrategy `json:"strategy"`
}

type InterventionType string

const (
	InterventionExpandOpportunity InterventionType = "expand_opportunity"
	InterventionToggleFavorite    InterventionType = "toggle_favorite"
	InterventionShiftFocus        InterventionType = "shift_focus"
	InterventionAdjustIntensity   InterventionType = "adjust_intensity"
	InterventionAddContext        InterventionType = "add_context"
)

type Intervention struct {
	ID          string           `json:"id" gorm:"primaryKey"`
	WorkspaceID string           `json:"workspace_id" gorm:"index"`
	Type        InterventionType `json:"type"`
	TargetID    string           `json:"target_id"`
	Note        string           `json:"note"`
	Status      string           `json:"status"` // e.g., "pending", "absorbed"
	CreatedAt   int64            `json:"created_at"`
}

type InterventionReq struct {
	Type     InterventionType `json:"type" binding:"required"`
	TargetID string           `json:"target_id" binding:"required"`
	Note     string           `json:"note"`
}

// Backward-compatible request with original frontend payload shape.
type FeedbackReq struct {
	Type   string `json:"type" binding:"required"`
	NodeID string `json:"nodeId" binding:"required"`
}

type CreateSessionReq struct {
	WorkspaceID string `json:"workspace_id"`
	Topic       string `json:"topic" binding:"required"`
	OutputGoal  string `json:"output_goal"`
	Constraints string `json:"constraints"`
}

type UpdateStrategyReq struct {
	IntervalMs        *int    `json:"interval_ms"`
	MaxRuns           *int    `json:"max_runs"`
	ExpansionMode     *string `json:"expansion_mode"`
	PreferredBranchID *string `json:"preferred_branch_id"`
}

type WorkspaceSummary struct {
	ID         string `json:"id"`
	Topic      string `json:"topic"`
	OutputGoal string `json:"output_goal"`
	UpdatedAt  int64  `json:"updated_at"`
}
