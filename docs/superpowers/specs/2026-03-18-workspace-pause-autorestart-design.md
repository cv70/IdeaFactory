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

GORM's `AutoMigrate` will add this column safely on existing databases (SQLite: ALTER TABLE adds the column as NULL for all existing rows, correctly defaulting to "active"). No manual migration is required.

Three new DAO methods:

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

Called synchronously at the end of `runAgentCycle` on **normal completion only** (not on panic — the panic `defer` exits before this call). Also called directly (no `go`) from `Start` during startup recovery. It returns quickly after launching a goroutine — it is never itself blocking, so no `go` wrapper is needed at any call site.

```go
func (d *ExplorationDomain) scheduleNextRun(workspaceID string)
```

**Locking discipline** (critical — must not hold `runtime.mu` while doing DB or blocking I/O, and must not hold `runtime.mu` while acquiring `store.mu`):

1. **Outside any lock** — check DB: if `PausedAt` is non-NULL, return immediately.
2. **Under `withWorkspaceState`** — read `state.Runs` length (run count) and `state.AgentRunning`. If `AgentRunning` is true, return without scheduling. If `MaxRuns > 0` and run count >= `MaxRuns`, return.
3. **Outside any lock** — read `IntervalMs` from the session store (`d.store.mu.RLock` → read `session.Strategy.IntervalMs` → unlock). This is a separate lock acquisition from step 2 — the two locks (`runtime.mu` and `store.mu`) must never be held simultaneously.
4. **Under `withWorkspaceState`** — create a cancellable context; store `cancel` in `state.cancelScheduler`.
5. **Outside any lock** — launch the scheduler goroutine:

```
select {
  case <-time.After(intervalMs):
      // Re-check paused state before firing — handles the narrow race where
      // a pause arrived after scheduleNextRun's DB check but before the context
      // was stored. If now paused, abort silently.
      dbState, _ := d.DB.GetWorkspaceState(workspaceID)
      if dbState != nil && dbState.PausedAt != nil {
          d.withWorkspaceState(workspaceID, func(s *RuntimeWorkspaceState) { s.cancelScheduler = nil })
          return
      }
      // clear cancelScheduler under withWorkspaceState before triggering
      d.withWorkspaceState(workspaceID, func(s *RuntimeWorkspaceState) {
          s.cancelScheduler = nil
      })
      triggerRun(ctx, workspaceID, "auto")
  case <-ctx.Done():
      // paused or domain shutting down — do nothing
}
```

`IntervalMs <= 0` is treated as `0` (trigger immediately after the goroutine starts).

**`AgentRunning` guard in `triggerRun`**: `triggerRun` preserves the existing `AgentRunning` check from `ApiV1CreateRun`. If a cycle is already running when the scheduler fires (e.g. a previous run lasted longer than `IntervalMs`), `triggerRun` returns `launched=false` and does not stack concurrent cycles.

**Fresh workspaces**: `scheduleNextRun` fires for any active non-paused workspace at startup, including ones with no prior runs. A workspace with no runs will have run count 0; if `MaxRuns == 0` (unlimited) or `MaxRuns > 0`, the scheduler will trigger a first run. This is intentional — a workspace that exists and is not paused should start exploring automatically.

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

The route is registered in the `v1` group inside `RegisterRoutes` (same group as all other `ApiV1*` handlers, not the legacy `/exploration` group):

```go
v1.PATCH("/workspaces/:workspaceID", domain.ApiV1PatchWorkspace)
```

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
3. Call `triggerRun(ctx, workspaceID, "resume")` directly — starts a run immediately without any delay. The run's normal completion will then call `scheduleNextRun`, which will use `Strategy.IntervalMs` for all subsequent intervals.
4. Return updated `WorkspaceView` with `status: "active"`.

**Error cases**:
- `400` if `status` value is not `"paused"` or `"active"`
- `404` if workspace not found
- `409` if trying to pause an already-paused workspace or resume an already-active workspace (idempotent is also acceptable — TBD at implementation)

### `GET /api/v1/workspaces/:id` — updated response

`toWorkspaceView` signature is updated to accept the DB state alongside the session:

```go
func toWorkspaceView(session ExplorationSession, dbState *dbdao.WorkspaceState) WorkspaceView
```

`ApiV1GetWorkspace` calls `d.DB.GetWorkspaceState(workspaceID)` to get `dbState`, then passes it to `toWorkspaceView`. `ApiV1CreateWorkspace` passes `nil` for `dbState` (a freshly created workspace is never paused). Status is derived as:

```go
status := WorkspaceStatusActive
if dbState != nil && dbState.PausedAt != nil {
    status = WorkspaceStatusPaused
}
```

When `DB == nil` (no-DB mode), `dbState` is `nil` and status defaults to `active`.

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
   c. `scheduleNextRun(id)` — called directly (no `go`); it is non-blocking and returns immediately after launching an internal goroutine. Each workspace's scheduler starts with the strategy's `IntervalMs` delay.
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
