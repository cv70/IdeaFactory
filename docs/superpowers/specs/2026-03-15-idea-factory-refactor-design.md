# Idea Factory — Refactor & Runtime Semantics Design

**Date:** 2026-03-15
**Scope:** Sub-project A (code structure) + Sub-project B (runtime semantics) — combined single-pass approach
**Constraint:** All existing tests remain green throughout; interface behavior is preserved

---

## 1. Motivation

The current backend has two distinct problems that compound each other:

**Structure problems (Sub-project A):**
- `domain.go` holds 10 loose maps with no cohesion; callers must coordinate many keys manually
- `api_v1.go` (864 lines) mixes HTTP handlers with domain logic and projection building
- `runtime_plan.go` and `runtime_tasks.go` are thin wrappers with misleading names (`dispatchPlanSteps` executes only the first step)
- Mutation events are never written, so SSE subscribers receive no incremental updates
- No `Planner` interface; the deterministic logic is baked into `runtime_agent.go` with no seam for extension

**Semantic problems (Sub-project B):**
- Deterministic runtime generates `NodeOpportunity`/`NodeIdea` (old ontology); spec requires Direction, Evidence, Claim, Decision, Unknown nodes
- `BalanceState` fields (`Divergence`, `Research`, `Aggression` as float64) are stored but never read during planning
- Intervention `intent` text is recorded but does not influence the graph or future runs
- No edges between nodes are generated (no `supports`, `contradicts`, `justifies`, `branches_from` relationships)

---

## 2. Target File Structure

```
backend/domain/exploration/
  domain.go               # Domain struct; holds Planner interface + workspace state map
  planner.go              # Planner interface definition
  deterministic.go        # DeterministicPlanner: node generation + BalanceState logic
  runtime_agent.go        # Coordinator: calls Planner, writes mutation events, manages lifecycle
  handler_workspace.go    # HTTP handlers: CreateWorkspace, GetWorkspace
  handler_run.go          # HTTP handlers: CreateRun, GetRun, GetTraceSummary, ListTraceEvents
  handler_intervention.go # HTTP handlers: CreateIntervention, GetIntervention, ListInterventionEvents
  projection_builder.go   # Pure function: ExplorationSession → ProjectionResponse
  schema.go               # Type definitions (node type constants added; BalanceState unchanged)
  routes.go               # Route registration (unchanged)
  api_test.go             # Tests: existing tests green + new semantic tests

  # Deleted:
  # api_v1.go      (split into handler_*.go + projection_builder.go)
  # runtime_plan.go  (merged into deterministic.go)
  # runtime_tasks.go (merged into deterministic.go)
```

---

## 3. Data Model Changes

### 3.1 RuntimeWorkspaceState

Replace the 10 loose maps in `Domain` with a single cohesive struct per workspace:

```go
type RuntimeWorkspaceState struct {
    Run          *Run
    Plan         *ExecutionPlan
    PlanSteps    []*PlanStep
    AgentTasks   []*AgentTask
    Results      []*Result
    Balance      BalanceState
    Mutations    []ExplorationMutation
    ReplanReason string
    Intervention *Intervention
    Running      bool
    Cursor       string
}
```

`Domain` becomes:

```go
type Domain struct {
    workspaces map[string]*RuntimeWorkspaceState  // keyed by workspace ID
    planner    Planner
    mu         sync.RWMutex
    // subscriptions, db, etc. unchanged
}
```

### 3.2 Planner Interface

```go
// planner.go
type Planner interface {
    BuildInitialPlan(ctx context.Context, ws *Workspace) (*ExecutionPlan, error)
    Replan(ctx context.Context, state *RuntimeWorkspaceState, trigger ReplanTrigger) (*ExecutionPlan, error)
}

type ReplanTrigger struct {
    Kind        string // "intervention" | "balance_shift" | "manual"
    Intervention *Intervention
}
```

`DeterministicPlanner` implements this interface. Domain is initialized with `DeterministicPlanner` by default.

### 3.3 Node Type Constants (schema.go additions)

```go
const (
    NodeTypeDirection = "direction"
    NodeTypeEvidence  = "evidence"
    NodeTypeClaim     = "claim"
    NodeTypeDecision  = "decision"
    NodeTypeUnknown   = "unknown"
    NodeTypeArtifact  = "artifact"
)

const (
    EdgeTypeSupports     = "supports"
    EdgeTypeContradicts  = "contradicts"
    EdgeTypeJustifies    = "justifies"
    EdgeTypeBranchesFrom = "branches_from"
    EdgeTypeRaises       = "raises"
    EdgeTypeResolves     = "resolves"
)
```

`BalanceState` struct fields remain float64 (0.0–1.0); no external API change.

---

## 4. Node Generation Semantics

`DeterministicPlanner` generates nodes deterministically based on the current graph state and `BalanceState`. Each call to `BuildInitialPlan` or `Replan` produces a plan whose steps emit specific node types when executed.

### 4.1 Generation Rules

| Graph state | BalanceState condition | Nodes generated |
|-------------|----------------------|-----------------|
| No nodes yet | any | 3–5 Direction nodes derived from topic |
| Directions exist, few Evidence | `Research > 0.5` | 1–2 Evidence per active Direction |
| Directions exist, few Evidence | `Research <= 0.5` | 1 Artifact node (produce output) |
| Evidence exists, `< 2` per Direction | `Aggression <= 0.6` | Additional Evidence nodes |
| `>= 2` Evidence per Direction | any | 1 Claim per Direction (synthesizes Evidence) |
| Claims exist | `Divergence < 0.4` (converge) | 1 Decision node (resolves Claims) |
| Open questions remain | `Divergence > 0.6` (diverge) | 1–2 Unknown nodes |

### 4.2 Edge Generation

Every newly generated node is connected to its parent(s):

- `Evidence` → `supports`/`contradicts` → `Direction`
- `Claim` → `justifies` → `Evidence` (primary evidence)
- `Decision` → `justifies` → `Claim`
- `Unknown` → `raises` → `Direction`

### 4.3 BalanceState Influence from Interventions

When an intervention is submitted, its `intent` text is scanned for keywords to adjust `BalanceState` before replan:

| Keyword match | Adjustment |
|--------------|-----------|
| "focus", "decide", "收敛" | `Divergence -= 0.2` (clamped to [0, 1]) |
| "explore", "expand", "发散" | `Divergence += 0.2` |
| "research", "evidence", "调研" | `Research += 0.2` |
| "produce", "output", "产出" | `Research -= 0.2` |
| "fast", "quick", "aggressive" | `Aggression += 0.2` |
| "careful", "thorough", "prudent" | `Aggression -= 0.2` |

After adjustment, `Replan` is called and the next run reflects the updated balance.

---

## 5. Mutation Events

Mutation events must be written at these points in `runtime_agent.go`:

| Moment | `kind` value | Payload |
|--------|-------------|---------|
| Run created | `run_created` | run ID, plan summary |
| Each node batch generated | `nodes_generated` | list of new node IDs and types |
| BalanceState changed by intervention | `balance_updated` | new BalanceState values |
| Intervention absorbed | `intervention_absorbed` | intervention ID, adjusted balance |
| Replan triggered | `replanned` | trigger kind, new plan summary |

These events are appended to `RuntimeWorkspaceState.Mutations` and fanned out to SSE subscribers via the existing `appendMutation` + subscription mechanism.

---

## 6. File Split: api_v1.go → handler_*.go + projection_builder.go

`api_v1.go` is split by responsibility. Function bodies are not modified during the split; only their location changes. This preserves test coverage.

| Destination | Functions moved |
|-------------|----------------|
| `handler_workspace.go` | `ApiV1CreateWorkspace`, `ApiV1GetWorkspace` |
| `handler_run.go` | `ApiV1CreateRun`, `ApiV1GetRun`, `ApiV1GetTraceSummary`, `ApiV1ListTraceEvents` |
| `handler_intervention.go` | `ApiV1CreateIntervention`, `ApiV1GetIntervention`, `ApiV1ListInterventionEvents`, `mapInterventionReq`, `storeInterventionRecord`, `advanceInterventionByRuntimeEvent` |
| `projection_builder.go` | `ApiV1GetProjection`, `buildProjectionResponse` and all helper functions |

`api_v1.go` is deleted after the split. Route registration stays in `routes.go`.

---

## 7. Renames

| Old name | New name | Reason |
|----------|----------|--------|
| `dispatchPlanSteps` | `executeFirstPlanStep` | MVP only executes first step; name must reflect this |
| `runtime_plan.go` | merged into `deterministic.go` | file is eliminated |
| `runtime_tasks.go` | merged into `deterministic.go` | file is eliminated |

---

## 8. Testing

### Existing tests (must stay green)
- All tests in `api_test.go` — interface behavior unchanged
- `TestV1ProjectionAndInterventionLifecycle`
- `TestV1CreateRunAndGetRun`
- `TestV1TraceSummary`
- `TestV1ListInterventionEvents`

### New tests to add
- `TestDeterministicPlannerInitialRun` — verifies first run produces Direction nodes, no other types
- `TestDeterministicPlannerResearchPhase` — verifies second run with `Research > 0.5` produces Evidence nodes connected to Directions
- `TestDeterministicPlannerConvergence` — verifies Claims generated when `>= 2` Evidence per Direction; Decision generated when `Divergence < 0.4`
- `TestBalanceStateAdjustedByIntervention` — verifies keyword matching adjusts BalanceState before replan
- `TestMutationEventsWritten` — verifies run completion writes at least one mutation event
- `TestEdgesGeneratedBetweenNodes` — verifies Evidence nodes have `supports`/`contradicts` edges to Directions

### Frontend
No new tests required. Existing `App.test.tsx` tests cover the intervention UX. Verify existing tests remain green after any minor type updates.

---

## 9. Out of Scope

- LLM-backed planner (remains a future extension point via the `Planner` interface)
- Frontend rendering changes beyond verifying existing node type display works
- Database persistence (runtime state remains in-memory)
- Authentication, multi-user, or workspace sharing
- BalanceState change to enum types (kept as float64 for now; enum refactor is a separate sub-project)

---

## 10. Acceptance Criteria

1. All existing `go test ./domain/exploration/...` tests pass
2. All existing frontend tests pass (`npm test`)
3. A fresh run on a new workspace produces Direction nodes (not NodeOpportunity)
4. Evidence nodes appear after the second run cycle and have edges to Directions
5. Submitting an intervention with "收敛" shifts `Divergence` down in the workspace state
6. `GET /api/v1/workspaces/:id/runs/:runId/mutations` returns non-empty list after a completed run
7. `api_v1.go` is deleted; no file in the package exceeds 300 lines
8. `DeterministicPlanner` implements the `Planner` interface (verified by compiler)
