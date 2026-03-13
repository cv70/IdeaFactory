package exploration

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type branchBlueprint struct {
	Slug       string
	Label      string
	Summary    string
	Question   string
	Hypothesis string
	Ideas      []string
}

var branchBlueprints = []branchBlueprint{
	{
		Slug:       "friction",
		Label:      "Learning friction",
		Summary:    "Trace the places where the current experience breaks down.",
		Question:   "Where does the current journey stall or become too expensive to sustain?",
		Hypothesis: "Reducing setup cost and feedback delay will unlock more participation.",
		Ideas:      []string{"Lightweight pilot loop", "Feedback checkpoint pack"},
	},
	{
		Slug:       "measurement",
		Label:      "Measurable outcomes",
		Summary:    "Frame the problem around signals that can be tested quickly.",
		Question:   "Which outcome can be observed in a short cycle without overfitting?",
		Hypothesis: "Smaller, better-instrumented experiments will surface clearer signals.",
		Ideas:      []string{"Outcome-first tracker", "Fast validation protocol"},
	},
	{
		Slug:       "adoption",
		Label:      "Adoption wedge",
		Summary:    "Find an entry point where the topic becomes easy to try and share.",
		Question:   "What narrow entry point would make the topic immediately usable?",
		Hypothesis: "A sharper entry wedge creates stronger pull than broader feature coverage.",
		Ideas:      []string{"Single-scenario launch kit", "Peer distribution loop"},
	},
}

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

func createWorkspace(req CreateWorkspaceReq) ExplorationSession {
	topic := strings.TrimSpace(req.Topic)
	outputGoal := strings.TrimSpace(req.OutputGoal)
	constraints := strings.TrimSpace(req.Constraints)
	workspaceID := fmt.Sprintf("workspace-%s-%d", slugify(topic), time.Now().UnixMilli())

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

	for idx, branch := range branchBlueprints {
		branchID := fmt.Sprintf("opportunity-%s-%d", branch.Slug, idx+1)
		opportunity := makeNode(
			workspaceID,
			branchID,
			NodeOpportunity,
			fmt.Sprintf("%s for %s", branch.Label, topic),
			fmt.Sprintf("%s Optimized for %s.", branch.Summary, firstNonEmpty(outputGoal, "open-ended exploration")),
			1,
			"",
		)
		question := makeNode(
			workspaceID,
			branchID,
			NodeQuestion,
			fmt.Sprintf("%s: %s", topic, branch.Question),
			fmt.Sprintf("Question path focused on %s.", strings.ToLower(branch.Label)),
			2,
			"",
		)
		hypothesis := makeNode(
			workspaceID,
			branchID,
			NodeHypothesis,
			branch.Hypothesis,
			fmt.Sprintf("Hypothesis shaped by %s.", firstNonEmpty(constraints, "default exploration constraints")),
			3,
			"",
		)

		nodes = append(nodes, opportunity, question, hypothesis)
		edges = append(edges,
			makeEdge(topicNode.ID, opportunity.ID, EdgeRefines),
			makeEdge(opportunity.ID, question.ID, EdgeSupports),
			makeEdge(question.ID, hypothesis.ID, EdgeLeadsTo),
		)

		for ideaIdx, idea := range branch.Ideas {
			ideaNode := makeNode(
				workspaceID,
				branchID,
				NodeIdea,
				fmt.Sprintf("%s: %s", idea, topic),
				fmt.Sprintf(
					"A concrete idea generated from %s for %s.",
					strings.ToLower(branch.Label),
					firstNonEmpty(outputGoal, "general exploration"),
				),
				4,
				fmt.Sprintf("idea-%d", ideaIdx+1),
			)
			nodes = append(nodes, ideaNode)
			edges = append(edges, makeEdge(hypothesis.ID, ideaNode.ID, EdgeLeadsTo))
		}
	}

	opportunities := filterNodesByType(nodes, NodeOpportunity)
	activeOpportunityID := topicNode.ID
	if len(opportunities) > 0 {
		activeOpportunityID = opportunities[0].ID
	}
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
				Summary: fmt.Sprintf("Mapped %s into %d initial opportunity branches.", topic, len(opportunities)),
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

func (d *ExplorationDomain) CreateWorkspace(req CreateWorkspaceReq) WorkspaceSnapshot {
	session := createWorkspace(req)
	d.store.mu.Lock()
	d.store.workspaces[session.ID] = &session
	d.store.mu.Unlock()
	d.persistWorkspace(session)
	d.startRuntime(session.ID)
	return WorkspaceSnapshot{
		Exploration:  session,
		Presentation: buildWorkbenchView(session, ""),
	}
}

func (d *ExplorationDomain) GetWorkspace(workspaceID string) (WorkspaceSnapshot, bool) {
	d.store.mu.RLock()
	session, ok := d.store.workspaces[workspaceID]
	d.store.mu.RUnlock()
	if !ok {
		loaded, found := d.loadWorkspace(workspaceID)
		if !found {
			return WorkspaceSnapshot{}, false
		}
		d.store.mu.Lock()
		d.store.workspaces[workspaceID] = loaded
		d.store.mu.Unlock()
		session = loaded
		d.startRuntime(workspaceID)
	}
	copySession := *session
	return WorkspaceSnapshot{
		Exploration:  copySession,
		Presentation: buildWorkbenchView(copySession, ""),
	}, true
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

func buildWorkbenchView(session ExplorationSession, activeOpportunityID string) WorkbenchView {
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

	return WorkbenchView{
		Opportunities:     opportunities,
		ActiveOpportunity: *active,
		QuestionTrail:     byBranch(session.Nodes, branchID, NodeQuestion),
		HypothesisTrail:   byBranch(session.Nodes, branchID, NodeHypothesis),
		IdeaCards:         ideaCards,
		SavedIdeas:        savedIdeas,
		RunNotes:          session.Runs,
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

func chooseRuntimeTarget(session ExplorationSession, runtime *runtimeState) string {
	opportunities := filterNodesByType(session.Nodes, NodeOpportunity)
	if len(opportunities) == 0 {
		return session.ActiveOpportunityID
	}
	strategy := normalizeRuntimeStrategy(&session.Strategy)
	if strategy.PreferredBranchID != "" {
		for _, opportunity := range opportunities {
			if opportunity.ID == strategy.PreferredBranchID {
				return opportunity.ID
			}
		}
	}
	if strategy.ExpansionMode == "round_robin" {
		idx := runtime.cursor[session.ID] % len(opportunities)
		runtime.cursor[session.ID] = (idx + 1) % len(opportunities)
		return opportunities[idx].ID
	}
	return session.ActiveOpportunityID
}

func (d *ExplorationDomain) UpdateStrategy(workspaceID string, req UpdateStrategyReq) (WorkspaceSnapshot, []MutationEvent, bool) {
	d.store.mu.Lock()
	session, ok := d.store.workspaces[workspaceID]
	if !ok {
		d.store.mu.Unlock()
		loaded, found := d.loadWorkspace(workspaceID)
		if !found {
			return WorkspaceSnapshot{}, nil, false
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
	d.startRuntime(workspaceID)

	return WorkspaceSnapshot{
		Exploration:  next,
		Presentation: buildWorkbenchView(next, ""),
	}, mutations, true
}

func (d *ExplorationDomain) ApplyIntervention(workspaceID string, req InterventionReq) (WorkspaceSnapshot, []MutationEvent, bool) {
	d.store.mu.Lock()
	session, ok := d.store.workspaces[workspaceID]
	if !ok {
		d.store.mu.Unlock()
		loaded, found := d.loadWorkspace(workspaceID)
		if !found {
			return WorkspaceSnapshot{}, nil, false
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
	default:
		d.store.mu.Unlock()
		return WorkspaceSnapshot{}, nil, false
	}
	d.store.workspaces[workspaceID] = &next
	d.store.mu.Unlock()
	d.persistWorkspace(next)
	d.persistIntervention(workspaceID, req)
	mutations := diffMutations(prev, next, "intervention")
	d.persistMutations(mutations)

	return WorkspaceSnapshot{
		Exploration:  next,
		Presentation: buildWorkbenchView(next, ""),
	}, mutations, true
}

func (d *ExplorationDomain) CreateSession(req *CreateSessionReq) (*ExplorationSession, error) {
	snapshot := d.CreateWorkspace(CreateWorkspaceReq{
		Topic:       req.Topic,
		OutputGoal:  req.OutputGoal,
		Constraints: req.Constraints,
	})
	session := snapshot.Exploration
	return &session, nil
}
