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
