# Workspace Pause & Auto-Restart Design

**Date**: 2026-03-18
**Status**: Approved
**Scope**: Backend only (`backend/`)

---

## Overview

Workspaces run an autonomous LLM-driven exploration cycle that grows a graph of nodes from a central Topic. This design adds:

1. **Automatic run scheduling** — after each run completes, a workspace automatically starts the next run after `Strategy.IntervalMs`, up to `Strategy.MaxRuns` (0 = unlimited).
2. **Pause / resume control** — users can pause a workspace via `PATCH /api/v1/workspaces/:id`, stopping the scheduling loop. Resuming restarts it immediately.
3. **Startup recovery** — on backend start, all active (non-paused, non-archived) workspaces automatically resume their scheduling loop.

---

## Data Model Changes

### `dbdao.WorkspaceState` (new field)

```go
PausedAt *time.Time `json:"paused_at" gorm:"index"`
```

- `NULL` → workspace is active (scheduling enabled)
- Non-NULL → workspace is paused (scheduling disabled)

Follows the same nullable-timestamp pattern as `ArchivedAt`. No separate `status` string column is introduced.

Two new DAO methods:

```go
func (d *DB) PauseWorkspaceState(workspaceID string) error   // sets PausedAt = now
func (d *DB) ResumeWorkspaceState(workspaceID string) error  // sets PausedAt = NULL
func (d *DB) ListActiveWorkspaceStates(limit int) ([]WorkspaceState, error) // archived_at IS NULL AND paused_at IS NULL
```

### `RuntimeWorkspaceState` (new field)

```go
cancelScheduler context.CancelFunc // non-nil when a scheduler goroutine is waiting
```

Non-nil means a goroutine is sleeping `IntervalMs` before triggering the next run. Calling it cancels the pending schedule.

---

## Scheduling Mechanism

### Core trigger: `triggerRun`

The run-creation logic inside `ApiV1CreateRun` is extracted into:

```go
func (d *ExplorationDomain) triggerRun(ctx context.Context, workspaceID string, source string) (runID string, launched bool)
```

Both the HTTP handler and the scheduler call this function. This avoids duplicating plan-build and state-mutation logic.

### `scheduleNextRun`

Called at the end of `runAgentCycle` (after a run completes successfully or fails):

```go
func (d *ExplorationDomain) scheduleNextRun(workspaceID string)
```

Steps:
1. Check DB: if `PausedAt` is non-NULL, return immediately.
2. Read `Strategy.MaxRuns` and current run count from `RuntimeWorkspaceState`. If `MaxRuns > 0` and run count >= MaxRuns, return (stay active, no more scheduling).
3. Read `Strategy.IntervalMs` from the workspace session.
4. Create a cancellable context; store `cancel` in `state.cancelScheduler`.
5. Launch goroutine:

```
select {
  case <-time.After(intervalMs):
      clear state.cancelScheduler
      triggerRun(ctx, workspaceID, "auto")
  case <-ctx.Done():
      // paused or domain shutting down — do nothing
}
```

`IntervalMs <= 0` is treated as `0` (trigger immediately after current run completes).

### `pauseScheduler`

```go
func (d *ExplorationDomain) pauseScheduler(workspaceID string) {
    d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
        if state.cancelScheduler != nil {
            state.cancelScheduler()
            state.cancelScheduler = nil
        }
    })
}
```

Does **not** interrupt an already-running `runAgentCycle`. The current run completes naturally; `scheduleNextRun` at the end will see `PausedAt` is set and exit.

---

## API Changes

### `PATCH /api/v1/workspaces/:id`

**Request body**:
```json
{ "status": "paused" | "active" }
```

**Pause flow**:
1. Verify workspace exists.
2. `DB.PauseWorkspaceState(workspaceID)`.
3. `pauseScheduler(workspaceID)`.
4. Return updated `WorkspaceView` with `status: "paused"`.

**Resume flow**:
1. Verify workspace exists.
2. `DB.ResumeWorkspaceState(workspaceID)`.
3. `scheduleNextRun(workspaceID)` with `intervalMs = 0` (start immediately).
4. Return updated `WorkspaceView` with `status: "active"`.

**Error cases**:
- `400` if `status` value is not `"paused"` or `"active"`
- `404` if workspace not found
- `409` if trying to pause an already-paused workspace or resume an already-active workspace (idempotent is also acceptable — TBD at implementation)

### `GET /api/v1/workspaces/:id` — updated response

`toWorkspaceView` is updated to read the actual status from DB:

```go
status := WorkspaceStatusActive
if state.PausedAt != nil {
    status = WorkspaceStatusPaused
}
```

---

## Startup Recovery

### New `Start` method on `ExplorationDomain`

```go
func (d *ExplorationDomain) Start(ctx context.Context) error
```

Steps:
1. `DB.ListActiveWorkspaceStates(limit=200)` — all non-archived, non-paused workspaces.
2. For each workspace:
   a. `loadWorkspace(id)` → add to `store`.
   b. `restoreRuntimeState(id)` → restore run/plan history into `RuntimeWorkspaceState`.
   c. `go scheduleNextRun(id)` — starts scheduler (delay = `IntervalMs`; if 0 or negative, triggers immediately).
3. Return nil (errors per-workspace are logged, not fatal).

### `main.go` change

```go
explorationDomain, err := exploration.BuildExplorationDomain(registry)
mistake.Unwrap(err)
exploration.RegisterRoutes(v1, explorationDomain)
go explorationDomain.Start(ctx)  // async, does not block HTTP server startup
```

`Start` runs after `RegisterRoutes` so that any WebSocket subscribers that connect immediately after server start can receive mutation events from the resumed runs.

---

## Error Handling

| Scenario | Behaviour |
|---|---|
| `runAgentCycle` panics | existing recover handler sets `AgentRunning=false`; `scheduleNextRun` is NOT called (panic path exits before it) |
| DB unavailable at `scheduleNextRun` check | treat as "not paused" — schedule proceeds (safe degradation) |
| `Start` fails to load a workspace | log and skip; does not block other workspaces |
| Backend restarts mid-run | previous run left as `running` in history; `Start` calls `scheduleNextRun` which triggers a new run; old interrupted run stays in history as-is |

---

## Files Changed

| File | Change |
|---|---|
| `datasource/dbdao/workspace_state.go` | Add `PausedAt` field; `PauseWorkspaceState`, `ResumeWorkspaceState`, `ListActiveWorkspaceStates` |
| `domain/exploration/domain.go` | Add `cancelScheduler` to `RuntimeWorkspaceState`; add `Start` method |
| `domain/exploration/runtime_agent.go` | Extract `triggerRun`; add `scheduleNextRun`, `pauseScheduler` |
| `domain/exploration/handler_workspace.go` | Add `ApiV1PatchWorkspace`; update `toWorkspaceView` to read real status |
| `domain/exploration/routes.go` | Register `PATCH /workspaces/:workspaceID` |
| `main.go` | Call `go explorationDomain.Start(ctx)` |

---

## Out of Scope

- Frontend UI changes (pause button, status indicator) — separate task
- `MaxRuns` enforcement beyond "stop scheduling" (e.g. auto-archive) — not requested
- Graceful shutdown (cancelling all schedulers on SIGTERM) — future improvement
