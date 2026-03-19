package exploration

import (
	"backend/datasource/dbdao"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

func (d *ExplorationDomain) persistWorkspace(session ExplorationSession) error {
	if d.DB == nil {
		return nil
	}
	workspaceID, err := parseWorkspaceID(session.ID)
	if err != nil {
		return err
	}

	strategyRaw, err := json.Marshal(session.Strategy)
	if err != nil {
		return err
	}
	favoritesRaw, err := json.Marshal(session.Favorites)
	if err != nil {
		return err
	}
	runsRaw, err := json.Marshal(session.Runs)
	if err != nil {
		return err
	}

	state := &dbdao.WorkspaceState{
		Model:               gorm.Model{ID: workspaceID},
		Topic:               session.Topic,
		OutputGoal:          session.OutputGoal,
		Constraints:         session.Constraints,
		Strategy:            string(strategyRaw),
		Favorites:           string(favoritesRaw),
		RunNotes:            string(runsRaw),
		ActiveOpportunityID: session.ActiveOpportunityID,
		LastRunRound:        len(session.Runs),
	}
	// Preserve lifecycle fields that are managed via dedicated APIs.
	if existing, err2 := d.DB.GetWorkspaceState(workspaceID); err2 == nil && existing != nil {
		state.CreatedAt = existing.CreatedAt
		state.PausedAt = existing.PausedAt
		state.ArchivedAt = existing.ArchivedAt
	}
	if err := d.DB.UpsertWorkspaceState(state); err != nil {
		return err
	}

	nodes := make([]dbdao.GraphNode, 0, len(session.Nodes))
	for _, node := range session.Nodes {
		metaRaw, _ := json.Marshal(node.Metadata)
		decisionRaw := ""
		if node.Decision != nil {
			decisionBytes, _ := json.Marshal(node.Decision)
			decisionRaw = string(decisionBytes)
		}
		nodes = append(nodes, dbdao.GraphNode{
			WorkspaceID: workspaceID,
			SessionID:   session.ID,
			NodeID:      node.ID,
			Type:        dbdao.NodeType(node.Type),
			Title:       node.Title,
			Summary:     node.Summary,
			Body:        node.ParentContext,
			Status:      dbdao.Status(node.Status),
			Score:       node.Score,
			Depth:       node.Depth,
			Metadata:    string(metaRaw),
			Evidence:    node.EvidenceSummary,
			Decision:    decisionRaw,
		})
	}
	edges := make([]dbdao.GraphEdge, 0, len(session.Edges))
	for _, edge := range session.Edges {
		edges = append(edges, dbdao.GraphEdge{
			WorkspaceID: workspaceID,
			SessionID:   session.ID,
			FromID:      edge.From,
			ToID:        edge.To,
			Type:        dbdao.EdgeType(edge.Type),
		})
	}
	return d.DB.ReplaceWorkspaceGraph(workspaceID, nodes, edges)
}

func (d *ExplorationDomain) loadWorkspace(workspaceID string) (*ExplorationSession, bool) {
	if d.DB == nil {
		return nil, false
	}
	workspaceDBID, err := parseWorkspaceID(workspaceID)
	if err != nil {
		return nil, false
	}
	workspaceID = formatWorkspaceID(workspaceDBID)
	state, err := d.DB.GetWorkspaceState(workspaceDBID)
	if err != nil || state == nil {
		return nil, false
	}
	nodes, edges, err := d.DB.GetWorkspaceGraph(workspaceDBID)
	if err != nil {
		return nil, false
	}

	var strategy RuntimeStrategy
	if strings.TrimSpace(state.Strategy) != "" {
		if err := json.Unmarshal([]byte(state.Strategy), &strategy); err != nil {
			return nil, false
		}
	}
	favorites := []string{}
	if strings.TrimSpace(state.Favorites) != "" {
		if err := json.Unmarshal([]byte(state.Favorites), &favorites); err != nil {
			return nil, false
		}
	}
	runNotes := []GenerationRun{}
	if strings.TrimSpace(state.RunNotes) != "" {
		if err := json.Unmarshal([]byte(state.RunNotes), &runNotes); err != nil {
			return nil, false
		}
	}

	outNodes := make([]Node, 0, len(nodes))
	for _, item := range nodes {
		var metadata NodeMetadata
		if strings.TrimSpace(item.Metadata) != "" {
			_ = json.Unmarshal([]byte(item.Metadata), &metadata)
		}
		var decision *Decision
		if strings.TrimSpace(item.Decision) != "" {
			var parsed Decision
			if err := json.Unmarshal([]byte(item.Decision), &parsed); err == nil {
				decision = &parsed
			}
		}
		outNodes = append(outNodes, Node{
			ID:              item.NodeID,
			WorkspaceID:     workspaceID,
			SessionID:       item.SessionID,
			Type:            NodeType(item.Type),
			Title:           item.Title,
			Summary:         item.Summary,
			Status:          NodeStatus(item.Status),
			Score:           item.Score,
			Depth:           item.Depth,
			ParentContext:   item.Body,
			Metadata:        metadata,
			EvidenceSummary: item.Evidence,
			Decision:        decision,
		})
	}
	outEdges := make([]Edge, 0, len(edges))
	for _, item := range edges {
		outEdges = append(outEdges, Edge{
			ID:          strconv.FormatUint(uint64(item.ID), 10),
			WorkspaceID: workspaceID,
			From:        item.FromID,
			To:          item.ToID,
			Type:        EdgeType(item.Type),
		})
	}

	session := ExplorationSession{
		ID:                  workspaceID,
		Topic:               state.Topic,
		OutputGoal:          state.OutputGoal,
		Constraints:         state.Constraints,
		Strategy:            strategy,
		ActiveOpportunityID: state.ActiveOpportunityID,
		Nodes:               outNodes,
		Edges:               outEdges,
		Favorites:           favorites,
		Runs:                runNotes,
	}
	return &session, true
}

func (d *ExplorationDomain) persistRuntimeState(workspaceID string) {
	if d.DB == nil {
		return
	}
	workspaceDBID, err := parseWorkspaceID(workspaceID)
	if err != nil {
		return
	}
	snapshot, ok := d.GetRuntimeState(workspaceID)
	if !ok {
		return
	}
	if snapshot.Balance.WorkspaceID != "" {
		balanceWorkspaceID, err := parseWorkspaceID(snapshot.Balance.WorkspaceID)
		if err != nil {
			balanceWorkspaceID = workspaceDBID
		}
		balance := &dbdao.RuntimeBalanceRecord{
			WorkspaceID:        balanceWorkspaceID,
			RunID:              snapshot.Balance.RunID,
			Divergence:         snapshot.Balance.Divergence,
			Research:           snapshot.Balance.Research,
			Aggression:         snapshot.Balance.Aggression,
			Reason:             snapshot.Balance.Reason,
			UpdatedAtMs:        snapshot.Balance.UpdatedAt,
			LatestReplanReason: snapshot.LatestReplanReason,
		}
		_ = d.DB.DB().
			Where("workspace_id = ?", balanceWorkspaceID).
			Delete(&dbdao.RuntimeBalanceRecord{}).Error
		_ = d.DB.DB().Create(balance).Error
	}
}

func (d *ExplorationDomain) loadRuntimeState(workspaceID string) (RuntimeStateSnapshot, bool) {
	if d.DB == nil {
		return RuntimeStateSnapshot{}, false
	}
	workspaceDBID, err := parseWorkspaceID(workspaceID)
	if err != nil {
		return RuntimeStateSnapshot{}, false
	}
	records, err := d.DB.ListAgentRunRecords(workspaceDBID)
	if err != nil {
		return RuntimeStateSnapshot{}, false
	}
	out := buildRuntimeStateFromAgentRunRecords(formatWorkspaceID(workspaceDBID), records)

	var balance dbdao.RuntimeBalanceRecord
	err = d.DB.DB().Where("workspace_id = ?", workspaceDBID).First(&balance).Error
	if err == nil {
		out.Balance = BalanceState{
			WorkspaceID: formatWorkspaceID(balance.WorkspaceID),
			RunID:       balance.RunID,
			Divergence:  balance.Divergence,
			Research:    balance.Research,
			Aggression:  balance.Aggression,
			Reason:      balance.Reason,
			UpdatedAt:   balance.UpdatedAtMs,
		}
		out.LatestReplanReason = balance.LatestReplanReason
	}
	if len(out.Runs) == 0 && out.Balance.WorkspaceID == "" {
		return RuntimeStateSnapshot{}, false
	}
	return out, true
}

func buildRuntimeStateFromAgentRunRecords(workspaceID string, records []dbdao.AgentRunRecord) RuntimeStateSnapshot {
	out := RuntimeStateSnapshot{
		Runs:       []Run{},
		AgentTasks: []AgentTask{},
		Results:    []AgentTaskResultSummary{},
		Events:     []AgentRunEvent{},
	}
	runIndex := map[string]int{}
	runTimelines := map[string][]string{}
	for _, record := range records {
		var payload map[string]any
		if strings.TrimSpace(record.Payload) != "" {
			_ = json.Unmarshal([]byte(record.Payload), &payload)
		}
		event := AgentRunEvent{
			ID:          strconv.FormatUint(uint64(record.ID), 10),
			WorkspaceID: formatWorkspaceID(record.WorkspaceID),
			RunID:       record.RunID,
			RootAgent:   record.RootAgent,
			EventType:   record.EventType,
			Actor:       record.Actor,
			Target:      record.Target,
			Summary:     record.Summary,
			Payload:     payload,
			CreatedAt:   record.CreatedAt.UnixMilli(),
		}
		out.Events = append(out.Events, event)

		idx, ok := runIndex[record.RunID]
		if !ok {
			out.Runs = append(out.Runs, Run{
				ID:          record.RunID,
				WorkspaceID: workspaceID,
				Status:      RunStatusRunning,
				StartedAt:   record.CreatedAt.UnixMilli(),
			})
			idx = len(out.Runs) - 1
			runIndex[record.RunID] = idx
		}
		run := &out.Runs[idx]
		switch record.EventType {
		case "agent_start":
			if source, _ := payload["source"].(string); source != "" {
				run.Source = source
			}
		case "agent_delegate":
			if record.Target != "" {
				runTimelines[record.RunID] = appendTimelineValue(runTimelines[record.RunID], record.Target)
			}
		case "tool_call":
			if record.Actor != "" {
				runTimelines[record.RunID] = appendTimelineValue(runTimelines[record.RunID], record.Actor)
			}
		case "run_summary":
			run.Status = RunStatusCompleted
			run.EndedAt = record.CreatedAt.UnixMilli()
			leadActor, _ := payload["lead_actor"].(string)
			timeline := append([]string{}, runTimelines[record.RunID]...)
			if len(timeline) == 0 && leadActor != "" {
				timeline = append(timeline, leadActor)
			}
			timeline = appendTimelineValue(timeline, "SUMMARY")
			taskID := "summary-" + strconv.FormatUint(uint64(record.ID), 10)
			subAgent := firstNonEmpty(leadActor, record.Actor)
			if subAgent == "" {
				subAgent = string(RuntimeActorMainAgent)
			}
			out.AgentTasks = append(out.AgentTasks, AgentTask{
				ID:          taskID,
				WorkspaceID: workspaceID,
				RunID:       record.RunID,
				SubAgent:    subAgent,
				Goal:        MainAgentCycleGoal,
				Status:      RuntimeTaskDone,
				UpdatedAt:   record.CreatedAt.UnixMilli(),
			})
			out.Results = append(out.Results, AgentTaskResultSummary{
				TaskID:    taskID,
				Summary:   record.Summary,
				Timeline:  timeline,
				IsSuccess: true,
				UpdatedAt: record.CreatedAt.UnixMilli(),
			})
		case "run_error":
			run.Status = RunStatusFailed
			run.EndedAt = record.CreatedAt.UnixMilli()
			taskID := "error-" + strconv.FormatUint(uint64(record.ID), 10)
			out.AgentTasks = append(out.AgentTasks, AgentTask{
				ID:          taskID,
				WorkspaceID: workspaceID,
				RunID:       record.RunID,
				SubAgent:    firstNonEmpty(record.Actor, string(RuntimeActorMainAgent)),
				Goal:        MainAgentCycleGoal,
				Status:      RuntimeTaskFailed,
				UpdatedAt:   record.CreatedAt.UnixMilli(),
			})
			out.Results = append(out.Results, AgentTaskResultSummary{
				TaskID:    taskID,
				Summary:   record.Summary,
				Timeline:  appendTimelineValue(runTimelines[record.RunID], "ERROR"),
				IsSuccess: false,
				UpdatedAt: record.CreatedAt.UnixMilli(),
			})
		}
	}
	return out
}

func appendTimelineValue(values []string, next string) []string {
	next = strings.TrimSpace(next)
	if next == "" {
		return values
	}
	if len(values) > 0 && values[len(values)-1] == next {
		return values
	}
	return append(values, next)
}

func (d *ExplorationDomain) persistAgentRunEvents(events []AgentRunEvent) {
	if d.DB == nil || len(events) == 0 {
		return
	}
	records := make([]dbdao.AgentRunRecord, 0, len(events))
	for _, event := range events {
		workspaceDBID, err := parseWorkspaceID(event.WorkspaceID)
		if err != nil {
			continue
		}
		payloadRaw := ""
		if len(event.Payload) > 0 {
			raw, _ := json.Marshal(event.Payload)
			payloadRaw = string(raw)
		}
		record := dbdao.AgentRunRecord{
			Model: gorm.Model{
				CreatedAt: time.UnixMilli(event.CreatedAt),
				UpdatedAt: time.UnixMilli(event.CreatedAt),
			},
			WorkspaceID: workspaceDBID,
			RunID:       event.RunID,
			RootAgent:   event.RootAgent,
			EventType:   event.EventType,
			Actor:       event.Actor,
			Target:      event.Target,
			Summary:     event.Summary,
			Payload:     payloadRaw,
		}
		if event.ID != "" {
			if parsedID, convErr := strconv.ParseUint(event.ID, 10, 64); convErr == nil {
				record.ID = uint(parsedID)
			}
		}
		records = append(records, record)
	}
	_ = d.DB.AppendAgentRunRecords(records)
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

	workspaceDBID, err := parseWorkspaceID(workspaceID)
	if err != nil {
		return
	}
	state, err := d.DB.GetWorkspaceState(workspaceDBID)
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
	event, err := d.DB.GetLatestInterventionEventByTarget(workspaceID, interventionID, "v1_intervention_snapshot")
	if err != nil || event == nil || event.Type != "v1_intervention_snapshot" {
		return InterventionView{}, false
	}
	var view InterventionView
	if err := json.Unmarshal([]byte(event.Note), &view); err != nil {
		return InterventionView{}, false
	}
	return view, true
}
