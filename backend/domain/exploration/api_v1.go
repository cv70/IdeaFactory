package exploration

import (
	"backend/datasource/dbdao"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (d *ExplorationDomain) ApiV1CreateWorkspace(c *gin.Context) {
	var req CreateWorkspaceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_argument", "failed to parse create workspace request")
		return
	}
	snapshot := d.CreateWorkspace(req)
	d.initializeWorkspaceGraph(c.Request.Context(), snapshot.Exploration.ID)
	// Re-fetch so the response includes seeded Direction nodes.
	if updated, ok := d.GetWorkspace(snapshot.Exploration.ID); ok {
		snapshot = updated
	}
	c.JSON(http.StatusCreated, WorkspaceResponse{Workspace: toWorkspaceView(snapshot.Exploration)})
}

func (d *ExplorationDomain) ApiV1GetWorkspace(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	snapshot, ok := d.GetWorkspace(workspaceID)
	if !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "workspace not found")
		return
	}
	c.JSON(http.StatusOK, WorkspaceResponse{Workspace: toWorkspaceView(snapshot.Exploration)})
}

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

func (d *ExplorationDomain) ApiV1GetProjection(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	snapshot, ok := d.GetWorkspace(workspaceID)
	if !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "workspace not found")
		return
	}
	projection := d.buildProjectionResponse(snapshot)
	c.JSON(http.StatusOK, projection)
}

func (d *ExplorationDomain) ApiV1CreateIntervention(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	var req CreateInterventionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_argument", "failed to parse intervention request")
		return
	}

	view := d.storeInterventionRecord(workspaceID, req)
	d.persistV1Intervention(view)
	mapped := mapInterventionReq(req, workspaceID)
	snapshot, mutations, ok := d.ApplyIntervention(workspaceID, mapped)
	if !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "workspace not found or invalid intervention")
		return
	}
	state, _ := d.GetRuntimeState(workspaceID)
	view = d.advanceInterventionByRuntimeEvent(workspaceID, view.ID, state, mutations)

	if snapshot.Exploration.ID != "" {
		_ = snapshot
	}
	c.JSON(http.StatusAccepted, InterventionResponse{Intervention: view})
}

func (d *ExplorationDomain) ApiV1GetIntervention(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	interventionID := c.Param("interventionID")
	view, ok := d.getInterventionRecord(workspaceID, interventionID)
	if !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "intervention not found")
		return
	}
	c.JSON(http.StatusOK, InterventionResponse{Intervention: view})
}

func (d *ExplorationDomain) ApiV1ListInterventionEvents(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	interventionID := c.Param("interventionID")
	if strings.TrimSpace(workspaceID) == "" || strings.TrimSpace(interventionID) == "" {
		writeV1Error(c, http.StatusBadRequest, "invalid_argument", "workspace_id or intervention_id is empty")
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
	status := InterventionLifecycleStatus(strings.TrimSpace(c.Query("status")))
	if status != "" &&
		status != InterventionReceived &&
		status != InterventionAbsorbed &&
		status != InterventionReplanned &&
		status != InterventionReflected {
		writeV1Error(c, http.StatusBadRequest, "invalid_argument", "invalid status")
		return
	}

	events, nextCursor, hasMore := d.listInterventionEvents(workspaceID, interventionID, cursor, limit, status)
	if len(events) == 0 {
		writeV1Error(c, http.StatusNotFound, "not_found", "intervention events not found")
		return
	}
	c.JSON(http.StatusOK, InterventionEventsResponse{
		WorkspaceID:    workspaceID,
		InterventionID: interventionID,
		Events:         events,
		NextCursor:     nextCursor,
		HasMore:        hasMore,
	})
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

func toWorkspaceView(session ExplorationSession) WorkspaceView {
	constraints := []string{}
	if strings.TrimSpace(session.Constraints) != "" {
		constraints = append(constraints, session.Constraints)
	}
	nowISO := time.Now().UTC().Format(time.RFC3339)
	return WorkspaceView{
		ID:          session.ID,
		Topic:       session.Topic,
		Goal:        session.OutputGoal,
		Constraints: constraints,
		Status:      WorkspaceStatusActive,
		CreatedAt:   nowISO,
		UpdatedAt:   nowISO,
	}
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

func (d *ExplorationDomain) buildProjectionResponse(snapshot WorkspaceSnapshot) ProjectionResponse {
	now := time.Now().UnixMilli()
	view := ProjectionView{
		WorkspaceID: snapshot.Exploration.ID,
		EventID:     fmt.Sprintf("event-%d", now),
		GeneratedAt: toRFC3339(now),
		Map: ProjectionMap{
			Nodes: snapshot.DirectionMap.Nodes,
			Edges: snapshot.DirectionMap.Edges,
		},
	}
	if state, ok := d.GetRuntimeState(snapshot.Exploration.ID); ok {
		for _, item := range state.Results {
			view.RecentChanges = append(view.RecentChanges, ProjectionChange{
				Type:    "task_result",
				Summary: item.Summary,
			})
		}
		if len(state.Runs) > 0 {
			lastRun := state.Runs[len(state.Runs)-1]
			view.RunSummary = RunSummaryView{
				RunID:  lastRun.ID,
				Status: normalizeRunStatus(lastRun.Status),
				Focus:  snapshot.Exploration.ActiveOpportunityID,
			}
		}
	}
	return ProjectionResponse{Projection: view}
}

func mapInterventionReq(req CreateInterventionRequest, workspaceID string) InterventionReq {
	target := strings.TrimSpace(req.TargetBranchID)
	if target == "" {
		target = workspaceID
	}
	intent := strings.ToLower(strings.TrimSpace(req.Intent))
	mapped := InterventionReq{Type: InterventionAddContext, TargetID: target, Note: req.Intent}
	switch {
	case strings.Contains(intent, "expand"):
		mapped.Type = InterventionExpandOpportunity
	case strings.Contains(intent, "focus"):
		mapped.Type = InterventionShiftFocus
	case strings.Contains(intent, "favorite") || strings.Contains(intent, "favourite") || strings.Contains(intent, "save"):
		mapped.Type = InterventionToggleFavorite
	case strings.Contains(intent, "intensity") || strings.Contains(intent, "aggressive"):
		mapped.Type = InterventionAdjustIntensity
	}
	return mapped
}

func (d *ExplorationDomain) storeInterventionRecord(workspaceID string, req CreateInterventionRequest) InterventionView {
	now := time.Now().UnixMilli()
	view := InterventionView{
		ID:          fmt.Sprintf("intervention-%s-%d", workspaceID, now),
		WorkspaceID: workspaceID,
		Intent:      strings.TrimSpace(req.Intent),
		Status:      InterventionReceived,
		CreatedAt:   toRFC3339(now),
		UpdatedAt:   toRFC3339(now),
	}
	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		state.Interventions[view.ID] = view
	})
	return view
}

func (d *ExplorationDomain) getInterventionRecord(workspaceID string, interventionID string) (InterventionView, bool) {
	var found InterventionView
	var ok bool
	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		found, ok = state.Interventions[interventionID]
	})
	if ok {
		return found, true
	}
	view, dbOk := d.loadV1Intervention(workspaceID, interventionID)
	if !dbOk {
		return InterventionView{}, false
	}
	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		state.Interventions[interventionID] = view
	})
	return view, true
}

func (d *ExplorationDomain) advanceInterventionByRuntimeEvent(workspaceID string, interventionID string, state RuntimeStateSnapshot, mutations []MutationEvent) InterventionView {
	now := time.Now().UnixMilli()
	var result InterventionView
	d.withWorkspaceState(workspaceID, func(ws *RuntimeWorkspaceState) {
		view, ok := ws.Interventions[interventionID]
		if !ok {
			return
		}
		if len(state.Runs) > 0 && view.Status == InterventionReceived {
			view.Status = InterventionAbsorbed
			view.AbsorbedByRunID = state.Runs[len(state.Runs)-1].ID
			view.UpdatedAt = toRFC3339(now)
		}
		if len(state.Plans) > 0 && (view.Status == InterventionReceived || view.Status == InterventionAbsorbed) {
			view.Status = InterventionReplanned
			view.ReplannedPlanID = state.Plans[len(state.Plans)-1].ID
			view.UpdatedAt = toRFC3339(now)
		}
		if len(mutations) > 0 {
			view.Status = InterventionReflected
			view.ReflectedEventID = fmt.Sprintf("event-%d", mutations[len(mutations)-1].CreatedAt)
			view.UpdatedAt = toRFC3339(now)
		}
		ws.Interventions[interventionID] = view
		result = view
	})
	d.persistV1Intervention(result)
	return result
}

func (d *ExplorationDomain) listInterventionEvents(
	workspaceID string,
	interventionID string,
	cursor string,
	limit int,
	status InterventionLifecycleStatus,
) ([]InterventionEventView, string, bool) {
	if limit <= 0 {
		limit = 50
	}
	if d.DB != nil {
		records, err := d.DB.ListInterventionEventsByPrefix(
			workspaceID,
			interventionID+"#",
			"v1_intervention_lifecycle_event",
			500,
		)
		if err == nil && len(records) > 0 {
			out := make([]InterventionEventView, 0, len(records))
			for _, item := range records {
				view, err := decodeInterventionEventView(item)
				if err != nil {
					continue
				}
				if status != "" && view.Status != status {
					continue
				}
				out = append(out, view)
			}
			if len(out) > 0 {
				return applyEventPagination(out, cursor, limit)
			}
		}
	}

	// Fallback when DB is unavailable: use in-memory latest snapshot as a single event.
	view, ok := d.getInterventionRecord(workspaceID, interventionID)
	if !ok {
		return nil, "", false
	}
	events := []InterventionEventView{{
		ID:             interventionID + "#snapshot",
		InterventionID: interventionID,
		WorkspaceID:    workspaceID,
		Status:         view.Status,
		CreatedAt:      view.UpdatedAt,
	}}
	if status != "" && events[0].Status != status {
		return []InterventionEventView{}, "", false
	}
	return applyEventPagination(events, cursor, limit)
}

func decodeInterventionEventView(event dbdao.InterventionEvent) (InterventionEventView, error) {
	var snapshot InterventionView
	if err := json.Unmarshal([]byte(event.Note), &snapshot); err != nil {
		return InterventionEventView{}, err
	}
	createdAt := snapshot.UpdatedAt
	if createdAt == "" {
		createdAt = event.CreatedAt.UTC().Format(time.RFC3339)
	}
	return InterventionEventView{
		ID:             event.ID,
		InterventionID: snapshot.ID,
		WorkspaceID:    snapshot.WorkspaceID,
		Status:         snapshot.Status,
		CreatedAt:      createdAt,
	}, nil
}

func applyEventPagination(events []InterventionEventView, cursor string, limit int) ([]InterventionEventView, string, bool) {
	if len(events) == 0 {
		return events, "", false
	}
	start := findStartIndexByCursor(events, cursor)
	if start >= len(events) {
		return []InterventionEventView{}, "", false
	}
	filtered := events[start:]
	if len(filtered) <= limit {
		return filtered, "", false
	}
	page := filtered[:limit]
	nextCursor := encodeEventCursor(page[len(page)-1])
	return page, nextCursor, true
}

func findStartIndexByCursor(events []InterventionEventView, cursor string) int {
	if cursor == "" {
		return 0
	}
	cTime, cID, ok := parseOrderedCursor(cursor)
	if !ok {
		// Backward compatibility for old cursor format that only carries event ID.
		for i := range events {
			if events[i].ID == cursor {
				return i + 1
			}
		}
		return 0
	}
	for i := range events {
		evTime, err := time.Parse(time.RFC3339, events[i].CreatedAt)
		if err != nil {
			continue
		}
		if evTime.After(cTime) || (evTime.Equal(cTime) && events[i].ID > cID) {
			return i
		}
	}
	return len(events)
}

func encodeEventCursor(event InterventionEventView) string {
	ts, err := time.Parse(time.RFC3339, event.CreatedAt)
	if err != nil {
		return event.ID
	}
	return buildOrderedCursor(ts, event.ID)
}

func buildTraceSummary(workspaceID string, runID string, state RuntimeStateSnapshot) TraceSummaryResponse {
	resp := TraceSummaryResponse{WorkspaceID: workspaceID, RunID: runID, Items: []TraceSummaryItem{}}
	for _, run := range state.Runs {
		resp.Items = append(resp.Items, TraceSummaryItem{
			ID:        "plan-" + run.ID,
			Timestamp: toRFC3339(run.StartedAt),
			Level:     "info",
			Category:  "plan",
			Message:   fmt.Sprintf("run %s started with source %s", run.ID, run.Source),
			RelatedIDs: []string{
				run.ID,
			},
		})
	}
	for _, result := range state.Results {
		resp.Items = append(resp.Items, TraceSummaryItem{
			ID:        "task-" + result.TaskID,
			Timestamp: toRFC3339(result.UpdatedAt),
			Level:     "info",
			Category:  "task",
			Message:   result.Summary,
			RelatedIDs: []string{
				result.TaskID,
			},
		})
	}
	if state.LatestReplanReason != "" {
		refID := ""
		if state.Balance.RunID != "" {
			refID = state.Balance.RunID
		} else {
			refID = "latest"
		}
		resp.Items = append(resp.Items, TraceSummaryItem{
			ID:        "intervention-" + refID,
			Timestamp: toRFC3339(state.Balance.UpdatedAt),
			Level:     "info",
			Category:  "intervention",
			Message:   state.LatestReplanReason,
		})
	}
	if len(resp.Items) == 0 {
		resp.Items = append(resp.Items, TraceSummaryItem{
			ID:        "projection-empty",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Level:     "info",
			Category:  "projection",
			Message:   "no trace events yet",
		})
	}
	return resp
}

func applyTracePagination(items []TraceSummaryItem, cursor string, limit int) ([]TraceSummaryItem, string, bool) {
	if len(items) == 0 {
		return items, "", false
	}
	start := 0
	if cursor != "" {
		cTime, cID, ok := parseOrderedCursor(cursor)
		if ok {
			start = len(items)
			for i := range items {
				ts, err := time.Parse(time.RFC3339, items[i].Timestamp)
				if err != nil {
					continue
				}
				if ts.After(cTime) || (ts.Equal(cTime) && items[i].ID > cID) {
					start = i
					break
				}
			}
		}
	}
	if start >= len(items) {
		return []TraceSummaryItem{}, "", false
	}
	filtered := items[start:]
	if len(filtered) <= limit {
		return filtered, "", false
	}
	page := filtered[:limit]
	last := page[len(page)-1]
	ts, err := time.Parse(time.RFC3339, last.Timestamp)
	if err != nil {
		return page, "", true
	}
	return page, buildOrderedCursor(ts, last.ID), true
}

func isValidTraceCategory(category string) bool {
	switch category {
	case "plan", "task", "tool", "projection", "intervention", "balance":
		return true
	default:
		return false
	}
}

func isValidTraceLevel(level string) bool {
	switch level {
	case "info", "warn", "error":
		return true
	default:
		return false
	}
}

func toRFC3339(ms int64) string {
	if ms <= 0 {
		return ""
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}

func writeV1Error(c *gin.Context, status int, code string, message string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
