# Workspace Pause & Auto-Restart Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add automatic run scheduling, user-controlled pause/resume, and startup recovery for workspace exploration cycles.

**Architecture:** Three layers of change — DB (new `PausedAt` column + DAO methods), runtime scheduling (`triggerRun` extraction, `scheduleNextRun`/`pauseScheduler` goroutine logic), and API (`PATCH /workspaces/:id` + corrected status in GET). On backend startup, `domain.Start()` re-schedules all active non-paused workspaces.

**Tech Stack:** Go 1.22+, Gin, GORM + SQLite, standard `context` and `sync` packages.

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `backend/datasource/dbdao/workspace_state.go` | Modify | Add `PausedAt *time.Time` field; add `PauseWorkspaceState`, `ResumeWorkspaceState`, `ListActiveWorkspaceStates` |
| `backend/datasource/dbdao/workspace_state_test.go` | Modify | Tests for the three new DAO methods |
| `backend/domain/exploration/domain.go` | Modify | Add `cancelScheduler context.CancelFunc` to `RuntimeWorkspaceState`; add `Start` method |
| `backend/domain/exploration/runtime_agent.go` | Modify | Extract `triggerRun`; add `scheduleNextRun`, `pauseScheduler`; wire into `runAgentCycle` normal-completion path |
| `backend/domain/exploration/runtime_agent_test.go` | Modify | Tests for `scheduleNextRun` (no-schedule-when-paused, no-stack-when-agent-running, maxruns-reached) |
| `backend/domain/exploration/handler_workspace.go` | Modify | Update `toWorkspaceView` signature; add `ApiV1PatchWorkspace` |
| `backend/domain/exploration/api_test.go` | Modify | Tests for `PATCH /workspaces/:id` pause and resume flows; `GET` returns correct status |
| `backend/domain/exploration/routes.go` | Modify | Register `PATCH /workspaces/:workspaceID` in `v1` group |
| `backend/main.go` | Modify | Call `go explorationDomain.Start(ctx)` after `RegisterRoutes` |

---

## Chunk 1: DB Layer

### Task 1: Add `PausedAt` field and DAO methods

**Files:**
- Modify: `backend/datasource/dbdao/workspace_state.go`
- Modify: `backend/datasource/dbdao/workspace_state_test.go`

Background: GORM's `AutoMigrate` adds new nullable columns safely; no manual migration needed. Each test gets its own isolated in-memory SQLite database via a unique name derived from `t.Name()` — this prevents row contamination between tests that share the same process.

- [ ] **Step 1.1: Write failing tests for the three new DAO methods**

Add to `backend/datasource/dbdao/workspace_state_test.go`:

```go
// newTestDB creates an isolated in-memory SQLite database for a single test.
// Each test gets its own DB instance via a unique name, preventing cross-test contamination.
func newTestDB(t *testing.T) *DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&WorkspaceState{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return NewDB(gdb)
}

func TestPauseAndResumeWorkspaceState(t *testing.T) {
	db := newTestDB(t)

	if err := db.UpsertWorkspaceState(&WorkspaceState{WorkspaceID: "ws-pause-test", Topic: "t"}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// pause
	if err := db.PauseWorkspaceState("ws-pause-test"); err != nil {
		t.Fatalf("pause: %v", err)
	}
	state, err := db.GetWorkspaceState("ws-pause-test")
	if err != nil || state == nil {
		t.Fatalf("get after pause: %v", err)
	}
	if state.PausedAt == nil {
		t.Fatal("expected PausedAt to be set after pause")
	}

	// resume
	if err := db.ResumeWorkspaceState("ws-pause-test"); err != nil {
		t.Fatalf("resume: %v", err)
	}
	state, err = db.GetWorkspaceState("ws-pause-test")
	if err != nil || state == nil {
		t.Fatalf("get after resume: %v", err)
	}
	if state.PausedAt != nil {
		t.Fatal("expected PausedAt to be nil after resume")
	}
}

func TestListActiveWorkspaceStates(t *testing.T) {
	db := newTestDB(t)

	for _, id := range []string{"ws-b", "ws-c", "ws-a"} { // ws-a last → highest updated_at
		if err := db.UpsertWorkspaceState(&WorkspaceState{WorkspaceID: id, Topic: id}); err != nil {
			t.Fatalf("upsert %s: %v", id, err)
		}
	}
	if err := db.PauseWorkspaceState("ws-b"); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if err := db.ArchiveWorkspaceState("ws-c"); err != nil {
		t.Fatalf("archive: %v", err)
	}

	active, err := db.ListActiveWorkspaceStates(50)
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	// Use set-membership check — ordering may vary if timestamps are equal.
	found := map[string]bool{}
	for _, s := range active {
		found[s.WorkspaceID] = true
	}
	if !found["ws-a"] {
		t.Fatalf("expected ws-a in active results, got: %v", active)
	}
	if found["ws-b"] {
		t.Fatal("expected ws-b to be excluded (paused)")
	}
	if found["ws-c"] {
		t.Fatal("expected ws-c to be excluded (archived)")
	}
}
```

- [ ] **Step 1.2: Run tests to confirm they fail**

```bash
cd backend && go test ./datasource/dbdao/... -run "TestPauseAndResume|TestListActive" -v
```
Expected: FAIL — `PauseWorkspaceState`, `ResumeWorkspaceState`, `ListActiveWorkspaceStates` undefined.

- [ ] **Step 1.3: Add `PausedAt` field to `WorkspaceState`**

In `backend/datasource/dbdao/workspace_state.go`, add `PausedAt` after `ArchivedAt`:

```go
ArchivedAt          *time.Time `json:"archived_at" gorm:"index"`
PausedAt            *time.Time `json:"paused_at" gorm:"index"`
```

- [ ] **Step 1.4: Implement the three new DAO methods**

Append to `backend/datasource/dbdao/workspace_state.go`:

```go
func (d *DB) PauseWorkspaceState(workspaceID string) error {
	now := time.Now()
	return d.DB().
		Model(&WorkspaceState{}).
		Where("workspace_id = ?", workspaceID).
		Update("paused_at", &now).Error
}

func (d *DB) ResumeWorkspaceState(workspaceID string) error {
	return d.DB().
		Model(&WorkspaceState{}).
		Where("workspace_id = ?", workspaceID).
		Update("paused_at", gorm.Expr("NULL")).Error
}

func (d *DB) ListActiveWorkspaceStates(limit int) ([]WorkspaceState, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var states []WorkspaceState
	err := d.DB().
		Where("archived_at IS NULL AND paused_at IS NULL").
		Order("updated_at desc").
		Limit(limit).
		Find(&states).Error
	return states, err
}
```

- [ ] **Step 1.5: Run tests to confirm they pass**

```bash
cd backend && go test ./datasource/dbdao/... -v
```
Expected: PASS for all tests including the two new ones.

- [ ] **Step 1.6: Build check**

```bash
cd backend && go build ./...
```
Expected: no errors.

- [ ] **Step 1.7: Commit**

```bash
git add backend/datasource/dbdao/workspace_state.go backend/datasource/dbdao/workspace_state_test.go
git commit -m "feat: add PausedAt field and pause/resume/list-active DAO methods"
```

---

## Chunk 2: Runtime Scheduling Layer

### Task 2: Extract `triggerRun` from `ApiV1CreateRun`

**Files:**
- Modify: `backend/domain/exploration/runtime_agent.go`
- Modify: `backend/domain/exploration/handler_run.go`

Background: `ApiV1CreateRun` in `handler_run.go` does run creation inline. Extract the core logic into `triggerRun` so both the HTTP handler and the scheduler can reuse it without duplication.

- [ ] **Step 2.1: Add `triggerRun` to `runtime_agent.go`**

Append to `backend/domain/exploration/runtime_agent.go`:

```go
// triggerRun creates a new run for the workspace and launches runAgentCycle in a goroutine.
// If a cycle is already running (AgentRunning==true), it returns the existing run ID with launched=false.
// Must be called while NOT holding runtime.mu or store.mu.
func (d *ExplorationDomain) triggerRun(ctx context.Context, workspaceID string, source string) (runID string, launched bool) {
	snapshot, ok := d.GetWorkspace(workspaceID)
	if !ok {
		return "", false
	}
	session := snapshot.Exploration

	// Build plan outside lock (deterministic, no I/O).
	plan, steps, _ := d.planner.BuildInitialPlan(ctx, &session, &RuntimeWorkspaceState{})

	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		if state.AgentRunning {
			if len(state.Runs) > 0 {
				runID = state.Runs[len(state.Runs)-1].ID
			}
			return
		}
		now := time.Now()
		runID = fmt.Sprintf("run-%s-%d", workspaceID, now.UnixNano())
		run := Run{
			ID:          runID,
			WorkspaceID: workspaceID,
			Source:      source,
			Status:      RunStatusRunning,
			StartedAt:   now.UnixMilli(),
		}
		state.Runs = append(state.Runs, run)
		if state.Balance.WorkspaceID == "" {
			state.Balance = buildInitialBalance(session, runID, now)
		}
		if plan != nil {
			plan.RunID = runID
			for i := range steps {
				steps[i].RunID = runID
			}
			if len(state.Plans) > 0 {
				plan.Version = state.Plans[len(state.Plans)-1].Version + 1
			}
			state.Plans = append(state.Plans, *plan)
			state.PlanSteps = append(state.PlanSteps, steps...)
		}
		state.Mutations = append(state.Mutations, MutationEvent{
			ID:          mutationID(workspaceID),
			WorkspaceID: workspaceID,
			Kind:        "run_created",
			Run:         &GenerationRun{ID: runID},
			CreatedAt:   now.UnixMilli(),
		})
		state.AgentRunning = true
		launched = true
	})

	if launched {
		go d.runAgentCycle(workspaceID)
	}
	return runID, launched
}
```

- [ ] **Step 2.2: Simplify `ApiV1CreateRun` to call `triggerRun`**

Replace the body of `ApiV1CreateRun` in `backend/domain/exploration/handler_run.go` with:

```go
func (d *ExplorationDomain) ApiV1CreateRun(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	var req CreateRunRequest
	if c.ContentType() == "application/json" {
		if err := c.ShouldBindJSON(&req); err != nil {
			writeV1Error(c, http.StatusBadRequest, "invalid_argument", "failed to parse create run request")
			return
		}
	}

	if _, ok := d.GetWorkspace(workspaceID); !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "workspace not found")
		return
	}

	source := strings.TrimSpace(req.Trigger)
	if source == "" {
		source = "manual"
	}

	runID, launched := d.triggerRun(c.Request.Context(), workspaceID, source)

	runtimeState, ok := d.GetRuntimeState(workspaceID)
	if !ok || runID == "" {
		writeV1Error(c, http.StatusInternalServerError, "internal", "failed to create run")
		return
	}
	var targetRun Run
	for _, r := range runtimeState.Runs {
		if r.ID == runID {
			targetRun = r
			break
		}
	}

	status := http.StatusAccepted
	if !launched {
		status = http.StatusOK
	}
	c.JSON(status, RunResponse{Run: d.buildRunView(runtimeState, targetRun)})
}
```

- [ ] **Step 2.3: Build and run existing tests to confirm no regression**

```bash
cd backend && go build ./... && go test ./domain/exploration/... -run "TestCreateWorkspace|TestV1CreateRun" -v
```
Expected: PASS (or skipped if not present). No compile errors.

- [ ] **Step 2.4: Commit**

```bash
git add backend/domain/exploration/runtime_agent.go backend/domain/exploration/handler_run.go
git commit -m "refactor: extract triggerRun from ApiV1CreateRun"
```

---

### Task 3: Add `cancelScheduler` and implement `scheduleNextRun` / `pauseScheduler`

**Files:**
- Modify: `backend/domain/exploration/domain.go`
- Modify: `backend/domain/exploration/runtime_agent.go`
- Modify: `backend/domain/exploration/runtime_agent_test.go`

- [ ] **Step 3.1: Add `cancelScheduler` to `RuntimeWorkspaceState`**

In `backend/domain/exploration/domain.go`, add to the `RuntimeWorkspaceState` struct after `Cursor`:

```go
cancelScheduler context.CancelFunc // non-nil while a scheduler goroutine is pending the next run
```

Also add `"context"` to the import block if not already present.

- [ ] **Step 3.2: Write failing tests for `scheduleNextRun` and `pauseScheduler`**

Add to `backend/domain/exploration/runtime_agent_test.go`:

```go
func TestScheduleNextRun_DoesNotScheduleWhenMaxRunsReached(t *testing.T) {
	// With nil DB, GetWorkspaceState returns nil → treated as "not paused" (safe degradation).
	// So we test the "MaxRuns reached" guard instead as a proxy for the early-exit paths.
	domain := newTestExplorationDomain()
	wsID := "ws-sched-maxruns"

	// Add workspace to store
	domain.store.mu.Lock()
	domain.store.workspaces[wsID] = &ExplorationSession{
		ID:       wsID,
		Topic:    "test",
		Strategy: RuntimeStrategy{MaxRuns: 1, IntervalMs: 0},
	}
	domain.store.mu.Unlock()

	// Seed one completed run so run count == MaxRuns
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		s.Runs = []Run{{ID: "run-1", WorkspaceID: wsID, Status: RunStatusCompleted}}
	})

	domain.scheduleNextRun(wsID)

	// cancelScheduler should NOT be set (no scheduler was launched)
	var cancelSet bool
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		cancelSet = s.cancelScheduler != nil
	})
	if cancelSet {
		t.Fatal("expected no scheduler to be launched when MaxRuns reached")
	}
}

func TestScheduleNextRun_DoesNotScheduleWhenAgentRunning(t *testing.T) {
	domain := newTestExplorationDomain()
	wsID := "ws-sched-running"

	domain.store.mu.Lock()
	domain.store.workspaces[wsID] = &ExplorationSession{
		ID:       wsID,
		Topic:    "test",
		Strategy: RuntimeStrategy{MaxRuns: 0, IntervalMs: 0},
	}
	domain.store.mu.Unlock()

	// Mark agent as already running
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		s.AgentRunning = true
	})

	domain.scheduleNextRun(wsID)

	var cancelSet bool
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		cancelSet = s.cancelScheduler != nil
	})
	if cancelSet {
		t.Fatal("expected no scheduler to be launched when AgentRunning=true")
	}
}

func TestPauseScheduler_CancelsWaitingScheduler(t *testing.T) {
	domain := newTestExplorationDomain()
	wsID := "ws-pause-sched"

	domain.store.mu.Lock()
	domain.store.workspaces[wsID] = &ExplorationSession{
		ID:       wsID,
		Topic:    "test",
		Strategy: RuntimeStrategy{MaxRuns: 0, IntervalMs: 60000}, // 60s — won't fire during test
	}
	domain.store.mu.Unlock()

	domain.scheduleNextRun(wsID)

	// cancelScheduler should be set now
	var cancelSetBefore bool
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		cancelSetBefore = s.cancelScheduler != nil
	})
	if !cancelSetBefore {
		t.Fatal("expected scheduler goroutine to be waiting (cancelScheduler set)")
	}

	domain.pauseScheduler(wsID)

	// cancelScheduler should be nil after pause
	var cancelSetAfter bool
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		cancelSetAfter = s.cancelScheduler != nil
	})
	if cancelSetAfter {
		t.Fatal("expected cancelScheduler to be nil after pauseScheduler")
	}
}
```

- [ ] **Step 3.3: Run tests to confirm they fail**

```bash
cd backend && go test ./domain/exploration/... -run "TestScheduleNextRun|TestPauseScheduler" -v
```
Expected: FAIL — `scheduleNextRun`, `pauseScheduler` undefined.

- [ ] **Step 3.4: Implement `scheduleNextRun` and `pauseScheduler`**

Append to `backend/domain/exploration/runtime_agent.go`:

```go
// scheduleNextRun inspects workspace state and, if scheduling is warranted,
// launches a goroutine that will call triggerRun after IntervalMs delay.
// It returns immediately (non-blocking). Never call with runtime.mu or store.mu held.
func (d *ExplorationDomain) scheduleNextRun(workspaceID string) {
	// Step 1: DB paused check (outside any lock).
	if d.DB != nil {
		dbState, err := d.DB.GetWorkspaceState(workspaceID)
		if err == nil && dbState != nil && dbState.PausedAt != nil {
			return
		}
	}

	// Step 2: Runtime guard checks (under runtime.mu).
	var maxRuns, runCount int
	var agentRunning bool
	d.withWorkspaceState(workspaceID, func(s *RuntimeWorkspaceState) {
		agentRunning = s.AgentRunning
		runCount = len(s.Runs)
	})
	if agentRunning {
		return
	}

	// Step 3: Read IntervalMs from session store (separate lock from runtime.mu).
	var intervalMs int
	d.store.mu.RLock()
	if session, ok := d.store.workspaces[workspaceID]; ok {
		maxRuns = session.Strategy.MaxRuns
		intervalMs = session.Strategy.IntervalMs
	}
	d.store.mu.RUnlock()

	if maxRuns > 0 && runCount >= maxRuns {
		return
	}

	if intervalMs < 0 {
		intervalMs = 0
	}

	// Step 4: Store cancel func (under runtime.mu).
	ctx, cancel := context.WithCancel(context.Background())
	d.withWorkspaceState(workspaceID, func(s *RuntimeWorkspaceState) {
		// If a scheduler is already waiting, cancel it first.
		if s.cancelScheduler != nil {
			s.cancelScheduler()
		}
		s.cancelScheduler = cancel
	})

	// Step 5: Launch scheduler goroutine (outside any lock).
	go func() {
		select {
		case <-time.After(time.Duration(intervalMs) * time.Millisecond):
			// Re-check paused state to close the narrow race where a pause arrived
			// after the DB check above but before the context was stored.
			if d.DB != nil {
				dbState, err := d.DB.GetWorkspaceState(workspaceID)
				if err == nil && dbState != nil && dbState.PausedAt != nil {
					d.withWorkspaceState(workspaceID, func(s *RuntimeWorkspaceState) {
						s.cancelScheduler = nil
					})
					return
				}
			}
			d.withWorkspaceState(workspaceID, func(s *RuntimeWorkspaceState) {
				s.cancelScheduler = nil
			})
			d.triggerRun(ctx, workspaceID, "auto")
		case <-ctx.Done():
			// Cancelled by pauseScheduler — do nothing.
		}
	}()
}

// pauseScheduler cancels any pending scheduler goroutine for the workspace.
// Does NOT interrupt a currently running runAgentCycle.
func (d *ExplorationDomain) pauseScheduler(workspaceID string) {
	d.withWorkspaceState(workspaceID, func(s *RuntimeWorkspaceState) {
		if s.cancelScheduler != nil {
			s.cancelScheduler()
			s.cancelScheduler = nil
		}
	})
}
```

- [ ] **Step 3.5: Run tests to confirm they pass**

```bash
cd backend && go test ./domain/exploration/... -run "TestScheduleNextRun|TestPauseScheduler" -v
```
Expected: PASS.

- [ ] **Step 3.6: Commit**

```bash
git add backend/domain/exploration/domain.go backend/domain/exploration/runtime_agent.go backend/domain/exploration/runtime_agent_test.go
git commit -m "feat: add cancelScheduler, scheduleNextRun, pauseScheduler"
```

---

### Task 4: Wire `scheduleNextRun` into `runAgentCycle`

**Files:**
- Modify: `backend/domain/exploration/runtime_agent.go`

- [ ] **Step 4.1: Add `scheduleNextRun` call at normal completion of `runAgentCycle`**

In `backend/domain/exploration/runtime_agent.go`, find the `runAgentCycle` function. After the block that sets `AgentRunning=false` and broadcasts `run_completed` (around line 537–544), add a call to `scheduleNextRun`. The final tail of `runAgentCycle` should look like:

```go
	// Mark complete and broadcast
	var completedRunID string
	d.withWorkspaceState(workspaceID, func(s *RuntimeWorkspaceState) {
		s.AgentRunning = false
		if len(s.Runs) > 0 {
			completedRunID = s.Runs[len(s.Runs)-1].ID
			if s.Runs[len(s.Runs)-1].Status == RunStatusRunning {
				s.Runs[len(s.Runs)-1].Status = RunStatusCompleted
				s.Runs[len(s.Runs)-1].EndedAt = time.Now().UnixMilli()
			}
		}
	})
	d.broadcastMutations(workspaceID, []MutationEvent{{
		ID:          mutationID(workspaceID),
		WorkspaceID: workspaceID,
		Kind:        "run_completed",
		Run:         &GenerationRun{ID: completedRunID},
		CreatedAt:   time.Now().UnixMilli(),
	}})
	// Schedule the next run (respects pause state, MaxRuns, and IntervalMs).
	// Not called on the panic path — the defer handler exits before reaching here.
	d.scheduleNextRun(workspaceID)
```

- [ ] **Step 4.2: Build and run full test suite**

```bash
cd backend && go build ./... && go test ./domain/exploration/... -v -timeout 60s
```
Expected: all existing tests pass; no new failures introduced.

- [ ] **Step 4.3: Commit**

```bash
git add backend/domain/exploration/runtime_agent.go
git commit -m "feat: wire scheduleNextRun into runAgentCycle normal-completion path"
```

---

## Chunk 3: API Layer

### Task 5: Update `toWorkspaceView` signature and `ApiV1GetWorkspace`

**Files:**
- Modify: `backend/domain/exploration/handler_workspace.go`

- [ ] **Step 5.1: Write a failing test for correct status in GET response**

Add to `backend/domain/exploration/api_test.go` (in the `exploration` package):

```go
func TestGetWorkspaceReturnsActiveStatus(t *testing.T) {
	r, _ := newTestRouterWithDomain()

	createBody := []byte(`{"topic":"status test","output_goal":"goal"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: unexpected status %d", w.Code)
	}

	var createResp WorkspaceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	wsID := createResp.Workspace.ID

	getReq, _ := http.NewRequest(http.MethodGet, "/api/v1/workspaces/"+wsID, nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("get: unexpected status %d", getW.Code)
	}

	var getResp WorkspaceResponse
	if err := json.Unmarshal(getW.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if getResp.Workspace.Status != WorkspaceStatusActive {
		t.Fatalf("expected status active, got %s", getResp.Workspace.Status)
	}
}
```

- [ ] **Step 5.2: Run to confirm it passes (it should already pass or reveal the hardcoded status)**

```bash
cd backend && go test ./domain/exploration/... -run "TestGetWorkspaceReturnsActiveStatus" -v
```
If it passes (hardcoded `active` returns same), that's fine — the test documents the contract. It will become meaningful once status can be `paused`.

- [ ] **Step 5.3: Update `toWorkspaceView` signature to accept `dbState`**

Replace `toWorkspaceView` in `backend/domain/exploration/handler_workspace.go`:

```go
func toWorkspaceView(session ExplorationSession, dbState *dbdao.WorkspaceState) WorkspaceView {
	constraints := []string{}
	if strings.TrimSpace(session.Constraints) != "" {
		constraints = append(constraints, session.Constraints)
	}
	nowISO := time.Now().UTC().Format(time.RFC3339)
	status := WorkspaceStatusActive
	if dbState != nil && dbState.PausedAt != nil {
		status = WorkspaceStatusPaused
	}
	return WorkspaceView{
		ID:          session.ID,
		Topic:       session.Topic,
		Goal:        session.OutputGoal,
		Constraints: constraints,
		Status:      status,
		CreatedAt:   nowISO,
		UpdatedAt:   nowISO,
	}
}
```

Add `"backend/datasource/dbdao"` to the import block if not already present.

- [ ] **Step 5.4: Update all callers of `toWorkspaceView`**

In `handler_workspace.go`, `ApiV1CreateWorkspace` passes `nil` (freshly created workspace is never paused):

```go
c.JSON(http.StatusCreated, WorkspaceResponse{Workspace: toWorkspaceView(snapshot.Exploration, nil)})
```

`ApiV1GetWorkspace` fetches the DB state and passes it:

```go
func (d *ExplorationDomain) ApiV1GetWorkspace(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	snapshot, ok := d.GetWorkspace(workspaceID)
	if !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "workspace not found")
		return
	}
	var dbState *dbdao.WorkspaceState
	if d.DB != nil {
		dbState, _ = d.DB.GetWorkspaceState(workspaceID)
	}
	c.JSON(http.StatusOK, WorkspaceResponse{Workspace: toWorkspaceView(snapshot.Exploration, dbState)})
}
```

- [ ] **Step 5.5: Build check**

```bash
cd backend && go build ./...
```
Expected: no errors.

- [ ] **Step 5.6: Run tests**

```bash
cd backend && go test ./domain/exploration/... -v -timeout 60s
```
Expected: all pass.

- [ ] **Step 5.7: Commit**

```bash
git add backend/domain/exploration/handler_workspace.go
git commit -m "feat: update toWorkspaceView to reflect real paused status from DB"
```

---

### Task 6: Add `ApiV1PatchWorkspace` handler

**Files:**
- Modify: `backend/domain/exploration/handler_workspace.go`
- Modify: `backend/domain/exploration/schema.go`
- Modify: `backend/domain/exploration/api_test.go`

- [ ] **Step 6.1: Add `PatchWorkspaceReq` to schema**

In `backend/domain/exploration/schema.go`, add after `UpdateStrategyReq`:

```go
type PatchWorkspaceReq struct {
	Status string `json:"status" binding:"required"`
}
```

- [ ] **Step 6.2: Write failing tests for pause and resume API flows**

Add to `backend/domain/exploration/api_test.go`:

```go
func TestPatchWorkspacePause(t *testing.T) {
	r, _ := newTestRouterWithDomain()

	// Create workspace
	createBody := []byte(`{"topic":"pause test","output_goal":"goal"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d", w.Code)
	}
	var createResp WorkspaceResponse
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	wsID := createResp.Workspace.ID

	// Pause it
	patchBody := []byte(`{"status":"paused"}`)
	patchReq, _ := http.NewRequest(http.MethodPatch, "/api/v1/workspaces/"+wsID, bytes.NewBuffer(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchW := httptest.NewRecorder()
	r.ServeHTTP(patchW, patchReq)
	if patchW.Code != http.StatusOK {
		t.Fatalf("patch pause: unexpected status %d body=%s", patchW.Code, patchW.Body.String())
	}
	var patchResp WorkspaceResponse
	if err := json.Unmarshal(patchW.Body.Bytes(), &patchResp); err != nil {
		t.Fatalf("decode patch: %v", err)
	}
	if patchResp.Workspace.Status != WorkspaceStatusPaused {
		t.Fatalf("expected paused, got %s", patchResp.Workspace.Status)
	}
}

func TestPatchWorkspaceResume(t *testing.T) {
	r, domain := newTestRouterWithDomain()

	createBody := []byte(`{"topic":"resume test","output_goal":"goal"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d", w.Code)
	}
	var createResp WorkspaceResponse
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	wsID := createResp.Workspace.ID

	// Pause first (if DB is present)
	if domain.DB != nil {
		if err := domain.DB.PauseWorkspaceState(wsID); err != nil {
			t.Fatalf("pause: %v", err)
		}
	}

	// Resume
	patchBody := []byte(`{"status":"active"}`)
	patchReq, _ := http.NewRequest(http.MethodPatch, "/api/v1/workspaces/"+wsID, bytes.NewBuffer(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchW := httptest.NewRecorder()
	r.ServeHTTP(patchW, patchReq)
	if patchW.Code != http.StatusOK {
		t.Fatalf("patch resume: unexpected status %d body=%s", patchW.Code, patchW.Body.String())
	}
	var patchResp WorkspaceResponse
	if err := json.Unmarshal(patchW.Body.Bytes(), &patchResp); err != nil {
		t.Fatalf("decode patch: %v", err)
	}
	if patchResp.Workspace.Status != WorkspaceStatusActive {
		t.Fatalf("expected active, got %s", patchResp.Workspace.Status)
	}
}

func TestPatchWorkspace_InvalidStatus(t *testing.T) {
	r, _ := newTestRouterWithDomain()

	createBody := []byte(`{"topic":"invalid test","output_goal":"goal"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/workspaces", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var createResp WorkspaceResponse
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	wsID := createResp.Workspace.ID

	patchBody := []byte(`{"status":"banana"}`)
	patchReq, _ := http.NewRequest(http.MethodPatch, "/api/v1/workspaces/"+wsID, bytes.NewBuffer(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchW := httptest.NewRecorder()
	r.ServeHTTP(patchW, patchReq)
	if patchW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", patchW.Code)
	}
}
```

- [ ] **Step 6.3: Run tests to confirm they fail**

```bash
cd backend && go test ./domain/exploration/... -run "TestPatchWorkspace" -v
```
Expected: FAIL — route not registered / handler undefined.

- [ ] **Step 6.4: Implement `ApiV1PatchWorkspace`**

Append to `backend/domain/exploration/handler_workspace.go`:

```go
func (d *ExplorationDomain) ApiV1PatchWorkspace(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	var req PatchWorkspaceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_argument", "failed to parse patch request")
		return
	}

	switch req.Status {
	case "paused", "active":
	default:
		writeV1Error(c, http.StatusBadRequest, "invalid_argument", "status must be 'paused' or 'active'")
		return
	}

	snapshot, ok := d.GetWorkspace(workspaceID)
	if !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "workspace not found")
		return
	}

	ctx := c.Request.Context()

	var dbState *dbdao.WorkspaceState
	if req.Status == "paused" {
		now := time.Now()
		if d.DB != nil {
			if err := d.DB.PauseWorkspaceState(workspaceID); err != nil {
				writeV1Error(c, http.StatusInternalServerError, "internal", "failed to pause workspace")
				return
			}
			dbState, _ = d.DB.GetWorkspaceState(workspaceID)
		}
		// In no-DB mode, construct a synthetic dbState so the response reflects "paused".
		if dbState == nil {
			dbState = &dbdao.WorkspaceState{PausedAt: &now}
		}
		d.pauseScheduler(workspaceID)
	} else {
		if d.DB != nil {
			if err := d.DB.ResumeWorkspaceState(workspaceID); err != nil {
				writeV1Error(c, http.StatusInternalServerError, "internal", "failed to resume workspace")
				return
			}
		}
		// Start a run immediately; subsequent runs will use IntervalMs from strategy.
		d.triggerRun(ctx, workspaceID, "resume")
		dbState = nil // PausedAt is nil → status = active
	}

	c.JSON(http.StatusOK, WorkspaceResponse{Workspace: toWorkspaceView(snapshot.Exploration, dbState)})
}
```

- [ ] **Step 6.5: Register the route in `routes.go`**

In `backend/domain/exploration/routes.go`, inside the `v1 := router.Group("")` block, add after the last `v1.GET` line:

```go
v1.PATCH("/workspaces/:workspaceID", domain.ApiV1PatchWorkspace)
```

- [ ] **Step 6.6: Run tests to confirm they pass**

```bash
cd backend && go test ./domain/exploration/... -run "TestPatchWorkspace" -v
```
Expected: PASS for all three patch tests.

In no-DB mode (test domain may or may not have a real DB): the pause handler constructs a synthetic `dbState` with `PausedAt` set, so `toWorkspaceView` correctly returns `status: "paused"`. The `if d.DB != nil` guards skip the DB writes when no DB is present.

- [ ] **Step 6.7: Run full test suite**

```bash
cd backend && go test ./domain/exploration/... -v -timeout 60s
```
Expected: all pass.

- [ ] **Step 6.8: Commit**

```bash
git add backend/domain/exploration/handler_workspace.go backend/domain/exploration/schema.go backend/domain/exploration/routes.go backend/domain/exploration/api_test.go
git commit -m "feat: add PATCH /workspaces/:id pause/resume endpoint"
```

---

## Chunk 4: Startup Recovery

### Task 7: Add `Start` method and wire into `main.go`

**Files:**
- Modify: `backend/domain/exploration/domain.go`
- Modify: `backend/main.go`

- [ ] **Step 7.1: Write a failing test for `Start`**

Add to `backend/domain/exploration/runtime_agent_test.go`:

```go
func TestStart_DoesNotPanicWithNilDB(t *testing.T) {
	// The test domain has no DB — Start should handle this gracefully (no-op for DB path).
	domain := newTestExplorationDomain()
	ctx := context.Background()

	// Should not panic or error even with nil DB.
	if err := domain.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
}
```

- [ ] **Step 7.2: Run test to confirm it fails**

```bash
cd backend && go test ./domain/exploration/... -run "TestStart" -v
```
Expected: FAIL — `Start` undefined.

- [ ] **Step 7.3: Implement `Start` on `ExplorationDomain`**

Append to `backend/domain/exploration/domain.go`:

```go
// Start loads all active (non-paused, non-archived) workspaces from the DB and
// schedules a run for each. It is non-fatal per workspace: errors are logged and skipped.
// Call this after RegisterRoutes so WS subscribers can receive mutation events.
// Run in a goroutine from main.go to avoid blocking HTTP server startup.
func (d *ExplorationDomain) Start(ctx context.Context) error {
	if d.DB == nil {
		return nil
	}
	states, err := d.DB.ListActiveWorkspaceStates(200)
	if err != nil {
		return fmt.Errorf("Start: list active workspaces: %w", err)
	}
	for _, state := range states {
		wsID := state.WorkspaceID
		session, ok := d.loadWorkspace(wsID)
		if !ok || session == nil {
			continue
		}
		d.store.mu.Lock()
		d.store.workspaces[wsID] = session
		d.store.mu.Unlock()

		d.restoreRuntimeState(wsID)
		d.scheduleNextRun(wsID)
	}
	return nil
}
```

Add `"fmt"` to the `domain.go` import if not already present.

- [ ] **Step 7.4: Run test to confirm it passes**

```bash
cd backend && go test ./domain/exploration/... -run "TestStart" -v
```
Expected: PASS.

- [ ] **Step 7.5: Wire `Start` into `main.go`**

In `backend/main.go`, after `exploration.RegisterRoutes(v1, explorationDomain)`, add:

```go
go explorationDomain.Start(ctx)
```

The full relevant section:
```go
explorationDomain, err := exploration.BuildExplorationDomain(registry)
mistake.Unwrap(err)
exploration.RegisterRoutes(v1, explorationDomain)
go explorationDomain.Start(ctx) // resume active workspaces after restart
```

- [ ] **Step 7.6: Build the entire backend**

```bash
cd backend && go build ./...
```
Expected: no errors.

- [ ] **Step 7.7: Run full test suite**

```bash
cd backend && go test ./... -timeout 120s
```
Expected: all tests pass.

- [ ] **Step 7.8: Commit**

```bash
git add backend/domain/exploration/domain.go backend/domain/exploration/runtime_agent_test.go backend/main.go
git commit -m "feat: add Start method for workspace auto-restart on backend startup"
```

---

## Final Verification

- [ ] **Step F.1: Full build and test**

```bash
cd backend && go build ./... && go test ./... -timeout 120s -v 2>&1 | tail -30
```
Expected: all PASS, no compile errors.

- [ ] **Step F.2: Smoke-test the API manually (optional but recommended)**

```bash
# Start backend
cd backend && go run main.go &

# Create a workspace
curl -s -X POST http://localhost:8888/api/v1/workspaces \
  -H "Content-Type: application/json" \
  -d '{"topic":"test pause","output_goal":"goal"}' | jq .workspace.status
# Expected: "active"

# Pause it (replace WS_ID with actual id from above)
curl -s -X PATCH http://localhost:8888/api/v1/workspaces/<WS_ID> \
  -H "Content-Type: application/json" \
  -d '{"status":"paused"}' | jq .workspace.status
# Expected: "paused"

# Resume it
curl -s -X PATCH http://localhost:8888/api/v1/workspaces/<WS_ID> \
  -H "Content-Type: application/json" \
  -d '{"status":"active"}' | jq .workspace.status
# Expected: "active"
```
