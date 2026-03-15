# Idea Factory — Refactor & Runtime Semantics Design (v3)

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
| `domain.go` | Modify | Replace `runtimeState`/`runtimeWorkspaces`; add `planner Planner` field; add accessor helpers |
| `schema.go` | Modify | Add missing `NodeDirection`, `NodeArtifact`, `EdgeJustifies`, `EdgeBranchesFrom`, `EdgeRaises`, `EdgeResolves` constants |
| `api_v1.go` | Split + Delete | Content distributed across `handler_shared.go`, `handler_workspace.go`, `handler_run.go`, `handler_intervention.go`, `projection_builder.go` |
| `runtime_agent.go` | Modify | Calls `Planner` interface; writes mutation events; removes direct `runtimeState` map access; adds `initializeWorkspaceGraph` |
| `runtime_plan.go` | Delete + Split | `buildInitialPlan` logic moves to `deterministic.go`; `generatePlanStepsWithModel` (method on `ExplorationDomain`) moves to `runtime_llm.go` |
| `runtime_tasks.go` | Delete | Content merged into `deterministic.go`; `dispatchPlanSteps` renamed to `executeFirstPlanStep` |
| `runtime_llm.go` | Modify | Receives `generatePlanStepsWithModel` method (moved from `runtime_plan.go`) |
| `workspace_management.go` | Modify | `ArchiveWorkspace`: replace `d.runtime.mu.Lock()`/delete calls with `withWorkspaceState` |
| `mutations.go` | Unchanged | `diffMutations` and `mutationID` helpers stay as-is |
| `runtime_context.go` | Unchanged | Utility types/functions; no changes needed |
| `runtime_operator.go` | Unchanged | `RuntimeOperator` interface |
| `persistence.go` | Unchanged | Persistence layer |
| `realtime.go` | Unchanged | WebSocket/SSE logic |
| `cursor.go` | Unchanged | Cursor helpers |
| `exploration.go` | Unchanged | Legacy route handlers |
| `api.go` | Unchanged | Legacy route definitions |
| `routes.go` | Unchanged | Route registration |

**New files to create:**

| File | Content |
|------|---------|
| `planner.go` | `Planner` interface, `ReplanTrigger`, `ReplanTriggerKind` constants |
| `deterministic.go` | `DeterministicPlanner` struct implementing `Planner`; node + edge generation logic |
| `handler_shared.go` | Shared handler utilities: `writeV1Error`, `toRFC3339` |
| `handler_workspace.go` | `ApiV1CreateWorkspace`, `ApiV1GetWorkspace`, `toWorkspaceView` |
| `handler_run.go` | Run handlers + run/trace helper functions (see Section 6) |
| `handler_intervention.go` | Intervention handlers + intervention helper functions (see Section 6) |
| `projection_builder.go` | `ApiV1GetProjection`, `buildProjectionResponse`, projection helpers |

---

## 3. Data Model Changes

### 3.1 RuntimeWorkspaceState

Replace the `runtimeState` struct and its 10 maps with one cohesive struct per workspace:

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
    Interventions map[string]InterventionView  // keyed by intervention ID; init to empty map
    Running       bool
    Cursor        int
}

type runtimeWorkspaces struct {
    mu         sync.Mutex
    workspaces map[string]*RuntimeWorkspaceState  // keyed by workspace ID
}

type ExplorationDomain struct {
    DB        *dbdao.DB
    DeepAgent adk.ResumableAgent
    General   adk.Agent
    Model     model.ToolCallingChatModel
    store     *workspaceStore
    ws        *wsState
    planner   Planner
    runtime   *runtimeWorkspaces  // replaces old runtimeState
}
```

**Accessor helpers defined in `domain.go`:**

```go
// getWorkspaceState returns the state for workspaceID, initializing it (with empty Interventions map) if absent.
// Callers must hold runtime.mu before calling.
func (d *ExplorationDomain) getWorkspaceState(workspaceID string) *RuntimeWorkspaceState

// withWorkspaceState locks runtime.mu, fetches or initializes the state, calls fn, then unlocks.
// fn MUST NOT call withWorkspaceState recursively.
func (d *ExplorationDomain) withWorkspaceState(workspaceID string, fn func(*RuntimeWorkspaceState))
```

**`Planner` methods are always called from within a `withWorkspaceState` callback.** This ensures all state mutations (appending nodes, updating balance, writing mutations) are covered by the lock.

**`GetRuntimeState` and `QueryRuntimeState`** (public methods in `runtime_agent.go`) are updated to use `withWorkspaceState` internally, copying the relevant state fields out into the return value before the callback returns.

**`ArchiveWorkspace` in `workspace_management.go`** replaces the two `d.runtime.mu.Lock()` / delete calls with:
```go
d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
    state.Running = false
    state.Cursor = 0
    state.Interventions = map[string]InterventionView{}
})
```

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
    Intervention *InterventionView  // non-nil when Kind == ReplanTriggerIntervention
}

type Planner interface {
    // BuildInitialPlan creates the first plan for a workspace. Called inside withWorkspaceState.
    BuildInitialPlan(ctx context.Context, session *ExplorationSession, state *RuntimeWorkspaceState) (*ExecutionPlan, []PlanStep, error)

    // Replan creates a new plan version. Called inside withWorkspaceState.
    Replan(ctx context.Context, session *ExplorationSession, state *RuntimeWorkspaceState, trigger ReplanTrigger) (*ExecutionPlan, []PlanStep, error)
}
```

`DeterministicPlanner` is a zero-dependency struct (no LLM, no Model):

```go
// deterministic.go

type DeterministicPlanner struct{}

func NewDeterministicPlanner() *DeterministicPlanner { return &DeterministicPlanner{} }

// Compile-time interface check
var _ Planner = &DeterministicPlanner{}
```

`NewExplorationDomain` sets `domain.planner = NewDeterministicPlanner()`.

### 3.3 Initial Workspace Graph Seeding

When `ApiV1CreateWorkspace` creates a new workspace, it must synchronously seed initial Direction nodes so the session is non-empty from the start. This is done by calling `initializeWorkspaceGraph` (a new function in `runtime_agent.go`) after the session is stored:

```go
// runtime_agent.go
// initializeWorkspaceGraph calls BuildInitialPlan, executes the first plan step synchronously,
// and appends the generated Direction nodes to the session.
// Must be called while NOT holding runtime.mu (it acquires its own lock via withWorkspaceState).
func (d *ExplorationDomain) initializeWorkspaceGraph(ctx context.Context, workspaceID string)
```

`ApiV1CreateWorkspace` calls this after `d.store` is updated. The generated Direction nodes are visible in the projection response within the same HTTP request lifecycle.

`TestV1ToggleFavoriteInterventionAffectsWorkspaceState` then finds `NodeDirection` nodes (not `NodeIdea`) and can favorite them.

### 3.4 Node and Edge Type Constants (schema.go additions)

Add to the existing `NodeType` constants block (do NOT rename or remove existing constants):

```go
NodeDirection NodeType = "direction"   // NEW
NodeArtifact  NodeType = "artifact"    // NEW
```

Add to the existing `EdgeType` constants block:

```go
EdgeJustifies    EdgeType = "justifies"      // NEW
EdgeBranchesFrom EdgeType = "branches_from"  // NEW — used when replan generates new Direction from existing Direction
EdgeRaises       EdgeType = "raises"          // NEW
EdgeResolves     EdgeType = "resolves"        // NEW
```

No existing constants are renamed or removed.

---

## 4. Node Generation Semantics

`DeterministicPlanner` inspects the current graph (`ExplorationSession.Nodes` and `Edges`) and `RuntimeWorkspaceState.Balance` to decide what to generate each cycle. Generated nodes and edges are appended to the session by `executeFirstPlanStep` (called by `runtime_agent.go`).

### 4.1 Definitions

- **"few Evidence"**: a Direction node has `< 2` Evidence nodes with an edge pointing to it.
- **"sufficient Evidence"**: a Direction node has `>= 2` Evidence nodes with an edge pointing to it.
- **`Aggression > 0.6` (fast path)**: treats all Direction nodes as having sufficient Evidence when evaluating rules, bypassing the Evidence generation phase.

### 4.2 Generation Rules (evaluated in priority order, first match wins)

| Priority | Precondition | BalanceState condition | Action |
|----------|-------------|----------------------|--------|
| 1 | No Direction nodes exist | any | Generate 3–5 Direction nodes derived from topic words |
| 2 | Any Direction has few Evidence AND `Aggression <= 0.6` | `Research >= 0.5` | Generate 1–2 Evidence nodes per under-evidenced Direction |
| 3 | Any Direction has few Evidence AND `Aggression <= 0.6` | `Research < 0.5` | Generate 1 Artifact node summarizing current knowledge |
| 4 | `Aggression > 0.6` OR all Directions have sufficient Evidence; no Claim per Direction | any | Generate 1 Claim node per Direction (synthesizes Evidence) |
| 5 | All Directions have a Claim; `Divergence < 0.4` (converge); no Decision exists | any | Generate 1 Decision node resolving Claims |
| 6 | All Directions have a Claim; `Divergence >= 0.6` (diverge); no Unknown exists | any | Generate 1–2 Unknown nodes representing open questions |
| 7 | Decision node exists | any | Generate 1 Artifact node (final output); mark step done |

**Rule 4 clarification (Aggression fast path):** When `Aggression > 0.6`, Rules 2 and 3 are skipped regardless of their preconditions. Rule 4 is evaluated directly after Rule 1 fails. This means Evidence nodes are never generated in high-aggression mode; Claim nodes are generated directly from Directions.

### 4.3 Edge Generation

| New node type | Edge type | Target |
|--------------|----------|--------|
| Direction (initial) | none | — |
| Direction (from replan on existing Direction) | `EdgeBranchesFrom` | Parent Direction |
| Evidence | `EdgeSupports` or `EdgeContradicts` (alternate per node) | Parent Direction |
| Claim | `EdgeJustifies` | Most-recent Evidence node for that Direction (or Direction itself if no Evidence in fast-path) |
| Decision | `EdgeJustifies` | All Claim nodes in the workspace |
| Unknown | `EdgeRaises` | Direction with fewest Evidence nodes |
| Artifact | `EdgeResolves` | Decision node if exists, else most-recent Claim |

### 4.4 BalanceState Adjustment from Interventions

When an intervention is submitted, scan its `intent` text (case-insensitive substring match) and accumulate adjustments to `BalanceState`, then call `Replan`. All values clamped to [0.0, 1.0].

| Keyword | Field | Delta |
|---------|-------|-------|
| "focus", "decide", "收敛", "converge" | Divergence | −0.2 |
| "explore", "expand", "发散", "diverge" | Divergence | +0.2 |
| "research", "evidence", "调研" | Research | +0.2 |
| "produce", "output", "产出" | Research | −0.2 |
| "fast", "quick", "aggressive" | Aggression | +0.2 |
| "careful", "thorough", "prudent" | Aggression | −0.2 |

---

## 5. Mutation Events

Use the existing `MutationEvent` struct (in `schema.go`) and the existing `mutationID` helper (in `mutations.go`). `runtime_agent.go` appends events to `state.Mutations` inside `withWorkspaceState`, then broadcasts via `realtime.go` after releasing the lock.

### 5.1 New Event Kinds

| When | `Kind` string | `MutationEvent` fields used |
|------|--------------|---------------------------|
| Run record created | `"run_created"` | `Run *GenerationRun` |
| BalanceState adjusted by intervention | `"balance_updated"` | `WorkspaceID` only (MVP: no additional payload field) |
| Intervention absorbed | `"intervention_absorbed"` | `ActiveOpportunityID` set to intervention ID |
| Replan triggered | `"replanned"` | `Run *GenerationRun` of the new plan's associated run |

**Node and edge mutation events** are produced by calling `diffMutations(prevSession, nextSession, "runtime")` after each cycle. This emits `"node_added"` and `"edge_added"` events through the existing mechanism. No separate `"nodes_generated"` event is needed.

### 5.2 Broadcast Flow

```
withWorkspaceState → append events to state.Mutations
→ release lock
→ broadcastMutations(workspaceID, newEvents)  // realtime.go
```

---

## 6. File Split: api_v1.go → handler_*.go + projection_builder.go

Function bodies are not modified during the move. Each destination file includes all private helpers required by its exported handlers.

### handler_shared.go
Shared utilities used by multiple handler files:
- `writeV1Error(c *gin.Context, code int, errCode, msg string)`
- `toRFC3339(ms int64) string`

### handler_workspace.go
- `ApiV1CreateWorkspace`
- `ApiV1GetWorkspace`
- `toWorkspaceView(session *ExplorationSession) WorkspaceView`

### handler_run.go
Exported handlers:
- `ApiV1CreateRun`
- `ApiV1GetRun`
- `ApiV1GetTraceSummary`
- `ApiV1ListTraceEvents`

Private helpers:
- `buildRunView`
- `buildTraceSummary`
- `applyTracePagination`
- `isValidTraceCategory`
- `isValidTraceLevel`
- `normalizeRunStatus`
- `normalizeStepStatus`
- `normalizeAgentName`
- `derivePlanStatus`
- `deriveRunStatus`
- `inferAgentFromStep`
- `indexOfPlan`

### handler_intervention.go
Exported handlers:
- `ApiV1CreateIntervention`
- `ApiV1GetIntervention`
- `ApiV1ListInterventionEvents`

Private helpers:
- `mapInterventionReq`
- `storeInterventionRecord`
- `advanceInterventionByRuntimeEvent`
- `getInterventionRecord`
- `listInterventionEvents`
- `decodeInterventionEventView`
- `applyEventPagination`
- `findStartIndexByCursor`
- `encodeEventCursor`

`storeInterventionRecord` and `advanceInterventionByRuntimeEvent` access runtime state through `d.withWorkspaceState(..., func(state) { state.Interventions[id] = ... })` instead of the old `d.runtime.intervention[wid][id]` pattern.

### projection_builder.go
- `ApiV1GetProjection`
- `buildProjectionResponse`
- `buildRunSummaryView`
- `buildInterventionEffects`
- `buildRecentChanges` (if present)
- Any other helper functions called only by projection building

`api_v1.go` is deleted after all functions are moved.

---

## 7. Renames

| Old location / name | New location / name | Reason |
|---------------------|---------------------|--------|
| `runtime_plan.go` → `dispatchPlanSteps` | `runtime_agent.go` (inlined or renamed) → `executeFirstPlanStep` | Only first step executed in MVP |
| `runtime_plan.go` → `buildInitialPlan` (package func) | `deterministic.go` → `DeterministicPlanner.BuildInitialPlan` | Moved to interface implementation |
| `runtime_plan.go` → `buildInitialBalance` | `deterministic.go` (kept as package-private helper) | Referenced only by DeterministicPlanner |
| `runtime_plan.go` → `generatePlanStepsWithModel` (method on `*ExplorationDomain`) | `runtime_llm.go` | Stays as domain method; file preserved as LLM extension point |
| `runtime_tasks.go` → entire file | `deterministic.go` (merged) | File eliminated; step execution logic integrated |

---

## 8. Testing

### 8.1 Existing tests (must stay green)

All HTTP-level tests in `api_test.go` test through the HTTP interface and must remain green with no API response shape changes.

### 8.2 Tests that access internal state directly

**`TestV1InterventionCanRecoverFromDB`** accesses `domain.runtime.intervention`. This test already skips in CI (`if domain.DB == nil { t.Skip(...) }`). Update the internal access to use the new accessor regardless:

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

**`TestV1ToggleFavoriteInterventionAffectsWorkspaceState`** checks for `NodeIdea`. Change to `NodeDirection`:

```go
// Before:
if node.Type == NodeIdea {
// After:
if node.Type == NodeDirection {
```

This is valid because `initializeWorkspaceGraph` (called during `ApiV1CreateWorkspace`) now seeds Direction nodes synchronously, so a Direction node is guaranteed to exist after workspace creation.

### 8.3 New tests to add

- `TestDeterministicPlannerInitialRun` — verifies first run produces Direction nodes; no Evidence/Claim/Decision present
- `TestDeterministicPlannerResearchPhase` — verifies second run with `Balance.Research >= 0.5` produces Evidence nodes with `EdgeSupports` edges to Direction nodes
- `TestDeterministicPlannerFastPath` — verifies that `Aggression > 0.6` skips Evidence and generates Claim nodes directly
- `TestDeterministicPlannerConvergence` — verifies Claim nodes generated when all Directions have >= 2 Evidence; Decision generated when `Divergence < 0.4`
- `TestInterventionAdjustsBalanceState` — verifies keyword "收敛" in intent shifts `Divergence` down by 0.2; `Balance.Divergence` observable via `GetRuntimeState`
- `TestMutationEventsWrittenOnRunComplete` — verifies `state.Mutations` is non-empty after a completed run; at minimum `"run_created"` and `"node_added"` events are present

---

## 9. Out of Scope

- LLM-backed planner (preserved via `runtime_llm.go`; `DeterministicPlanner` does not call it)
- Frontend rendering changes
- Database persistence (runtime state remains in-memory)
- Authentication, multi-user, workspace sharing
- BalanceState change to enum types (kept as float64)
- Changes to `exploration.go`, `api.go`, `persistence.go`, `realtime.go`, `cursor.go`, `runtime_context.go`, `runtime_operator.go`, `mutations.go`

---

## 10. Acceptance Criteria

1. `go test ./domain/exploration/...` — all tests pass (including updated internal-access tests)
2. `npm test` — all frontend tests pass
3. `ApiV1CreateWorkspace` response includes `NodeDirection` nodes in the projection (seeded by `initializeWorkspaceGraph`)
4. Evidence nodes appear after the second run cycle and have `EdgeSupports` edges to Direction nodes
5. Submitting an intervention with intent "收敛" reduces `BalanceState.Divergence` by 0.2 (verified by `TestInterventionAdjustsBalanceState`)
6. `state.Mutations` is non-empty after a completed run (verified by `TestMutationEventsWrittenOnRunComplete`)
7. `api_v1.go`, `runtime_plan.go`, `runtime_tasks.go` are deleted
8. `var _ Planner = &DeterministicPlanner{}` compiles without error
9. No new file created in this PR exceeds 400 lines
10. All runtime state mutations go through `withWorkspaceState`; no code accesses `d.runtime.workspaces[id]` directly outside `domain.go`
