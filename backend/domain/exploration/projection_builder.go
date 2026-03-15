package exploration

import (
	"fmt"
	"time"
)

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
