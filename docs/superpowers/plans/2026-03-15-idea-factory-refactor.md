# Idea Factory — Refactor & Runtime Semantics Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate the backend runtime state into a typed struct, extract a `Planner` interface with a working `DeterministicPlanner` that generates Direction/Evidence/Claim/Decision/Unknown nodes, split `api_v1.go` into focused handler files, and write mutation events at key lifecycle points.

**Architecture:** `runtimeState` (10 loose maps) becomes `runtimeWorkspaces` (one `RuntimeWorkspaceState` per workspace), accessed exclusively via `withWorkspaceState`. `DeterministicPlanner` implements the new `Planner` interface and generates graph nodes deterministically based on `BalanceState`. `api_v1.go` (864 lines) splits into five focused files with no behavior change.

**Tech Stack:** Go 1.23+, Gin web framework, existing `schema.go` types (`MutationEvent`, `NodeType`, `EdgeType`), Vitest for frontend.

---

## Chunk 1: Foundation — schema constants, Planner interface, domain struct

### Task 1: Add node and edge type constants to schema.go

**Files:**
- Modify: `backend/domain/exploration/schema.go`

- [ ] **Step 1: Add `NodeDirection` and `NodeArtifact` to the `NodeType` constants block**

In `schema.go`, after the last existing `NodeType` constant (`NodeUnknown NodeType = "unknown"` at line 16), add:

```go
NodeDirection NodeType = "direction"
NodeArtifact  NodeType = "artifact"
```

- [ ] **Step 2: Add four new `EdgeType` constants**

In `schema.go`, after the last existing `EdgeType` constant (`EdgeWeakens EdgeType = "weakens"` at line 29), add:

```go
EdgeJustifies    EdgeType = "justifies"
EdgeBranchesFrom EdgeType = "branches_from"
EdgeRaises       EdgeType = "raises"
EdgeResolves     EdgeType = "resolves"
```

- [ ] **Step 3: Verify build passes**

```bash
cd /home/o/space/IdeaFactory/backend && go build ./domain/exploration/...
```
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add backend/domain/exploration/schema.go
git commit -m "feat: add NodeDirection, NodeArtifact, and four new EdgeType constants"
```

---

### Task 2: Create planner.go with Planner interface

**Files:**
- Create: `backend/domain/exploration/planner.go`
- Create: `backend/domain/exploration/deterministic.go` (stub — real implementation in Task 4)

- [ ] **Step 1: Create `planner.go` with the Planner interface**

```go
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
```

- [ ] **Step 2: Create a minimal stub `deterministic.go`**

This stub satisfies the `Planner` interface so the build stays green through Tasks 3–4. Task 4 (Chunk 2) replaces the stub bodies with real logic. Do NOT add `buildInitialBalance` here — it is already defined in `runtime_plan.go` and will be moved in Task 5.

```go
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
```

- [ ] **Step 3: Verify build**

```bash
cd /home/o/space/IdeaFactory/backend && go build ./domain/exploration/...
```

- [ ] **Step 4: Commit**

```bash
git add backend/domain/exploration/planner.go backend/domain/exploration/deterministic.go
git commit -m "feat: add Planner interface + DeterministicPlanner stub (build stays green)"
```

---

### Task 3: Refactor domain.go — RuntimeWorkspaceState + withWorkspaceState

This task replaces the old `runtimeState` struct with `runtimeWorkspaces`/`RuntimeWorkspaceState` AND updates every caller in the same commit so the build stays green.

**Files:**
- Modify: `backend/domain/exploration/domain.go`
- Modify: `backend/domain/exploration/runtime_agent.go` (all `d.runtime.*` accesses)
- Modify: `backend/domain/exploration/workspace_management.go`
- Modify: `backend/domain/exploration/api_v1.go` (only the three intervention methods that access `d.runtime.intervention`)

- [ ] **Step 1: Replace `domain.go` entirely**

Replace the entire contents of `domain.go` with:

```go
package exploration

import (
	"backend/agents"
	"backend/datasource/dbdao"
	"context"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/gorilla/websocket"
)

// RuntimeWorkspaceState holds all per-workspace runtime data.
// Access exclusively via withWorkspaceState.
type RuntimeWorkspaceState struct {
	Runs          []Run
	Plans         []ExecutionPlan
	PlanSteps     []PlanStep
	AgentTasks    []AgentTask
	Results       []AgentTaskResultSummary
	Balance       BalanceState
	Mutations     []MutationEvent
	ReplanReason  string
	Interventions map[string]InterventionView // keyed by intervention ID
	Running       bool
	Cursor        int
}

type workspaceStore struct {
	mu         sync.RWMutex
	workspaces map[string]*ExplorationSession
}

type wsClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

type wsState struct {
	mu          sync.RWMutex
	subscribers map[string]map[*wsClient]struct{}
}

type runtimeWorkspaces struct {
	mu         sync.Mutex
	workspaces map[string]*RuntimeWorkspaceState // keyed by workspace ID
}

type ExplorationDomain struct {
	DB        *dbdao.DB
	DeepAgent adk.ResumableAgent
	General   adk.Agent
	Model     model.ToolCallingChatModel
	store     *workspaceStore
	ws        *wsState
	planner   Planner
	runtime   *runtimeWorkspaces
}

// getWorkspaceState returns the state for workspaceID, initializing it if absent.
// Callers MUST hold runtime.mu before calling.
func (d *ExplorationDomain) getWorkspaceState(workspaceID string) *RuntimeWorkspaceState {
	state, ok := d.runtime.workspaces[workspaceID]
	if !ok {
		state = &RuntimeWorkspaceState{
			Interventions: map[string]InterventionView{},
		}
		d.runtime.workspaces[workspaceID] = state
	}
	return state
}

// withWorkspaceState locks runtime.mu, fetches or initializes the state, calls fn, then unlocks.
// fn MUST NOT call withWorkspaceState (no re-entry).
func (d *ExplorationDomain) withWorkspaceState(workspaceID string, fn func(*RuntimeWorkspaceState)) {
	d.runtime.mu.Lock()
	state := d.getWorkspaceState(workspaceID)
	fn(state)
	d.runtime.mu.Unlock()
}

func NewExplorationDomain(db *dbdao.DB, lm model.ToolCallingChatModel) *ExplorationDomain {
	domain := &ExplorationDomain{
		DB:      db,
		Model:   lm,
		planner: NewDeterministicPlanner(),
		store: &workspaceStore{
			workspaces: map[string]*ExplorationSession{},
		},
		ws: &wsState{
			subscribers: map[string]map[*wsClient]struct{}{},
		},
		runtime: &runtimeWorkspaces{
			workspaces: map[string]*RuntimeWorkspaceState{},
		},
	}
	if lm != nil {
		if agent, err := agents.BuildExplorationAgent(context.Background(), lm); err == nil {
			domain.DeepAgent = agent
		}
		if general, err := agents.NewGeneralAgent(context.Background(), lm); err == nil {
			domain.General = general
		}
	} else {
		if general, err := agents.NewGeneralAgent(context.Background(), nil); err == nil {
			domain.General = general
		}
	}
	return domain
}
```

- [ ] **Step 2: Update `runtime_agent.go` — replace all `d.runtime.*[workspaceID]` accesses**

The file has four functions that touch `d.runtime.*` directly. Rewrite each:

**`initializeRuntimeState`** — replace body:
```go
func (d *ExplorationDomain) initializeRuntimeState(session ExplorationSession, source string) {
	ctx := context.Background()
	now := time.Now()
	var skip bool
	d.withWorkspaceState(session.ID, func(state *RuntimeWorkspaceState) {
		if len(state.Runs) > 0 {
			skip = true
			return
		}
		runID := fmt.Sprintf("run-%s-%d", session.ID, now.UnixNano())
		run := Run{
			ID:          runID,
			WorkspaceID: session.ID,
			Source:      source,
			Status:      RunStatusRunning,
			StartedAt:   now.UnixMilli(),
		}
		state.Runs = []Run{run}
		state.Balance = buildInitialBalance(session, runID, now)
		state.ReplanReason = ""

		plan, steps, _ := d.planner.BuildInitialPlan(ctx, &session, state)
		if plan != nil {
			state.Plans = []ExecutionPlan{*plan}
			state.PlanSteps = steps
		}
		nodes, edges := d.planner.GenerateNodesForCycle(ctx, &session, state)
		state.Mutations = append(state.Mutations, MutationEvent{
			ID:          mutationID(session.ID),
			WorkspaceID: session.ID,
			Kind:        "run_created",
			Run:         &GenerationRun{ID: runID},
			CreatedAt:   now.UnixMilli(),
		})
		run.Status = RunStatusCompleted
		run.EndedAt = now.UnixMilli()
		state.Runs[0] = run

		_ = nodes
		_ = edges
	})
	if skip {
		return
	}
	d.persistRuntimeState(session.ID)
}
```

**`GetRuntimeState`** — stays as:
```go
func (d *ExplorationDomain) GetRuntimeState(workspaceID string) (RuntimeStateSnapshot, bool) {
	return d.QueryRuntimeState(workspaceID, RuntimeStateQuery{})
}
```

**`QueryRuntimeState`** — replace body:
```go
func (d *ExplorationDomain) QueryRuntimeState(workspaceID string, query RuntimeStateQuery) (RuntimeStateSnapshot, bool) {
	var snapshot RuntimeStateSnapshot
	var found bool
	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		if len(state.Runs) == 0 {
			return
		}
		found = true
		snapshot = RuntimeStateSnapshot{
			Runs:               append([]Run{}, state.Runs...),
			Plans:              append([]ExecutionPlan{}, state.Plans...),
			PlanSteps:          append([]PlanStep{}, state.PlanSteps...),
			AgentTasks:         append([]AgentTask{}, state.AgentTasks...),
			Results:            append([]AgentTaskResultSummary{}, state.Results...),
			Balance:            state.Balance,
			LatestReplanReason: state.ReplanReason,
		}
	})
	if !found {
		return RuntimeStateSnapshot{}, false
	}
	return filterRuntimeSnapshot(snapshot, query), true
}
```

**`replanRuntimeState`** — replace body:
```go
func (d *ExplorationDomain) replanRuntimeState(session ExplorationSession, intervention InterventionReq) {
	ctx := context.Background()
	now := time.Now()
	var skip bool
	d.withWorkspaceState(session.ID, func(state *RuntimeWorkspaceState) {
		if len(state.Runs) == 0 {
			skip = true
			return
		}
		currentRun := state.Runs[len(state.Runs)-1]

		// Skip pending steps on current plan
		if len(state.Plans) > 0 {
			currentPlan := state.Plans[len(state.Plans)-1]
			for i := range state.PlanSteps {
				if state.PlanSteps[i].PlanID != currentPlan.ID {
					continue
				}
				if state.PlanSteps[i].Status == PlanStepTodo || state.PlanSteps[i].Status == PlanStepDoing {
					state.PlanSteps[i].Status = PlanStepSkipped
					state.PlanSteps[i].UpdatedAt = now.UnixMilli()
				}
			}
		}

		// Adjust balance for intervention intent keywords
		state.Balance = adjustBalanceForIntent(state.Balance, intervention.Note, now)
		state.ReplanReason = fmt.Sprintf("%s:%s", intervention.Type, strings.TrimSpace(intervention.Note))

		trigger := ReplanTrigger{
			Kind: ReplanTriggerIntervention,
		}
		plan, steps, _ := d.planner.Replan(ctx, &session, state, trigger)
		if plan != nil {
			if len(state.Plans) > 0 {
				plan.Version = state.Plans[len(state.Plans)-1].Version + 1
			}
			state.Plans = append(state.Plans, *plan)
			state.PlanSteps = append(state.PlanSteps, steps...)
		}

		nodes, edges := d.planner.GenerateNodesForCycle(ctx, &session, state)
		state.Mutations = append(state.Mutations, MutationEvent{
			ID:          mutationID(session.ID),
			WorkspaceID: session.ID,
			Kind:        "replanned",
			Run:         &GenerationRun{ID: currentRun.ID},
			CreatedAt:   now.UnixMilli(),
		})
		state.Mutations = append(state.Mutations, MutationEvent{
			ID:          mutationID(session.ID),
			WorkspaceID: session.ID,
			Kind:        "balance_updated",
			CreatedAt:   now.UnixMilli(),
		})

		_ = nodes
		_ = edges
	})
	if skip {
		return
	}
	d.persistRuntimeState(session.ID)
}
```

**`executeRuntimeCycle`** — replace body:
```go
func (d *ExplorationDomain) executeRuntimeCycle(session ExplorationSession, source string) {
	ctx := context.Background()
	now := time.Now()

	d.withWorkspaceState(session.ID, func(state *RuntimeWorkspaceState) {
		if len(state.Runs) == 0 {
			d.startRuntimeRunLocked(ctx, session, source, now, state)
			return
		}
		if !d.executeNextTodoStepLocked(session.ID, now, state) {
			d.startRuntimeRunLocked(ctx, session, source, now, state)
		}
		state.Balance.Divergence += 0.01
		if state.Balance.Divergence > 1 {
			state.Balance.Divergence = 1
		}
		state.Balance.UpdatedAt = now.UnixMilli()
	})
	d.persistRuntimeState(session.ID)
}
```

**`restoreRuntimeState`** — replace body:
```go
func (d *ExplorationDomain) restoreRuntimeState(workspaceID string) bool {
	snapshot, ok := d.loadRuntimeState(workspaceID)
	if !ok {
		return false
	}
	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		if len(state.Runs) > 0 {
			return
		}
		state.Runs = append([]Run{}, snapshot.Runs...)
		state.Plans = append([]ExecutionPlan{}, snapshot.Plans...)
		state.PlanSteps = append([]PlanStep{}, snapshot.PlanSteps...)
		state.AgentTasks = append([]AgentTask{}, snapshot.AgentTasks...)
		state.Results = append([]AgentTaskResultSummary{}, snapshot.Results...)
		state.Balance = snapshot.Balance
		state.ReplanReason = snapshot.LatestReplanReason
	})
	return true
}
```

**`startRuntimeRunLocked`** — change signature (takes `state *RuntimeWorkspaceState` instead of locking itself) and add `ctx`:
```go
func (d *ExplorationDomain) startRuntimeRunLocked(ctx context.Context, session ExplorationSession, source string, now time.Time, state *RuntimeWorkspaceState) {
	runID := fmt.Sprintf("run-%s-%d", session.ID, now.UnixNano())
	run := Run{
		ID:          runID,
		WorkspaceID: session.ID,
		Source:      source,
		Status:      RunStatusRunning,
		StartedAt:   now.UnixMilli(),
	}
	state.Runs = append(state.Runs, run)

	plan, steps, _ := d.planner.BuildInitialPlan(ctx, &session, state)
	if plan != nil {
		if len(state.Plans) > 0 {
			plan.Version = state.Plans[len(state.Plans)-1].Version + 1
		}
		state.Plans = append(state.Plans, *plan)
		state.PlanSteps = append(state.PlanSteps, steps...)
	}

	_ = d.executeNextTodoStepLocked(session.ID, now, state)

	newBalance := buildInitialBalance(session, runID, now)
	if state.Balance.RunID != "" {
		newBalance.Divergence = (state.Balance.Divergence + newBalance.Divergence) / 2
		newBalance.Research = (state.Balance.Research + newBalance.Research) / 2
		newBalance.Aggression = (state.Balance.Aggression + newBalance.Aggression) / 2
	}
	state.Balance = newBalance

	state.Runs[len(state.Runs)-1].Status = RunStatusCompleted
	state.Runs[len(state.Runs)-1].EndedAt = now.UnixMilli()

	state.Mutations = append(state.Mutations, MutationEvent{
		ID:          mutationID(session.ID),
		WorkspaceID: session.ID,
		Kind:        "run_created",
		Run:         &GenerationRun{ID: runID},
		CreatedAt:   now.UnixMilli(),
	})
}
```

**`executeNextTodoStepLocked`** — change signature (add `state *RuntimeWorkspaceState`):
```go
func (d *ExplorationDomain) executeNextTodoStepLocked(workspaceID string, now time.Time, state *RuntimeWorkspaceState) bool {
	if len(state.Plans) == 0 {
		return false
	}
	currentPlan := state.Plans[len(state.Plans)-1]

	targetIndex := -1
	for i := len(state.PlanSteps) - 1; i >= 0; i-- {
		if state.PlanSteps[i].PlanID == currentPlan.ID && state.PlanSteps[i].Status == PlanStepTodo {
			targetIndex = i
		}
	}
	if targetIndex == -1 {
		return false
	}

	step := state.PlanSteps[targetIndex]
	step.Status = PlanStepDoing
	step.UpdatedAt = now.UnixMilli()

	taskID := fmt.Sprintf("task-%s-%d", currentPlan.ID, step.Index)
	task := AgentTask{
		ID:          taskID,
		WorkspaceID: workspaceID,
		RunID:       currentPlan.RunID,
		PlanID:      currentPlan.ID,
		PlanStepID:  step.ID,
		SubAgent:    subAgentForStep(step.Index),
		Goal:        step.Desc,
		Status:      PlanStepDone,
		UpdatedAt:   now.UnixMilli(),
	}

	step.Status = PlanStepDone
	step.UpdatedAt = now.UnixMilli()
	state.PlanSteps[targetIndex] = step
	state.AgentTasks = append(state.AgentTasks, task)
	state.Results = append(state.Results, AgentTaskResultSummary{
		TaskID:    taskID,
		Summary:   subAgentForStep(step.Index) + " step completed",
		IsSuccess: true,
		UpdatedAt: now.UnixMilli(),
	})
	return true
}
```

Add `adjustBalanceForIntent` helper at the bottom of `runtime_agent.go`:
```go
// adjustBalanceForIntent adjusts BalanceState fields based on intent keyword scanning.
// Adjustments accumulate; all fields are clamped to [0, 1].
func adjustBalanceForIntent(prev BalanceState, intent string, now time.Time) BalanceState {
	next := prev
	next.UpdatedAt = now.UnixMilli()
	lower := strings.ToLower(intent)

	clamp := func(v float64) float64 {
		if v < 0 { return 0 }
		if v > 1 { return 1 }
		return v
	}

	if strings.Contains(lower, "focus") || strings.Contains(lower, "decide") ||
		strings.Contains(lower, "收敛") || strings.Contains(lower, "converge") {
		next.Divergence = clamp(next.Divergence - 0.2)
	}
	if strings.Contains(lower, "explore") || strings.Contains(lower, "expand") ||
		strings.Contains(lower, "发散") || strings.Contains(lower, "diverge") {
		next.Divergence = clamp(next.Divergence + 0.2)
	}
	if strings.Contains(lower, "research") || strings.Contains(lower, "evidence") ||
		strings.Contains(lower, "调研") {
		next.Research = clamp(next.Research + 0.2)
	}
	if strings.Contains(lower, "produce") || strings.Contains(lower, "output") ||
		strings.Contains(lower, "产出") {
		next.Research = clamp(next.Research - 0.2)
	}
	if strings.Contains(lower, "fast") || strings.Contains(lower, "quick") ||
		strings.Contains(lower, "aggressive") {
		next.Aggression = clamp(next.Aggression + 0.2)
	}
	if strings.Contains(lower, "careful") || strings.Contains(lower, "thorough") ||
		strings.Contains(lower, "prudent") {
		next.Aggression = clamp(next.Aggression - 0.2)
	}

	if next.Reason == "" {
		next.Reason = "adjusted by intervention intent"
	}
	return next
}
```

**Remove `updateBalanceForIntervention`** — it is replaced by `adjustBalanceForIntent`. Search `replanRuntimeState` to ensure it calls `adjustBalanceForIntent`.

Also **remove `dispatchAgentTask`** from `runtime_agent.go` (the `executeNextTodoStepLocked` rewrite above no longer calls it — it just marks steps done deterministically). The function can be left in place for now and cleaned up in Task 6.

- [ ] **Step 3: Update `workspace_management.go` — `ArchiveWorkspace`**

Replace the runtime map clearing in `ArchiveWorkspace` (lines 61-64). Note: the replacement also clears `Interventions`, which is new behavior (the old code did not clear the intervention map on archive — this is intentional per spec).

```go
// Before:
d.runtime.mu.Lock()
delete(d.runtime.running, workspaceID)
delete(d.runtime.cursor, workspaceID)
d.runtime.mu.Unlock()

// After:
d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
    state.Running = false
    state.Cursor = 0
    state.Interventions = map[string]InterventionView{}
})
```

- [ ] **Step 4: Update `api_v1.go` — the three intervention methods that access `d.runtime.intervention`**

**`storeInterventionRecord`** — replace the runtime access:
```go
func (d *ExplorationDomain) storeInterventionRecord(workspaceID string, req CreateInterventionRequest) InterventionView {
	now := time.Now().UnixMilli()
	view := InterventionView{
		ID:          fmt.Sprintf("intervention-%s-%d", workspaceID, now),
		WorkspaceID: workspaceID,
		Intent:      strings.TrimSpace(req.Intent),
		Status:      InterventionReceived,
		CreatedAt:   toRFC3339(now),
		UpdatedAt:   toRFC3339(now),
	}
	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		state.Interventions[view.ID] = view
	})
	return view
}
```

**`getInterventionRecord`** — replace the runtime access:
```go
func (d *ExplorationDomain) getInterventionRecord(workspaceID string, interventionID string) (InterventionView, bool) {
	var found InterventionView
	var ok bool
	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		found, ok = state.Interventions[interventionID]
	})
	if ok {
		return found, true
	}
	view, dbOk := d.loadV1Intervention(workspaceID, interventionID)
	if !dbOk {
		return InterventionView{}, false
	}
	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		state.Interventions[interventionID] = view
	})
	return view, true
}
```

**`advanceInterventionByRuntimeEvent`** — replace the runtime access:
```go
func (d *ExplorationDomain) advanceInterventionByRuntimeEvent(workspaceID string, interventionID string, state RuntimeStateSnapshot, mutations []MutationEvent) InterventionView {
	now := time.Now().UnixMilli()
	var result InterventionView
	d.withWorkspaceState(workspaceID, func(ws *RuntimeWorkspaceState) {
		view, ok := ws.Interventions[interventionID]
		if !ok {
			return
		}
		if len(state.Runs) > 0 && view.Status == InterventionReceived {
			view.Status = InterventionAbsorbed
			view.AbsorbedByRunID = state.Runs[len(state.Runs)-1].ID
			view.UpdatedAt = toRFC3339(now)
		}
		if len(state.Plans) > 0 && (view.Status == InterventionReceived || view.Status == InterventionAbsorbed) {
			view.Status = InterventionReplanned
			view.ReplannedPlanID = state.Plans[len(state.Plans)-1].ID
			view.UpdatedAt = toRFC3339(now)
		}
		if len(mutations) > 0 {
			view.Status = InterventionReflected
			view.ReflectedEventID = fmt.Sprintf("event-%d", mutations[len(mutations)-1].CreatedAt)
			view.UpdatedAt = toRFC3339(now)
		}
		ws.Interventions[interventionID] = view
		result = view
	})
	d.persistV1Intervention(result)
	return result
}
```

- [ ] **Step 5: Add `initializeWorkspaceGraph` to `runtime_agent.go`**

```go
// initializeWorkspaceGraph calls BuildInitialPlan and GenerateNodesForCycle synchronously,
// adding the generated Direction nodes to the session's in-memory store.
// Must be called while NOT holding runtime.mu.
func (d *ExplorationDomain) initializeWorkspaceGraph(ctx context.Context, workspaceID string) {
	d.store.mu.Lock()
	session, ok := d.store.workspaces[workspaceID]
	if !ok {
		d.store.mu.Unlock()
		return
	}
	sessionCopy := *session
	d.store.mu.Unlock()

	var newNodes []Node
	var newEdges []Edge
	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		if state.Balance.WorkspaceID == "" {
			now := time.Now()
			runID := fmt.Sprintf("init-%s-%d", workspaceID, now.UnixNano())
			state.Balance = buildInitialBalance(sessionCopy, runID, now)
		}
		newNodes, newEdges = d.planner.GenerateNodesForCycle(ctx, &sessionCopy, state)
	})

	if len(newNodes) == 0 {
		return
	}

	d.store.mu.Lock()
	if current, ok := d.store.workspaces[workspaceID]; ok {
		prev := *current
		current.Nodes = append(current.Nodes, newNodes...)
		current.Edges = append(current.Edges, newEdges...)
		// Write node_added mutation events via diffMutations
		mutations := diffMutations(prev, *current, "init")
		updatedCopy := *current // capture before unlocking
		d.store.mu.Unlock()
		d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
			state.Mutations = append(state.Mutations, mutations...)
		})
		d.broadcastMutations(workspaceID, mutations)
		d.persistWorkspace(updatedCopy) // persistWorkspace takes ExplorationSession by value
	} else {
		d.store.mu.Unlock()
	}
}
```

- [ ] **Step 6: Update `ApiV1CreateWorkspace` in `api_v1.go` to call `initializeWorkspaceGraph`**

Replace the handler body:
```go
func (d *ExplorationDomain) ApiV1CreateWorkspace(c *gin.Context) {
	var req CreateWorkspaceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_argument", "failed to parse create workspace request")
		return
	}
	snapshot := d.CreateWorkspace(req)
	d.initializeWorkspaceGraph(c.Request.Context(), snapshot.Exploration.ID)
	// Re-fetch so the response includes seeded Direction nodes.
	if updated, ok := d.GetWorkspace(snapshot.Exploration.ID); ok {
		snapshot = updated
	}
	c.JSON(http.StatusCreated, WorkspaceResponse{Workspace: toWorkspaceView(snapshot.Exploration)})
}
```

- [ ] **Step 7: Verify build**

```bash
cd /home/o/space/IdeaFactory/backend && go build ./domain/exploration/...
```
Expected: no errors. Fix any compilation errors before continuing.

- [ ] **Step 8: Run existing tests**

```bash
cd /home/o/space/IdeaFactory/backend && go test ./domain/exploration/... -count=1 -timeout=60s
```
Expected: same pass/skip counts as before (existing tests should pass; tests requiring DB will skip).

- [ ] **Step 9: Commit**

```bash
git add backend/domain/exploration/domain.go \
        backend/domain/exploration/runtime_agent.go \
        backend/domain/exploration/workspace_management.go \
        backend/domain/exploration/api_v1.go
git commit -m "refactor: replace runtimeState maps with RuntimeWorkspaceState + withWorkspaceState accessor"
```

---

## Chunk 2: DeterministicPlanner

### Task 4: Implement DeterministicPlanner with TDD

**Files:**
- Modify: `backend/domain/exploration/deterministic.go` (replace stub bodies with real logic)
- Modify: `backend/domain/exploration/api_test.go` (add 5 new test functions at the end)

- [ ] **Step 1: Write failing tests for `GenerateNodesForCycle`**

Add these test functions to the end of `api_test.go`:

```go
func TestDeterministicPlannerInitialRun(t *testing.T) {
	p := NewDeterministicPlanner()
	session := &ExplorationSession{
		ID:    "ws-test",
		Topic: "machine learning for healthcare",
	}
	state := &RuntimeWorkspaceState{
		Balance: BalanceState{
			Divergence: 0.6,
			Research:   0.7,
			Aggression: 0.4,
		},
	}
	nodes, edges := p.GenerateNodesForCycle(context.Background(), session, state)
	if len(nodes) == 0 {
		t.Fatal("expected Direction nodes on first cycle, got none")
	}
	for _, n := range nodes {
		if n.Type != NodeDirection {
			t.Errorf("expected all initial nodes to be NodeDirection, got %s", n.Type)
		}
	}
	// No edges expected for initial Direction nodes
	for _, e := range edges {
		_ = e
	}
	if len(nodes) < 3 || len(nodes) > 5 {
		t.Errorf("expected 3-5 Direction nodes, got %d", len(nodes))
	}
}

func TestDeterministicPlannerResearchPhase(t *testing.T) {
	p := NewDeterministicPlanner()
	dirNode := Node{ID: "dir-1", Type: NodeDirection, Title: "ML for diagnosis", WorkspaceID: "ws-test"}
	session := &ExplorationSession{
		ID:    "ws-test",
		Topic: "machine learning for healthcare",
		Nodes: []Node{dirNode},
	}
	state := &RuntimeWorkspaceState{
		Balance: BalanceState{
			Divergence: 0.6,
			Research:   0.7,
			Aggression: 0.4,
		},
	}
	nodes, edges := p.GenerateNodesForCycle(context.Background(), session, state)
	if len(nodes) == 0 {
		t.Fatal("expected Evidence nodes in research phase, got none")
	}
	for _, n := range nodes {
		if n.Type != NodeEvidence {
			t.Errorf("expected Evidence nodes, got %s", n.Type)
		}
	}
	// Each Evidence node should have an edge to the Direction
	if len(edges) == 0 {
		t.Error("expected edges from Evidence to Direction, got none")
	}
	for _, e := range edges {
		if e.Type != EdgeSupports && e.Type != EdgeContradicts {
			t.Errorf("expected EdgeSupports or EdgeContradicts, got %s", e.Type)
		}
		if e.To != dirNode.ID {
			t.Errorf("expected edge to direction ID %s, got %s", dirNode.ID, e.To)
		}
	}
}

func TestDeterministicPlannerFastPath(t *testing.T) {
	p := NewDeterministicPlanner()
	dirNode := Node{ID: "dir-1", Type: NodeDirection, Title: "ML for diagnosis", WorkspaceID: "ws-test"}
	session := &ExplorationSession{
		ID:    "ws-test",
		Topic: "machine learning for healthcare",
		Nodes: []Node{dirNode}, // Direction but no Evidence
	}
	state := &RuntimeWorkspaceState{
		Balance: BalanceState{
			Aggression: 0.8, // High aggression: skip Evidence, go straight to Claims
		},
	}
	nodes, _ := p.GenerateNodesForCycle(context.Background(), session, state)
	if len(nodes) == 0 {
		t.Fatal("expected Claim nodes on fast path, got none")
	}
	for _, n := range nodes {
		if n.Type != NodeClaim {
			t.Errorf("expected NodeClaim on fast path, got %s", n.Type)
		}
	}
}

func TestDeterministicPlannerConvergence(t *testing.T) {
	p := NewDeterministicPlanner()
	dir := Node{ID: "dir-1", Type: NodeDirection, WorkspaceID: "ws-test"}
	ev1 := Node{ID: "ev-1", Type: NodeEvidence, Metadata: NodeMetadata{BranchID: "dir-1"}, WorkspaceID: "ws-test"}
	ev2 := Node{ID: "ev-2", Type: NodeEvidence, Metadata: NodeMetadata{BranchID: "dir-1"}, WorkspaceID: "ws-test"}
	claim := Node{ID: "cl-1", Type: NodeClaim, Metadata: NodeMetadata{BranchID: "dir-1"}, WorkspaceID: "ws-test"}
	session := &ExplorationSession{
		ID:    "ws-test",
		Topic: "machine learning for healthcare",
		Nodes: []Node{dir, ev1, ev2, claim},
		Edges: []Edge{
			{ID: "e1", From: "ev-1", To: "dir-1", Type: EdgeSupports},
			{ID: "e2", From: "ev-2", To: "dir-1", Type: EdgeContradicts},
		},
	}
	state := &RuntimeWorkspaceState{
		Balance: BalanceState{
			Divergence: 0.3, // Converge: should produce Decision
		},
	}
	nodes, _ := p.GenerateNodesForCycle(context.Background(), session, state)
	if len(nodes) == 0 {
		t.Fatal("expected Decision node in convergence, got none")
	}
	for _, n := range nodes {
		if n.Type != NodeDecision {
			t.Errorf("expected NodeDecision in convergence, got %s", n.Type)
		}
	}
}

func TestInterventionAdjustsBalanceState(t *testing.T) {
	domain := newTestExplorationDomain()
	req := CreateWorkspaceReq{Topic: "quantum computing", OutputGoal: "summary"}
	snapshot := domain.CreateWorkspace(req)
	wsID := snapshot.Exploration.ID

	// Set known balance state
	domain.withWorkspaceState(wsID, func(state *RuntimeWorkspaceState) {
		state.Balance = BalanceState{WorkspaceID: wsID, Divergence: 0.6, Research: 0.6, Aggression: 0.4}
	})

	// Apply intervention with converging intent
	intReq := InterventionReq{Type: InterventionAddContext, Note: "please focus and 收敛"}
	domain.replanRuntimeState(snapshot.Exploration, intReq)

	var balance BalanceState
	domain.withWorkspaceState(wsID, func(state *RuntimeWorkspaceState) {
		balance = state.Balance
	})
	// "focus" and "收敛" both trigger Divergence -= 0.2, accumulated = -0.4 → clamped
	if balance.Divergence >= 0.6 {
		t.Errorf("expected Divergence to decrease after '收敛' intent, got %f", balance.Divergence)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd /home/o/space/IdeaFactory/backend && go test ./domain/exploration/... -run "TestDeterministicPlanner|TestInterventionAdjustsBalance" -v 2>&1 | head -40
```
Expected: tests compile and run but FAIL — the stub `GenerateNodesForCycle` returns `nil, nil`, so assertions like "expected Direction nodes" fail.

- [ ] **Step 3: Replace `deterministic.go` with full implementation**

Replace the entire contents of `deterministic.go` (overwriting the stub from Task 2):

```go
package exploration

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// DeterministicPlanner implements Planner without any LLM calls.
// Node generation is deterministic, based solely on graph state and BalanceState.
type DeterministicPlanner struct{}

// Compile-time interface check
var _ Planner = &DeterministicPlanner{}

func NewDeterministicPlanner() *DeterministicPlanner {
	return &DeterministicPlanner{}
}

// BuildInitialPlan creates a 3-step plan skeleton. Node generation happens
// separately via GenerateNodesForCycle.
func (p *DeterministicPlanner) BuildInitialPlan(ctx context.Context, session *ExplorationSession, state *RuntimeWorkspaceState) (*ExecutionPlan, []PlanStep, error) {
	return p.buildPlan(ctx, session, state, 1)
}

// Replan creates a new plan version after a trigger.
func (p *DeterministicPlanner) Replan(ctx context.Context, session *ExplorationSession, state *RuntimeWorkspaceState, trigger ReplanTrigger) (*ExecutionPlan, []PlanStep, error) {
	version := 1
	if len(state.Plans) > 0 {
		version = state.Plans[len(state.Plans)-1].Version + 1
	}
	return p.buildPlan(ctx, session, state, version)
}

func (p *DeterministicPlanner) buildPlan(_ context.Context, session *ExplorationSession, _ *RuntimeWorkspaceState, version int) (*ExecutionPlan, []PlanStep, error) {
	now := time.Now()
	planID := fmt.Sprintf("plan-%s-%d", session.ID, now.UnixNano())
	plan := &ExecutionPlan{
		ID:          planID,
		WorkspaceID: session.ID,
		Version:     version,
		CreatedAt:   now.UnixMilli(),
	}
	descs := []string{
		"generate directions and evidence nodes",
		"synthesize claims from evidence",
		"produce decisions and artifacts",
	}
	steps := make([]PlanStep, len(descs))
	for i, desc := range descs {
		steps[i] = PlanStep{
			ID:          fmt.Sprintf("%s-step-%d", planID, i+1),
			WorkspaceID: session.ID,
			PlanID:      planID,
			Index:       i + 1,
			Desc:        desc,
			Status:      PlanStepTodo,
			UpdatedAt:   now.UnixMilli(),
		}
	}
	return plan, steps, nil
}

// GenerateNodesForCycle applies the 7-rule priority table from the spec to determine
// what nodes and edges to generate next.
func (p *DeterministicPlanner) GenerateNodesForCycle(_ context.Context, session *ExplorationSession, state *RuntimeWorkspaceState) ([]Node, []Edge) {
	balance := state.Balance

	// Collect existing nodes by type
	var dirNodes []Node
	evidenceByDir := map[string][]Node{} // dirID → Evidence nodes pointing to it
	claimByDir := map[string]bool{}
	var hasDecision bool

	for _, n := range session.Nodes {
		switch n.Type {
		case NodeDirection:
			dirNodes = append(dirNodes, n)
		case NodeEvidence:
			evidenceByDir[n.Metadata.BranchID] = append(evidenceByDir[n.Metadata.BranchID], n)
		case NodeClaim:
			claimByDir[n.Metadata.BranchID] = true
		case NodeDecision:
			hasDecision = true
		}
	}

	wsID := session.ID
	now := time.Now()

	// Rule 1: No directions yet
	if len(dirNodes) == 0 {
		return p.generateDirections(session, wsID, now)
	}

	// Find under-evidenced directions (< 2 Evidence)
	var underEvidenced []Node
	for _, dir := range dirNodes {
		if len(evidenceByDir[dir.ID]) < 2 {
			underEvidenced = append(underEvidenced, dir)
		}
	}

	// Rules 2, 3 (slow path — only when Aggression <= 0.6)
	if len(underEvidenced) > 0 && balance.Aggression <= 0.6 {
		if balance.Research >= 0.5 {
			// Rule 2: Generate Evidence
			return p.generateEvidence(underEvidenced, wsID, now)
		}
		// Rule 3: Generate Artifact (produce output)
		return p.generateArtifact(wsID, now, nil, p.latestClaim(session))
	}

	// Rule 4/5: Generate Claims for directions that don't have one
	var claimlessDirs []Node
	for _, dir := range dirNodes {
		if !claimByDir[dir.ID] {
			claimlessDirs = append(claimlessDirs, dir)
		}
	}
	if len(claimlessDirs) > 0 {
		return p.generateClaims(claimlessDirs, evidenceByDir, wsID, now)
	}

	// Rule 6: Converge → Decision
	if !hasDecision && balance.Divergence < 0.4 {
		return p.generateDecision(session, wsID, now)
	}

	// Rule 7: Diverge → Unknowns
	var hasUnknown bool
	for _, n := range session.Nodes {
		if n.Type == NodeUnknown {
			hasUnknown = true
			break
		}
	}
	if balance.Divergence >= 0.6 && !hasUnknown {
		return p.generateUnknowns(dirNodes, evidenceByDir, wsID, now)
	}

	// Rule 8: Decision exists → Artifact
	if hasDecision {
		var decNode *Node
		for i := range session.Nodes {
			if session.Nodes[i].Type == NodeDecision {
				decNode = &session.Nodes[i]
				break
			}
		}
		return p.generateArtifact(wsID, now, decNode, nil)
	}

	return nil, nil
}

func (p *DeterministicPlanner) generateDirections(session *ExplorationSession, wsID string, now time.Time) ([]Node, []Edge) {
	words := topicWords(session.Topic, 5)
	if len(words) == 0 {
		words = []string{"exploration", "direction", "approach"}
	}
	nodes := make([]Node, 0, len(words))
	for i, w := range words {
		nodes = append(nodes, Node{
			ID:          fmt.Sprintf("dir-%s-%d-%d", wsID, now.UnixNano(), i),
			WorkspaceID: session.ID,
			Type:        NodeDirection,
			Title:       strings.ToUpper(w[:1]) + w[1:] + " direction",
			Summary:     "Explore " + w + " as a strategic direction",
			Status:      NodeActive,
			Score:       0.5,
			Depth:       0,
		})
	}
	return nodes, nil // Direction nodes have no edges
}

func (p *DeterministicPlanner) generateEvidence(dirs []Node, wsID string, now time.Time) ([]Node, []Edge) {
	var nodes []Node
	var edges []Edge
	edgeTypes := []EdgeType{EdgeSupports, EdgeContradicts}
	i := 0
	for _, dir := range dirs {
		count := 2
		if len(dirs) > 3 {
			count = 1 // Limit total output for large graphs
		}
		for j := 0; j < count; j++ {
			nID := fmt.Sprintf("ev-%s-%d-%d", wsID, now.UnixNano(), i)
			nodes = append(nodes, Node{
				ID:          nID,
				WorkspaceID: wsID,
				Type:        NodeEvidence,
				Title:       fmt.Sprintf("Evidence for %s", dir.Title),
				Summary:     fmt.Sprintf("Research signal supporting or challenging: %s", dir.Title),
				Status:      NodeActive,
				Score:       0.6,
				Metadata:    NodeMetadata{BranchID: dir.ID},
			})
			edges = append(edges, Edge{
				ID:          fmt.Sprintf("edge-%s-%d-%d", wsID, now.UnixNano(), i),
				WorkspaceID: wsID,
				From:        nID,
				To:          dir.ID,
				Type:        edgeTypes[i%2],
			})
			i++
		}
	}
	return nodes, edges
}

func (p *DeterministicPlanner) generateClaims(dirs []Node, evidenceByDir map[string][]Node, wsID string, now time.Time) ([]Node, []Edge) {
	var nodes []Node
	var edges []Edge
	for i, dir := range dirs {
		nID := fmt.Sprintf("cl-%s-%d-%d", wsID, now.UnixNano(), i)
		claim := Node{
			ID:          nID,
			WorkspaceID: wsID,
			Type:        NodeClaim,
			Title:       "Claim: " + dir.Title,
			Summary:     "Synthesized assertion based on evidence for: " + dir.Title,
			Status:      NodeActive,
			Score:       0.7,
			Metadata:    NodeMetadata{BranchID: dir.ID},
		}
		nodes = append(nodes, claim)
		// Connect claim to most recent Evidence (or Direction if fast-path)
		evs := evidenceByDir[dir.ID]
		if len(evs) > 0 {
			target := evs[len(evs)-1]
			edges = append(edges, Edge{
				ID:          fmt.Sprintf("edge-cl-ev-%s-%d-%d", wsID, now.UnixNano(), i),
				WorkspaceID: wsID,
				From:        nID,
				To:          target.ID,
				Type:        EdgeJustifies,
			})
		} else {
			// Fast path: connect claim to Direction
			edges = append(edges, Edge{
				ID:          fmt.Sprintf("edge-cl-dir-%s-%d-%d", wsID, now.UnixNano(), i),
				WorkspaceID: wsID,
				From:        nID,
				To:          dir.ID,
				Type:        EdgeJustifies,
			})
		}
	}
	return nodes, edges
}

func (p *DeterministicPlanner) generateDecision(session *ExplorationSession, wsID string, now time.Time) ([]Node, []Edge) {
	nID := fmt.Sprintf("dec-%s-%d", wsID, now.UnixNano())
	decision := Node{
		ID:          nID,
		WorkspaceID: wsID,
		Type:        NodeDecision,
		Title:       "Decision for: " + session.Topic,
		Summary:     "Resolved decision synthesizing all claims",
		Status:      NodeActive,
		Score:       0.85,
	}
	var edges []Edge
	for i, n := range session.Nodes {
		if n.Type == NodeClaim {
			edges = append(edges, Edge{
				ID:          fmt.Sprintf("edge-dec-%s-%d-%d", wsID, now.UnixNano(), i),
				WorkspaceID: wsID,
				From:        nID,
				To:          n.ID,
				Type:        EdgeJustifies,
			})
		}
	}
	return []Node{decision}, edges
}

func (p *DeterministicPlanner) generateUnknowns(dirs []Node, evidenceByDir map[string][]Node, wsID string, now time.Time) ([]Node, []Edge) {
	// Find direction with fewest evidence
	minDir := dirs[0]
	for _, d := range dirs[1:] {
		if len(evidenceByDir[d.ID]) < len(evidenceByDir[minDir.ID]) {
			minDir = d
		}
	}
	nID := fmt.Sprintf("unk-%s-%d", wsID, now.UnixNano())
	unknown := Node{
		ID:          nID,
		WorkspaceID: wsID,
		Type:        NodeUnknown,
		Title:       "Open question in: " + minDir.Title,
		Summary:     "Unresolved question that needs further exploration",
		Status:      NodeActive,
		Score:       0.4,
	}
	edge := Edge{
		ID:          fmt.Sprintf("edge-unk-%s-%d", wsID, now.UnixNano()),
		WorkspaceID: wsID,
		From:        nID,
		To:          minDir.ID,
		Type:        EdgeRaises,
	}
	return []Node{unknown}, []Edge{edge}
}

func (p *DeterministicPlanner) generateArtifact(wsID string, now time.Time, decision *Node, claim *Node) ([]Node, []Edge) {
	nID := fmt.Sprintf("art-%s-%d", wsID, now.UnixNano())
	artifact := Node{
		ID:          nID,
		WorkspaceID: wsID,
		Type:        NodeArtifact,
		Title:       "Output artifact",
		Summary:     "Synthesized output from exploration",
		Status:      NodeActive,
		Score:       0.9,
	}
	var edges []Edge
	if decision != nil {
		edges = append(edges, Edge{
			ID:          fmt.Sprintf("edge-art-%s-%d", wsID, now.UnixNano()),
			WorkspaceID: wsID,
			From:        nID,
			To:          decision.ID,
			Type:        EdgeResolves,
		})
	} else if claim != nil {
		edges = append(edges, Edge{
			ID:          fmt.Sprintf("edge-art-%s-%d", wsID, now.UnixNano()),
			WorkspaceID: wsID,
			From:        nID,
			To:          claim.ID,
			Type:        EdgeResolves,
		})
	}
	return []Node{artifact}, edges
}

func (p *DeterministicPlanner) latestClaim(session *ExplorationSession) *Node {
	for i := len(session.Nodes) - 1; i >= 0; i-- {
		if session.Nodes[i].Type == NodeClaim {
			n := session.Nodes[i]
			return &n
		}
	}
	return nil
}

// topicWords splits the topic into significant words, returning up to max.
func topicWords(topic string, max int) []string {
	stopwords := map[string]bool{
		"for": true, "the": true, "and": true, "a": true, "an": true,
		"of": true, "in": true, "to": true, "with": true, "on": true,
	}
	raw := strings.Fields(strings.ToLower(topic))
	var out []string
	for _, w := range raw {
		w = strings.Trim(w, ".,!?;:\"'")
		if len(w) < 3 || stopwords[w] {
			continue
		}
		out = append(out, w)
		if len(out) >= max {
			break
		}
	}
	return out
}
```

- [ ] **Step 4: Add `buildInitialBalance` to `deterministic.go`**

`buildInitialBalance` was NOT included in the Task 2 stub (it lived in `runtime_plan.go` until Task 5 deletes that file). Add it now at the bottom of `deterministic.go`:

```go
// buildInitialBalance returns the default BalanceState for a new run.
func buildInitialBalance(session ExplorationSession, runID string, now time.Time) BalanceState {
	return BalanceState{
		WorkspaceID: session.ID,
		RunID:       runID,
		Divergence:  0.6,
		Research:    0.7,
		Aggression:  0.45,
		Reason:      "bootstrap exploration state from initial workspace graph",
		UpdatedAt:   now.UnixMilli(),
	}
}
```

Then delete `buildInitialBalance` from `runtime_plan.go` (the remaining functions in that file will be handled in Task 5). Verify the build still passes after this edit.

- [ ] **Step 5: Run the new tests**

```bash
cd /home/o/space/IdeaFactory/backend && go test ./domain/exploration/... -run "TestDeterministicPlanner|TestInterventionAdjustsBalance" -v
```
Expected: all 5 new tests PASS.

- [ ] **Step 6: Run full test suite**

```bash
cd /home/o/space/IdeaFactory/backend && go test ./domain/exploration/... -count=1 -timeout=60s
```
Expected: same pass/skip as before + 5 new passing tests.

- [ ] **Step 7: Commit**

```bash
git add backend/domain/exploration/deterministic.go backend/domain/exploration/api_test.go
git commit -m "feat: implement DeterministicPlanner with 7-rule node generation (TDD)"
```

---

## Chunk 3: Runtime cleanup — delete dead files, move LLM function

### Task 5: Move `generatePlanStepsWithModel`; delete `runtime_plan.go` and `runtime_tasks.go`

**Files:**
- Modify: `backend/domain/exploration/runtime_llm.go` (receive moved method)
- Delete: `backend/domain/exploration/runtime_plan.go`
- Delete: `backend/domain/exploration/runtime_tasks.go`

- [ ] **Step 1: Move `generatePlanStepsWithModel` to `runtime_llm.go`**

`runtime_plan.go` currently contains:
1. `buildInitialBalance` — moved to `deterministic.go` in Task 4 Step 4 (and removed from `runtime_plan.go` in that same step)
2. `buildInitialPlan` — replaced by `DeterministicPlanner.BuildInitialPlan`
3. `generatePlanStepsWithModel` — a method on `*ExplorationDomain` that uses the LLM

Append `generatePlanStepsWithModel` to `runtime_llm.go` by copying the entire function body from `runtime_plan.go` lines 88–127.

Then **explicitly add** the two imports that `runtime_llm.go` currently lacks:
- `"encoding/json"`
- `"github.com/kaptinlin/jsonrepair"`

The full updated import block for `runtime_llm.go` should be:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/kaptinlin/jsonrepair"
)
```

- [ ] **Step 2: Delete `runtime_plan.go`**

```bash
rm backend/domain/exploration/runtime_plan.go
```

- [ ] **Step 3: Delete `runtime_tasks.go`**

`runtime_tasks.go` contains `subAgentForStep` and `dispatchPlanSteps`. The latter is no longer called. `subAgentForStep` is still called by `executeNextTodoStepLocked` in `runtime_agent.go`.

Move `subAgentForStep` to `runtime_agent.go` (add it after the last function in the file), then delete `runtime_tasks.go`:

```go
func subAgentForStep(index int) string {
	switch index {
	case 1:
		return "research"
	case 2:
		return "graph"
	case 3:
		return "artifact"
	default:
		return "general"
	}
}
```

```bash
rm backend/domain/exploration/runtime_tasks.go
```

- [ ] **Step 4: Verify build**

```bash
cd /home/o/space/IdeaFactory/backend && go build ./domain/exploration/...
```

- [ ] **Step 5: Run full test suite**

```bash
cd /home/o/space/IdeaFactory/backend && go test ./domain/exploration/... -count=1 -timeout=60s
```
Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add backend/domain/exploration/runtime_llm.go \
        backend/domain/exploration/runtime_agent.go
git rm backend/domain/exploration/runtime_plan.go \
       backend/domain/exploration/runtime_tasks.go
git commit -m "refactor: delete runtime_plan.go and runtime_tasks.go; move generatePlanStepsWithModel to runtime_llm.go"
```

---

## Chunk 4: api_v1.go split into focused handler files

### Task 6: Create handler_shared.go and handler_workspace.go

**Files:**
- Create: `backend/domain/exploration/handler_shared.go`
- Create: `backend/domain/exploration/handler_workspace.go`

- [ ] **Step 1: Create `handler_shared.go`**

```go
package exploration

import (
	"time"
	"github.com/gin-gonic/gin"
)

func writeV1Error(c *gin.Context, status int, code string, message string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func toRFC3339(ms int64) string {
	if ms <= 0 {
		return ""
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}
```

- [ ] **Step 2: Create `handler_workspace.go`**

```go
package exploration

import (
	"net/http"
	"strings"
	"time"
	"github.com/gin-gonic/gin"
)

func (d *ExplorationDomain) ApiV1CreateWorkspace(c *gin.Context) {
	// (copy from api_v1.go — already updated in Task 3 to call initializeWorkspaceGraph)
}

func (d *ExplorationDomain) ApiV1GetWorkspace(c *gin.Context) {
	// (copy from api_v1.go)
}

func toWorkspaceView(session ExplorationSession) WorkspaceView {
	// (copy from api_v1.go)
}
```

- [ ] **Step 3: Remove the moved functions from `api_v1.go`**

Delete `ApiV1CreateWorkspace`, `ApiV1GetWorkspace`, `toWorkspaceView`, `writeV1Error`, `toRFC3339` from `api_v1.go`. The file will still build because these definitions now live in the new files.

- [ ] **Step 4: Verify build and tests**

```bash
cd /home/o/space/IdeaFactory/backend && go build ./domain/exploration/... && go test ./domain/exploration/... -count=1 -timeout=60s
```

- [ ] **Step 5: Commit**

```bash
git add backend/domain/exploration/handler_shared.go \
        backend/domain/exploration/handler_workspace.go \
        backend/domain/exploration/api_v1.go
git commit -m "refactor: extract handler_shared.go and handler_workspace.go from api_v1.go"
```

---

### Task 7: Create handler_run.go and handler_intervention.go

**Files:**
- Create: `backend/domain/exploration/handler_run.go`
- Create: `backend/domain/exploration/handler_intervention.go`

- [ ] **Step 1: Create `handler_run.go`**

Move these from `api_v1.go`:

**Exported handlers:** `ApiV1CreateRun`, `ApiV1GetRun`, `ApiV1GetTraceSummary`, `ApiV1ListTraceEvents`

**Private helpers (all move together):** `buildRunView`, `buildTraceSummary`, `applyTracePagination`, `isValidTraceCategory`, `isValidTraceLevel`, `normalizeRunStatus`, `normalizeStepStatus`, `normalizeAgentName`, `derivePlanStatus`, `deriveRunStatus`, `inferAgentFromStep`, `indexOfPlan`

The file header:
```go
package exploration

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"github.com/gin-gonic/gin"
)
```

- [ ] **Step 2: Create `handler_intervention.go`**

Move these from `api_v1.go`:

**Exported handlers:** `ApiV1CreateIntervention`, `ApiV1GetIntervention`, `ApiV1ListInterventionEvents`

**Private helpers:** `mapInterventionReq`, `storeInterventionRecord`, `getInterventionRecord`, `advanceInterventionByRuntimeEvent`, `listInterventionEvents`, `decodeInterventionEventView`, `applyEventPagination`, `findStartIndexByCursor`, `encodeEventCursor`

The file header:
```go
package exploration

import (
	"backend/datasource/dbdao"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"github.com/gin-gonic/gin"
)
```

- [ ] **Step 3: Remove moved functions from `api_v1.go`**

After moving all run and intervention functions, `api_v1.go` should contain only `ApiV1GetProjection`, `buildProjectionResponse`, and any remaining helpers.

- [ ] **Step 4: Verify build and tests**

```bash
cd /home/o/space/IdeaFactory/backend && go build ./domain/exploration/... && go test ./domain/exploration/... -count=1 -timeout=60s
```

- [ ] **Step 5: Commit**

```bash
git add backend/domain/exploration/handler_run.go \
        backend/domain/exploration/handler_intervention.go \
        backend/domain/exploration/api_v1.go
git commit -m "refactor: extract handler_run.go and handler_intervention.go from api_v1.go"
```

---

### Task 8: Create projection_builder.go; delete api_v1.go

**Files:**
- Create: `backend/domain/exploration/projection_builder.go`
- Delete: `backend/domain/exploration/api_v1.go`

- [ ] **Step 1: Create `projection_builder.go`**

Move these from `api_v1.go`:
- `ApiV1GetProjection`
- `buildProjectionResponse`
- Any remaining helper functions in `api_v1.go`

The file header:
```go
package exploration

import (
	"fmt"
	"net/http"
	"time"
	"github.com/gin-gonic/gin"
)
```

- [ ] **Step 2: Verify `api_v1.go` is now empty (only package declaration)**

At this point `api_v1.go` should have no functions remaining. Delete it:

```bash
rm backend/domain/exploration/api_v1.go
```

- [ ] **Step 3: Verify build and tests**

```bash
cd /home/o/space/IdeaFactory/backend && go build ./domain/exploration/... && go test ./domain/exploration/... -count=1 -timeout=60s
```
Expected: all tests pass; no compilation errors.

- [ ] **Step 4: Commit**

```bash
git add backend/domain/exploration/projection_builder.go
git rm backend/domain/exploration/api_v1.go
git commit -m "refactor: extract projection_builder.go; delete api_v1.go (split complete)"
```

---

## Chunk 5: Tests — update internal-access tests, add mutation event test

### Task 9: Update existing tests and add mutation event test

**Files:**
- Modify: `backend/domain/exploration/api_test.go`

- [ ] **Step 1: Update `TestV1ToggleFavoriteInterventionAffectsWorkspaceState`**

Find the line:
```go
if node.Type == NodeIdea {
```
Change to:
```go
if node.Type == NodeDirection {
```

- [ ] **Step 2: Update `TestV1InterventionCanRecoverFromDB`**

Find the lines:
```go
domain.runtime.mu.Lock()
delete(domain.runtime.intervention, created.Workspace.ID)
domain.runtime.mu.Unlock()
```
Replace with:
```go
domain.withWorkspaceState(created.Workspace.ID, func(state *RuntimeWorkspaceState) {
    state.Interventions = map[string]InterventionView{}
})
```

- [ ] **Step 3: Add `TestMutationEventsWrittenOnRunComplete`**

First, ensure `api_test.go` imports `"context"`. If the existing import block does not already include it, add it.

Add at the end of `api_test.go`:

```go
func TestMutationEventsWrittenOnRunComplete(t *testing.T) {
	router, domain := newTestRouterWithDomain()
	_ = router

	// Create workspace (initializeWorkspaceGraph is called inside CreateWorkspace handler)
	wsID := ""
	{
		req := CreateWorkspaceReq{Topic: "autonomous vehicles safety", OutputGoal: "risk summary"}
		snapshot := domain.CreateWorkspace(req)
		wsID = snapshot.Exploration.ID
		// Also call initializeWorkspaceGraph (normally called from handler)
		domain.initializeWorkspaceGraph(context.Background(), wsID)
	}

	var mutations []MutationEvent
	domain.withWorkspaceState(wsID, func(state *RuntimeWorkspaceState) {
		mutations = append(mutations, state.Mutations...)
	})

	if len(mutations) == 0 {
		t.Fatal("expected at least one mutation event after initializeWorkspaceGraph, got none")
	}

	// Check that at least one node_added event exists
	hasNodeAdded := false
	for _, m := range mutations {
		if m.Kind == "node_added" {
			hasNodeAdded = true
			break
		}
	}
	if !hasNodeAdded {
		t.Error("expected at least one 'node_added' mutation event")
	}
}
```

- [ ] **Step 4: Run all tests**

```bash
cd /home/o/space/IdeaFactory/backend && go test ./domain/exploration/... -count=1 -timeout=60s -v 2>&1 | tail -30
```
Expected: all tests PASS (plus 6 new tests PASS).

- [ ] **Step 5: Run frontend tests to confirm no regression**

```bash
cd /home/o/space/IdeaFactory/frontend && npm test -- --run 2>&1 | tail -20
```
Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add backend/domain/exploration/api_test.go
git commit -m "test: update internal-access tests + add TestMutationEventsWrittenOnRunComplete"
```

---

## Final Verification

- [ ] **Full backend test suite**

```bash
cd /home/o/space/IdeaFactory/backend && go test ./domain/exploration/... -count=1 -timeout=60s
```
Expected: all tests pass.

- [ ] **Deleted files confirmed**

```bash
ls backend/domain/exploration/api_v1.go backend/domain/exploration/runtime_plan.go backend/domain/exploration/runtime_tasks.go 2>&1
```
Expected: `ls: cannot access` for all three files.

- [ ] **Interface check present**

```bash
grep "var _ Planner" backend/domain/exploration/deterministic.go
```
Expected: `var _ Planner = &DeterministicPlanner{}`

- [ ] **No file exceeds 400 lines**

```bash
wc -l backend/domain/exploration/handler_*.go backend/domain/exploration/deterministic.go backend/domain/exploration/projection_builder.go backend/domain/exploration/planner.go | sort -rn | head -10
```
Expected: all new files < 400 lines.
