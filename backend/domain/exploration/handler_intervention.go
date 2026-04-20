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

	controlReq := CreateControlActionRequest{
		Kind:           ControlActionIntervention,
		Intent:         req.Intent,
		TargetBranchID: req.TargetBranchID,
		Priority:       normalizeControlActionPriority(req.Priority),
	}
	controlView := d.storeControlActionRecord(workspaceID, controlReq)
	d.persistControlAction(controlView)
	d.persistV1Intervention(interventionViewFromControlAction(controlView))
	mapped := mapInterventionReq(req, workspaceID)
	snapshot, mutations, ok := d.ApplyIntervention(workspaceID, mapped)
	if !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "workspace not found or invalid intervention")
		return
	}
	state, _ := d.GetRuntimeState(workspaceID)
	controlView = d.advanceControlActionByRuntimeEvent(workspaceID, controlView.ID, state, mutations)
	view := interventionViewFromControlAction(controlView)
	d.persistV1Intervention(view)

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

func (d *ExplorationDomain) ApiV1CreateControlAction(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	var req CreateControlActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeV1Error(c, http.StatusBadRequest, "invalid_argument", "failed to parse control action request")
		return
	}
	if !isValidControlActionKind(req.Kind) {
		writeV1Error(c, http.StatusBadRequest, "invalid_argument", "invalid control action kind")
		return
	}
	if _, ok := d.GetWorkspace(workspaceID); !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "workspace not found")
		return
	}

	view := d.storeControlActionRecord(workspaceID, req)
	d.persistControlAction(view)

	if req.Kind == ControlActionIntervention {
		legacyReq := mapInterventionReq(CreateInterventionRequest{
			Intent:         req.Intent,
			TargetBranchID: req.TargetBranchID,
			Priority:       string(req.Priority),
		}, workspaceID)
		_, mutations, ok := d.ApplyIntervention(workspaceID, legacyReq)
		if !ok {
			writeV1Error(c, http.StatusNotFound, "not_found", "workspace not found or invalid control action")
			return
		}
		state, _ := d.GetRuntimeState(workspaceID)
		view = d.advanceControlActionByRuntimeEvent(workspaceID, view.ID, state, mutations)
	}

	c.JSON(http.StatusAccepted, ControlActionResponse{ControlAction: view})
}

func (d *ExplorationDomain) ApiV1GetControlAction(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	controlActionID := c.Param("controlActionID")
	view, ok := d.getControlActionRecord(workspaceID, controlActionID)
	if !ok {
		writeV1Error(c, http.StatusNotFound, "not_found", "control action not found")
		return
	}
	c.JSON(http.StatusOK, ControlActionResponse{ControlAction: view})
}

func (d *ExplorationDomain) ApiV1ListControlActionEvents(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	controlActionID := c.Param("controlActionID")
	if strings.TrimSpace(workspaceID) == "" || strings.TrimSpace(controlActionID) == "" {
		writeV1Error(c, http.StatusBadRequest, "invalid_argument", "workspace_id or control_action_id is empty")
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
	events, nextCursor, hasMore := d.listControlActionEvents(workspaceID, controlActionID, cursor, limit)
	if len(events) == 0 {
		writeV1Error(c, http.StatusNotFound, "not_found", "control action events not found")
		return
	}
	c.JSON(http.StatusOK, ControlActionEventsResponse{
		WorkspaceID:     workspaceID,
		ControlActionID: controlActionID,
		Events:          events,
		NextCursor:      nextCursor,
		HasMore:         hasMore,
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

func normalizeControlActionPriority(raw string) ControlActionPriority {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "low":
		return ControlActionPriorityLow
	case "high", "urgent":
		return ControlActionPriorityHigh
	default:
		return ControlActionPriorityNormal
	}
}

func isValidControlActionKind(kind ControlActionKind) bool {
	switch kind {
	case ControlActionIntervention,
		ControlActionReviewRequest,
		ControlActionArtifactRequest,
		ControlActionResumeRequest,
		ControlActionPolicyAdjustment,
		ControlActionMemoryPin:
		return true
	default:
		return false
	}
}

func interventionViewFromControlAction(action ControlActionView) InterventionView {
	return InterventionView{
		ID:               action.ID,
		WorkspaceID:      action.WorkspaceID,
		Intent:           action.Intent,
		Status:           InterventionLifecycleStatus(action.Status),
		AbsorbedByRunID:  action.AbsorbedByRunID,
		ReflectedEventID: action.ReflectedEventID,
		CreatedAt:        action.CreatedAt,
		UpdatedAt:        action.UpdatedAt,
	}
}

func (d *ExplorationDomain) storeInterventionRecord(workspaceID string, req CreateInterventionRequest) InterventionView {
	return interventionViewFromControlAction(d.storeControlActionRecord(workspaceID, CreateControlActionRequest{
		Kind:           ControlActionIntervention,
		Intent:         req.Intent,
		TargetBranchID: req.TargetBranchID,
		Priority:       normalizeControlActionPriority(req.Priority),
	}))
}

func (d *ExplorationDomain) storeControlActionRecord(workspaceID string, req CreateControlActionRequest) ControlActionView {
	now := time.Now().UnixMilli()
	view := ControlActionView{
		ID:             fmt.Sprintf("control-action-%s-%d", workspaceID, now),
		WorkspaceID:    workspaceID,
		Kind:           req.Kind,
		Intent:         strings.TrimSpace(req.Intent),
		Status:         ControlActionReceived,
		Priority:       req.Priority,
		TargetBranchID: strings.TrimSpace(req.TargetBranchID),
		CreatedAt:      toRFC3339(now),
		UpdatedAt:      toRFC3339(now),
	}
	if view.Priority == "" {
		view.Priority = ControlActionPriorityNormal
	}
	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		state.ControlActions = upsertControlAction(state.ControlActions, view)
		state.Interventions[view.ID] = interventionViewFromControlAction(view)
	})
	return view
}

func (d *ExplorationDomain) getInterventionRecord(workspaceID string, interventionID string) (InterventionView, bool) {
	controlAction, ok := d.getControlActionRecord(workspaceID, interventionID)
	if !ok {
		return InterventionView{}, false
	}
	return interventionViewFromControlAction(controlAction), true
}

func (d *ExplorationDomain) getControlActionRecord(workspaceID string, controlActionID string) (ControlActionView, bool) {
	var found ControlActionView
	var ok bool
	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		found, ok = findControlAction(state.ControlActions, controlActionID)
	})
	if ok {
		return found, true
	}
	view, dbOk := d.loadControlAction(workspaceID, controlActionID)
	if !dbOk {
		return ControlActionView{}, false
	}
	d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
		state.ControlActions = upsertControlAction(state.ControlActions, view)
		state.Interventions[controlActionID] = interventionViewFromControlAction(view)
	})
	return view, true
}

func (d *ExplorationDomain) advanceInterventionByRuntimeEvent(workspaceID string, interventionID string, state RuntimeStateSnapshot, mutations []MutationEvent) InterventionView {
	return interventionViewFromControlAction(d.advanceControlActionByRuntimeEvent(workspaceID, interventionID, state, mutations))
}

func (d *ExplorationDomain) advanceControlActionByRuntimeEvent(workspaceID string, controlActionID string, state RuntimeStateSnapshot, mutations []MutationEvent) ControlActionView {
	now := time.Now().UnixMilli()
	var result ControlActionView
	d.withWorkspaceState(workspaceID, func(ws *RuntimeWorkspaceState) {
		view, ok := findControlAction(ws.ControlActions, controlActionID)
		if !ok {
			return
		}
		if len(state.Runs) > 0 && view.Status == ControlActionReceived {
			view.Status = ControlActionAbsorbed
			view.AbsorbedByRunID = state.Runs[len(state.Runs)-1].ID
			view.UpdatedAt = toRFC3339(now)
		}
		if len(mutations) > 0 {
			view.Status = ControlActionReflected
			view.ReflectedEventID = fmt.Sprintf("event-%d", mutations[len(mutations)-1].CreatedAt)
			view.UpdatedAt = toRFC3339(now)
		}
		ws.ControlActions = upsertControlAction(ws.ControlActions, view)
		ws.Interventions[controlActionID] = interventionViewFromControlAction(view)
		result = view
	})
	d.persistControlAction(result)
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
		ID:             strconv.FormatUint(uint64(event.ID), 10),
		InterventionID: snapshot.ID,
		WorkspaceID:    snapshot.WorkspaceID,
		Status:         snapshot.Status,
		CreatedAt:      createdAt,
	}, nil
}

func (d *ExplorationDomain) listControlActionEvents(workspaceID string, controlActionID string, cursor string, limit int) ([]ControlActionEventView, string, bool) {
	if limit <= 0 {
		limit = 50
	}
	if d.DB != nil {
		records, err := d.DB.ListInterventionEventsByPrefix(
			workspaceID,
			controlActionID+"#",
			"v1_control_action_lifecycle_event",
			500,
		)
		if err == nil && len(records) > 0 {
			out := make([]ControlActionEventView, 0, len(records))
			for _, item := range records {
				view, err := decodeControlActionEventView(item)
				if err != nil {
					continue
				}
				out = append(out, view)
			}
			if len(out) > 0 {
				return applyControlActionEventPagination(out, cursor, limit)
			}
		}
	}

	view, ok := d.getControlActionRecord(workspaceID, controlActionID)
	if !ok {
		return nil, "", false
	}
	events := []ControlActionEventView{{
		ID:              controlActionID + "#snapshot",
		ControlActionID: controlActionID,
		WorkspaceID:     workspaceID,
		Status:          view.Status,
		Summary:         firstNonEmpty(view.Intent, string(view.Kind)),
		CreatedAt:       view.UpdatedAt,
	}}
	return applyControlActionEventPagination(events, cursor, limit)
}

func decodeControlActionEventView(event dbdao.InterventionEvent) (ControlActionEventView, error) {
	var snapshot ControlActionView
	if err := json.Unmarshal([]byte(event.Note), &snapshot); err != nil {
		return ControlActionEventView{}, err
	}
	createdAt := snapshot.UpdatedAt
	if createdAt == "" {
		createdAt = event.CreatedAt.UTC().Format(time.RFC3339)
	}
	return ControlActionEventView{
		ID:              strconv.FormatUint(uint64(event.ID), 10),
		ControlActionID: snapshot.ID,
		WorkspaceID:     snapshot.WorkspaceID,
		Status:          snapshot.Status,
		Summary:         firstNonEmpty(snapshot.Intent, string(snapshot.Kind)),
		CreatedAt:       createdAt,
	}, nil
}

func applyControlActionEventPagination(events []ControlActionEventView, cursor string, limit int) ([]ControlActionEventView, string, bool) {
	if len(events) == 0 {
		return events, "", false
	}
	start := 0
	if cursor != "" {
		for i, event := range events {
			if encodeControlActionEventCursor(event) == cursor {
				start = i + 1
				break
			}
		}
	}
	if start >= len(events) {
		return []ControlActionEventView{}, "", false
	}
	filtered := events[start:]
	if len(filtered) <= limit {
		return filtered, "", false
	}
	page := filtered[:limit]
	return page, encodeControlActionEventCursor(page[len(page)-1]), true
}

func encodeControlActionEventCursor(event ControlActionEventView) string {
	return event.CreatedAt + "|" + event.ID
}

func findControlAction(actions []ControlActionView, id string) (ControlActionView, bool) {
	for _, action := range actions {
		if action.ID == id {
			return action, true
		}
	}
	return ControlActionView{}, false
}

func upsertControlAction(actions []ControlActionView, action ControlActionView) []ControlActionView {
	for i := range actions {
		if actions[i].ID == action.ID {
			actions[i] = action
			return actions
		}
	}
	return append(actions, action)
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
	for i := range events {
		if encodeEventCursor(events[i]) == cursor || events[i].ID == cursor {
			return i + 1
		}
	}
	cTime, cID, ok := parseOrderedCursor(cursor)
	if !ok {
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
