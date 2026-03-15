# Idea Factory MVP Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a runnable end-to-end MVP that creates workspaces, executes deterministic runs, projects a direction map, accepts interventions, and renders the loop in the frontend against the v1 API.

**Architecture:** Keep the implementation inside the existing Go `exploration` domain and React workbench. Introduce a deterministic runtime path that owns run/plan/task/projection/intervention state in memory, wire the v1 API handlers to that runtime, then adapt the frontend's exploration-oriented model to the workspace/projection contract with minimal component churn.

**Tech Stack:** Go, Gin, in-memory domain state, React 19, TypeScript, Vite, Vitest

---

### Task 1: Lock the backend MVP contract with failing tests

**Files:**
- Modify: `backend/domain/exploration/api_test.go`
- Modify: `backend/domain/exploration/api_v1.go`
- Modify: `backend/domain/exploration/routes.go`
- Test: `backend/domain/exploration/api_test.go`

**Step 1: Write the failing test**

```go
func TestV1WorkspaceRunProjectionInterventionLifecycle(t *testing.T) {
	domain := NewExplorationDomain(nil, nil)
	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterRoutes(v1, domain)

	createBody := `{"topic":"AI travel planner","goal":"find product directions","constraints":["b2c","mobile-first"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusCreated, resp.Code)

	var workspaceResp WorkspaceResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &workspaceResp))
	workspaceID := workspaceResp.Workspace.ID

	runReq := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+workspaceID+"/runs", nil)
	runResp := httptest.NewRecorder()
	router.ServeHTTP(runResp, runReq)
	require.Equal(t, http.StatusAccepted, runResp.Code)

	projectionReq := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/"+workspaceID+"/projection", nil)
	projectionResp := httptest.NewRecorder()
	router.ServeHTTP(projectionResp, projectionReq)
	require.Equal(t, http.StatusOK, projectionResp.Code)
	require.Contains(t, projectionResp.Body.String(), `"recent_changes"`)

	interventionBody := `{"kind":"refocus","summary":"prioritize creator workflows"}`
	interventionReq := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/"+workspaceID+"/interventions", strings.NewReader(interventionBody))
	interventionReq.Header.Set("Content-Type", "application/json")
	interventionResp := httptest.NewRecorder()
	router.ServeHTTP(interventionResp, interventionReq)
	require.Equal(t, http.StatusAccepted, interventionResp.Code)
	require.Contains(t, interventionResp.Body.String(), `"status":"reflected"`)
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./domain/exploration -run TestV1WorkspaceRunProjectionInterventionLifecycle -v`
Expected: FAIL because route wiring and lifecycle payloads are incomplete or inconsistent.

**Step 3: Write minimal implementation**

Update the existing v1 handlers and router registration so the test can create a workspace, run the deterministic cycle, fetch a projection, and submit an intervention without relying on the legacy exploration endpoints.

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./domain/exploration -run TestV1WorkspaceRunProjectionInterventionLifecycle -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/domain/exploration/api_test.go backend/domain/exploration/api_v1.go backend/domain/exploration/routes.go
git commit -m "test: lock v1 exploration mvp contract"
```

### Task 2: Implement deterministic run, plan, and projection state transitions

**Files:**
- Modify: `backend/domain/exploration/domain.go`
- Modify: `backend/domain/exploration/runtime_plan.go`
- Modify: `backend/domain/exploration/runtime_tasks.go`
- Modify: `backend/domain/exploration/runtime_context.go`
- Modify: `backend/domain/exploration/runtime_agent.go`
- Modify: `backend/domain/exploration/exploration.go`
- Test: `backend/domain/exploration/realtime_test.go`

**Step 1: Write the failing test**

```go
func TestExecuteRuntimeCycleBuildsProjection(t *testing.T) {
	domain := NewExplorationDomain(nil, nil)
	snapshot := domain.CreateWorkspace(CreateWorkspaceReq{
		Topic:       "AI coding copilots for teams",
		OutputGoal:  "identify direction map",
		Constraints: []string{"enterprise", "developer workflow"},
	})

	domain.executeRuntimeCycle(snapshot.Exploration, "manual")

	state, ok := domain.GetRuntimeState(snapshot.Exploration.ID)
	require.True(t, ok)
	require.NotEmpty(t, state.Runs)
	require.Equal(t, RunStatusCompleted, state.Runs[len(state.Runs)-1].Status)
	require.NotEmpty(t, state.PlanSteps)
	require.NotEmpty(t, state.Mutations)

	projection := domain.buildProjectionResponse(snapshot)
	require.NotEmpty(t, projection.Projection.Map.Nodes)
	require.NotEmpty(t, projection.Projection.RecentChanges)
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./domain/exploration -run TestExecuteRuntimeCycleBuildsProjection -v`
Expected: FAIL because the runtime cycle does not yet guarantee deterministic plan/task/mutation/projection outputs for MVP.

**Step 3: Write minimal implementation**

Implement a deterministic runtime pipeline with explicit state transitions:

```go
func (d *ExplorationDomain) executeRuntimeCycle(exploration Exploration, trigger string) {
	run := d.startRun(exploration.ID, trigger)
	plan := d.activatePlan(run)
	steps := d.planDeterministicSteps(exploration, plan)
	tasks := d.dispatchPlanSteps(run, steps)
	results := d.executeDeterministicTasks(exploration, run, tasks)
	d.integrateTaskResults(exploration.ID, run, plan, steps, tasks, results)
	d.completeRun(exploration.ID, run.ID)
}
```

The implementation should emit:

- 2-3 initial direction branches on first run
- supporting evidence and open questions
- a focus branch and recent change summary
- traceable mutation events attached to the run and tasks

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./domain/exploration -run TestExecuteRuntimeCycleBuildsProjection -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/domain/exploration/domain.go backend/domain/exploration/runtime_plan.go backend/domain/exploration/runtime_tasks.go backend/domain/exploration/runtime_context.go backend/domain/exploration/runtime_agent.go backend/domain/exploration/exploration.go backend/domain/exploration/realtime_test.go
git commit -m "feat: add deterministic exploration runtime"
```

### Task 3: Implement intervention lifecycle and replanning behavior

**Files:**
- Modify: `backend/domain/exploration/api_v1.go`
- Modify: `backend/domain/exploration/runtime_plan.go`
- Modify: `backend/domain/exploration/workspace_management.go`
- Modify: `backend/domain/exploration/realtime.go`
- Test: `backend/domain/exploration/realtime_test.go`
- Test: `backend/domain/exploration/api_test.go`

**Step 1: Write the failing test**

```go
func TestInterventionTriggersReplanAndReflection(t *testing.T) {
	domain := NewExplorationDomain(nil, nil)
	snapshot := domain.CreateWorkspace(CreateWorkspaceReq{
		Topic:       "AI finance assistant",
		OutputGoal:  "map product directions",
		Constraints: []string{"consumer trust"},
	})
	domain.executeRuntimeCycle(snapshot.Exploration, "manual")

	view := domain.storeInterventionRecord(snapshot.Exploration.ID, CreateInterventionRequest{
		Kind:    "refocus",
		Summary: "increase emphasis on explainability",
	})
	domain.persistV1Intervention(view)

	_, mutations, ok := domain.ApplyIntervention(snapshot.Exploration.ID, mapInterventionReq(CreateInterventionRequest{
		Kind:    "refocus",
		Summary: "increase emphasis on explainability",
	}, snapshot.Exploration.ID))
	require.True(t, ok)
	state, _ := domain.GetRuntimeState(snapshot.Exploration.ID)
	updated := domain.advanceInterventionByRuntimeEvent(snapshot.Exploration.ID, view.ID, state, mutations)

	require.Equal(t, InterventionReflected, updated.Status)
	require.NotEmpty(t, updated.Effects)
	require.NotEmpty(t, state.Plans)
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./domain/exploration -run TestInterventionTriggersReplanAndReflection -v`
Expected: FAIL because intervention absorption, replanning, and projection reflection are not fully connected.

**Step 3: Write minimal implementation**

Implement the intervention path so it:

- records `received`
- updates to `absorbed` when bound to the current or next run
- supersedes or invalidates pending plan work
- creates a new plan version with a changed focus
- updates projection/intervention effects and advances to `reflected`

Use explicit helper functions instead of embedding lifecycle logic directly in handlers.

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./domain/exploration -run TestInterventionTriggersReplanAndReflection -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/domain/exploration/api_v1.go backend/domain/exploration/runtime_plan.go backend/domain/exploration/workspace_management.go backend/domain/exploration/realtime.go backend/domain/exploration/realtime_test.go backend/domain/exploration/api_test.go
git commit -m "feat: add intervention replanning lifecycle"
```

### Task 4: Switch the frontend to the v1 workspace and projection loop

**Files:**
- Modify: `frontend/src/lib/explorationApi.ts`
- Modify: `frontend/src/lib/workbench.ts`
- Modify: `frontend/src/types/api.ts`
- Modify: `frontend/src/types/exploration.ts`
- Modify: `frontend/src/App.tsx`
- Test: `frontend/src/lib/explorationApi.test.ts`
- Test: `frontend/src/lib/workbench.test.ts`

**Step 1: Write the failing test**

```ts
it('creates a workspace and maps projection data into the workbench payload', async () => {
  vi.spyOn(global, 'fetch')
    .mockResolvedValueOnce(new Response(JSON.stringify({
      workspace: { id: 'ws_1', topic: 'AI sales copilot', goal: 'map opportunities', constraints: ['b2b'] }
    }), { status: 201 }))
    .mockResolvedValueOnce(new Response(JSON.stringify({
      run: { id: 'run_1', status: 'completed' }
    }), { status: 202 }))
    .mockResolvedValueOnce(new Response(JSON.stringify({
      workspace: { id: 'ws_1', topic: 'AI sales copilot', goal: 'map opportunities', constraints: ['b2b'] }
    }), { status: 200 }))
    .mockResolvedValueOnce(new Response(JSON.stringify({
      projection: {
        workspace_id: 'ws_1',
        map: { nodes: [{ id: 'dir_1', type: 'opportunity', title: 'SMB outbound' }], edges: [] },
        run_summary: { run_id: 'run_1', focus: 'dir_1', status: 'completed' },
        recent_changes: [{ summary: 'Created initial direction map' }]
      }
    }), { status: 200 }))

  const result = await createExploration({
    topic: 'AI sales copilot',
    outputGoal: 'map opportunities',
    constraints: 'b2b',
  })

  expect(result.code).toBe(200)
  expect(result.data.exploration.id).toBe('ws_1')
  expect(result.data.exploration.runs[0]?.id).toBe('run_1')
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- --run frontend/src/lib/explorationApi.test.ts`
Expected: FAIL because create flow still treats workspace creation and projection loading as separate mock-first paths.

**Step 3: Write minimal implementation**

Update the frontend API layer to:

- create the workspace via v1 API
- immediately trigger the initial run
- load the workspace and projection snapshots
- map projection nodes and edges into the existing workbench shape
- keep mock fallback only for explicit failure paths

**Step 4: Run test to verify it passes**

Run: `cd frontend && npm test -- --run frontend/src/lib/explorationApi.test.ts frontend/src/lib/workbench.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/lib/explorationApi.ts frontend/src/lib/workbench.ts frontend/src/types/api.ts frontend/src/types/exploration.ts frontend/src/App.tsx frontend/src/lib/explorationApi.test.ts frontend/src/lib/workbench.test.ts
git commit -m "feat: wire frontend to v1 workspace projection flow"
```

### Task 5: Add intervention UX and verify the end-to-end MVP

**Files:**
- Modify: `frontend/src/components/SidebarPanel.tsx`
- Modify: `frontend/src/components/WorkbenchColumns.tsx`
- Modify: `frontend/src/components/WorkspaceManager.tsx`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/App.test.tsx`
- Modify: `frontend/README.md`
- Test: `frontend/src/components/App.test.tsx`

**Step 1: Write the failing test**

```ts
it('submits an intervention and refreshes the reflected workspace state', async () => {
  render(<App />)

  await userEvent.type(screen.getByLabelText(/topic/i), 'AI wellness coach')
  await userEvent.type(screen.getByLabelText(/output goal/i), 'find promising directions')
  await userEvent.click(screen.getByRole('button', { name: /launch/i }))

  await screen.findByText(/direction map/i)
  await userEvent.type(screen.getByLabelText(/intervention/i), 'focus on retention loops')
  await userEvent.click(screen.getByRole('button', { name: /submit intervention/i }))

  await screen.findByText(/reflected/i)
  expect(screen.getByText(/focus on retention loops/i)).toBeInTheDocument()
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- --run frontend/src/components/App.test.tsx`
Expected: FAIL because the current UI does not expose a complete intervention submission and reflected-status flow against the v1 API.

**Step 3: Write minimal implementation**

Add intervention UX that:

- accepts a high-level direction change request
- posts to the v1 intervention endpoint
- refreshes intervention and projection state
- surfaces reflected status and recent directional change in the workbench

Update the frontend README with the backend/frontend dev flow needed to run the MVP.

**Step 4: Run test to verify it passes**

Run: `cd frontend && npm test -- --run frontend/src/components/App.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/components/SidebarPanel.tsx frontend/src/components/WorkbenchColumns.tsx frontend/src/components/WorkspaceManager.tsx frontend/src/App.tsx frontend/src/components/App.test.tsx frontend/README.md
git commit -m "feat: add intervention flow to workbench"
```

### Task 6: Full verification before completion

**Files:**
- Modify: `docs/plans/2026-03-15-idea-factory-mvp.md`
- Test: `backend/domain/exploration/...`
- Test: `frontend/src/...`

**Step 1: Run backend test suite**

Run: `cd backend && go test ./...`
Expected: PASS

**Step 2: Run frontend test suite**

Run: `cd frontend && npm test -- --run`
Expected: PASS

**Step 3: Run frontend production checks**

Run: `cd frontend && npm run build && npm run lint`
Expected: PASS

**Step 4: Update plan status notes if needed**

Record any deviations or follow-up items directly in this plan document if implementation required a scoped change from the original design.

**Step 5: Commit**

```bash
git add docs/plans/2026-03-15-idea-factory-mvp.md
git commit -m "docs: finalize idea factory mvp implementation plan"
```
