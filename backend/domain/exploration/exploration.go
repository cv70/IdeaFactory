package exploration

import (
	"backend/datasource/dbdao"
	"fmt"
	"slices"
	"strings"
	"time"
)

func defaultRuntimeStrategy() RuntimeStrategy {
	return RuntimeStrategy{
		IntervalMs:    4000,
		MaxRuns:       30,
		ExpansionMode: "active",
	}
}

func normalizeRuntimeStrategy(input *RuntimeStrategy) RuntimeStrategy {
	base := defaultRuntimeStrategy()
	if input == nil {
		return base
	}
	out := *input
	if out.IntervalMs < 500 {
		out.IntervalMs = base.IntervalMs
	}
	if out.MaxRuns <= 0 {
		out.MaxRuns = base.MaxRuns
	}
	if out.ExpansionMode != "active" && out.ExpansionMode != "round_robin" {
		out.ExpansionMode = base.ExpansionMode
	}
	return out
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "workspace"
	}
	return out
}

func makeNode(
	workspaceID string,
	branchID string,
	nodeType NodeType,
	title string,
	summary string,
	depth int,
	slot string,
) Node {
	nodeID := fmt.Sprintf("%s-%s-%s", nodeType, slugify(title), "root")
	if branchID != "" {
		nodeID = fmt.Sprintf("%s-%s-%s", nodeType, slugify(title), branchID)
	}

	evidenceSummary := "Derived from the user topic and launch intent."
	if branchID != "" {
		evidenceSummary = fmt.Sprintf("Derived from branch %s during structured exploration.", branchID)
	}

	return Node{
		ID:            nodeID,
		SessionID:     workspaceID,
		Type:          nodeType,
		Title:         title,
		Summary:       summary,
		Status:        NodeActive,
		Score:         max(0.45, 0.92-float64(depth)*0.1),
		Depth:         depth,
		ParentContext: branchID,
		Metadata: NodeMetadata{
			BranchID: branchID,
			Slot:     slot,
		},
		EvidenceSummary: evidenceSummary,
	}
}

func makeEdge(from string, to string, edgeType EdgeType) Edge {
	return Edge{
		ID:   fmt.Sprintf("edge-%s-%s-%s", from, to, edgeType),
		From: from,
		To:   to,
		Type: edgeType,
	}
}

func createWorkspace(workspaceID string, req CreateWorkspaceReq) ExplorationSession {
	topic := strings.TrimSpace(req.Topic)
	outputGoal := strings.TrimSpace(req.OutputGoal)
	constraints := strings.TrimSpace(req.Constraints)

	topicNode := makeNode(
		workspaceID,
		"",
		NodeTopic,
		topic,
		fmt.Sprintf("Explore %s through problems, hypotheses, and idea branches.", topic),
		0,
		"",
	)

	nodes := []Node{topicNode}
	edges := []Edge{}
	activeOpportunityID := topicNode.ID
	strategy := normalizeRuntimeStrategy(req.Strategy)

	return ExplorationSession{
		ID:                  workspaceID,
		Topic:               topic,
		OutputGoal:          outputGoal,
		Constraints:         constraints,
		Strategy:            strategy,
		ActiveOpportunityID: activeOpportunityID,
		Nodes:               nodes,
		Edges:               edges,
		Favorites:           []string{},
		Runs: []GenerationRun{
			{
				ID:      "run-1",
				Round:   1,
				Focus:   activeOpportunityID,
				Summary: "Initialized workspace with topic anchor; waiting for agent-driven graph growth.",
			},
		},
	}
}

func firstNonEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func filterNodesByType(nodes []Node, nodeType NodeType) []Node {
	out := make([]Node, 0)
	for _, n := range nodes {
		if n.Type == nodeType {
			out = append(out, n)
		}
	}
	return out
}

func (d *ExplorationDomain) CreateWorkspace(req CreateWorkspaceReq) (*WorkspaceSnapshot, error) {
	workspaceID := fmt.Sprintf("workspace-%s-%d", slugify(strings.TrimSpace(req.Topic)), time.Now().UnixMilli())
	if d.DB != nil {
		state := &dbdao.WorkspaceState{
			Topic:       strings.TrimSpace(req.Topic),
			OutputGoal:  strings.TrimSpace(req.OutputGoal),
			Constraints: strings.TrimSpace(req.Constraints),
		}
		if err := d.DB.UpsertWorkspaceState(state); err != nil {
			return nil, err
		}
		workspaceID = formatWorkspaceID(state.ID)
	}
	session := createWorkspace(workspaceID, req)
	d.store.mu.Lock()
	d.store.workspaces[session.ID] = &session
	d.store.mu.Unlock()
	err := d.persistWorkspace(session)
	if err != nil {
		return nil, err
	}
	d.initializeRuntimeState(session, string(RunSourceWorkspaceCreate))
	snapshot := d.buildWorkspaceSnapshot(session, "")
	return snapshot, nil
}

func (d *ExplorationDomain) GetWorkspace(workspaceID string) (*WorkspaceSnapshot, bool) {
	if d.DB != nil {
		if workspaceDBID, err := parseWorkspaceID(workspaceID); err == nil {
			workspaceID = formatWorkspaceID(workspaceDBID)
		}
	}
	d.store.mu.RLock()
	session, ok := d.store.workspaces[workspaceID]
	d.store.mu.RUnlock()
	if !ok {
		loaded, found := d.loadWorkspace(workspaceID)
		if !found {
			return nil, false
		}
		d.store.mu.Lock()
		d.store.workspaces[workspaceID] = loaded
		d.store.mu.Unlock()
		session = loaded
		d.restoreRuntimeState(workspaceID)
		d.initializeRuntimeState(*session, string(RunSourceWorkspaceLoad))
	}
	copySession := *session
	d.restoreRuntimeState(workspaceID)
	d.initializeRuntimeState(copySession, string(RunSourceWorkspaceRead))
	snapshot := d.buildWorkspaceSnapshot(copySession, "")
	return snapshot, true
}

func byBranch(nodes []Node, branchID string, nodeType NodeType) []Node {
	out := make([]Node, 0)
	for _, n := range nodes {
		if n.Type == nodeType && n.Metadata.BranchID == branchID {
			out = append(out, n)
		}
	}
	return out
}

func buildDirectionMapProjection(session ExplorationSession) DirectionMapProjection {
	return DirectionMapProjection{
		WorkspaceID: session.ID,
		Nodes:       append([]Node{}, session.Nodes...),
		Edges:       append([]Edge{}, session.Edges...),
	}
}

func buildWorkbenchProjection(session ExplorationSession, activeOpportunityID string) WorkbenchProjection {
	opportunities := filterNodesByType(session.Nodes, NodeOpportunity)
	targetID := firstNonEmpty(activeOpportunityID, session.ActiveOpportunityID)
	var active *Node
	for idx := range opportunities {
		if opportunities[idx].ID == targetID {
			active = &opportunities[idx]
			break
		}
	}
	if active == nil && len(opportunities) > 0 {
		active = &opportunities[0]
	}
	if active == nil {
		active = &Node{
			ID:      session.ID,
			Type:    NodeTopic,
			Title:   session.Topic,
			Summary: "",
		}
	}

	branchID := firstNonEmpty(active.Metadata.BranchID, active.ID)
	ideaCards := byBranch(session.Nodes, branchID, NodeIdea)
	savedIdeas := make([]Node, 0)
	for _, idea := range ideaCards {
		if slices.Contains(session.Favorites, idea.ID) {
			savedIdeas = append(savedIdeas, idea)
		}
	}

	return WorkbenchProjection{
		Opportunities:     opportunities,
		ActiveOpportunity: *active,
		QuestionTrail:     byBranch(session.Nodes, branchID, NodeQuestion),
		HypothesisTrail:   byBranch(session.Nodes, branchID, NodeHypothesis),
		IdeaCards:         ideaCards,
		SavedIdeas:        savedIdeas,
		RunNotes:          session.Runs,
	}
}

func (d *ExplorationDomain) buildWorkspaceSnapshot(session ExplorationSession, activeOpportunityID string) *WorkspaceSnapshot {
	workbench := buildWorkbenchProjection(session, activeOpportunityID)
	if runtimeState, ok := d.GetRuntimeState(session.ID); ok {
		workbench.CurrentFocus = session.ActiveOpportunityID
		if len(runtimeState.Results) > 0 {
			workbench.LatestChange = runtimeState.Results[len(runtimeState.Results)-1].Summary
		}
		if len(runtimeState.Runs) > 0 {
			workbench.LatestRunStatus = string(runtimeState.Runs[len(runtimeState.Runs)-1].Status)
		}
		workbench.LatestReplanReason = runtimeState.LatestReplanReason
	}

	return &WorkspaceSnapshot{
		Exploration:  session,
		DirectionMap: buildDirectionMapProjection(session),
		Workbench:    workbench,
	}
}

func (d *ExplorationDomain) applyExpandOpportunity(session ExplorationSession, opportunityID string) ExplorationSession {
	var opportunity *Node
	for idx := range session.Nodes {
		if session.Nodes[idx].ID == opportunityID && session.Nodes[idx].Type == NodeOpportunity {
			opportunity = &session.Nodes[idx]
			break
		}
	}
	if opportunity == nil {
		return session
	}

	branchID := firstNonEmpty(opportunity.Metadata.BranchID, opportunity.ID)
	branchIdeas := byBranch(session.Nodes, branchID, NodeIdea)
	hypotheses := byBranch(session.Nodes, branchID, NodeHypothesis)
	if len(hypotheses) == 0 {
		return session
	}

	nextIdea := makeNode(
		session.ID,
		branchID,
		NodeIdea,
		fmt.Sprintf("Expansion %d: %s", len(branchIdeas)+1, session.Topic),
		fmt.Sprintf("A deeper materialization for %s.", strings.ToLower(opportunity.Title)),
		4,
		fmt.Sprintf("idea-%d", len(branchIdeas)+1),
	)

	session.ActiveOpportunityID = opportunityID
	session.Nodes = append(session.Nodes, nextIdea)
	session.Edges = append(session.Edges, makeEdge(hypotheses[0].ID, nextIdea.ID, EdgeExpands))
	session.Runs = append(session.Runs, GenerationRun{
		ID:      fmt.Sprintf("run-%d", len(session.Runs)+1),
		Round:   len(session.Runs) + 1,
		Focus:   opportunityID,
		Summary: fmt.Sprintf("Expanded %s with one more materialized idea.", opportunity.Title),
	})
	return session
}

func (d *ExplorationDomain) applyToggleFavorite(session ExplorationSession, ideaID string) ExplorationSession {
	exists := slices.Contains(session.Favorites, ideaID)
	if exists {
		next := make([]string, 0, len(session.Favorites)-1)
		for _, id := range session.Favorites {
			if id != ideaID {
				next = append(next, id)
			}
		}
		session.Favorites = next
		return session
	}
	session.Favorites = append(session.Favorites, ideaID)
	return session
}

func (d *ExplorationDomain) UpdateStrategy(workspaceID string, req UpdateStrategyReq) (*WorkspaceSnapshot, []MutationEvent, bool) {
	d.store.mu.Lock()
	session, ok := d.store.workspaces[workspaceID]
	if !ok {
		d.store.mu.Unlock()
		loaded, found := d.loadWorkspace(workspaceID)
		if !found {
			return nil, nil, false
		}
		d.store.mu.Lock()
		d.store.workspaces[workspaceID] = loaded
		session = loaded
	}

	prev := *session
	next := *session
	strategy := normalizeRuntimeStrategy(&next.Strategy)
	if req.IntervalMs != nil {
		strategy.IntervalMs = *req.IntervalMs
	}
	if req.MaxRuns != nil {
		strategy.MaxRuns = *req.MaxRuns
	}
	if req.ExpansionMode != nil {
		strategy.ExpansionMode = *req.ExpansionMode
	}
	if req.PreferredBranchID != nil {
		strategy.PreferredBranchID = *req.PreferredBranchID
	}
	next.Strategy = normalizeRuntimeStrategy(&strategy)

	d.store.workspaces[workspaceID] = &next
	d.store.mu.Unlock()
	d.persistWorkspace(next)

	mutations := diffMutations(prev, next, "intervention")
	if len(mutations) == 0 {
		mutations = append(mutations, MutationEvent{
			ID:          mutationID(workspaceID),
			WorkspaceID: workspaceID,
			Kind:        "strategy_updated",
			Source:      "intervention",
			Strategy:    &next.Strategy,
			CreatedAt:   time.Now().UnixMilli(),
		})
	}
	d.persistMutations(mutations)
	snapshot := d.buildWorkspaceSnapshot(next, "")
	return snapshot, mutations, true
}

func (d *ExplorationDomain) ApplyIntervention(workspaceID string, req InterventionReq) (*WorkspaceSnapshot, []MutationEvent, bool) {
	d.store.mu.Lock()
	session, ok := d.store.workspaces[workspaceID]
	if !ok {
		d.store.mu.Unlock()
		loaded, found := d.loadWorkspace(workspaceID)
		if !found {
			return nil, nil, false
		}
		d.store.mu.Lock()
		d.store.workspaces[workspaceID] = loaded
		session = loaded
	}

	next := *session
	prev := *session
	switch req.Type {
	case InterventionExpandOpportunity:
		next = d.applyExpandOpportunity(next, req.TargetID)
	case InterventionToggleFavorite:
		next = d.applyToggleFavorite(next, req.TargetID)
	case InterventionShiftFocus:
		next.ActiveOpportunityID = req.TargetID
	case InterventionAdjustIntensity:
		// Runtime balance state absorbs this intervention.
	case InterventionAddContext:
		// Runtime replanning absorbs this intervention.
	default:
		d.store.mu.Unlock()
		return nil, nil, false
	}
	d.store.workspaces[workspaceID] = &next
	d.store.mu.Unlock()
	d.persistWorkspace(next)
	d.persistIntervention(workspaceID, req)
	d.replanRuntimeState(next, req)
	mutations := diffMutations(prev, next, "intervention")
	d.persistMutations(mutations)

	return d.buildWorkspaceSnapshot(next, ""), mutations, true
}

func (d *ExplorationDomain) CreateSession(req *CreateSessionReq) (*ExplorationSession, error) {
	snapshot, err := d.CreateWorkspace(CreateWorkspaceReq{
		Topic:       req.Topic,
		OutputGoal:  req.OutputGoal,
		Constraints: req.Constraints,
	})
	if err != nil {
		return nil, err
	}
	session := snapshot.Exploration
	return &session, nil
}
