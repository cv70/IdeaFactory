package exploration

import (
	"net/http"
	"strings"
	"time"

	"backend/datasource/dbdao"
	"backend/utils"

	"github.com/gin-gonic/gin"
)

func (d *ExplorationDomain) ApiV1CreateWorkspace(c *gin.Context) {
	var req CreateWorkspaceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespError(c, http.StatusBadRequest, "failed to parse create workspace request")
		return
	}
	snapshot, err := d.CreateWorkspace(req)
	if err != nil {
		utils.RespError(c, http.StatusInternalServerError, "failed to create workspace")
		return
	}
	// Re-fetch so the response includes any initial run-side mutations applied during creation.
	if updated, ok := d.GetWorkspace(snapshot.Exploration.ID); ok {
		snapshot = updated
	}
	c.JSON(http.StatusCreated, WorkspaceResponse{Workspace: toWorkspaceView(snapshot.Exploration, nil)})
}

func (d *ExplorationDomain) ApiV1GetWorkspace(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	workspaceDBID, err := parseWorkspaceID(workspaceID)
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_argument", "workspaceID must be a positive integer")
		return
	}
	snapshot, ok := d.GetWorkspace(workspaceID)
	if !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "workspace not found")
		return
	}
	var dbState *dbdao.WorkspaceState
	if d.DB != nil {
		dbState, _ = d.DB.GetWorkspaceState(workspaceDBID)
	}
	c.JSON(http.StatusOK, WorkspaceResponse{Workspace: toWorkspaceView(snapshot.Exploration, dbState)})
}

func (d *ExplorationDomain) ApiV1PatchWorkspace(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	workspaceDBID, err := parseWorkspaceID(workspaceID)
	if err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_argument", "workspaceID must be a positive integer")
		return
	}
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
			if err := d.DB.PauseWorkspaceState(workspaceDBID); err != nil {
				writeV1Error(c, http.StatusInternalServerError, "internal", "failed to pause workspace")
				return
			}
			dbState, _ = d.DB.GetWorkspaceState(workspaceDBID)
		}
		// In no-DB mode, construct a synthetic dbState so the response reflects "paused".
		if dbState == nil {
			dbState = &dbdao.WorkspaceState{PausedAt: &now}
		}
		d.pauseScheduler(workspaceID)
	} else {
		if d.DB != nil {
			if err := d.DB.ResumeWorkspaceState(workspaceDBID); err != nil {
				writeV1Error(c, http.StatusInternalServerError, "internal", "failed to resume workspace")
				return
			}
		}
		// Start a run immediately; subsequent runs will use IntervalMs from strategy.
		d.triggerRun(ctx, workspaceID, string(RunSourceResume))
		dbState = nil // PausedAt is nil → status = active
	}

	c.JSON(http.StatusOK, WorkspaceResponse{Workspace: toWorkspaceView(snapshot.Exploration, dbState)})
}

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
