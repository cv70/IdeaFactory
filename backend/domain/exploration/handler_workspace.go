package exploration

import (
	"net/http"
	"strings"
	"time"

	"backend/datasource/dbdao"

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
	c.JSON(http.StatusCreated, WorkspaceResponse{Workspace: toWorkspaceView(snapshot.Exploration, nil)})
}

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
