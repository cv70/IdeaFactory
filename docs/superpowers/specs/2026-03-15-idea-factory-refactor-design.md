# Idea Factory — Refactor & Runtime Semantics Design (v2)

**Date:** 2026-03-15
**Scope:** Sub-project A (code structure) + Sub-project B (runtime semantics) — combined single-pass approach
**Constraint:** All existing tests remain green throughout; HTTP API behavior is preserved

---

## 1. Motivation

The current backend has two distinct problems that compound each other:

**Structure problems (Sub-project A):**
- `domain.go` holds 10 loose maps inside `runtimeState` with no cohesion; callers coordinate many keys manually
- `api_v1.go` (864 lines) mixes HTTP handlers with domain logic and projection building
- `runtime_plan.go` and `runtime_tasks.go` are thin wrappers with misleading names (`dispatchPlanSteps` executes only the first step)
- Mutation events are never written from the runtime path, so SSE subscribers receive no incremental updates
- No `Planner` interface; the deterministic logic is baked into `runtime_agent.go` with no seam for extension

**Semantic problems (Sub-project B):**
- Deterministic runtime generates `NodeOpportunity`/`NodeIdea` (old ontology); spec requires Direction, Evidence, Claim, Decision, Unknown nodes
- `BalanceState` fields (`Divergence`, `Research`, `Aggression` as float64) are stored but never read during planning
- Intervention `intent` text is recorded but does not influence the graph or future runs
- No edges between nodes are generated

---

## 2. File Disposition

Every existing file in `backend/domain/exploration/` is accounted for:

| File | Action | Notes |
|------|--------|-------|
| `domain.go` | Modify | Replace `runtimeState` with `RuntimeWorkspaceState`; add `Planner` field |
| `schema.go` | Modify | Add missing `NodeDirection`, `NodeArtifact`, `EdgeJustifies`, `EdgeBranchesFrom`, `EdgeRaises`, `EdgeResolves` constants |
| `api_v1.go` | Split + Delete | Content distributed across `handler_workspace.go`, `handler_run.go`, `handler_intervention.go`, `projection_builder.go` |
| `runtime_agent.go` | Modify | Calls `Planner` interface; writes mutation events; removes direct `runtimeState` map access |
| `runtime_plan.go` | Delete | Content merged into `deterministic.go` |
| `runtime_tasks.go` | Delete | Content merged into `deterministic.go` |
| `mutations.go` | Unchanged | `diffMutations` and `mutationID` helpers stay as-is |
| `workspace_management.go` | Modify | Update to use `RuntimeWorkspaceState` instead of `runtimeState` maps |
| `runtime_context.go` | Unchanged | Utility types/functions; no changes needed |
| `runtime_operator.go` | Unchanged | `RuntimeOperator` interface; used by `runtime_llm.go` |
| `runtime_llm.go` | Unchanged | LLM path; not used by `DeterministicPlanner` |
| `persistence.go` | Unchanged | Persistence layer; no changes needed |
| `realtime.go` | Unchanged | WebSocket/SSE logic; no changes needed |
| `cursor.go` | Unchanged | Cursor helpers; no changes needed |
| `exploration.go` | Unchanged | Legacy route handlers; no changes needed |
| `api.go` | Unchanged | Legacy route definitions; no changes needed |
| `routes.go` | Unchanged | Route registration; no changes needed |

**New files to create:**

| File | Content |
|------|---------|
| `planner.go` | `Planner` interface + `ReplanTrigger` type |
| `deterministic.go` | `DeterministicPlanner` struct implementing `Planner` |
| `handler_workspace.go` | `ApiV1CreateWorkspace`, `ApiV1GetWorkspace` |
| `handler_run.go` | `ApiV1CreateRun`, `ApiV1GetRun`, `ApiV1GetTraceSummary`, `ApiV1ListTraceEvents` |
| `handler_intervention.go` | `ApiV1CreateIntervention`, `ApiV1GetIntervention`, `ApiV1ListInterventionEvents`, `mapInterventionReq`, `storeInterventionRecord`, `advanceInterventionByRuntimeEvent` |
| `projection_builder.go` | `ApiV1GetProjection`, `buildProjectionResponse`, all projection helper functions |

---

## 3. Data Model Changes

### 3.1 RuntimeWorkspaceState

Replace `runtimeState`'s 10 loose maps with one cohesive struct per workspace:

```go
// domain.go
type RuntimeWorkspaceState struct {
    Runs          []Run
    Plans         []ExecutionPlan
    PlanSteps     []PlanStep
    AgentTasks    []AgentTask
    Results       []AgentTaskResultSummary
    Balance       BalanceState
    Mutations     []MutationEvent
    ReplanReason  string
    Interventions map[string]InterventionView  // keyed by intervention ID
    Running       bool
    Cursor        int
}

type ExplorationDomain struct {
    DB        *dbdao.DB
    DeepAgent adk.ResumableAgent
    General   adk.Agent
    Model     model.ToolCallingChatModel
    store     *workspaceStore
    ws        *wsState
    planner   Planner
    runtime   *runtimeWorkspaces   // replaces runtimeState
}

type runtimeWorkspaces struct {
    mu         sync.Mutex
    workspaces map[string]*RuntimeWorkspaceState  // keyed by workspace ID
}
```

**Accessor pattern:** `runtime_agent.go` and `workspace_management.go` access runtime state through two helpers defined in `domain.go`:

```go
// getWorkspaceState returns state, initializing if absent.
func (d *ExplorationDomain) getWorkspaceState(workspaceID string) *RuntimeWorkspaceState

// withWorkspaceState locks the mutex and calls fn with the state.
// fn must not call other withWorkspaceState calls (no re-entry).
func (d *ExplorationDomain) withWorkspaceState(workspaceID string, fn func(*RuntimeWorkspaceState))
```

`runtimeState` (the old struct) is removed entirely. All previous `d.runtime.runs[id]`, `d.runtime.plans[id]`, etc. usages are rewritten to use `state.Runs`, `state.Plans`, etc. via `withWorkspaceState`.

**Initialization:** `getWorkspaceState` lazily creates a `RuntimeWorkspaceState` with zero values and an empty `Interventions` map on first access.

### 3.2 Planner Interface

```go
// planner.go

type ReplanTriggerKind string

const (
    ReplanTriggerIntervention ReplanTriggerKind = "intervention"
    ReplanTriggerBalanceShift ReplanTriggerKind = "balance_shift"
    ReplanTriggerManual       ReplanTriggerKind = "manual"
)

type ReplanTrigger struct {
    Kind         ReplanTriggerKind
    Intervention *InterventionView  // set when Kind == ReplanTriggerIntervention
}

type Planner interface {
    // BuildInitialPlan creates the first plan for a workspace.
    BuildInitialPlan(ctx context.Context, session *ExplorationSession, state *RuntimeWorkspaceState) (*ExecutionPlan, []PlanStep, error)

    // Replan creates a new plan version given the current runtime state and a trigger.
    Replan(ctx context.Context, session *ExplorationSession, state *RuntimeWorkspaceState, trigger ReplanTrigger) (*ExecutionPlan, []PlanStep, error)
}
```

`DeterministicPlanner` is a zero-dependency struct (no LLM, no Model field):

```go
// deterministic.go

type DeterministicPlanner struct{}

func NewDeterministicPlanner() *DeterministicPlanner { return &DeterministicPlanner{} }
```

Domain is initialized with `NewDeterministicPlanner()` in `NewExplorationDomain`. `runtime_llm.go` (and its `generatePlanStepsWithModel`) is left unchanged as a separate, unused-by-default path.

### 3.3 Node and Edge Type Constants (schema.go additions)

Add to existing `NodeType` constants block (do NOT rename or remove existing constants):

```go
const (
    // existing constants unchanged ...

    NodeDirection NodeType = "direction"   // NEW
    NodeArtifact  NodeType = "artifact"    // NEW
)
```

Add to existing `EdgeType` constants block:

```go
const (
    // existing constants unchanged ...

    EdgeJustifies    EdgeType = "justifies"     // NEW
    EdgeBranchesFrom EdgeType = "branches_from" // NEW
    EdgeRaises       EdgeType = "raises"         // NEW
    EdgeResolves     EdgeType = "resolves"       // NEW
)
```

No existing constants are renamed or removed.

---

## 4. Node Generation Semantics

`DeterministicPlanner` inspects the current graph (nodes in `ExplorationSession.Nodes`) and `RuntimeWorkspaceState.Balance` to decide what to generate. Each plan step, when executed by `runtime_agent.go`, produces new `Node` and `Edge` records that are appended to the session.

### 4.1 "Few Evidence" Threshold

A Direction node has "sufficient Evidence" when `>= 2` Evidence nodes with edges to it exist. Fewer than 2 = "few Evidence."

### 4.2 Generation Rules (evaluated in priority order)

| Priority | Precondition | BalanceState condition | Action |
|----------|-------------|----------------------|--------|
| 1 | No Direction nodes exist | any | Generate 3–5 Direction nodes from topic words |
| 2 | Direction nodes exist, any Direction has < 2 Evidence AND `Aggression <= 0.6` | `Research >= 0.5` | Generate 1–2 Evidence nodes per under-evidenced Direction |
| 3 | Direction nodes exist, any Direction has < 2 Evidence AND `Aggression <= 0.6` | `Research < 0.5` | Generate 1 Artifact node summarizing current knowledge |
| 4 | Direction nodes exist AND `Aggression > 0.6` (fast path) | any | Skip evidence; proceed directly to rule 5 |
| 5 | Every active Direction has >= 2 Evidence, no Claim yet per Direction | any | Generate 1 Claim node per Direction (synthesizes Evidence) |
| 6 | Claim nodes exist, `Divergence < 0.4` (converge mode), no Decision exists | any | Generate 1 Decision node resolving Claims |
| 7 | Claim nodes exist, `Divergence >= 0.6` (diverge mode) | any | Generate 1–2 Unknown nodes representing open questions |
| 8 | Decision node exists | any | Generate 1 Artifact node (final output); mark run complete |

Rules are evaluated top-to-bottom; first matching rule wins.

### 4.3 Edge Generation

Every newly generated node is connected to its parent(s):

| New node type | Edge type | Connected to |
|--------------|----------|--------------|
| Evidence | `EdgeSupports` or `EdgeContradicts` (alternating per node for variety) | Parent Direction |
| Claim | `EdgeJustifies` | Its most-recent Evidence node |
| Decision | `EdgeJustifies` | All Claim nodes in the workspace |
| Unknown | `EdgeRaises` | The Direction with fewest Evidence nodes |
| Artifact | `EdgeResolves` | The Decision node (if exists), or the most-recent Claim |
| Direction | none | — |

### 4.4 BalanceState Adjustment from Interventions

When an intervention is submitted, its `intent` text is scanned (case-insensitive substring match) to adjust `BalanceState.Divergence`, `Research`, and `Aggression` before `Replan` is called. All values are clamped to [0.0, 1.0].

| Keyword | Field | Delta |
|---------|-------|-------|
| "focus", "decide", "收敛", "converge" | Divergence | −0.2 |
| "explore", "expand", "发散", "diverge" | Divergence | +0.2 |
| "research", "evidence", "调研" | Research | +0.2 |
| "produce", "output", "产出" | Research | −0.2 |
| "fast", "quick", "aggressive" | Aggression | +0.2 |
| "careful", "thorough", "prudent" | Aggression | −0.2 |

Multiple keywords may match; adjustments accumulate.

---

## 5. Mutation Events

Use the existing `MutationEvent` struct (defined in `schema.go`) and the existing `mutationID` helper (in `mutations.go`). The runtime path appends to `RuntimeWorkspaceState.Mutations` and fans out to subscribers via the existing `realtime.go` broadcast mechanism.

### 5.1 New Event Kinds

These events are emitted by `runtime_agent.go` using existing `MutationEvent` fields:

| When | `Kind` string | Payload fields used |
|------|--------------|-------------------|
| Run record created | `"run_created"` | `Run *GenerationRun` |
| BalanceState adjusted by intervention | `"balance_updated"` | (no standard field; store adjustment summary in future `Reason` field — for now emit event with just Kind + WorkspaceID) |
| Intervention absorbed into runtime | `"intervention_absorbed"` | `ActiveOpportunityID` = intervention ID (repurposed as a string carrier) |
| Replan triggered | `"replanned"` | `Run *GenerationRun` of the new plan's associated run |

**Note:** Node and edge generation mutations are handled by calling `diffMutations(prevSession, nextSession, "runtime")` after each cycle, which already emits `"node_added"` and `"edge_added"` events. Do not emit a separate `"nodes_generated"` event.

### 5.2 Where Events Are Written

In `runtime_agent.go`, after each of the following operations:

```
CreateRun → emit "run_created"
Intervention absorbed → emit "intervention_absorbed" then call diffMutations
BalanceState adjusted → emit "balance_updated"
Replan executed → emit "replanned"
Run cycle completes (nodes/edges added) → call diffMutations(prev, next, "runtime") and broadcast results
```

All events are appended to `state.Mutations` and passed to `broadcastMutations(workspaceID, events)` from `realtime.go`.

---

## 6. File Split: api_v1.go → handler_*.go + projection_builder.go

`api_v1.go` is split by responsibility. Function bodies are not modified during the move.

| Destination | Functions moved |
|-------------|----------------|
| `handler_workspace.go` | `ApiV1CreateWorkspace`, `ApiV1GetWorkspace` |
| `handler_run.go` | `ApiV1CreateRun`, `ApiV1GetRun`, `ApiV1GetTraceSummary`, `ApiV1ListTraceEvents` |
| `handler_intervention.go` | `ApiV1CreateIntervention`, `ApiV1GetIntervention`, `ApiV1ListInterventionEvents`, `mapInterventionReq`, `storeInterventionRecord`, `advanceInterventionByRuntimeEvent` |
| `projection_builder.go` | `ApiV1GetProjection`, `buildProjectionResponse` and all helper functions |

`api_v1.go` is deleted after the split. Route registration stays in `routes.go`.

`handler_intervention.go` accesses runtime state through the `withWorkspaceState` accessor defined in Section 3.1. The `storeInterventionRecord` and `advanceInterventionByRuntimeEvent` functions are updated to use `state.Interventions[id]` instead of `d.runtime.intervention[workspaceID][id]`.

**File size guidance:** Target < 400 lines per new handler file. `projection_builder.go` may reach ~300 lines given all the projection helpers it contains; this is acceptable.

---

## 7. Renames

| Old name | New name | Reason |
|----------|----------|--------|
| `dispatchPlanSteps` | `executeFirstPlanStep` | MVP only executes first step; name must reflect this |
| `runtime_plan.go` | merged into `deterministic.go` | file is eliminated |
| `runtime_tasks.go` | merged into `deterministic.go` | file is eliminated |

`runtime_llm.go` stays as-is. `DeterministicPlanner` does not call `generatePlanStepsWithModel`.

---

## 8. Testing

### 8.1 Existing tests (must stay green)

All HTTP-level tests in `api_test.go` test through the HTTP interface and must remain green. No API response shapes change.

### 8.2 Tests that access `domain.runtime.*` directly

`TestV1InterventionCanRecoverFromDB` accesses `domain.runtime.intervention` directly to simulate a restart. Update this test to use the new accessor:

```go
// Before:
domain.runtime.mu.Lock()
delete(domain.runtime.intervention, created.Workspace.ID)
domain.runtime.mu.Unlock()

// After:
domain.withWorkspaceState(created.Workspace.ID, func(state *RuntimeWorkspaceState) {
    state.Interventions = map[string]InterventionView{}
})
```

`TestV1ToggleFavoriteInterventionAffectsWorkspaceState` checks for `NodeIdea`. Update to check for `NodeDirection` instead, since the deterministic planner now generates Direction nodes as the primary type:

```go
// Before:
if node.Type == NodeIdea {
// After:
if node.Type == NodeDirection {
```

### 8.3 New tests to add

- `TestDeterministicPlannerInitialRun` — verifies first run produces Direction nodes; no Evidence/Claim/Decision nodes yet
- `TestDeterministicPlannerResearchPhase` — verifies second run with default `Balance.Research >= 0.5` produces Evidence nodes with `EdgeSupports` edges to Direction nodes
- `TestDeterministicPlannerConvergence` — verifies Claim nodes generated when all Directions have >= 2 Evidence; Decision generated when `Divergence < 0.4`
- `TestInterventionAdjustsBalanceState` — verifies keyword "收敛" in intent shifts `Divergence` down by 0.2 before replan
- `TestMutationEventsWrittenOnRunComplete` — verifies `state.Mutations` is non-empty after a completed run cycle

---

## 9. Out of Scope

- LLM-backed planner (remains a future extension point via the `Planner` interface; `runtime_llm.go` is preserved unchanged)
- Frontend rendering changes beyond verifying existing node type display works
- Database persistence (runtime state remains in-memory)
- Authentication, multi-user, or workspace sharing
- BalanceState change to enum types (kept as float64; enum refactor is a separate sub-project)
- Changes to `exploration.go`, `api.go`, `persistence.go`, `realtime.go`, `cursor.go`, `runtime_context.go`, `runtime_operator.go`, `mutations.go`

---

## 10. Acceptance Criteria

1. `go test ./domain/exploration/...` — all tests pass (including updated `TestV1InterventionCanRecoverFromDB` and `TestV1ToggleFavoriteInterventionAffectsWorkspaceState`)
2. `npm test` — all frontend tests pass
3. A fresh run on a new workspace produces Direction nodes (type `"direction"`) in the projection response
4. Evidence nodes appear after the second run cycle and have `EdgeSupports` edges to Direction nodes
5. Submitting an intervention with intent containing "收敛" reduces `BalanceState.Divergence` by 0.2 (observable via trace or direct state check in test)
6. `state.Mutations` is non-empty after a completed run (verified by `TestMutationEventsWrittenOnRunComplete`)
7. `api_v1.go` is deleted; `runtime_plan.go` and `runtime_tasks.go` are deleted
8. `DeterministicPlanner` implements the `Planner` interface (verified by compiler: `var _ Planner = &DeterministicPlanner{}`)
9. No new file created in this PR exceeds 400 lines
10. `withWorkspaceState` is the only path for mutating `RuntimeWorkspaceState` fields in handler and runtime files
