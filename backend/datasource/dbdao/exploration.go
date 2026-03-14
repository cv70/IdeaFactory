package dbdao

import (
	"time"

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

type WorkspaceState struct {
	WorkspaceID         string     `json:"workspace_id" gorm:"primaryKey"`
	Topic               string     `json:"topic"`
	OutputGoal          string     `json:"output_goal"`
	Constraints         string     `json:"constraints"`
	ActiveOpportunityID string     `json:"active_opportunity_id"`
	LastRunRound        int        `json:"last_run_round"`
	LastCompactedAt     time.Time  `json:"last_compacted_at"`
	ArchivedAt          *time.Time `json:"archived_at" gorm:"index"`
	Snapshot            string     `json:"snapshot" gorm:"type:text"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type WorkspaceRuntimeState struct {
	WorkspaceID string    `json:"workspace_id" gorm:"primaryKey"`
	Snapshot    string    `json:"snapshot" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RuntimeRunRecord struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	Source      string    `json:"source"`
	Status      string    `json:"status"`
	StartedAt   int64     `json:"started_at"`
	EndedAt     int64     `json:"ended_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RuntimePlanRecord struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	RunID       string    `json:"run_id" gorm:"index"`
	Version     int       `json:"version"`
	CreatedAtMs int64     `json:"created_at_ms"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RuntimePlanStepRecord struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	RunID       string    `json:"run_id" gorm:"index"`
	PlanID      string    `json:"plan_id" gorm:"index"`
	StepIndex   int       `json:"step_index"`
	Desc        string    `json:"desc"`
	Status      string    `json:"status"`
	UpdatedAtMs int64     `json:"updated_at_ms"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RuntimeAgentTaskRecord struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	RunID       string    `json:"run_id" gorm:"index"`
	PlanID      string    `json:"plan_id" gorm:"index"`
	PlanStepID  string    `json:"plan_step_id" gorm:"index"`
	SubAgent    string    `json:"sub_agent"`
	Goal        string    `json:"goal"`
	Status      string    `json:"status"`
	UpdatedAtMs int64     `json:"updated_at_ms"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RuntimeTaskResultRecord struct {
	TaskID      string    `json:"task_id" gorm:"primaryKey"`
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	Summary     string    `json:"summary"`
	IsSuccess   bool      `json:"is_success"`
	UpdatedAtMs int64     `json:"updated_at_ms"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RuntimeBalanceRecord struct {
	WorkspaceID        string    `json:"workspace_id" gorm:"primaryKey"`
	RunID              string    `json:"run_id" gorm:"index"`
	Divergence         float64   `json:"divergence"`
	Research           float64   `json:"research"`
	Aggression         float64   `json:"aggression"`
	Reason             string    `json:"reason"`
	UpdatedAtMs        int64     `json:"updated_at_ms"`
	LatestReplanReason string    `json:"latest_replan_reason"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type InterventionEvent struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	Type        string    `json:"type"`
	TargetID    string    `json:"target_id"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
}

type MutationLog struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	Kind        string    `json:"kind"`
	Source      string    `json:"source"`
	Payload     string    `json:"payload" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at" gorm:"index"`
}

func (d *DB) UpsertWorkspaceState(state *WorkspaceState) error {
	state.UpdatedAt = time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = state.UpdatedAt
	}
	return d.DB().Save(state).Error
}

func (d *DB) GetWorkspaceState(workspaceID string) (*WorkspaceState, error) {
	var state WorkspaceState
	err := d.DB().Where("workspace_id = ?", workspaceID).First(&state).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &state, nil
}

func (d *DB) UpsertWorkspaceRuntimeState(state *WorkspaceRuntimeState) error {
	state.UpdatedAt = time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = state.UpdatedAt
	}
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

type RuntimeStateProjection struct {
	WorkspaceID        string
	Runs               []RuntimeRunRecord
	Plans              []RuntimePlanRecord
	PlanSteps          []RuntimePlanStepRecord
	AgentTasks         []RuntimeAgentTaskRecord
	Results            []RuntimeTaskResultRecord
	Balance            *RuntimeBalanceRecord
	LatestReplanReason string
}

func (d *DB) ReplaceWorkspaceRuntimeProjection(state RuntimeStateProjection) error {
	tx := d.DB().Begin()
	if tx.Error != nil {
		return tx.Error
	}
	rollback := func(err error) error {
		_ = tx.Rollback()
		return err
	}

	tables := []any{
		&RuntimeRunRecord{},
		&RuntimePlanRecord{},
		&RuntimePlanStepRecord{},
		&RuntimeAgentTaskRecord{},
		&RuntimeTaskResultRecord{},
		&RuntimeBalanceRecord{},
	}
	for _, table := range tables {
		if err := tx.Where("workspace_id = ?", state.WorkspaceID).Delete(table).Error; err != nil {
			return rollback(err)
		}
	}

	if len(state.Runs) > 0 {
		if err := tx.Create(&state.Runs).Error; err != nil {
			return rollback(err)
		}
	}
	if len(state.Plans) > 0 {
		if err := tx.Create(&state.Plans).Error; err != nil {
			return rollback(err)
		}
	}
	if len(state.PlanSteps) > 0 {
		if err := tx.Create(&state.PlanSteps).Error; err != nil {
			return rollback(err)
		}
	}
	if len(state.AgentTasks) > 0 {
		if err := tx.Create(&state.AgentTasks).Error; err != nil {
			return rollback(err)
		}
	}
	if len(state.Results) > 0 {
		if err := tx.Create(&state.Results).Error; err != nil {
			return rollback(err)
		}
	}
	if state.Balance != nil {
		if err := tx.Create(state.Balance).Error; err != nil {
			return rollback(err)
		}
	}

	return tx.Commit().Error
}

func (d *DB) LoadWorkspaceRuntimeProjection(workspaceID string) (*RuntimeStateProjection, error) {
	var runs []RuntimeRunRecord
	if err := d.DB().
		Where("workspace_id = ?", workspaceID).
		Order("started_at asc, id asc").
		Find(&runs).Error; err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return nil, nil
	}

	var plans []RuntimePlanRecord
	if err := d.DB().
		Where("workspace_id = ?", workspaceID).
		Order("created_at_ms asc, id asc").
		Find(&plans).Error; err != nil {
		return nil, err
	}

	var steps []RuntimePlanStepRecord
	if err := d.DB().
		Where("workspace_id = ?", workspaceID).
		Order("updated_at_ms asc, step_index asc, id asc").
		Find(&steps).Error; err != nil {
		return nil, err
	}

	var tasks []RuntimeAgentTaskRecord
	if err := d.DB().
		Where("workspace_id = ?", workspaceID).
		Order("updated_at_ms asc, id asc").
		Find(&tasks).Error; err != nil {
		return nil, err
	}

	var results []RuntimeTaskResultRecord
	if err := d.DB().
		Where("workspace_id = ?", workspaceID).
		Order("updated_at_ms asc, task_id asc").
		Find(&results).Error; err != nil {
		return nil, err
	}

	var balance RuntimeBalanceRecord
	err := d.DB().Where("workspace_id = ?", workspaceID).First(&balance).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	var balancePtr *RuntimeBalanceRecord
	if err == nil {
		balancePtr = &balance
	}

	out := &RuntimeStateProjection{
		WorkspaceID: workspaceID,
		Runs:        runs,
		Plans:       plans,
		PlanSteps:   steps,
		AgentTasks:  tasks,
		Results:     results,
		Balance:     balancePtr,
	}
	if balancePtr != nil {
		out.LatestReplanReason = balancePtr.LatestReplanReason
	}
	return out, nil
}

func (d *DB) ListWorkspaceStates(limit int, includeArchived bool) ([]WorkspaceState, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := d.DB().Order("updated_at desc").Limit(limit)
	if !includeArchived {
		query = query.Where("archived_at IS NULL")
	}
	var states []WorkspaceState
	if err := query.Find(&states).Error; err != nil {
		return nil, err
	}
	return states, nil
}

func (d *DB) ArchiveWorkspaceState(workspaceID string) error {
	now := time.Now()
	return d.DB().
		Model(&WorkspaceState{}).
		Where("workspace_id = ?", workspaceID).
		Update("archived_at", &now).Error
}

func (d *DB) CreateInterventionEvent(event *InterventionEvent) error {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	return d.DB().Create(event).Error
}

func (d *DB) UpsertInterventionEvent(event *InterventionEvent) error {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	return d.DB().Save(event).Error
}

func (d *DB) GetInterventionEvent(workspaceID string, id string) (*InterventionEvent, error) {
	var event InterventionEvent
	err := d.DB().
		Where("workspace_id = ? AND id = ?", workspaceID, id).
		First(&event).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &event, nil
}

func (d *DB) ListInterventionEventsByPrefix(workspaceID string, idPrefix string, eventType string, limit int) ([]InterventionEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := d.DB().Where("workspace_id = ?", workspaceID)
	if idPrefix != "" {
		query = query.Where("id LIKE ?", idPrefix+"%")
	}
	if eventType != "" {
		query = query.Where("type = ?", eventType)
	}
	var events []InterventionEvent
	if err := query.Order("created_at asc, id asc").Limit(limit).Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

func (d *DB) CreateMutationLogs(logs []MutationLog) error {
	if len(logs) == 0 {
		return nil
	}
	return d.DB().Create(&logs).Error
}

func (d *DB) ListMutationLogs(workspaceID string, sinceUnixMs int64, limit int) ([]MutationLog, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	query := d.DB().Where("workspace_id = ?", workspaceID).Order("created_at asc").Limit(limit)
	if sinceUnixMs > 0 {
		sinceTime := time.UnixMilli(sinceUnixMs)
		query = query.Where("created_at > ?", sinceTime)
	}
	var logs []MutationLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func (d *DB) ListMutationLogsByCursor(workspaceID string, cursorTime time.Time, cursorID string, limit int) ([]MutationLog, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	query := d.DB().Where("workspace_id = ?", workspaceID).Order("created_at asc, id asc").Limit(limit)
	if !cursorTime.IsZero() {
		if cursorID != "" {
			query = query.Where("(created_at > ?) OR (created_at = ? AND id > ?)", cursorTime, cursorTime, cursorID)
		} else {
			query = query.Where("created_at > ?", cursorTime)
		}
	}
	var logs []MutationLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func (d *DB) CountMutationLogs(workspaceID string) (int64, error) {
	var count int64
	if err := d.DB().Model(&MutationLog{}).Where("workspace_id = ?", workspaceID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (d *DB) DeleteMutationLogsBefore(workspaceID string, cutoff time.Time) error {
	return d.DB().Where("workspace_id = ? AND created_at < ?", workspaceID, cutoff).Delete(&MutationLog{}).Error
}

func (d *DB) GetMutationCutoffForRecent(workspaceID string, keepRecent int) (*MutationLog, error) {
	if keepRecent <= 0 {
		keepRecent = 1
	}
	var log MutationLog
	err := d.DB().
		Where("workspace_id = ?", workspaceID).
		Order("created_at desc, id desc").
		Offset(keepRecent - 1).
		Limit(1).
		Find(&log).Error
	if err != nil {
		return nil, err
	}
	if log.ID == "" {
		return nil, nil
	}
	return &log, nil
}
