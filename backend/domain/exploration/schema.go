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
)

type EdgeType string

const (
	EdgeSupports EdgeType = "supports"
	EdgeRefines  EdgeType = "refines"
	EdgeLeadsTo  EdgeType = "leads_to"
	EdgeExpands  EdgeType = "expands"
)

type NodeStatus string

const (
	NodeActive NodeStatus = "active"
)

type NodeMetadata struct {
	BranchID string `json:"branchId,omitempty"`
	Slot     string `json:"slot,omitempty"`
}

type Node struct {
	ID              string      `json:"id"`
	SessionID       string      `json:"sessionId"`
	Type            NodeType    `json:"type"`
	Title           string      `json:"title"`
	Summary         string      `json:"summary"`
	Status          NodeStatus  `json:"status"`
	Score           float64     `json:"score"`
	Depth           int         `json:"depth"`
	ParentContext   string      `json:"parentContext,omitempty"`
	Metadata        NodeMetadata `json:"metadata"`
	EvidenceSummary string      `json:"evidenceSummary"`
}

type Edge struct {
	ID   string   `json:"id"`
	From string   `json:"from"`
	To   string   `json:"to"`
	Type EdgeType `json:"type"`
}

type GenerationRun struct {
	ID      string `json:"id"`
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
	ID                  string          `json:"id"`
	Topic               string          `json:"topic"`
	OutputGoal          string          `json:"outputGoal"`
	Constraints         string          `json:"constraints"`
	Strategy            RuntimeStrategy `json:"strategy"`
	ActiveOpportunityID string          `json:"activeOpportunityId"`
	Nodes               []Node          `json:"nodes"`
	Edges               []Edge          `json:"edges"`
	Favorites           []string        `json:"favorites"`
	Runs                []GenerationRun `json:"runs"`
}

type WorkbenchView struct {
	Opportunities   []Node          `json:"opportunities"`
	ActiveOpportunity Node          `json:"activeOpportunity"`
	QuestionTrail   []Node          `json:"questionTrail"`
	HypothesisTrail []Node          `json:"hypothesisTrail"`
	IdeaCards       []Node          `json:"ideaCards"`
	SavedIdeas      []Node          `json:"savedIdeas"`
	RunNotes        []GenerationRun `json:"runNotes"`
}

type WorkspaceSnapshot struct {
	Exploration  ExplorationSession `json:"exploration"`
	Presentation WorkbenchView      `json:"presentation"`
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
)

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
