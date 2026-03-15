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

func (p *DeterministicPlanner) BuildInitialPlan(_ context.Context, session *ExplorationSession, _ *RuntimeWorkspaceState) (*ExecutionPlan, []PlanStep, error) {
	now := time.Now()
	planID := fmt.Sprintf("plan-%s-%d", session.ID, now.UnixNano())
	plan := &ExecutionPlan{
		ID: planID, WorkspaceID: session.ID, Version: 1, CreatedAt: now.UnixMilli(),
	}
	steps := []PlanStep{{
		ID: planID + "-step-1", WorkspaceID: session.ID, PlanID: planID,
		Index: 1, Desc: "generate nodes", Status: PlanStepTodo, UpdatedAt: now.UnixMilli(),
	}}
	return plan, steps, nil
}

func (p *DeterministicPlanner) Replan(_ context.Context, session *ExplorationSession, state *RuntimeWorkspaceState, _ ReplanTrigger) (*ExecutionPlan, []PlanStep, error) {
	return p.BuildInitialPlan(context.Background(), session, state)
}

func (p *DeterministicPlanner) GenerateNodesForCycle(_ context.Context, _ *ExplorationSession, _ *RuntimeWorkspaceState) ([]Node, []Edge) {
	return nil, nil // stub — replaced in Task 4
}
