package exploration

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func (d *ExplorationDomain) ApiV1CreateRun(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	var req CreateRunRequest
	if c.ContentType() == "application/json" {
		if err := c.ShouldBindJSON(&req); err != nil {
			writeV1Error(c, http.StatusBadRequest, "invalid_argument", "failed to parse create run request")
			return
		}
	}
	snapshot, ok := d.GetWorkspace(workspaceID)
	if !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "workspace not found")
		return
	}

	source := strings.TrimSpace(req.Trigger)
	if source == "" {
		source = "manual"
	}
	d.executeRuntimeCycle(snapshot.Exploration, source)

	state, ok := d.GetRuntimeState(workspaceID)
	if !ok || len(state.Runs) == 0 {
		writeV1Error(c, http.StatusInternalServerError, "internal", "failed to create run")
		return
	}
	latest := state.Runs[len(state.Runs)-1]
	c.JSON(http.StatusAccepted, RunResponse{Run: d.buildRunView(state, latest)})
}

func (d *ExplorationDomain) ApiV1GetRun(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	runID := c.Param("runID")
	if _, ok := d.GetWorkspace(workspaceID); !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "workspace not found")
		return
	}
	state, ok := d.QueryRuntimeState(workspaceID, RuntimeStateQuery{RunID: runID})
	if !ok || len(state.Runs) == 0 {
		writeV1Error(c, http.StatusNotFound, "not_found", "run not found")
		return
	}
	c.JSON(http.StatusOK, RunResponse{Run: d.buildRunView(state, state.Runs[0])})
}

func (d *ExplorationDomain) ApiV1GetTraceSummary(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	runID := strings.TrimSpace(c.Query("run_id"))
	if _, ok := d.GetWorkspace(workspaceID); !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "workspace not found")
		return
	}

	query := RuntimeStateQuery{}
	if runID != "" {
		query.RunID = runID
	}
	state, ok := d.QueryRuntimeState(workspaceID, query)
	if !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "runtime state not found")
		return
	}
	c.JSON(http.StatusOK, buildTraceSummary(workspaceID, runID, state))
}

func (d *ExplorationDomain) ApiV1ListTraceEvents(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	runID := strings.TrimSpace(c.Query("run_id"))
	category := strings.TrimSpace(c.Query("category"))
	level := strings.TrimSpace(c.Query("level"))
	if category != "" && !isValidTraceCategory(category) {
		writeV1Error(c, http.StatusBadRequest, "invalid_argument", "invalid category")
		return
	}
	if level != "" && !isValidTraceLevel(level) {
		writeV1Error(c, http.StatusBadRequest, "invalid_argument", "invalid level")
		return
	}
	if _, ok := d.GetWorkspace(workspaceID); !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "workspace not found")
		return
	}
	limit := 50
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			writeV1Error(c, http.StatusBadRequest, "invalid_argument", "invalid limit")
			return
		}
		if value > 200 {
			value = 200
		}
		limit = value
	}
	cursor := strings.TrimSpace(c.Query("cursor"))
	query := RuntimeStateQuery{}
	if runID != "" {
		query.RunID = runID
	}
	state, ok := d.QueryRuntimeState(workspaceID, query)
	if !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "runtime state not found")
		return
	}
	full := buildTraceSummary(workspaceID, runID, state).Items
	if category != "" {
		filtered := make([]TraceSummaryItem, 0, len(full))
		for _, item := range full {
			if item.Category == category {
				filtered = append(filtered, item)
			}
		}
		full = filtered
	}
	if level != "" {
		filtered := make([]TraceSummaryItem, 0, len(full))
		for _, item := range full {
			if item.Level == level {
				filtered = append(filtered, item)
			}
		}
		full = filtered
	}
	items, nextCursor, hasMore := applyTracePagination(full, cursor, limit)
	if len(items) == 0 {
		writeV1Error(c, http.StatusNotFound, "not_found", "trace events not found")
		return
	}
	c.JSON(http.StatusOK, TraceEventsResponse{
		WorkspaceID: workspaceID,
		RunID:       runID,
		Items:       items,
		NextCursor:  nextCursor,
		HasMore:     hasMore,
	})
}

func normalizeRunStatus(status RunStatus) string {
	switch status {
	case RunStatusPending:
		return "queued"
	case RunStatusRunning:
		return "planning"
	case RunStatusCompleted:
		return "completed"
	case RunStatusFailed:
		return "failed"
	default:
		return "completed"
	}
}

func (d *ExplorationDomain) buildRunView(state RuntimeStateSnapshot, run Run) RunView {
	view := RunView{
		ID:          run.ID,
		WorkspaceID: run.WorkspaceID,
		TriggerType: run.Source,
		Status:      deriveRunStatus(state, run),
		StartedAt:   toRFC3339(run.StartedAt),
		FinishedAt:  toRFC3339(run.EndedAt),
	}
	stepToAgent := map[string]string{}
	for _, task := range state.AgentTasks {
		if task.RunID != run.ID {
			continue
		}
		if task.PlanStepID == "" {
			continue
		}
		stepToAgent[task.PlanStepID] = normalizeAgentName(task.SubAgent)
	}

	latestPlanIndex := -1
	for _, plan := range state.Plans {
		if plan.RunID != run.ID {
			continue
		}
		if latestPlanIndex == -1 || plan.Version >= state.Plans[latestPlanIndex].Version {
			p := plan
			view.CurrentPlan = &PlanView{
				ID:      p.ID,
				Version: p.Version,
				Status:  derivePlanStatus(state, p, run),
			}
			latestPlanIndex = indexOfPlan(state.Plans, p.ID)
		}
	}
	if view.CurrentPlan != nil {
		for _, step := range state.PlanSteps {
			if step.RunID != run.ID || step.PlanID != view.CurrentPlan.ID {
				continue
			}
			assigned := stepToAgent[step.ID]
			if assigned == "" {
				assigned = inferAgentFromStep(step.Desc)
			}
			view.CurrentPlan.Steps = append(view.CurrentPlan.Steps, PlanStepV1{
				ID:            step.ID,
				Order:         step.Index,
				Kind:          step.Desc,
				AssignedAgent: assigned,
				Status:        normalizeStepStatus(step.Status),
			})
		}
	}
	return view
}

func indexOfPlan(plans []ExecutionPlan, planID string) int {
	for i := range plans {
		if plans[i].ID == planID {
			return i
		}
	}
	return -1
}

func normalizeStepStatus(status PlanStepStatus) string {
	if status == "" {
		return "todo"
	}
	return string(status)
}

func normalizeAgentName(name string) string {
	low := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.Contains(low, "research"):
		return "research"
	case strings.Contains(low, "graph"):
		return "graph"
	case strings.Contains(low, "artifact"):
		return "artifact"
	default:
		return "general"
	}
}

func derivePlanStatus(state RuntimeStateSnapshot, plan ExecutionPlan, run Run) string {
	hasNewer := false
	for _, p := range state.Plans {
		if p.RunID == plan.RunID && p.Version > plan.Version {
			hasNewer = true
			break
		}
	}
	if hasNewer {
		return "superseded"
	}

	total := 0
	done := 0
	failed := 0
	for _, step := range state.PlanSteps {
		if step.PlanID != plan.ID {
			continue
		}
		total++
		if step.Status == PlanStepDone || step.Status == PlanStepSkipped {
			done++
		}
		if step.Status == PlanStepFailed {
			failed++
		}
	}
	if failed > 0 {
		return "failed"
	}
	if total > 0 && done == total && run.EndedAt > 0 {
		return "completed"
	}
	return "active"
}

func deriveRunStatus(state RuntimeStateSnapshot, run Run) string {
	if run.Status == RunStatusFailed {
		return "failed"
	}
	if run.Status == RunStatusPending {
		return "queued"
	}
	if run.Status == RunStatusRunning {
		hasPlan := false
		hasDoing := false
		hasResult := false
		for _, plan := range state.Plans {
			if plan.RunID == run.ID {
				hasPlan = true
				break
			}
		}
		for _, step := range state.PlanSteps {
			if step.RunID != run.ID {
				continue
			}
			if step.Status == PlanStepDoing || step.Status == PlanStepTodo {
				hasDoing = true
			}
		}
		for _, result := range state.Results {
			taskID := result.TaskID
			for _, task := range state.AgentTasks {
				if task.ID == taskID && task.RunID == run.ID {
					hasResult = true
					break
				}
			}
		}
		if !hasPlan {
			return "planning"
		}
		if hasDoing {
			return "dispatching"
		}
		if hasResult {
			return "integrating"
		}
		return "planning"
	}

	if run.Status == RunStatusCompleted {
		for _, result := range state.Results {
			taskID := result.TaskID
			for _, task := range state.AgentTasks {
				if task.ID == taskID && task.RunID == run.ID {
					return "projected"
				}
			}
		}
		return "completed"
	}
	return normalizeRunStatus(run.Status)
}

func inferAgentFromStep(desc string) string {
	lower := strings.ToLower(desc)
	switch {
	case strings.Contains(lower, "research"):
		return "research"
	case strings.Contains(lower, "graph"):
		return "graph"
	case strings.Contains(lower, "artifact"):
		return "artifact"
	default:
		return "general"
	}
}
