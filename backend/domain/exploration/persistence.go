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

func (d *ExplorationDomain) persistRuntimeState(workspaceID string) {
	if d.DB == nil {
		return
	}
	snapshot, ok := d.GetRuntimeState(workspaceID)
	if !ok {
		return
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	_ = d.DB.UpsertWorkspaceRuntimeState(&dbdao.WorkspaceRuntimeState{
		WorkspaceID: workspaceID,
		Snapshot:    string(raw),
	})

	projection := dbdao.RuntimeStateProjection{
		WorkspaceID: workspaceID,
		Runs:        make([]dbdao.RuntimeRunRecord, 0, len(snapshot.Runs)),
		Plans:       make([]dbdao.RuntimePlanRecord, 0, len(snapshot.Plans)),
		PlanSteps:   make([]dbdao.RuntimePlanStepRecord, 0, len(snapshot.PlanSteps)),
		AgentTasks:  make([]dbdao.RuntimeAgentTaskRecord, 0, len(snapshot.AgentTasks)),
		Results:     make([]dbdao.RuntimeTaskResultRecord, 0, len(snapshot.Results)),
	}
	for _, item := range snapshot.Runs {
		projection.Runs = append(projection.Runs, dbdao.RuntimeRunRecord{
			WorkspaceID: item.WorkspaceID,
			Source:      item.Source,
			Status:      string(item.Status),
			StartedAt:   item.StartedAt,
			EndedAt:     item.EndedAt,
		})
	}
	for _, item := range snapshot.Plans {
		projection.Plans = append(projection.Plans, dbdao.RuntimePlanRecord{
			WorkspaceID: item.WorkspaceID,
			RunID:       item.RunID,
			Version:     item.Version,
		})
	}
	for _, item := range snapshot.PlanSteps {
		projection.PlanSteps = append(projection.PlanSteps, dbdao.RuntimePlanStepRecord{
			WorkspaceID: item.WorkspaceID,
			RunID:       item.RunID,
			PlanID:      item.PlanID,
			StepIndex:   item.Index,
			Desc:        item.Desc,
			Status:      string(item.Status),
		})
	}
	for _, item := range snapshot.AgentTasks {
		projection.AgentTasks = append(projection.AgentTasks, dbdao.RuntimeAgentTaskRecord{
			WorkspaceID: item.WorkspaceID,
			RunID:       item.RunID,
			PlanID:      item.PlanID,
			PlanStepID:  item.PlanStepID,
			SubAgent:    item.SubAgent,
			Goal:        item.Goal,
			Status:      string(item.Status),
		})
	}
	for _, item := range snapshot.Results {
		projection.Results = append(projection.Results, dbdao.RuntimeTaskResultRecord{
			TaskID:      item.TaskID,
			WorkspaceID: workspaceID,
			Summary:     item.Summary,
			IsSuccess:   item.IsSuccess,
		})
	}
	if snapshot.Balance.WorkspaceID != "" {
		projection.Balance = &dbdao.RuntimeBalanceRecord{
			WorkspaceID:        snapshot.Balance.WorkspaceID,
			RunID:              snapshot.Balance.RunID,
			Divergence:         snapshot.Balance.Divergence,
			Research:           snapshot.Balance.Research,
			Aggression:         snapshot.Balance.Aggression,
			Reason:             snapshot.Balance.Reason,
			UpdatedAtMs:        snapshot.Balance.UpdatedAt,
			LatestReplanReason: snapshot.LatestReplanReason,
		}
	}
	_ = d.DB.ReplaceWorkspaceRuntimeProjection(projection)
}

func (d *ExplorationDomain) loadRuntimeState(workspaceID string) (RuntimeStateSnapshot, bool) {
	if d.DB == nil {
		return RuntimeStateSnapshot{}, false
	}
	projection, err := d.DB.LoadWorkspaceRuntimeProjection(workspaceID)
	if err == nil && projection != nil && len(projection.Runs) > 0 {
		out := RuntimeStateSnapshot{
			Runs:               make([]Run, 0, len(projection.Runs)),
			Plans:              make([]ExecutionPlan, 0, len(projection.Plans)),
			PlanSteps:          make([]PlanStep, 0, len(projection.PlanSteps)),
			AgentTasks:         make([]AgentTask, 0, len(projection.AgentTasks)),
			Results:            make([]AgentTaskResultSummary, 0, len(projection.Results)),
			LatestReplanReason: projection.LatestReplanReason,
		}
		for _, item := range projection.Runs {
			out.Runs = append(out.Runs, Run{
				ID:          strconv.FormatUint(uint64(item.ID), 10),
				WorkspaceID: item.WorkspaceID,
				Source:      item.Source,
				Status:      RunStatus(item.Status),
				StartedAt:   item.StartedAt,
				EndedAt:     item.EndedAt,
			})
		}
		for _, item := range projection.Plans {
			out.Plans = append(out.Plans, ExecutionPlan{
				ID:          strconv.FormatUint(uint64(item.ID), 10),
				WorkspaceID: item.WorkspaceID,
				RunID:       item.RunID,
				Version:     item.Version,
				CreatedAt:   item.CreatedAt.UnixMilli(),
			})
		}
		for _, item := range projection.PlanSteps {
			out.PlanSteps = append(out.PlanSteps, PlanStep{
				ID:          strconv.FormatUint(uint64(item.ID), 10),
				WorkspaceID: item.WorkspaceID,
				RunID:       item.RunID,
				PlanID:      item.PlanID,
				Index:       item.StepIndex,
				Desc:        item.Desc,
				Status:      PlanStepStatus(item.Status),
				UpdatedAt:   item.UpdatedAt.UnixMilli(),
			})
		}
		for _, item := range projection.AgentTasks {
			out.AgentTasks = append(out.AgentTasks, AgentTask{
				ID:          strconv.FormatUint(uint64(item.ID), 10),
				WorkspaceID: item.WorkspaceID,
				RunID:       item.RunID,
				PlanID:      item.PlanID,
				PlanStepID:  item.PlanStepID,
				SubAgent:    item.SubAgent,
				Goal:        item.Goal,
				Status:      PlanStepStatus(item.Status),
				UpdatedAt:   item.UpdatedAt.UnixMilli(),
			})
		}
		for _, item := range projection.Results {
			out.Results = append(out.Results, AgentTaskResultSummary{
				TaskID:    item.TaskID,
				Summary:   item.Summary,
				IsSuccess: item.IsSuccess,
				UpdatedAt: item.UpdatedAt.UnixMilli(),
			})
		}
		if projection.Balance != nil {
			out.Balance = BalanceState{
				WorkspaceID: projection.Balance.WorkspaceID,
				RunID:       projection.Balance.RunID,
				Divergence:  projection.Balance.Divergence,
				Research:    projection.Balance.Research,
				Aggression:  projection.Balance.Aggression,
				Reason:      projection.Balance.Reason,
				UpdatedAt:   projection.Balance.UpdatedAtMs,
			}
		}
		return out, true
	}

	state, err := d.DB.GetWorkspaceRuntimeState(workspaceID)
	if err != nil || state == nil {
		return RuntimeStateSnapshot{}, false
	}
	var snapshot RuntimeStateSnapshot
	if err := json.Unmarshal([]byte(state.Snapshot), &snapshot); err != nil {
		return RuntimeStateSnapshot{}, false
	}
	if len(snapshot.Runs) == 0 {
		return RuntimeStateSnapshot{}, false
	}
	return snapshot, true
}

func (d *ExplorationDomain) persistIntervention(workspaceID string, req InterventionReq) {
	if d.DB == nil {
		return
	}
	event := &dbdao.InterventionEvent{
		WorkspaceID: workspaceID,
		Type:        string(req.Type),
		TargetID:    req.TargetID,
		Note:        req.Note,
	}
	_ = d.DB.CreateInterventionEvent(event)
}

func (d *ExplorationDomain) persistMutations(mutations []MutationEvent) {
	if len(mutations) == 0 {
		return
	}
	if d.DB == nil {
		workspaceID := mutations[0].WorkspaceID
		d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
			state.Mutations = append(state.Mutations, mutations...)
			if len(state.Mutations) > 3000 {
				state.Mutations = append([]MutationEvent{}, state.Mutations[len(state.Mutations)-2000:]...)
			}
		})
		return
	}

	logs := make([]dbdao.MutationLog, 0, len(mutations))
	for _, mutation := range mutations {
		raw, err := json.Marshal(mutation)
		if err != nil {
			continue
		}
		log := dbdao.MutationLog{
			WorkspaceID: mutation.WorkspaceID,
			Kind:        mutation.Kind,
			Source:      mutation.Source,
			Payload:     string(raw),
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
	ts, id, ok := parseOrderedCursor(cursor)
	if !ok {
		return time.Time{}, "", fmt.Errorf("invalid cursor")
	}
	return ts, id, nil
}

func buildCursor(createdAt time.Time, id string) string {
	// Keep legacy cursor shape for existing mutation consumers.
	return buildOrderedCursorUnixMilli(createdAt, id)
}

func (d *ExplorationDomain) replayMutations(workspaceID string, cursor string, limit int) (MutationReplayPage, error) {
	if d.DB == nil {
		cursorTime, cursorID, err := parseCursor(cursor)
		if err != nil {
			return MutationReplayPage{}, err
		}
		fetchLimit := limit
		if fetchLimit <= 0 || fetchLimit > 1000 {
			fetchLimit = 200
		}

		var logs []MutationEvent
		d.withWorkspaceState(workspaceID, func(state *RuntimeWorkspaceState) {
			logs = append([]MutationEvent{}, state.Mutations...)
		})

		filtered := make([]MutationEvent, 0, len(logs))
		for _, event := range logs {
			if cursorTime.IsZero() {
				filtered = append(filtered, event)
				continue
			}
			eventTime := time.UnixMilli(event.CreatedAt)
			if eventTime.After(cursorTime) || (eventTime.Equal(cursorTime) && event.ID > cursorID) {
				filtered = append(filtered, event)
			}
		}

		hasMore := len(filtered) > fetchLimit
		if hasMore {
			filtered = filtered[:fetchLimit]
		}
		page := MutationReplayPage{
			Mutations: filtered,
			HasMore:   hasMore,
		}
		if hasMore && len(filtered) > 0 {
			last := filtered[len(filtered)-1]
			page.NextCursor = buildCursor(time.UnixMilli(last.CreatedAt), last.ID)
		}
		return page, nil
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
		page.NextCursor = buildCursor(last.CreatedAt, strconv.FormatUint(uint64(last.ID), 10))
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

func (d *ExplorationDomain) persistV1Intervention(view InterventionView) {
	if d.DB == nil || view.ID == "" || view.WorkspaceID == "" {
		return
	}
	raw, err := json.Marshal(view)
	if err != nil {
		return
	}
	// Snapshot record keeps the latest lifecycle state for fast point-read.
	snapshot := &dbdao.InterventionEvent{
		WorkspaceID: view.WorkspaceID,
		Type:        "v1_intervention_snapshot",
		TargetID:    view.ID,
		Note:        string(raw),
	}
	_ = d.DB.UpsertInterventionEvent(snapshot)

	// History record appends each lifecycle change for replay/audit.
	history := &dbdao.InterventionEvent{
		WorkspaceID: view.WorkspaceID,
		Type:        "v1_intervention_lifecycle_event",
		TargetID:    view.ID,
		Note:        string(raw),
	}
	_ = d.DB.CreateInterventionEvent(history)
}

func (d *ExplorationDomain) loadV1Intervention(workspaceID string, interventionID string) (InterventionView, bool) {
	if d.DB == nil || workspaceID == "" || interventionID == "" {
		return InterventionView{}, false
	}
	event, err := d.DB.GetInterventionEvent(workspaceID, interventionID)
	if err != nil || event == nil || event.Type != "v1_intervention_snapshot" {
		return InterventionView{}, false
	}
	var view InterventionView
	if err := json.Unmarshal([]byte(event.Note), &view); err != nil {
		return InterventionView{}, false
	}
	return view, true
}
