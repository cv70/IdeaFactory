package exploration

import (
	"backend/agentools"
	"backend/agents"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func (d *ExplorationDomain) initializeRuntimeState(session ExplorationSession, source string) {
	d.runtime.mu.Lock()
	if len(d.runtime.runs[session.ID]) > 0 {
		d.runtime.mu.Unlock()
		return
	}

	now := time.Now()
	runID := fmt.Sprintf("run-%s-%d", session.ID, now.UnixNano())
	run := Run{
		ID:          runID,
		WorkspaceID: session.ID,
		Source:      source,
		Status:      RunStatusRunning,
		StartedAt:   now.UnixMilli(),
	}
	d.runtime.runs[session.ID] = []Run{run}

	balance := buildInitialBalance(session, runID, now)
	d.runtime.balance[session.ID] = balance
	d.runtime.replanReason[session.ID] = ""

	plan, steps := buildInitialPlan(d, session, runID, now)
	d.runtime.plans[session.ID] = []ExecutionPlan{plan}

	nextSteps, tasks, results := dispatchPlanSteps(session, runID, plan, steps, now)
	d.runtime.planSteps[session.ID] = nextSteps
	d.runtime.agentTasks[session.ID] = tasks
	d.runtime.results[session.ID] = results

	run.Status = RunStatusCompleted
	run.EndedAt = now.UnixMilli()
	d.runtime.runs[session.ID][0] = run
	d.runtime.mu.Unlock()
	d.persistRuntimeState(session.ID)
}

func (d *ExplorationDomain) GetRuntimeState(workspaceID string) (RuntimeStateSnapshot, bool) {
	return d.QueryRuntimeState(workspaceID, RuntimeStateQuery{})
}

func (d *ExplorationDomain) QueryRuntimeState(workspaceID string, query RuntimeStateQuery) (RuntimeStateSnapshot, bool) {
	d.runtime.mu.Lock()
	defer d.runtime.mu.Unlock()

	runs, ok := d.runtime.runs[workspaceID]
	if !ok || len(runs) == 0 {
		return RuntimeStateSnapshot{}, false
	}
	snapshot := RuntimeStateSnapshot{
		Runs:               append([]Run{}, runs...),
		Plans:              append([]ExecutionPlan{}, d.runtime.plans[workspaceID]...),
		PlanSteps:          append([]PlanStep{}, d.runtime.planSteps[workspaceID]...),
		AgentTasks:         append([]AgentTask{}, d.runtime.agentTasks[workspaceID]...),
		Results:            append([]AgentTaskResultSummary{}, d.runtime.results[workspaceID]...),
		Balance:            d.runtime.balance[workspaceID],
		LatestReplanReason: d.runtime.replanReason[workspaceID],
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
	d.runtime.mu.Lock()

	runs := d.runtime.runs[session.ID]
	if len(runs) == 0 {
		d.runtime.mu.Unlock()
		return
	}
	now := time.Now()
	currentRun := runs[len(runs)-1]

	plans := d.runtime.plans[session.ID]
	if len(plans) == 0 {
		d.runtime.mu.Unlock()
		return
	}
	currentPlan := plans[len(plans)-1]

	steps := d.runtime.planSteps[session.ID]
	for i := range steps {
		if steps[i].PlanID != currentPlan.ID {
			continue
		}
		if steps[i].Status == PlanStepTodo || steps[i].Status == PlanStepDoing {
			steps[i].Status = PlanStepSkipped
			steps[i].UpdatedAt = now.UnixMilli()
		}
	}
	d.runtime.planSteps[session.ID] = steps

	nextPlan, nextSteps := buildInitialPlan(d, session, currentRun.ID, now)
	nextPlan.Version = currentPlan.Version + 1
	d.runtime.plans[session.ID] = append(d.runtime.plans[session.ID], nextPlan)
	dispatchedSteps, tasks, results := dispatchPlanSteps(session, currentRun.ID, nextPlan, nextSteps, now)
	d.runtime.planSteps[session.ID] = append(d.runtime.planSteps[session.ID], dispatchedSteps...)
	d.runtime.agentTasks[session.ID] = append(d.runtime.agentTasks[session.ID], tasks...)
	d.runtime.results[session.ID] = append(d.runtime.results[session.ID], results...)

	d.runtime.balance[session.ID] = updateBalanceForIntervention(d.runtime.balance[session.ID], intervention, now)
	d.runtime.replanReason[session.ID] = fmt.Sprintf("%s:%s", intervention.Type, strings.TrimSpace(intervention.Note))
	d.runtime.mu.Unlock()
	d.persistRuntimeState(session.ID)
}

func updateBalanceForIntervention(prev BalanceState, intervention InterventionReq, now time.Time) BalanceState {
	next := prev
	next.UpdatedAt = now.UnixMilli()
	switch intervention.Type {
	case InterventionExpandOpportunity:
		next.Divergence += 0.08
		next.Reason = "replan for opportunity expansion"
	case InterventionShiftFocus:
		next.Research += 0.05
		next.Divergence -= 0.03
		next.Reason = "replan for focus shift"
	case InterventionAdjustIntensity:
		next.Aggression += 0.08
		next.Reason = "replan for intensity adjustment"
	case InterventionAddContext:
		next.Research += 0.1
		next.Reason = "replan after adding context"
	default:
		next.Reason = "replan after intervention"
	}
	if next.Divergence < 0 {
		next.Divergence = 0
	}
	if next.Research < 0 {
		next.Research = 0
	}
	if next.Aggression < 0 {
		next.Aggression = 0
	}
	if next.Divergence > 1 {
		next.Divergence = 1
	}
	if next.Research > 1 {
		next.Research = 1
	}
	if next.Aggression > 1 {
		next.Aggression = 1
	}
	return next
}

func (d *ExplorationDomain) executeRuntimeCycle(session ExplorationSession, source string) {
	now := time.Now()

	d.runtime.mu.Lock()

	if len(d.runtime.runs[session.ID]) == 0 {
		d.startRuntimeRunLocked(session, source, now)
		d.runtime.mu.Unlock()
		d.persistRuntimeState(session.ID)
		return
	}

	if !d.executeNextTodoStepLocked(session.ID, now) {
		d.startRuntimeRunLocked(session, source, now)
	}

	balance := d.runtime.balance[session.ID]
	balance.Divergence += 0.01
	if balance.Divergence > 1 {
		balance.Divergence = 1
	}
	balance.UpdatedAt = now.UnixMilli()
	d.runtime.balance[session.ID] = balance
	d.runtime.mu.Unlock()
	d.persistRuntimeState(session.ID)
}

func (d *ExplorationDomain) restoreRuntimeState(workspaceID string) bool {
	snapshot, ok := d.loadRuntimeState(workspaceID)
	if !ok {
		return false
	}

	d.runtime.mu.Lock()
	defer d.runtime.mu.Unlock()
	if len(d.runtime.runs[workspaceID]) > 0 {
		return true
	}
	d.runtime.runs[workspaceID] = append([]Run{}, snapshot.Runs...)
	d.runtime.plans[workspaceID] = append([]ExecutionPlan{}, snapshot.Plans...)
	d.runtime.planSteps[workspaceID] = append([]PlanStep{}, snapshot.PlanSteps...)
	d.runtime.agentTasks[workspaceID] = append([]AgentTask{}, snapshot.AgentTasks...)
	d.runtime.results[workspaceID] = append([]AgentTaskResultSummary{}, snapshot.Results...)
	d.runtime.balance[workspaceID] = snapshot.Balance
	d.runtime.replanReason[workspaceID] = snapshot.LatestReplanReason
	return true
}

func (d *ExplorationDomain) startRuntimeRunLocked(session ExplorationSession, source string, now time.Time) {
	runID := fmt.Sprintf("run-%s-%d", session.ID, now.UnixNano())
	run := Run{
		ID:          runID,
		WorkspaceID: session.ID,
		Source:      source,
		Status:      RunStatusRunning,
		StartedAt:   now.UnixMilli(),
	}
	d.runtime.runs[session.ID] = append(d.runtime.runs[session.ID], run)

	plan, steps := buildInitialPlan(d, session, runID, now)
	if plans := d.runtime.plans[session.ID]; len(plans) > 0 {
		plan.Version = plans[len(plans)-1].Version + 1
	}
	d.runtime.plans[session.ID] = append(d.runtime.plans[session.ID], plan)
	d.runtime.planSteps[session.ID] = append(d.runtime.planSteps[session.ID], steps...)

	_ = d.executeNextTodoStepLocked(session.ID, now)

	balance := buildInitialBalance(session, runID, now)
	if prev, ok := d.runtime.balance[session.ID]; ok {
		balance.Divergence = (prev.Divergence + balance.Divergence) / 2
		balance.Research = (prev.Research + balance.Research) / 2
		balance.Aggression = (prev.Aggression + balance.Aggression) / 2
	}
	d.runtime.balance[session.ID] = balance

	run.Status = RunStatusCompleted
	run.EndedAt = now.UnixMilli()
	d.runtime.runs[session.ID][len(d.runtime.runs[session.ID])-1] = run
}

func (d *ExplorationDomain) dispatchAgentTask(task AgentTask) (AgentTaskResultSummary, error) {
	runWithAgent := func(name string, agent adk.Agent) (AgentTaskResultSummary, error) {
		iter := agent.Run(context.Background(), &adk.AgentInput{
			Messages: []adk.Message{schema.UserMessage(task.Goal)},
		})

		eventCount := 0
		summary := ""
		for {
			ev, ok := iter.Next()
			if !ok {
				break
			}
			eventCount++
			if ev != nil && ev.Err != nil {
				return AgentTaskResultSummary{}, ev.Err
			}
			if ev != nil && ev.Output != nil && ev.Output.MessageOutput != nil && ev.Output.MessageOutput.Message != nil {
				content := strings.TrimSpace(ev.Output.MessageOutput.Message.Content)
				if content != "" {
					summary = content
				}
			}
		}
		if summary == "" {
			summary = fmt.Sprintf("%s executed task (%d events)", name, eventCount)
		}
		return AgentTaskResultSummary{
			TaskID:    task.ID,
			Summary:   summary,
			IsSuccess: true,
			UpdatedAt: time.Now().UnixMilli(),
		}, nil
	}

	if d.DeepAgent != nil {
		return runWithAgent("DeepAgent", d.DeepAgent)
	}

	agent := d.General
	if agent == nil {
		var err error
		agent, err = agents.NewGeneralAgent(context.Background(), d.Model)
		if err != nil || agent == nil {
			return AgentTaskResultSummary{}, fmt.Errorf("no agent available")
		}
	}
	return runWithAgent(agent.Name(context.Background()), agent)
}

func (d *ExplorationDomain) executeNextTodoStepLocked(workspaceID string, now time.Time) bool {
	plans := d.runtime.plans[workspaceID]
	if len(plans) == 0 {
		return false
	}
	currentPlan := plans[len(plans)-1]

	steps := d.runtime.planSteps[workspaceID]
	targetIndex := -1
	for i := len(steps) - 1; i >= 0; i-- {
		if steps[i].PlanID == currentPlan.ID && steps[i].Status == PlanStepTodo {
			targetIndex = i
		}
	}
	if targetIndex == -1 {
		return false
	}

	step := steps[targetIndex]
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
		Status:      PlanStepDoing,
		UpdatedAt:   now.UnixMilli(),
	}

	wrapper := agentools.NewRuntimeToolWrapper()
	resp, err := wrapper.NormalizeToolCall("read_file", "{path:'runtime.md'}")
	if err != nil {
		step.Status = PlanStepFailed
		task.Status = PlanStepFailed
		d.runtime.results[workspaceID] = append(d.runtime.results[workspaceID], AgentTaskResultSummary{
			TaskID:    task.ID,
			Summary:   "runtime tool normalization failed",
			IsSuccess: false,
			UpdatedAt: now.UnixMilli(),
		})
	} else {
		_ = resp
		result, runErr := d.dispatchAgentTask(task)
		if runErr != nil {
			step.Status = PlanStepFailed
			task.Status = PlanStepFailed
			d.runtime.results[workspaceID] = append(d.runtime.results[workspaceID], AgentTaskResultSummary{
				TaskID:    task.ID,
				Summary:   task.SubAgent + " adapter execution failed",
				IsSuccess: false,
				UpdatedAt: now.UnixMilli(),
			})
		} else {
			step.Status = PlanStepDone
			task.Status = PlanStepDone
			result.UpdatedAt = now.UnixMilli()
			d.runtime.results[workspaceID] = append(d.runtime.results[workspaceID], result)
		}
	}

	step.UpdatedAt = now.UnixMilli()
	task.UpdatedAt = now.UnixMilli()
	steps[targetIndex] = step
	d.runtime.planSteps[workspaceID] = steps
	d.runtime.agentTasks[workspaceID] = append(d.runtime.agentTasks[workspaceID], task)
	return true
}
