package exploration

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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

func (d *ExplorationDomain) buildProjectionResponse(snapshot *WorkspaceSnapshot) ProjectionResponse {
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
				Type:     "task_result",
				Summary:  item.Summary,
				Timeline: append([]string{}, item.Timeline...),
			})
		}
		if len(state.Runs) > 0 {
			lastRun := state.Runs[len(state.Runs)-1]
			view.RunSummary = RunSummaryView{
				RunID:  lastRun.ID,
				Status: normalizeRunStatus(lastRun.Status),
				Mode:   string(lastRun.Mode),
				Focus:  snapshot.Exploration.ActiveOpportunityID,
			}
		}
		if len(state.Turns) > 0 {
			lastTurn := state.Turns[len(state.Turns)-1]
			view.TurnSummary = TurnSummaryView{
				TurnID:         lastTurn.ID,
				Index:          lastTurn.TurnIndex,
				Status:         string(lastTurn.Status),
				ContinueReason: lastTurn.ContinueReason,
			}
		}
		for _, action := range state.ControlActions {
			if action.Status != ControlActionReflected {
				continue
			}
			view.ControlEffects = append(view.ControlEffects, ControlEffectView{
				ControlActionID: action.ID,
				Kind:            string(action.Kind),
				EffectSummary:   firstNonEmpty(action.Intent, string(action.Kind)),
			})
		}
	}
	return ProjectionResponse{Projection: view}
}
