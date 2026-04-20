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
	NodeDirection   NodeType = "direction"
	NodeArtifact    NodeType = "artifact"
)

type EdgeType string

const (
	EdgeSupports     EdgeType = "supports"
	EdgeRefines      EdgeType = "refines"
	EdgeLeadsTo      EdgeType = "leads_to"
	EdgeExpands      EdgeType = "expands"
	EdgeContradicts  EdgeType = "contradicts"
	EdgeQuestions    EdgeType = "questions"
	EdgeExplains     EdgeType = "explains"
	EdgeWeakens      EdgeType = "weakens"
	EdgeJustifies    EdgeType = "justifies"
	EdgeBranchesFrom EdgeType = "branches_from"
	EdgeRaises       EdgeType = "raises"
	EdgeResolves     EdgeType = "resolves"
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
	ID              string       `json:"id"`
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
	ID          string   `json:"id"`
	WorkspaceID string   `json:"workspace_id" gorm:"index"`
	From        string   `json:"from"`
	To          string   `json:"to"`
	Type        EdgeType `json:"type"`
}

type GenerationRun struct {
	ID      string `json:"id"`
	Round   int    `json:"round"`
	Focus   string `json:"focus"`
	Summary string `json:"summary"`
}

type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusWaiting   RunStatus = "waiting"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
)

type RunMode string

const (
	RunModeExplore  RunMode = "explore"
	RunModeReview   RunMode = "review"
	RunModeArtifact RunMode = "artifact"
	RunModeResume   RunMode = "resume"
)

type Run struct {
	ID                 string    `json:"id"`
	WorkspaceID        string    `json:"workspace_id"`
	Source             string    `json:"source"`
	Mode               RunMode   `json:"mode,omitempty"`
	Status             RunStatus `json:"status"`
	WaitingReason      string    `json:"waiting_reason,omitempty"`
	LatestCheckpointID string    `json:"latest_checkpoint_id,omitempty"`
	StartedAt          int64     `json:"started_at"`
	EndedAt            int64     `json:"ended_at,omitempty"`
}

type RuntimeTaskStatus string

const (
	RuntimeTaskTodo    RuntimeTaskStatus = "todo"
	RuntimeTaskDoing   RuntimeTaskStatus = "doing"
	RuntimeTaskDone    RuntimeTaskStatus = "done"
	RuntimeTaskFailed  RuntimeTaskStatus = "failed"
	RuntimeTaskSkipped RuntimeTaskStatus = "skipped"
)

type AgentTask struct {
	ID          string            `json:"id"`
	WorkspaceID string            `json:"workspace_id"`
	RunID       string            `json:"run_id"`
	SubAgent    string            `json:"sub_agent"`
	Goal        string            `json:"goal"`
	Status      RuntimeTaskStatus `json:"status"`
	UpdatedAt   int64             `json:"updated_at"`
}

type AgentTaskResultSummary struct {
	TaskID    string   `json:"task_id"`
	Summary   string   `json:"summary"`
	Timeline  []string `json:"timeline,omitempty"`
	IsSuccess bool     `json:"is_success"`
	UpdatedAt int64    `json:"updated_at"`
}

type AgentRunEvent struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspace_id"`
	RunID       string         `json:"run_id"`
	RootAgent   string         `json:"root_agent"`
	EventType   string         `json:"event_type"`
	Actor       string         `json:"actor"`
	Target      string         `json:"target,omitempty"`
	Summary     string         `json:"summary"`
	Payload     map[string]any `json:"payload,omitempty"`
	CreatedAt   int64          `json:"created_at"`
}

type BalanceState struct {
	WorkspaceID string  `json:"workspace_id"`
	RunID       string  `json:"run_id"`
	Divergence  float64 `json:"divergence"`
	Research    float64 `json:"research"`
	Aggression  float64 `json:"aggression"`
	Reason      string  `json:"reason"`
	UpdatedAt   int64   `json:"updated_at"`
}

type RuntimeStateSnapshot struct {
	Runs               []Run                    `json:"runs"`
	AgentTasks         []AgentTask              `json:"agent_tasks"`
	Results            []AgentTaskResultSummary `json:"results"`
	Events             []AgentRunEvent          `json:"events,omitempty"`
	Turns              []RunTurn                `json:"turns,omitempty"`
	Checkpoints        []RunCheckpoint          `json:"checkpoints,omitempty"`
	Mutations          []MutationEvent          `json:"mutations,omitempty"`
	ControlActions     []ControlActionView      `json:"control_actions,omitempty"`
	LoadedSkills       []string                 `json:"loaded_skills,omitempty"`
	ActiveMemories     []string                 `json:"active_memories,omitempty"`
	Balance            BalanceState             `json:"balance"`
	LatestReplanReason string                   `json:"latest_replan_reason,omitempty"`
}

type RuntimeStateQuery struct {
	RunID      string `json:"run_id,omitempty"`
	LatestRuns int    `json:"latest_runs,omitempty"`
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
	Opportunities      []Node          `json:"opportunities"`
	ActiveOpportunity  Node            `json:"activeOpportunity"`
	QuestionTrail      []Node          `json:"questionTrail"`
	HypothesisTrail    []Node          `json:"hypothesisTrail"`
	IdeaCards          []Node          `json:"ideaCards"`
	SavedIdeas         []Node          `json:"savedIdeas"`
	RunNotes           []GenerationRun `json:"runNotes"`
	CurrentFocus       string          `json:"currentFocus,omitempty"`
	LatestChange       string          `json:"latestChange,omitempty"`
	LatestRunStatus    string          `json:"latestRunStatus,omitempty"`
	LatestReplanReason string          `json:"latestReplanReason,omitempty"`
}

type WorkspaceSnapshot struct {
	Exploration  ExplorationSession     `json:"exploration"`
	DirectionMap DirectionMapProjection `json:"directionMap"`
	Workbench    WorkbenchProjection    `json:"workbench"`
}

type MutationEvent struct {
	ID                  string           `json:"id"`
	WorkspaceID         string           `json:"workspace_id"`
	Kind                string           `json:"kind"`
	Source              string           `json:"source"`
	Node                *Node            `json:"node,omitempty"`
	Edge                *Edge            `json:"edge,omitempty"`
	Run                 *GenerationRun   `json:"run,omitempty"`
	Favorites           []string         `json:"favorites,omitempty"`
	ActiveOpportunityID string           `json:"active_opportunity_id,omitempty"`
	Strategy            *RuntimeStrategy `json:"strategy,omitempty"`
	CreatedAt           int64            `json:"created_at"`
}

type CreateWorkspaceReq struct {
	Topic       string           `json:"topic" binding:"required"`
	OutputGoal  string           `json:"output_goal"`
	Constraints string           `json:"constraints"`
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
	ID          string           `json:"id"`
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

type PatchWorkspaceReq struct {
	Status string `json:"status" binding:"required"`
}

type WorkspaceSummary struct {
	ID         string `json:"id"`
	Topic      string `json:"topic"`
	OutputGoal string `json:"output_goal"`
	UpdatedAt  int64  `json:"updated_at"`
}

type WorkspaceStatus string

const (
	WorkspaceStatusDraft    WorkspaceStatus = "draft"
	WorkspaceStatusActive   WorkspaceStatus = "active"
	WorkspaceStatusPaused   WorkspaceStatus = "paused"
	WorkspaceStatusArchived WorkspaceStatus = "archived"
)

type WorkspaceView struct {
	ID          string          `json:"id"`
	Topic       string          `json:"topic"`
	Goal        string          `json:"goal"`
	Constraints []string        `json:"constraints,omitempty"`
	Status      WorkspaceStatus `json:"status"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

type WorkspaceResponse struct {
	Workspace WorkspaceView `json:"workspace"`
}

type RunView struct {
	ID                 string `json:"id"`
	WorkspaceID        string `json:"workspace_id"`
	TriggerType        string `json:"trigger_type"`
	Mode               string `json:"mode,omitempty"`
	Status             string `json:"status"`
	WaitingReason      string `json:"waiting_reason,omitempty"`
	StartedAt          string `json:"started_at"`
	FinishedAt         string `json:"finished_at,omitempty"`
	TurnCount          int    `json:"turn_count,omitempty"`
	LatestTurnID       string `json:"latest_turn_id,omitempty"`
	LatestCheckpointID string `json:"latest_checkpoint_id,omitempty"`
	ResumeCursor       string `json:"resume_cursor,omitempty"`
}

type RunResponse struct {
	Run RunView `json:"run"`
}

type ProjectionMap struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type RunSummaryView struct {
	RunID  string `json:"run_id,omitempty"`
	Status string `json:"status,omitempty"`
	Mode   string `json:"mode,omitempty"`
	Focus  string `json:"focus,omitempty"`
}

type ControlEffectView struct {
	ControlActionID string `json:"control_action_id"`
	Kind            string `json:"kind,omitempty"`
	EffectSummary   string `json:"effect_summary"`
}

type TurnSummaryView struct {
	TurnID         string `json:"turn_id,omitempty"`
	Index          int    `json:"index,omitempty"`
	Status         string `json:"status,omitempty"`
	ContinueReason string `json:"continue_reason,omitempty"`
}

type ProjectionView struct {
	WorkspaceID    string              `json:"workspace_id"`
	EventID        string              `json:"event_id"`
	GeneratedAt    string              `json:"generated_at"`
	Map            ProjectionMap       `json:"map"`
	RecentChanges  []ProjectionChange  `json:"recent_changes,omitempty"`
	RunSummary     RunSummaryView      `json:"run_summary,omitempty"`
	TurnSummary    TurnSummaryView     `json:"turn_summary,omitempty"`
	ControlEffects []ControlEffectView `json:"control_effects,omitempty"`
}

type ProjectionChange struct {
	Type     string   `json:"type"`
	Summary  string   `json:"summary"`
	Timeline []string `json:"timeline,omitempty"`
}

type ProjectionResponse struct {
	Projection ProjectionView `json:"projection"`
}

type CreateRunRequest struct {
	Trigger string `json:"trigger"`
	Notes   string `json:"notes"`
}

type CreateInterventionRequest struct {
	Intent         string `json:"intent" binding:"required"`
	TargetBranchID string `json:"target_branch_id"`
	Priority       string `json:"priority"`
}

type ControlActionKind string

const (
	ControlActionIntervention     ControlActionKind = "intervention"
	ControlActionReviewRequest    ControlActionKind = "review_request"
	ControlActionArtifactRequest  ControlActionKind = "artifact_request"
	ControlActionResumeRequest    ControlActionKind = "resume_request"
	ControlActionPolicyAdjustment ControlActionKind = "policy_adjustment"
	ControlActionMemoryPin        ControlActionKind = "memory_pin"
)

type ControlActionStatus string

const (
	ControlActionReceived  ControlActionStatus = "received"
	ControlActionAbsorbed  ControlActionStatus = "absorbed"
	ControlActionReflected ControlActionStatus = "reflected"
	ControlActionRejected  ControlActionStatus = "rejected"
)

type ControlActionPriority string

const (
	ControlActionPriorityLow    ControlActionPriority = "low"
	ControlActionPriorityNormal ControlActionPriority = "normal"
	ControlActionPriorityHigh   ControlActionPriority = "high"
)

type CreateControlActionRequest struct {
	Kind           ControlActionKind     `json:"kind" binding:"required"`
	Intent         string                `json:"intent"`
	TargetBranchID string                `json:"target_branch_id"`
	CheckpointID   string                `json:"checkpoint_id"`
	Priority       ControlActionPriority `json:"priority"`
	Payload        map[string]any        `json:"payload"`
}

type ControlActionView struct {
	ID               string                `json:"id"`
	WorkspaceID      string                `json:"workspace_id"`
	Kind             ControlActionKind     `json:"kind"`
	Intent           string                `json:"intent,omitempty"`
	Status           ControlActionStatus   `json:"status"`
	Priority         ControlActionPriority `json:"priority,omitempty"`
	TargetBranchID   string                `json:"target_branch_id,omitempty"`
	AbsorbedByRunID  string                `json:"absorbed_by_run_id,omitempty"`
	ReflectedEventID string                `json:"reflected_event_id,omitempty"`
	CreatedAt        string                `json:"created_at"`
	UpdatedAt        string                `json:"updated_at"`
}

type ControlActionResponse struct {
	ControlAction ControlActionView `json:"control_action"`
}

type ControlActionEventView struct {
	ID              string              `json:"id"`
	ControlActionID string              `json:"control_action_id"`
	WorkspaceID     string              `json:"workspace_id"`
	Status          ControlActionStatus `json:"status"`
	Summary         string              `json:"summary,omitempty"`
	CreatedAt       string              `json:"created_at"`
}

type ControlActionEventsResponse struct {
	WorkspaceID     string                   `json:"workspace_id"`
	ControlActionID string                   `json:"control_action_id"`
	Events          []ControlActionEventView `json:"events"`
	NextCursor      string                   `json:"next_cursor,omitempty"`
	HasMore         bool                     `json:"has_more"`
}

type InterventionLifecycleStatus string

const (
	InterventionReceived  InterventionLifecycleStatus = "received"
	InterventionAbsorbed  InterventionLifecycleStatus = "absorbed"
	InterventionReplanned InterventionLifecycleStatus = "replanned"
	InterventionReflected InterventionLifecycleStatus = "reflected"
)

type InterventionView struct {
	ID               string                      `json:"id"`
	WorkspaceID      string                      `json:"workspace_id"`
	Intent           string                      `json:"intent"`
	Status           InterventionLifecycleStatus `json:"status"`
	AbsorbedByRunID  string                      `json:"absorbed_by_run_id,omitempty"`
	ReflectedEventID string                      `json:"reflected_event_id,omitempty"`
	CreatedAt        string                      `json:"created_at"`
	UpdatedAt        string                      `json:"updated_at"`
}

type InterventionResponse struct {
	Intervention InterventionView `json:"intervention"`
}

type TraceSummaryItem struct {
	ID         string   `json:"id,omitempty"`
	Timestamp  string   `json:"timestamp"`
	Level      string   `json:"level"`
	Category   string   `json:"category"`
	Message    string   `json:"message"`
	RelatedIDs []string `json:"related_ids,omitempty"`
}

type TraceSummaryResponse struct {
	WorkspaceID string             `json:"workspace_id"`
	RunID       string             `json:"run_id,omitempty"`
	Items       []TraceSummaryItem `json:"items"`
}

type TraceEventsResponse struct {
	WorkspaceID string             `json:"workspace_id"`
	RunID       string             `json:"run_id,omitempty"`
	Items       []TraceSummaryItem `json:"items"`
	Events      []AgentRunEvent    `json:"events,omitempty"`
	NextCursor  string             `json:"next_cursor,omitempty"`
	HasMore     bool               `json:"has_more"`
}

type InterventionEventView struct {
	ID             string                      `json:"id"`
	InterventionID string                      `json:"intervention_id"`
	WorkspaceID    string                      `json:"workspace_id"`
	Status         InterventionLifecycleStatus `json:"status"`
	CreatedAt      string                      `json:"created_at"`
}

type InterventionEventsResponse struct {
	WorkspaceID    string                  `json:"workspace_id"`
	InterventionID string                  `json:"intervention_id"`
	Events         []InterventionEventView `json:"events"`
	NextCursor     string                  `json:"next_cursor,omitempty"`
	HasMore        bool                    `json:"has_more"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

type RunTurnStatus string

const (
	RunTurnStatusRunning   RunTurnStatus = "running"
	RunTurnStatusCompleted RunTurnStatus = "completed"
	RunTurnStatusFailed    RunTurnStatus = "failed"
)

type RunTurn struct {
	ID                 string        `json:"id"`
	WorkspaceID        string        `json:"workspace_id"`
	RunID              string        `json:"run_id"`
	TurnIndex          int           `json:"turn_index"`
	Status             RunTurnStatus `json:"status"`
	InputContextDigest string        `json:"input_context_digest,omitempty"`
	ToolCallCount      int           `json:"tool_call_count,omitempty"`
	GraphMutationCount int           `json:"graph_mutation_count,omitempty"`
	ContinueReason     string        `json:"continue_reason,omitempty"`
	StartedAt          int64         `json:"started_at"`
	FinishedAt         int64         `json:"finished_at,omitempty"`
	Summary            string        `json:"summary,omitempty"`
	LeadActor          string        `json:"lead_actor,omitempty"`
	Timeline           []string      `json:"timeline,omitempty"`
	ResumeCursor       string        `json:"resume_cursor,omitempty"`
}

type RunCheckpoint struct {
	ID           string `json:"id"`
	WorkspaceID  string `json:"workspace_id"`
	RunID        string `json:"run_id"`
	TurnID       string `json:"turn_id"`
	ResumeCursor string `json:"resume_cursor"`
	Reason       string `json:"reason,omitempty"`
	CreatedAt    int64  `json:"created_at"`
}
