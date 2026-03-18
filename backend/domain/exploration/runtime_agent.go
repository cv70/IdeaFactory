package exploration

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (d *ExplorationDomain) initializeRuntimeState(session ExplorationSession, source string) {
	ctx := context.Background()
	now := time.Now()
	var skip bool
	var newNodes []Node
	var newEdges []Edge
	var stateCopy RuntimeWorkspaceState
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
		// Execute the first step synchronously so callers immediately see tasks.
		d.executeNextTodoStepLocked(session.ID, now, state)
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
		stateCopy.Plans = append([]ExecutionPlan{}, state.Plans...)
		stateCopy.PlanSteps = append([]PlanStep{}, state.PlanSteps...)
		stateCopy.Balance = state.Balance
		stateCopy.AgentTasks = append([]AgentTask{}, state.AgentTasks...)
		stateCopy.Results = append([]AgentTaskResultSummary{}, state.Results...)
		stateCopy.ReplanReason = state.ReplanReason
	})
	if skip {
		return
	}
	newNodes, newEdges = d.planner.GenerateNodesForCycle(ctx, &session, &stateCopy)
	d.applyGeneratedNodes(session.ID, newNodes, newEdges)
	d.persistRuntimeState(session.ID)
}

func (d *ExplorationDomain) GetRuntimeState(workspaceID string) (RuntimeStateSnapshot, bool) {
	return d.QueryRuntimeState(workspaceID, RuntimeStateQuery{})
}

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

func filterRuntimeSnapshot(snapshot RuntimeStateSnapshot, query RuntimeStateQuery) RuntimeStateSnapshot {
	if query.RunID != "" {
		snapshot = filterRuntimeByRunIDs(snapshot, map[string]struct{}{query.RunID: {}})
	}
	if query.LatestRuns > 0 && len(snapshot.Runs) > query.LatestRuns {
		keepRuns := snapshot.Runs[len(snapshot.Runs)-query.LatestRuns:]
		keepIDs := map[string]struct{}{}
		for _, run := range keepRuns {
			keepIDs[run.ID] = struct{}{}
		}
		snapshot = filterRuntimeByRunIDs(snapshot, keepIDs)
	}
	return snapshot
}

func filterRuntimeByRunIDs(snapshot RuntimeStateSnapshot, runIDs map[string]struct{}) RuntimeStateSnapshot {
	out := RuntimeStateSnapshot{
		Runs:               []Run{},
		Plans:              []ExecutionPlan{},
		PlanSteps:          []PlanStep{},
		AgentTasks:         []AgentTask{},
		Results:            []AgentTaskResultSummary{},
		LatestReplanReason: snapshot.LatestReplanReason,
	}

	for _, run := range snapshot.Runs {
		if _, ok := runIDs[run.ID]; ok {
			out.Runs = append(out.Runs, run)
		}
	}

	planIDs := map[string]struct{}{}
	for _, plan := range snapshot.Plans {
		if _, ok := runIDs[plan.RunID]; !ok {
			continue
		}
		out.Plans = append(out.Plans, plan)
		planIDs[plan.ID] = struct{}{}
	}

	for _, step := range snapshot.PlanSteps {
		if _, ok := runIDs[step.RunID]; !ok {
			continue
		}
		if _, ok := planIDs[step.PlanID]; !ok {
			continue
		}
		out.PlanSteps = append(out.PlanSteps, step)
	}

	taskIDs := map[string]struct{}{}
	for _, task := range snapshot.AgentTasks {
		if _, ok := runIDs[task.RunID]; !ok {
			continue
		}
		if _, ok := planIDs[task.PlanID]; !ok {
			continue
		}
		out.AgentTasks = append(out.AgentTasks, task)
		taskIDs[task.ID] = struct{}{}
	}

	for _, result := range snapshot.Results {
		if _, ok := taskIDs[result.TaskID]; ok {
			out.Results = append(out.Results, result)
		}
	}

	if _, ok := runIDs[snapshot.Balance.RunID]; ok {
		out.Balance = snapshot.Balance
	}
	return out
}

func (d *ExplorationDomain) replanRuntimeState(session ExplorationSession, intervention InterventionReq) {
	ctx := context.Background()
	now := time.Now()
	var skip bool
	var newNodes []Node
	var newEdges []Edge
	var stateCopy RuntimeWorkspaceState
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
		stateCopy.Plans = append([]ExecutionPlan{}, state.Plans...)
		stateCopy.PlanSteps = append([]PlanStep{}, state.PlanSteps...)
		stateCopy.Balance = state.Balance
		stateCopy.AgentTasks = append([]AgentTask{}, state.AgentTasks...)
		stateCopy.Results = append([]AgentTaskResultSummary{}, state.Results...)
		stateCopy.ReplanReason = state.ReplanReason
	})
	if skip {
		return
	}
	newNodes, newEdges = d.planner.GenerateNodesForCycle(ctx, &session, &stateCopy)
	d.applyGeneratedNodes(session.ID, newNodes, newEdges)
	d.persistRuntimeState(session.ID)
}

func (d *ExplorationDomain) executeRuntimeCycle(session ExplorationSession, source string) {
	ctx := context.Background()
	now := time.Now()
	var newNodes []Node
	var newEdges []Edge

	var skip bool
	d.withWorkspaceState(session.ID, func(s *RuntimeWorkspaceState) {
		skip = s.AgentRunning
	})
	if skip {
		return
	}

	var stateCopy RuntimeWorkspaceState
	d.withWorkspaceState(session.ID, func(state *RuntimeWorkspaceState) {
		if len(state.Runs) == 0 {
			d.startRuntimeRunLocked(ctx, session, source, now, state)
		} else {
			if !d.executeNextTodoStepLocked(session.ID, now, state) {
				d.startRuntimeRunLocked(ctx, session, source, now, state)
			}
			state.Balance.Divergence += 0.01
			if state.Balance.Divergence > 1 {
				state.Balance.Divergence = 1
			}
			state.Balance.UpdatedAt = now.UnixMilli()
		}
		stateCopy.Plans = append([]ExecutionPlan{}, state.Plans...)
		stateCopy.PlanSteps = append([]PlanStep{}, state.PlanSteps...)
		stateCopy.Balance = state.Balance
		stateCopy.AgentTasks = append([]AgentTask{}, state.AgentTasks...)
		stateCopy.Results = append([]AgentTaskResultSummary{}, state.Results...)
		stateCopy.ReplanReason = state.ReplanReason
	})
	newNodes, newEdges = d.planner.GenerateNodesForCycle(ctx, &session, &stateCopy)
	d.applyGeneratedNodes(session.ID, newNodes, newEdges)
	d.persistRuntimeState(session.ID)
}

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

// snapshotForCycle reads a consistent copy of session and runtime state for use by GenerateNodesForCycle.
// Acquires store.mu then runtime.mu separately — never both simultaneously.
// hasTodo is true if the current plan has at least one PlanStepTodo step.
func (d *ExplorationDomain) snapshotForCycle(workspaceID string) (session ExplorationSession, state RuntimeWorkspaceState, hasTodo bool) {
	// Step 1: copy session under store.mu
	d.store.mu.RLock()
	if ws, ok := d.store.workspaces[workspaceID]; ok {
		session = *ws
	}
	d.store.mu.RUnlock()

	// Step 2: copy runtime state under runtime.mu
	d.withWorkspaceState(workspaceID, func(s *RuntimeWorkspaceState) {
		state.Plans = append([]ExecutionPlan{}, s.Plans...)
		state.PlanSteps = append([]PlanStep{}, s.PlanSteps...)
		state.Balance = s.Balance
		state.AgentTasks = append([]AgentTask{}, s.AgentTasks...)
		state.Results = append([]AgentTaskResultSummary{}, s.Results...)
		state.ReplanReason = s.ReplanReason
		if len(s.Plans) > 0 {
			currentPlanID := s.Plans[len(s.Plans)-1].ID
			for _, step := range s.PlanSteps {
				if step.PlanID == currentPlanID && step.Status == PlanStepTodo {
					hasTodo = true
					break
				}
			}
		}
	})
	return
}

// markStepDoneAndCheck marks the first Todo step in the current plan as Done,
// appends an AgentTask result, and returns true if no Todo steps remain.
func (d *ExplorationDomain) markStepDoneAndCheck(workspaceID string) (allDone bool) {
	now := time.Now()
	d.withWorkspaceState(workspaceID, func(s *RuntimeWorkspaceState) {
		if len(s.Plans) == 0 {
			allDone = true
			return
		}
		currentPlan := s.Plans[len(s.Plans)-1]
		targetIdx := -1
		for i, step := range s.PlanSteps {
			if step.PlanID == currentPlan.ID && step.Status == PlanStepTodo {
				targetIdx = i
				break
			}
		}
		if targetIdx == -1 {
			allDone = true
			return
		}
		step := &s.PlanSteps[targetIdx]
		taskID := fmt.Sprintf("task-%s-%d", currentPlan.ID, step.Index)
		s.AgentTasks = append(s.AgentTasks, AgentTask{
			ID:          taskID,
			WorkspaceID: workspaceID,
			RunID:       currentPlan.RunID,
			PlanID:      currentPlan.ID,
			PlanStepID:  step.ID,
			SubAgent:    subAgentForStep(step.Index),
			Goal:        step.Desc,
			Status:      PlanStepDone,
			UpdatedAt:   now.UnixMilli(),
		})
		s.Results = append(s.Results, AgentTaskResultSummary{
			TaskID:    taskID,
			Summary:   subAgentForStep(step.Index) + " completed",
			IsSuccess: true,
			UpdatedAt: now.UnixMilli(),
		})
		step.Status = PlanStepDone
		step.UpdatedAt = now.UnixMilli()
		// Check if any Todo steps remain
		for _, st := range s.PlanSteps {
			if st.PlanID == currentPlan.ID && st.Status == PlanStepTodo {
				return // more work to do
			}
		}
		allDone = true
	})
	return
}

// runAgentCycle drives the agent execution loop for a workspace in a background goroutine.
// It reads state snapshots, calls LLMPlanner.GenerateNodesForCycle outside locks,
// applies nodes, and broadcasts via WebSocket. Sets AgentRunning=false on completion or panic.
func (d *ExplorationDomain) runAgentCycle(workspaceID string) {
	// Read current run ID before defer (needed in panic handler)
	var currentRunID string
	d.withWorkspaceState(workspaceID, func(s *RuntimeWorkspaceState) {
		if len(s.Runs) > 0 {
			currentRunID = s.Runs[len(s.Runs)-1].ID
		}
	})

	defer func() {
		if r := recover(); r != nil {
			d.withWorkspaceState(workspaceID, func(s *RuntimeWorkspaceState) {
				s.AgentRunning = false
				if len(s.Runs) > 0 && s.Runs[len(s.Runs)-1].Status != RunStatusCompleted {
					s.Runs[len(s.Runs)-1].Status = RunStatusFailed
					s.Runs[len(s.Runs)-1].EndedAt = time.Now().UnixMilli()
				}
			})
			d.broadcastMutations(workspaceID, []MutationEvent{{
				ID:          mutationID(workspaceID),
				WorkspaceID: workspaceID,
				Kind:        "run_failed",
				Run:         &GenerationRun{ID: currentRunID},
				CreatedAt:   time.Now().UnixMilli(),
			}})
		}
	}()

	totalCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	for {
		sessionCopy, stateCopy, hasTodo := d.snapshotForCycle(workspaceID)
		if !hasTodo {
			break
		}

		stepCtx, stepCancel := context.WithTimeout(totalCtx, 2*time.Minute)
		nodes, edges := d.planner.GenerateNodesForCycle(stepCtx, &sessionCopy, &stateCopy)
		stepCancel()

		d.applyGeneratedNodes(workspaceID, nodes, edges)
		allDone := d.markStepDoneAndCheck(workspaceID)
		d.persistRuntimeState(workspaceID)
		if allDone {
			break
		}
	}

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
}

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

// adjustBalanceForIntent adjusts BalanceState fields based on intent keyword scanning.
// Adjustments accumulate; all fields are clamped to [0, 1].
func adjustBalanceForIntent(prev BalanceState, intent string, now time.Time) BalanceState {
	next := prev
	next.UpdatedAt = now.UnixMilli()
	lower := strings.ToLower(intent)

	clamp := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
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

// applyGeneratedNodes adds newNodes/newEdges to the session store,
// writes node_added/edge_added mutation events, broadcasts them, and persists the session.
// Must be called while NOT holding store.mu or runtime.mu.
func (d *ExplorationDomain) applyGeneratedNodes(workspaceID string, newNodes []Node, newEdges []Edge) {
	if len(newNodes) == 0 && len(newEdges) == 0 {
		return
	}
	d.store.mu.Lock()
	current, ok := d.store.workspaces[workspaceID]
	if !ok {
		d.store.mu.Unlock()
		return
	}
	prev := *current
	current.Nodes = append(current.Nodes, newNodes...)
	current.Edges = append(current.Edges, newEdges...)
	mutations := diffMutations(prev, *current, "runtime")
	updatedCopy := *current
	d.store.mu.Unlock()

	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		state.Mutations = append(state.Mutations, mutations...)
	})
	d.broadcastMutations(workspaceID, mutations)
	d.persistWorkspace(updatedCopy)
}

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

	var stateCopy RuntimeWorkspaceState
	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		if state.Balance.WorkspaceID == "" {
			now := time.Now()
			runID := fmt.Sprintf("init-%s-%d", workspaceID, now.UnixNano())
			state.Balance = buildInitialBalance(sessionCopy, runID, now)
		}
		stateCopy.Plans = append([]ExecutionPlan{}, state.Plans...)
		stateCopy.PlanSteps = append([]PlanStep{}, state.PlanSteps...)
		stateCopy.Balance = state.Balance
		stateCopy.AgentTasks = append([]AgentTask{}, state.AgentTasks...)
		stateCopy.Results = append([]AgentTaskResultSummary{}, state.Results...)
		stateCopy.ReplanReason = state.ReplanReason
	})

	newNodes, newEdges := d.planner.GenerateNodesForCycle(ctx, &sessionCopy, &stateCopy)
	d.applyGeneratedNodes(workspaceID, newNodes, newEdges)
}
