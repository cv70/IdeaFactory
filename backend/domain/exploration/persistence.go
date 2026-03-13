package exploration

import (
	"backend/datasource/dbdao"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (d *ExplorationDomain) persistWorkspace(session ExplorationSession) {
	if d.DB == nil {
		return
	}
	raw, err := json.Marshal(session)
	if err != nil {
		return
	}

	state := &dbdao.WorkspaceState{
		WorkspaceID:         session.ID,
		Topic:               session.Topic,
		OutputGoal:          session.OutputGoal,
		Constraints:         session.Constraints,
		ActiveOpportunityID: session.ActiveOpportunityID,
		LastRunRound:        len(session.Runs),
		Snapshot:            string(raw),
	}
	_ = d.DB.UpsertWorkspaceState(state)
}

func (d *ExplorationDomain) loadWorkspace(workspaceID string) (*ExplorationSession, bool) {
	if d.DB == nil {
		return nil, false
	}
	state, err := d.DB.GetWorkspaceState(workspaceID)
	if err != nil || state == nil {
		return nil, false
	}
	var session ExplorationSession
	if err := json.Unmarshal([]byte(state.Snapshot), &session); err != nil {
		return nil, false
	}
	return &session, true
}

func (d *ExplorationDomain) persistIntervention(workspaceID string, req InterventionReq) {
	if d.DB == nil {
		return
	}
	event := &dbdao.InterventionEvent{
		ID:          fmt.Sprintf("intervention-%s-%d", workspaceID, time.Now().UnixNano()),
		WorkspaceID: workspaceID,
		Type:        string(req.Type),
		TargetID:    req.TargetID,
		Note:        req.Note,
	}
	_ = d.DB.CreateInterventionEvent(event)
}

func (d *ExplorationDomain) persistMutations(mutations []MutationEvent) {
	if d.DB == nil || len(mutations) == 0 {
		return
	}

	logs := make([]dbdao.MutationLog, 0, len(mutations))
	for _, mutation := range mutations {
		raw, err := json.Marshal(mutation)
		if err != nil {
			continue
		}
		log := dbdao.MutationLog{
			ID:          mutation.ID,
			WorkspaceID: mutation.WorkspaceID,
			Kind:        mutation.Kind,
			Source:      mutation.Source,
			Payload:     string(raw),
			CreatedAt:   time.UnixMilli(mutation.CreatedAt),
		}
		logs = append(logs, log)
	}
	_ = d.DB.CreateMutationLogs(logs)
	if len(logs) > 0 {
		d.compactMutationLogs(logs[0].WorkspaceID, 3000, 2000)
	}
}

type MutationReplayPage struct {
	Mutations  []MutationEvent `json:"mutations"`
	NextCursor string          `json:"next_cursor,omitempty"`
	HasMore    bool            `json:"has_more"`
}

func parseCursor(cursor string) (time.Time, string, error) {
	if strings.TrimSpace(cursor) == "" {
		return time.Time{}, "", nil
	}
	parts := strings.SplitN(cursor, "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", fmt.Errorf("invalid cursor")
	}
	unixMs, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, "", err
	}
	return time.UnixMilli(unixMs), parts[1], nil
}

func buildCursor(createdAt time.Time, id string) string {
	return fmt.Sprintf("%d|%s", createdAt.UnixMilli(), id)
}

func (d *ExplorationDomain) replayMutations(workspaceID string, cursor string, limit int) (MutationReplayPage, error) {
	if d.DB == nil {
		return MutationReplayPage{}, nil
	}

	cursorTime, cursorID, err := parseCursor(cursor)
	if err != nil {
		return MutationReplayPage{}, err
	}

	fetchLimit := limit
	if fetchLimit <= 0 || fetchLimit > 1000 {
		fetchLimit = 200
	}
	logs, err := d.DB.ListMutationLogsByCursor(workspaceID, cursorTime, cursorID, fetchLimit+1)
	if err != nil {
		return MutationReplayPage{}, err
	}

	hasMore := len(logs) > fetchLimit
	if hasMore {
		logs = logs[:fetchLimit]
	}

	events := make([]MutationEvent, 0, len(logs))
	for _, log := range logs {
		var event MutationEvent
		if err := json.Unmarshal([]byte(log.Payload), &event); err != nil {
			continue
		}
		events = append(events, event)
	}

	page := MutationReplayPage{
		Mutations: events,
		HasMore:   hasMore,
	}
	if hasMore && len(logs) > 0 {
		last := logs[len(logs)-1]
		page.NextCursor = buildCursor(last.CreatedAt, last.ID)
	}
	return page, nil
}

func (d *ExplorationDomain) compactMutationLogs(workspaceID string, hardLimit int64, keepRecent int) {
	if d.DB == nil {
		return
	}
	count, err := d.DB.CountMutationLogs(workspaceID)
	if err != nil {
		return
	}
	if count <= hardLimit {
		return
	}

	cutoffLog, err := d.DB.GetMutationCutoffForRecent(workspaceID, keepRecent)
	if err != nil || cutoffLog == nil {
		return
	}
	_ = d.DB.DeleteMutationLogsBefore(workspaceID, cutoffLog.CreatedAt)

	state, err := d.DB.GetWorkspaceState(workspaceID)
	if err != nil || state == nil {
		return
	}
	state.LastCompactedAt = time.Now()
	_ = d.DB.UpsertWorkspaceState(state)
}
