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

	if _, ok := d.GetWorkspace(workspaceID); !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "workspace not found")
		return
	}

	source := strings.TrimSpace(req.Trigger)
	if source == "" {
		source = string(RunSourceManual)
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
		return "running"
	case RunStatusFailed:
		return "failed"
	case RunStatusCompleted:
		return "completed"
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
	return view
}

func deriveRunStatus(state RuntimeStateSnapshot, run Run) string {
	_ = state
	if run.Status == RunStatusFailed {
		return "failed"
	}
	if run.Status == RunStatusPending {
		return "queued"
	}
	if run.Status == RunStatusRunning {
		return "running"
	}

	if run.Status == RunStatusCompleted {
		return "completed"
	}
	return normalizeRunStatus(run.Status)
}
