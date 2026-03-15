package exploration

import (
	"context"
	"fmt"
	"time"
)

// DeterministicPlanner implements Planner without any LLM calls.
// Node generation is deterministic based on graph state and BalanceState.
type DeterministicPlanner struct{}

// Compile-time interface check
var _ Planner = &DeterministicPlanner{}

func NewDeterministicPlanner() *DeterministicPlanner {
	return &DeterministicPlanner{}
}

func (p *DeterministicPlanner) BuildInitialPlan(_ context.Context, session *ExplorationSession, state *RuntimeWorkspaceState) (*ExecutionPlan, []PlanStep, error) {
	now := time.Now()
	planID := fmt.Sprintf("plan-%s-%d", session.ID, now.UnixNano())
	runID := ""
	if state != nil && len(state.Runs) > 0 {
		runID = state.Runs[len(state.Runs)-1].ID
	}
	plan := &ExecutionPlan{
		ID: planID, WorkspaceID: session.ID, RunID: runID, Version: 1, CreatedAt: now.UnixMilli(),
	}
	stepDescs := []string{
		"collect research signals for current opportunities",
		"structure research into graph mutations and decisions",
		"materialize high-confidence idea cards and summary",
	}
	steps := make([]PlanStep, 0, len(stepDescs))
	for i, desc := range stepDescs {
		steps = append(steps, PlanStep{
			ID:          fmt.Sprintf("%s-step-%d", planID, i+1),
			WorkspaceID: session.ID,
			RunID:       runID,
			PlanID:      planID,
			Index:       i + 1,
			Desc:        desc,
			Status:      PlanStepTodo,
			UpdatedAt:   now.UnixMilli(),
		})
	}
	return plan, steps, nil
}

func (p *DeterministicPlanner) Replan(_ context.Context, session *ExplorationSession, state *RuntimeWorkspaceState, _ ReplanTrigger) (*ExecutionPlan, []PlanStep, error) {
	return p.BuildInitialPlan(context.Background(), session, state)
}

func (p *DeterministicPlanner) GenerateNodesForCycle(_ context.Context, _ *ExplorationSession, _ *RuntimeWorkspaceState) ([]Node, []Edge) {
	return nil, nil // stub — replaced in Task 4
}
