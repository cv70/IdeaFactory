package dbdao

import (
	"gorm.io/gorm"
)

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

type RuntimeRunRecord struct {
	gorm.Model
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	Source      string    `json:"source"`
	Status      string    `json:"status"`
	StartedAt   int64     `json:"started_at"`
	EndedAt     int64     `json:"ended_at"`
}

type RuntimePlanRecord struct {
	gorm.Model
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	RunID       string    `json:"run_id" gorm:"index"`
	Version     int       `json:"version"`
}

type RuntimePlanStepRecord struct {
	gorm.Model
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	RunID       string    `json:"run_id" gorm:"index"`
	PlanID      string    `json:"plan_id" gorm:"index"`
	StepIndex   int       `json:"step_index"`
	Desc        string    `json:"desc"`
	Status      string    `json:"status"`
}

type RuntimeAgentTaskRecord struct {
	gorm.Model
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	RunID       string    `json:"run_id" gorm:"index"`
	PlanID      string    `json:"plan_id" gorm:"index"`
	PlanStepID  string    `json:"plan_step_id" gorm:"index"`
	SubAgent    string    `json:"sub_agent"`
	Goal        string    `json:"goal"`
	Status      string    `json:"status"`
}

type RuntimeTaskResultRecord struct {
	gorm.Model
	TaskID      string    `json:"task_id" gorm:"index"`
	WorkspaceID string    `json:"workspace_id" gorm:"index"`
	Summary     string    `json:"summary"`
	IsSuccess   bool      `json:"is_success"`
}

type RuntimeBalanceRecord struct {
	gorm.Model
	WorkspaceID        string    `json:"workspace_id" gorm:"index"`
	RunID              string    `json:"run_id" gorm:"index"`
	Divergence         float64   `json:"divergence"`
	Research           float64   `json:"research"`
	Aggression         float64   `json:"aggression"`
	Reason             string    `json:"reason"`
	UpdatedAtMs        int64     `json:"updated_at_ms"`
	LatestReplanReason string    `json:"latest_replan_reason"`
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
