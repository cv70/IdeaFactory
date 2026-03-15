package exploration

import "context"

// ReplanTriggerKind identifies what caused a replan.
type ReplanTriggerKind string

const (
	ReplanTriggerIntervention ReplanTriggerKind = "intervention"
	ReplanTriggerBalanceShift ReplanTriggerKind = "balance_shift"
	ReplanTriggerManual       ReplanTriggerKind = "manual"
)

// ReplanTrigger carries context about what triggered a replan.
type ReplanTrigger struct {
	Kind         ReplanTriggerKind
	Intervention *InterventionView // non-nil when Kind == ReplanTriggerIntervention
}

// Planner builds and adapts execution plans for a workspace.
// All methods are called from within a withWorkspaceState callback,
// so they must not call withWorkspaceState themselves.
type Planner interface {
	// BuildInitialPlan creates the first plan for a workspace.
	BuildInitialPlan(ctx context.Context, session *ExplorationSession, state *RuntimeWorkspaceState) (*ExecutionPlan, []PlanStep, error)

	// Replan creates a new plan version given the current runtime state and a trigger.
	Replan(ctx context.Context, session *ExplorationSession, state *RuntimeWorkspaceState, trigger ReplanTrigger) (*ExecutionPlan, []PlanStep, error)

	// GenerateNodesForCycle inspects the current graph and balance state,
	// and returns the nodes and edges to add in the next cycle.
	GenerateNodesForCycle(ctx context.Context, session *ExplorationSession, state *RuntimeWorkspaceState) ([]Node, []Edge)
}
