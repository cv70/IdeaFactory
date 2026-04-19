package exploration

import (
	"backend/agents"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino-examples/adk/multiagent/deep/utils"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func (d *ExplorationDomain) initializeRuntimeState(session ExplorationSession, source string) {
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
		state.Mutations = append(state.Mutations, MutationEvent{
			ID:          mutationID(session.ID),
			WorkspaceID: session.ID,
			Kind:        string(MutationKindRunCreated),
			Run:         &GenerationRun{ID: runID},
			CreatedAt:   now.UnixMilli(),
		})
	})
	if skip {
		return
	}
	d.runSingleAgentPass(session.ID)
	d.withWorkspaceState(session.ID, func(state *RuntimeWorkspaceState) {
		if len(state.Runs) > 0 {
			state.Runs[0].Status = RunStatusCompleted
			state.Runs[0].EndedAt = time.Now().UnixMilli()
		}
	})
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
			AgentTasks:         append([]AgentTask{}, state.AgentTasks...),
			Results:            append([]AgentTaskResultSummary{}, state.Results...),
			Events:             append([]AgentRunEvent{}, state.Events...),
			Turns:              append([]RunTurn{}, state.Turns...),
			Checkpoints:        append([]RunCheckpoint{}, state.Checkpoints...),
			Mutations:          append([]MutationEvent{}, state.Mutations...),
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
		AgentTasks:         []AgentTask{},
		Results:            []AgentTaskResultSummary{},
		Events:             []AgentRunEvent{},
		Turns:              []RunTurn{},
		Checkpoints:        []RunCheckpoint{},
		Mutations:          []MutationEvent{},
		LatestReplanReason: snapshot.LatestReplanReason,
	}

	for _, run := range snapshot.Runs {
		if _, ok := runIDs[run.ID]; ok {
			out.Runs = append(out.Runs, run)
		}
	}

	taskIDs := map[string]struct{}{}
	for _, task := range snapshot.AgentTasks {
		if _, ok := runIDs[task.RunID]; !ok {
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
	for _, event := range snapshot.Events {
		if _, ok := runIDs[event.RunID]; ok {
			out.Events = append(out.Events, event)
		}
	}
	for _, turn := range snapshot.Turns {
		if _, ok := runIDs[turn.RunID]; ok {
			out.Turns = append(out.Turns, turn)
		}
	}
	for _, checkpoint := range snapshot.Checkpoints {
		if _, ok := runIDs[checkpoint.RunID]; ok {
			out.Checkpoints = append(out.Checkpoints, checkpoint)
		}
	}

	if _, ok := runIDs[snapshot.Balance.RunID]; ok {
		out.Balance = snapshot.Balance
	}
	for _, mutation := range snapshot.Mutations {
		if mutation.Run == nil {
			out.Mutations = append(out.Mutations, mutation)
			continue
		}
		if _, ok := runIDs[mutation.Run.ID]; ok {
			out.Mutations = append(out.Mutations, mutation)
		}
	}
	return out
}

func (d *ExplorationDomain) replanRuntimeState(session ExplorationSession, intervention InterventionReq) {
	now := time.Now()
	var shouldRun bool
	d.withWorkspaceState(session.ID, func(state *RuntimeWorkspaceState) {
		currentRunID := latestRunID(state.Runs)
		state.Balance = adjustBalanceForIntent(state.Balance, intervention.Note, now)
		state.ReplanReason = fmt.Sprintf("%s:%s", intervention.Type, strings.TrimSpace(intervention.Note))
		state.Mutations = append(state.Mutations, MutationEvent{
			ID:          mutationID(session.ID),
			WorkspaceID: session.ID,
			Kind:        string(MutationKindInterventionAbsorbed),
			Run:         &GenerationRun{ID: currentRunID},
			CreatedAt:   now.UnixMilli(),
		})
		state.Mutations = append(state.Mutations, MutationEvent{
			ID:          mutationID(session.ID),
			WorkspaceID: session.ID,
			Kind:        string(MutationKindBalanceUpdated),
			CreatedAt:   now.UnixMilli(),
		})
		shouldRun = true
	})
	if !shouldRun {
		return
	}
	d.runSingleAgentPass(session.ID)
	d.persistRuntimeState(session.ID)
}

func (d *ExplorationDomain) executeRuntimeCycle(session ExplorationSession, source string) {
	now := time.Now()

	var skip bool
	d.withWorkspaceState(session.ID, func(s *RuntimeWorkspaceState) {
		skip = s.AgentRunning
	})
	if skip {
		return
	}

	d.withWorkspaceState(session.ID, func(state *RuntimeWorkspaceState) {
		if len(state.Runs) == 0 {
			d.startRuntimeRunLocked(session, source, now, state)
		} else {
			d.startRuntimeRunLocked(session, source, now, state)
		}
	})
	d.runSingleAgentPass(session.ID)
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
		state.AgentTasks = append([]AgentTask{}, snapshot.AgentTasks...)
		state.Results = append([]AgentTaskResultSummary{}, snapshot.Results...)
		state.Events = append([]AgentRunEvent{}, snapshot.Events...)
		state.Turns = append([]RunTurn{}, snapshot.Turns...)
		state.Checkpoints = append([]RunCheckpoint{}, snapshot.Checkpoints...)
		state.Balance = snapshot.Balance
		state.ReplanReason = snapshot.LatestReplanReason
	})
	return true
}

func (d *ExplorationDomain) startRuntimeRunLocked(session ExplorationSession, source string, now time.Time, state *RuntimeWorkspaceState) {
	runID := fmt.Sprintf("run-%s-%d", session.ID, now.UnixNano())
	run := Run{
		ID:          runID,
		WorkspaceID: session.ID,
		Source:      source,
		Status:      RunStatusRunning,
		StartedAt:   now.UnixMilli(),
	}
	state.Runs = append(state.Runs, run)

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
		Kind:        string(MutationKindRunCreated),
		Run:         &GenerationRun{ID: runID},
		CreatedAt:   now.UnixMilli(),
	})
}

// runAgentCycle drives the single-pass MainAgent execution loop for a workspace in a background goroutine.
// It performs one runtime pass, persists state, and sets AgentRunning=false on completion or panic.
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
				Kind:        string(MutationKindRunFailed),
				Run:         &GenerationRun{ID: currentRunID},
				CreatedAt:   time.Now().UnixMilli(),
			}})
		}
	}()

	d.runSingleAgentPass(workspaceID)
	d.persistRuntimeState(workspaceID)

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
		Kind:        string(MutationKindRunCompleted),
		Run:         &GenerationRun{ID: completedRunID},
		CreatedAt:   time.Now().UnixMilli(),
	}})
	// Schedule the next run (respects pause state, MaxRuns, and IntervalMs).
	// Not called on the panic path — the defer handler exits before reaching here.
	d.scheduleNextRun(workspaceID)
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
	_ = ctx

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
		state.Mutations = append(state.Mutations, MutationEvent{
			ID:          mutationID(workspaceID),
			WorkspaceID: workspaceID,
			Kind:        string(MutationKindRunCreated),
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

func (d *ExplorationDomain) runSingleAgentPass(workspaceID string) {
	d.store.mu.RLock()
	session, ok := d.store.workspaces[workspaceID]
	if !ok {
		d.store.mu.RUnlock()
		return
	}
	sessionCopy := *session
	d.store.mu.RUnlock()

	var stateCopy RuntimeWorkspaceState
	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		stateCopy.Balance = state.Balance
		stateCopy.ReplanReason = state.ReplanReason
		stateCopy.Runs = append([]Run{}, state.Runs...)
		stateCopy.Turns = append([]RunTurn{}, state.Turns...)
	})

	cycleResult, runErr := d.runMainAgentCycle(context.Background(), workspaceID, &sessionCopy, &stateCopy)

	now := time.Now().UnixMilli()
	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		summary := cycleResult.Summary
		if summary == "" {
			summary = "main agent completed without graph changes"
		}
		taskID := fmt.Sprintf("main-agent-%s-%d", workspaceID, now)
		isSuccess := runErr == nil
		subAgent := cycleResult.LeadActor
		if strings.TrimSpace(subAgent) == "" {
			subAgent = string(RuntimeActorMainAgent)
		}
		if subAgent != string(RuntimeActorMainAgent) {
			summary = fmt.Sprintf("%s led this run: %s", subAgent, summary)
		}
		if runErr != nil {
			summary = runErr.Error()
		}
		turnIndex := cycleResult.TurnIndex
		if turnIndex <= 0 {
			turnIndex = nextRunTurnIndex(latestRunID(state.Runs), state.Turns)
		}
		turnID := cycleResult.TurnID
		if strings.TrimSpace(turnID) == "" {
			turnID = fmt.Sprintf("turn-%s-%02d", latestRunID(state.Runs), turnIndex)
		}
		resumeCursor := cycleResult.ResumeCursor
		if strings.TrimSpace(resumeCursor) == "" {
			resumeCursor = turnID
		}
		turn := RunTurn{
			ID:           turnID,
			WorkspaceID:  workspaceID,
			RunID:        latestRunID(state.Runs),
			TurnIndex:    turnIndex,
			Status:       RunTurnStatusCompleted,
			StartedAt:    cycleResult.TurnStartedAt,
			FinishedAt:   now,
			Summary:      summary,
			LeadActor:    subAgent,
			Timeline:     append([]string{}, cycleResult.Timeline...),
			ResumeCursor: resumeCursor,
		}
		if turn.StartedAt == 0 {
			turn.StartedAt = now
		}
		if runErr != nil {
			turn.Status = RunTurnStatusFailed
		}
		state.Turns = append(state.Turns, turn)
		checkpointID := cycleResult.CheckpointID
		if strings.TrimSpace(checkpointID) == "" && turn.Status == RunTurnStatusCompleted {
			checkpointID = fmt.Sprintf("checkpoint-%s-%02d", latestRunID(state.Runs), turnIndex)
		}
		if checkpointID != "" {
			checkpointAt := cycleResult.CheckpointAt
			if checkpointAt == 0 {
				checkpointAt = now
			}
			state.Checkpoints = append(state.Checkpoints, RunCheckpoint{
				ID:           checkpointID,
				WorkspaceID:  workspaceID,
				RunID:        latestRunID(state.Runs),
				TurnID:       turnID,
				ResumeCursor: resumeCursor,
				Reason:       cycleResult.CheckpointReason,
				CreatedAt:    checkpointAt,
			})
		}
		state.AgentTasks = append(state.AgentTasks, AgentTask{
			ID:          taskID,
			WorkspaceID: workspaceID,
			RunID:       latestRunID(state.Runs),
			SubAgent:    subAgent,
			Goal:        MainAgentCycleGoal,
			Status:      RuntimeTaskDone,
			UpdatedAt:   now,
		})
		state.Results = append(state.Results, AgentTaskResultSummary{
			TaskID:    taskID,
			Summary:   summary,
			Timeline:  cycleResult.Timeline,
			IsSuccess: isSuccess,
			UpdatedAt: now,
		})
	})
	d.persistAgentRunEvents(cycleResult.Events)
}

type mainAgentCycleResult struct {
	Summary          string
	LeadActor        string
	Timeline         []string
	Events           []AgentRunEvent
	TurnID           string
	TurnIndex        int
	TurnStartedAt    int64
	CheckpointID     string
	CheckpointAt     int64
	CheckpointReason string
	ResumeCursor     string
}

func (d *ExplorationDomain) runMainAgentCycle(ctx context.Context, workspaceID string, session *ExplorationSession, state *RuntimeWorkspaceState) (mainAgentCycleResult, error) {
	if d.DeepAgent == nil {
		return mainAgentCycleResult{Summary: "main agent completed without graph changes"}, nil
	}
	iter := d.DeepAgent.Run(ctx, &adk.AgentInput{
		Messages: []adk.Message{
			schema.UserMessage(d.buildMainAgentGraphPrompt(session, state)),
		},
	})

	lastAssistantMessage := ""
	agentActivity := map[string]int{}
	lastAgentActivityAt := map[string]int{}
	activityIndex := 0
	timeline := make([]string, 0, 4)
	events := make([]AgentRunEvent, 0, 8)
	rootAgent := d.DeepAgent.Name(ctx)
	runID := latestRunID(state.Runs)
	turnIndex := nextRunTurnIndex(runID, state.Turns)
	turnID := fmt.Sprintf("turn-%s-%02d", runID, turnIndex)
	turnStartedAt := time.Now().UnixMilli()
	events = append(events, AgentRunEvent{
		ID:          fmt.Sprintf("event-%s-%d", workspaceID, time.Now().UnixNano()),
		WorkspaceID: workspaceID,
		RunID:       runID,
		RootAgent:   rootAgent,
		EventType:   "turn_started",
		Actor:       string(RuntimeActorMainAgent),
		Summary:     fmt.Sprintf("turn %d started", turnIndex),
		Payload: map[string]any{
			"turn_id":    turnID,
			"turn_index": turnIndex,
		},
		CreatedAt: turnStartedAt,
	})
	for event, ok := iter.Next(); ok; event, ok = iter.Next() {
		if event == nil {
			continue
		}
		if event.Err != nil {
			errAt := time.Now().UnixMilli()
			events = append(events, AgentRunEvent{
				ID:          fmt.Sprintf("event-%s-%d", workspaceID, time.Now().UnixNano()),
				WorkspaceID: workspaceID,
				RunID:       runID,
				RootAgent:   rootAgent,
				EventType:   "turn_failed",
				Actor:       string(RuntimeActorMainAgent),
				Summary:     fmt.Sprintf("turn %d failed", turnIndex),
				Payload: map[string]any{
					"turn_id":    turnID,
					"turn_index": turnIndex,
				},
				CreatedAt: errAt,
			})
			events = append(events, AgentRunEvent{
				ID:          fmt.Sprintf("event-%s-%d", workspaceID, time.Now().UnixNano()),
				WorkspaceID: workspaceID,
				RunID:       runID,
				RootAgent:   rootAgent,
				EventType:   "run_error",
				Actor:       string(RuntimeActorMainAgent),
				Summary:     event.Err.Error(),
				Payload: map[string]any{
					"turn_id":    turnID,
					"turn_index": turnIndex,
				},
				CreatedAt:   errAt,
			})
			return mainAgentCycleResult{
				Events:        events,
				TurnID:        turnID,
				TurnIndex:     turnIndex,
				TurnStartedAt: turnStartedAt,
			}, event.Err
		}
		if event.Output == nil {
			continue
		}
		if runtimeEvent, ok := event.Output.CustomizedOutput.(agents.RuntimeEvent); ok {
			payload := runtimeEvent.Payload
			if runtimeEvent.EventType == agents.RuntimeEventAgentStart {
				if payload == nil {
					payload = map[string]any{}
				}
				if len(state.Runs) > 0 {
					payload["source"] = state.Runs[len(state.Runs)-1].Source
				}
			}
			events = append(events, AgentRunEvent{
				ID:          fmt.Sprintf("event-%s-%d", workspaceID, time.Now().UnixNano()),
				WorkspaceID: workspaceID,
				RunID:       runID,
				RootAgent:   rootAgent,
				EventType:   runtimeEvent.EventType,
				Actor:       runtimeEvent.Actor,
				Target:      runtimeEvent.Target,
				Summary:     runtimeEvent.Summary,
				Payload:     payload,
				CreatedAt:   time.Now().UnixMilli(),
			})
		}
		if event.Output.MessageOutput == nil {
			continue
		}
		activityIndex++
		if actor := normalizeLeadAgentName(event.AgentName); actor != "" {
			agentActivity[actor]++
			lastAgentActivityAt[actor] = activityIndex
			timeline = appendTimelineStep(timeline, actor)
		}
		if event.Output.MessageOutput.Role != schema.Assistant {
			continue
		}
		msg, err := event.Output.MessageOutput.GetMessage()
		if err != nil {
			return mainAgentCycleResult{}, err
		}
		if msg != nil && strings.TrimSpace(msg.Content) != "" {
			lastAssistantMessage = msg.Content
		}
	}
	result := mainAgentCycleResult{
		Summary:       parseMainAgentSummary(lastAssistantMessage),
		LeadActor:     selectLeadAgent(agentActivity, lastAgentActivityAt),
		Timeline:      appendSummaryStep(timeline, lastAssistantMessage),
		Events:        events,
		TurnID:        turnID,
		TurnIndex:     turnIndex,
		TurnStartedAt: turnStartedAt,
	}
	completedAt := time.Now().UnixMilli()
	result.Events = append(result.Events, AgentRunEvent{
		ID:          fmt.Sprintf("event-%s-%d", workspaceID, time.Now().UnixNano()),
		WorkspaceID: workspaceID,
		RunID:       runID,
		RootAgent:   rootAgent,
		EventType:   "turn_completed",
		Actor:       string(RuntimeActorMainAgent),
		Summary:     fmt.Sprintf("turn %d completed", turnIndex),
		Payload: map[string]any{
			"turn_id":    turnID,
			"turn_index": turnIndex,
			"lead_actor": result.LeadActor,
			"timeline":   result.Timeline,
			"summary":    result.Summary,
		},
		CreatedAt: completedAt,
	})
	checkpointID := fmt.Sprintf("checkpoint-%s-%02d", runID, turnIndex)
	resumeCursor := turnID
	result.CheckpointID = checkpointID
	result.CheckpointAt = completedAt
	result.CheckpointReason = "run_turn_completed"
	result.ResumeCursor = resumeCursor
	result.Events = append(result.Events, AgentRunEvent{
		ID:          fmt.Sprintf("event-%s-%d", workspaceID, time.Now().UnixNano()),
		WorkspaceID: workspaceID,
		RunID:       runID,
		RootAgent:   rootAgent,
		EventType:   "run_checkpoint",
		Actor:       string(RuntimeActorMainAgent),
		Summary:     fmt.Sprintf("checkpoint created for turn %d", turnIndex),
		Payload: map[string]any{
			"checkpoint_id": checkpointID,
			"turn_id":       turnID,
			"turn_index":    turnIndex,
			"resume_cursor": resumeCursor,
			"reason":        result.CheckpointReason,
		},
		CreatedAt: completedAt,
	})
	result.Events = append(result.Events, AgentRunEvent{
		ID:          fmt.Sprintf("event-%s-%d", workspaceID, time.Now().UnixNano()),
		WorkspaceID: workspaceID,
		RunID:       runID,
		RootAgent:   rootAgent,
		EventType:   "run_summary",
		Actor:       firstNonEmpty(result.LeadActor, string(RuntimeActorMainAgent)),
		Summary:     result.Summary,
		Payload: map[string]any{
			"lead_actor": result.LeadActor,
			"timeline":   result.Timeline,
			"turn_id":    turnID,
			"turn_index": turnIndex,
			"checkpoint": checkpointID,
		},
		CreatedAt: completedAt,
	})
	return result, nil
}

func nextRunTurnIndex(runID string, turns []RunTurn) int {
	next := 1
	for _, turn := range turns {
		if turn.RunID != runID {
			continue
		}
		if turn.TurnIndex >= next {
			next = turn.TurnIndex + 1
		}
	}
	return next
}

func appendTimelineStep(timeline []string, step string) []string {
	step = strings.TrimSpace(step)
	if step == "" {
		return timeline
	}
	if len(timeline) > 0 && timeline[len(timeline)-1] == step {
		return timeline
	}
	return append(timeline, step)
}

func appendSummaryStep(timeline []string, lastAssistantMessage string) []string {
	if strings.Contains(strings.TrimSpace(lastAssistantMessage), "SUMMARY:") {
		return appendTimelineStep(timeline, "SUMMARY")
	}
	return timeline
}

func normalizeLeadAgentName(name string) string {
	name = strings.TrimSpace(name)
	switch name {
	case "", "exploration-main-agent", string(RuntimeActorMainAgent):
		return ""
	default:
		return name
	}
}

func selectLeadAgent(activity map[string]int, lastSeen map[string]int) string {
	bestName := ""
	bestCount := 0
	bestLastSeen := -1
	for name, count := range activity {
		if count > bestCount || (count == bestCount && lastSeen[name] > bestLastSeen) {
			bestName = name
			bestCount = count
			bestLastSeen = lastSeen[name]
		}
	}
	return bestName
}

func parseMainAgentSummary(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "main agent completed without graph changes"
	}

	lines := strings.Split(trimmed, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "SUMMARY:") {
			continue
		}
		summary := strings.TrimSpace(strings.TrimPrefix(line, "SUMMARY:"))
		if summary != "" {
			return summary
		}
		break
	}

	// Keep a readable fallback even when the agent misses the format contract.
	firstLine := strings.TrimSpace(strings.Split(utils.RepairJSON(trimmed), "\n")[0])
	if firstLine != "" {
		return firstLine
	}
	return "main agent completed without graph changes"
}

func latestRunID(runs []Run) string {
	if len(runs) == 0 {
		return ""
	}
	return runs[len(runs)-1].ID
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

// initializeWorkspaceGraph is a compatibility helper for tests and legacy bootstrap paths.
// It uses the deterministic generator to append a one-time graph batch without involving planner state.
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
		stateCopy.Balance = state.Balance
		stateCopy.AgentTasks = append([]AgentTask{}, state.AgentTasks...)
		stateCopy.Results = append([]AgentTaskResultSummary{}, state.Results...)
		stateCopy.ReplanReason = state.ReplanReason
	})

	newNodes, newEdges := NewDeterministicPlanner().GenerateNodesForCycle(ctx, &sessionCopy, &stateCopy)
	_, _ = d.appendGraphBatch(workspaceID, newNodes, newEdges, string(MutationSourceRuntime))
}

// scheduleNextRun inspects workspace state and, if scheduling is warranted,
// launches a goroutine that will call triggerRun after IntervalMs delay.
// It returns immediately (non-blocking). Never call with runtime.mu or store.mu held.
func (d *ExplorationDomain) scheduleNextRun(workspaceID string) {
	// Step 1: DB paused check (outside any lock).
	if d.DB != nil {
		if workspaceDBID, parseErr := parseWorkspaceID(workspaceID); parseErr == nil {
			dbState, err := d.DB.GetWorkspaceState(workspaceDBID)
			if err == nil && dbState != nil && dbState.PausedAt != nil {
				return
			}
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
		defer cancel()
		select {
		case <-time.After(time.Duration(intervalMs) * time.Millisecond):
			// Re-check paused state to close the narrow race where a pause arrived
			// after the DB check above but before the context was stored.
			if d.DB != nil {
				if workspaceDBID, parseErr := parseWorkspaceID(workspaceID); parseErr == nil {
					dbState, err := d.DB.GetWorkspaceState(workspaceDBID)
					if err == nil && dbState != nil && dbState.PausedAt != nil {
						d.withWorkspaceState(workspaceID, func(s *RuntimeWorkspaceState) {
							s.cancelScheduler = nil
						})
						return
					}
				}
			}
			d.withWorkspaceState(workspaceID, func(s *RuntimeWorkspaceState) {
				s.cancelScheduler = nil
			})
			d.triggerRun(ctx, workspaceID, string(RunSourceAuto))
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
