# Exploration Deep-Agent Runtime Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate `backend/domain/exploration` from the current timer-driven prototype into a deep-agent runtime with explicit internal runs, plans, tasks, replanning, and a self-balancing exploration engine.

**Architecture:** Keep the current `workspace` and map projection contract, but replace implicit runtime behavior with a `MainAgent + SubAgents + TaskTool` execution pipeline plus an internal balance engine. The migration should be incremental: first add durable run/plan/task/balance state and adapters, then route existing exploration behavior through the new orchestration layer, and only after that replace the fake auto-expansion loop.

**Tech Stack:** Go, Gin, GORM, Eino ADK / Deep, WebSocket, existing `backend/deep` reference implementation

---

### Task 1: Stabilize the Current Exploration Contract

**Files:**
- Modify: `backend/domain/exploration/schema.go`
- Modify: `backend/domain/exploration/exploration.go`
- Modify: `backend/domain/exploration/api_test.go`
- Modify: `backend/domain/exploration/realtime_test.go`

**Step 1: Write the failing tests**

Add tests that lock the current external contract before runtime migration:

- `CreateWorkspace` returns a workspace snapshot with exploration state and presentation state.
- `ApplyIntervention` changes workspace output without breaking current REST responses.
- WebSocket / runtime tests assert run-history growth or, if semantics change, are updated to assert explicit run-status events instead of timer side effects.

**Step 2: Run test to verify it fails**

Run: `go test ./domain/exploration -run 'Test(CreateWorkspaceAndReadProjection|InterventionExpandOpportunity|RuntimeContinuouslyExpandsWorkspace)' -v`

Expected: At least one failure if the contract and current schema naming are inconsistent.

**Step 3: Write minimal implementation**

Normalize the current snapshot shape and type naming so tests and implementation agree on:

- `WorkspaceSnapshot`
- product-facing projection field names
- `GenerationRun` vs future `Run` transition boundary

Do not introduce deep-agent orchestration in this task. This task only creates a reliable baseline for later migration.

**Step 4: Run test to verify it passes**

Run: `go test ./domain/exploration -run 'Test(CreateWorkspaceAndReadProjection|InterventionExpandOpportunity|RuntimeContinuouslyExpandsWorkspace)' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add backend/domain/exploration/schema.go backend/domain/exploration/exploration.go backend/domain/exploration/api_test.go backend/domain/exploration/realtime_test.go
git commit -m "test: stabilize exploration workspace contract"
```

### Task 2: Introduce Durable Run / Plan / Task / Balance Models

**Files:**
- Modify: `backend/domain/exploration/schema.go`
- Modify: `backend/domain/exploration/persistence.go`
- Modify: `backend/domain/exploration/domain.go`
- Test: `backend/domain/exploration/api_test.go`

**Step 1: Write the failing test**

Add tests that create a workspace and assert that the backend can persist and retrieve:

- a `run`
- an `execution_plan`
- `plan_step` records
- `agent_task` summaries
- a `balance_state`

The test may start with an in-memory fallback if DB-backed persistence is not wired yet, but the API shape must be explicit.

**Step 2: Run test to verify it fails**

Run: `go test ./domain/exploration -run 'TestRunPlanTaskPersistence' -v`

Expected: FAIL because the plan/task/balance models do not exist yet.

**Step 3: Write minimal implementation**

Extend `schema.go` with new types:

- `Run`
- `ExecutionPlan`
- `PlanStep`
- `AgentTask`
- `BalanceState`
- `AgentTaskResultSummary`

Extend `persistence.go` and `domain.go` to store and retrieve these models, initially using the same mixed persistence strategy as the current workspace prototype if needed:

- in-memory runtime state for fast local iteration
- DB-backed snapshot/event persistence where already available

Do not remove existing `GenerationRun` immediately; keep a compatibility boundary until projection code is migrated.

**Step 4: Run test to verify it passes**

Run: `go test ./domain/exploration -run 'TestRunPlanTaskPersistence' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add backend/domain/exploration/schema.go backend/domain/exploration/persistence.go backend/domain/exploration/domain.go backend/domain/exploration/api_test.go
git commit -m "feat: add exploration run plan task models"
```

### Task 3: Add Workdir Context and Runtime Operator Abstractions

**Files:**
- Create: `backend/domain/exploration/runtime_context.go`
- Create: `backend/domain/exploration/runtime_operator.go`
- Modify: `backend/domain/exploration/domain.go`
- Test: `backend/domain/exploration/realtime_test.go`

**Step 1: Write the failing test**

Add tests for:

- creating a per-run work directory
- attaching run-scoped context parameters
- ensuring runtime helpers resolve the correct workdir for a run

**Step 2: Run test to verify it fails**

Run: `go test ./domain/exploration -run 'TestRuntimeContext' -v`

Expected: FAIL because no run-scoped workdir/context abstraction exists.

**Step 3: Write minimal implementation**

Port the shape of these concepts from `backend/deep`:

- context parameter helpers similar to `backend/deep/params/params.go`
- operator abstraction similar to `backend/deep/operator.go`

Adapt them to exploration semantics:

- context keys should include `workspace_id`, `run_id`, `plan_id`, `work_dir`, and input summaries
- operator should remain generic and testable

Do not expose these details to API clients.

**Step 4: Run test to verify it passes**

Run: `go test ./domain/exploration -run 'TestRuntimeContext' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add backend/domain/exploration/runtime_context.go backend/domain/exploration/runtime_operator.go backend/domain/exploration/domain.go backend/domain/exploration/realtime_test.go
git commit -m "feat: add exploration runtime context and operator"
```

### Task 4: Introduce Main-Agent Planning, Balance, and Task Dispatch

**Files:**
- Create: `backend/domain/exploration/runtime_agent.go`
- Create: `backend/domain/exploration/runtime_plan.go`
- Create: `backend/domain/exploration/runtime_tasks.go`
- Modify: `backend/domain/exploration/realtime.go`
- Test: `backend/domain/exploration/realtime_test.go`

**Step 1: Write the failing test**

Add tests that assert:

- starting a run creates an explicit plan
- starting a run also computes an initial `balance_state`
- plan steps transition through `todo -> doing -> done` or failure states
- task dispatch records which sub-agent handled a step

**Step 2: Run test to verify it fails**

Run: `go test ./domain/exploration -run 'TestRunCreatesExplicitPlan|TestPlanStepTransitions' -v`

Expected: FAIL because the current runtime only auto-expands opportunities on a timer.

**Step 3: Write minimal implementation**

Create a first-pass orchestrator:

- `MainAgent` equivalent that reads workspace state, computes `BalanceState`, and creates an `ExecutionPlan`
- a dispatcher that can assign steps to logical sub-agent roles such as `research`, `graph`, `artifact`, `general`
- task status updates wired into the new run/plan/task models

At this stage, the actual sub-agent execution can still call stubbed exploration helpers. The important change is moving control flow into explicit orchestration state plus internal rhythm control.

**Step 4: Run test to verify it passes**

Run: `go test ./domain/exploration -run 'TestRunCreatesExplicitPlan|TestPlanStepTransitions' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add backend/domain/exploration/runtime_agent.go backend/domain/exploration/runtime_plan.go backend/domain/exploration/runtime_tasks.go backend/domain/exploration/realtime.go backend/domain/exploration/realtime_test.go
git commit -m "feat: add exploration main-agent planning loop"
```

### Task 5: Integrate Wrapped Tooling and Sub-Agent Adapters

**Files:**
- Create: `backend/domain/exploration/runtime_tools.go`
- Create: `backend/domain/exploration/runtime_subagents.go`
- Modify: `backend/domain/exploration/domain.go`
- Modify: `backend/go.mod`
- Test: `backend/domain/exploration/realtime_test.go`

**Step 1: Write the failing test**

Add tests that assert:

- sub-agent adapters can be constructed for `research`, `graph`, `artifact`, and `general`
- tool calls are routed through a wrapper layer that normalizes responses
- invalid tool payloads are repaired or rejected consistently

**Step 2: Run test to verify it fails**

Run: `go test ./domain/exploration -run 'TestSubAgentAdapterConstruction|TestWrappedToolResponses' -v`

Expected: FAIL because exploration currently has no deep-style tool wrapper or sub-agent adapter layer.

**Step 3: Write minimal implementation**

Reuse the shape from `backend/deep`:

- wrapper layer similar to `backend/deep/tools/wrap.go`
- logical tool set for file reads, tree, shell, structured file edits, and optional web search
- sub-agent adapter constructors modeled after `backend/deep/agents/code_agent.go` and `backend/deep/agents/web_search.go`

Keep this layer exploration-specific:

- `ResearchAgent` should focus on retrieval and evidence extraction
- `GraphAgent` should focus on structure and decision proposals
- `ArtifactAgent` should focus on materialization packaging

**Step 4: Run test to verify it passes**

Run: `go test ./domain/exploration -run 'TestSubAgentAdapterConstruction|TestWrappedToolResponses' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add backend/domain/exploration/runtime_tools.go backend/domain/exploration/runtime_subagents.go backend/domain/exploration/domain.go backend/go.mod backend/domain/exploration/realtime_test.go
git commit -m "feat: add exploration sub-agent adapters and wrapped tools"
```

### Task 6: Rewire Interventions into Directional Replanning

**Files:**
- Modify: `backend/domain/exploration/api.go`
- Modify: `backend/domain/exploration/exploration.go`
- Modify: `backend/domain/exploration/realtime.go`
- Modify: `backend/domain/exploration/schema.go`
- Test: `backend/domain/exploration/api_test.go`

**Step 1: Write the failing test**

Add tests that assert:

- posting an `intervention` changes plan status and creates a new plan version
- posting an `intervention` updates or recomputes `balance_state`
- affected pending steps are marked `skipped`, `invalidated`, or equivalent
- API responses expose that replanning occurred

**Step 2: Run test to verify it fails**

Run: `go test ./domain/exploration -run 'TestInterventionTriggersReplanning' -v`

Expected: FAIL because current interventions mutate workspace state directly instead of replanning.

**Step 3: Write minimal implementation**

Refactor intervention handling so that:

- interventions are persisted as governance events
- main runtime converts them into a directional replanning request
- the balance engine recomputes internal exploration rhythm
- the next visible result is a new plan version plus downstream graph/projection updates

Keep the existing user-facing intervention API path intact during migration.

**Step 4: Run test to verify it passes**

Run: `go test ./domain/exploration -run 'TestInterventionTriggersReplanning' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add backend/domain/exploration/api.go backend/domain/exploration/exploration.go backend/domain/exploration/realtime.go backend/domain/exploration/schema.go backend/domain/exploration/api_test.go
git commit -m "feat: replan exploration runs on intervention"
```

### Task 7: Make Projection and Streaming Consume Lightweight Run State

**Files:**
- Modify: `backend/domain/exploration/schema.go`
- Modify: `backend/domain/exploration/exploration.go`
- Modify: `backend/domain/exploration/realtime.go`
- Modify: `backend/domain/exploration/api.go`
- Test: `backend/domain/exploration/api_test.go`
- Test: `backend/domain/exploration/realtime_test.go`

**Step 1: Write the failing tests**

Add tests that assert the frontend-facing snapshot or stream can expose:

- current focus summary
- latest meaningful change
- latest replanning reason
- graph and artifact updates alongside run status

**Step 2: Run test to verify it fails**

Run: `go test ./domain/exploration -run 'TestWorkspaceSnapshotIncludesPlanState|TestWebSocketBroadcastsRunStatus' -v`

Expected: FAIL because current snapshot/stream payloads only reflect graph-ish state and mutation events or expose the wrong granularity.

**Step 3: Write minimal implementation**

Extend projection and stream payloads so they surface:

- `run` summary
- latest focus summary
- latest meaningful system change
- mutation or projection updates tied to task execution

Do not expose raw chain-of-thought, full task lists, or full tool transcripts in the default frontend projection.

**Step 4: Run test to verify it passes**

Run: `go test ./domain/exploration -run 'TestWorkspaceSnapshotIncludesPlanState|TestWebSocketBroadcastsRunStatus' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add backend/domain/exploration/schema.go backend/domain/exploration/exploration.go backend/domain/exploration/realtime.go backend/domain/exploration/api.go backend/domain/exploration/api_test.go backend/domain/exploration/realtime_test.go
git commit -m "feat: expose exploration plan and run status in projections"
```

### Task 8: Replace Timer Expansion with Real Deep-Agent Execution

**Files:**
- Modify: `backend/domain/exploration/realtime.go`
- Modify: `backend/domain/exploration/runtime_agent.go`
- Modify: `backend/domain/exploration/runtime_tasks.go`
- Modify: `backend/domain/exploration/runtime_subagents.go`
- Test: `backend/domain/exploration/realtime_test.go`

**Step 1: Write the failing test**

Add an end-to-end test that asserts:

- a run starts
- an internal balance state is computed
- a plan is created
- at least one delegated task completes
- graph/projection state changes as a result of task output

**Step 2: Run test to verify it fails**

Run: `go test ./domain/exploration -run 'TestDeepAgentRunE2E' -v`

Expected: FAIL while the runtime still depends on the old `time.Sleep`-driven expansion loop.

**Step 3: Write minimal implementation**

Remove the prototype loop that periodically calls `applyExpandOpportunity` as the primary runtime driver. Replace it with:

- explicit run startup
- balance-state computation
- plan generation
- task dispatch
- normalized task results
- graph/projection writes

Weak directions should be folded or cooled rather than deleted when they stop receiving primary resources.

If a fallback loop is still needed for local prototyping, isolate it behind a clearly named test-only or prototype-only path.

**Step 4: Run test to verify it passes**

Run: `go test ./domain/exploration -run 'TestDeepAgentRunE2E' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add backend/domain/exploration/realtime.go backend/domain/exploration/runtime_agent.go backend/domain/exploration/runtime_tasks.go backend/domain/exploration/runtime_subagents.go backend/domain/exploration/realtime_test.go
git commit -m "feat: replace prototype loop with deep-agent runtime"
```

### Task 9: Run Full Verification and Clean Up Compatibility Layers

**Files:**
- Modify: `backend/domain/exploration/schema.go`
- Modify: `backend/domain/exploration/exploration.go`
- Modify: `backend/domain/exploration/api.go`
- Modify: `backend/domain/exploration/realtime.go`
- Test: `backend/domain/exploration/api_test.go`
- Test: `backend/domain/exploration/realtime_test.go`

**Step 1: Re-read requirements**

Verify the implementation still satisfies:

- `workspace` as top-level contract
- explicit planning and replanning
- internal self-balancing across `发散/收敛`、`研究/产出`、`激进/稳健`
- isolated sub-agent execution
- graph / projection as durable result layer
- intervention-driven governance

**Step 2: Remove transitional compatibility shims**

Delete or simplify temporary dual-model code such as:

- obsolete `GenerationRun` compatibility fields
- prototype-only snapshot aliases
- any direct intervention-to-mutation shortcuts left from the prototype
- any hard-delete or hard-elimination behavior that conflicts with fold/suppress/reactivate semantics

**Step 3: Run package tests**

Run: `go test ./domain/exploration -v`

Expected: PASS

**Step 4: Run broader backend verification**

Run: `go test ./...`

Expected: PASS, or explicit documentation of unrelated existing failures.

**Step 5: Commit**

```bash
git add backend/domain/exploration backend/go.mod backend/go.sum
git commit -m "feat: migrate exploration runtime to deep-agent orchestration"
```
