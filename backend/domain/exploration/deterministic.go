package exploration

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// DeterministicPlanner implements Planner without any LLM calls.
// Node generation is deterministic, based solely on graph state and BalanceState.
type DeterministicPlanner struct{}

// Compile-time interface check
var _ Planner = &DeterministicPlanner{}

func NewDeterministicPlanner() *DeterministicPlanner {
	return &DeterministicPlanner{}
}

// BuildInitialPlan creates a 3-step plan skeleton. Node generation happens
// separately via GenerateNodesForCycle.
func (p *DeterministicPlanner) BuildInitialPlan(ctx context.Context, session *ExplorationSession, state *RuntimeWorkspaceState) (*ExecutionPlan, []PlanStep, error) {
	return p.buildPlan(ctx, session, state, 1)
}

// Replan creates a new plan version after a trigger.
func (p *DeterministicPlanner) Replan(ctx context.Context, session *ExplorationSession, state *RuntimeWorkspaceState, trigger ReplanTrigger) (*ExecutionPlan, []PlanStep, error) {
	version := 1
	if len(state.Plans) > 0 {
		version = state.Plans[len(state.Plans)-1].Version + 1
	}
	return p.buildPlan(ctx, session, state, version)
}

func (p *DeterministicPlanner) buildPlan(_ context.Context, session *ExplorationSession, state *RuntimeWorkspaceState, version int) (*ExecutionPlan, []PlanStep, error) {
	now := time.Now()
	planID := fmt.Sprintf("plan-%s-%d", session.ID, now.UnixNano())
	runID := ""
	if len(state.Runs) > 0 {
		runID = state.Runs[len(state.Runs)-1].ID
	}
	plan := &ExecutionPlan{
		ID:          planID,
		WorkspaceID: session.ID,
		RunID:       runID,
		Version:     version,
		CreatedAt:   now.UnixMilli(),
	}
	descs := []string{
		"generate directions and evidence nodes",
		"synthesize claims from evidence",
		"produce decisions and artifacts",
	}
	steps := make([]PlanStep, len(descs))
	for i, desc := range descs {
		steps[i] = PlanStep{
			ID:          fmt.Sprintf("%s-step-%d", planID, i+1),
			WorkspaceID: session.ID,
			RunID:       runID,
			PlanID:      planID,
			Index:       i + 1,
			Desc:        desc,
			Status:      PlanStepTodo,
			UpdatedAt:   now.UnixMilli(),
		}
	}
	return plan, steps, nil
}

// GenerateNodesForCycle applies the 7-rule priority table from the spec to determine
// what nodes and edges to generate next.
func (p *DeterministicPlanner) GenerateNodesForCycle(_ context.Context, session *ExplorationSession, state *RuntimeWorkspaceState) ([]Node, []Edge) {
	balance := state.Balance

	// Collect existing nodes by type
	var dirNodes []Node
	evidenceByDir := map[string][]Node{} // dirID → Evidence nodes pointing to it
	claimByDir := map[string]bool{}
	var hasDecision bool

	for _, n := range session.Nodes {
		switch n.Type {
		case NodeDirection:
			dirNodes = append(dirNodes, n)
		case NodeEvidence:
			evidenceByDir[n.Metadata.BranchID] = append(evidenceByDir[n.Metadata.BranchID], n)
		case NodeClaim:
			claimByDir[n.Metadata.BranchID] = true
		case NodeDecision:
			hasDecision = true
		}
	}

	wsID := session.ID
	now := time.Now()

	// Rule 1: No directions yet
	if len(dirNodes) == 0 {
		return p.generateDirections(session, wsID, now)
	}

	// Find under-evidenced directions (< 2 Evidence)
	var underEvidenced []Node
	for _, dir := range dirNodes {
		if len(evidenceByDir[dir.ID]) < 2 {
			underEvidenced = append(underEvidenced, dir)
		}
	}

	// Rules 2, 3 (slow path — only when Aggression <= 0.6)
	if len(underEvidenced) > 0 && balance.Aggression <= 0.6 {
		if balance.Research >= 0.5 {
			// Rule 2: Generate Evidence
			return p.generateEvidence(underEvidenced, wsID, now)
		}
		// Rule 3: Generate Artifact (produce output)
		return p.generateArtifact(wsID, now, nil, p.latestClaim(session))
	}

	// Rule 4/5: Generate Claims for directions that don't have one
	var claimlessDirs []Node
	for _, dir := range dirNodes {
		if !claimByDir[dir.ID] {
			claimlessDirs = append(claimlessDirs, dir)
		}
	}
	if len(claimlessDirs) > 0 {
		return p.generateClaims(claimlessDirs, evidenceByDir, wsID, now)
	}

	// Rule 6: Converge → Decision
	if !hasDecision && balance.Divergence < 0.4 {
		return p.generateDecision(session, wsID, now)
	}

	// Rule 7: Diverge → Unknowns
	var hasUnknown bool
	for _, n := range session.Nodes {
		if n.Type == NodeUnknown {
			hasUnknown = true
			break
		}
	}
	if balance.Divergence >= 0.6 && !hasUnknown {
		return p.generateUnknowns(dirNodes, evidenceByDir, wsID, now)
	}

	// Rule 8: Decision exists → Artifact
	if hasDecision {
		var decNode *Node
		for i := range session.Nodes {
			if session.Nodes[i].Type == NodeDecision {
				decNode = &session.Nodes[i]
				break
			}
		}
		return p.generateArtifact(wsID, now, decNode, nil)
	}

	return nil, nil
}

func (p *DeterministicPlanner) generateDirections(session *ExplorationSession, wsID string, now time.Time) ([]Node, []Edge) {
	words := topicWords(session.Topic, 5)
	if len(words) == 0 {
		words = []string{"exploration", "direction", "approach"}
	}
	nodes := make([]Node, 0, len(words))
	for i, w := range words {
		nodes = append(nodes, Node{
			ID:          fmt.Sprintf("dir-%s-%d-%d", wsID, now.UnixNano(), i),
			WorkspaceID: session.ID,
			Type:        NodeDirection,
			Title:       strings.ToUpper(w[:1]) + w[1:] + " direction",
			Summary:     "Explore " + w + " as a strategic direction",
			Status:      NodeActive,
			Score:       0.5,
			Depth:       0,
		})
	}
	return nodes, nil // Direction nodes have no edges
}

func (p *DeterministicPlanner) generateEvidence(dirs []Node, wsID string, now time.Time) ([]Node, []Edge) {
	var nodes []Node
	var edges []Edge
	edgeTypes := []EdgeType{EdgeSupports, EdgeContradicts}
	i := 0
	for _, dir := range dirs {
		count := 2
		if len(dirs) > 3 {
			count = 1 // Limit total output for large graphs
		}
		for j := 0; j < count; j++ {
			nID := fmt.Sprintf("ev-%s-%d-%d", wsID, now.UnixNano(), i)
			nodes = append(nodes, Node{
				ID:          nID,
				WorkspaceID: wsID,
				Type:        NodeEvidence,
				Title:       fmt.Sprintf("Evidence for %s", dir.Title),
				Summary:     fmt.Sprintf("Research signal supporting or challenging: %s", dir.Title),
				Status:      NodeActive,
				Score:       0.6,
				Metadata:    NodeMetadata{BranchID: dir.ID},
			})
			edges = append(edges, Edge{
				ID:          fmt.Sprintf("edge-%s-%d-%d", wsID, now.UnixNano(), i),
				WorkspaceID: wsID,
				From:        nID,
				To:          dir.ID,
				Type:        edgeTypes[i%2],
			})
			i++
		}
	}
	return nodes, edges
}

func (p *DeterministicPlanner) generateClaims(dirs []Node, evidenceByDir map[string][]Node, wsID string, now time.Time) ([]Node, []Edge) {
	var nodes []Node
	var edges []Edge
	for i, dir := range dirs {
		nID := fmt.Sprintf("cl-%s-%d-%d", wsID, now.UnixNano(), i)
		claim := Node{
			ID:          nID,
			WorkspaceID: wsID,
			Type:        NodeClaim,
			Title:       "Claim: " + dir.Title,
			Summary:     "Synthesized assertion based on evidence for: " + dir.Title,
			Status:      NodeActive,
			Score:       0.7,
			Metadata:    NodeMetadata{BranchID: dir.ID},
		}
		nodes = append(nodes, claim)
		// Connect claim to most recent Evidence (or Direction if fast-path)
		evs := evidenceByDir[dir.ID]
		if len(evs) > 0 {
			target := evs[len(evs)-1]
			edges = append(edges, Edge{
				ID:          fmt.Sprintf("edge-cl-ev-%s-%d-%d", wsID, now.UnixNano(), i),
				WorkspaceID: wsID,
				From:        nID,
				To:          target.ID,
				Type:        EdgeJustifies,
			})
		} else {
			// Fast path: connect claim to Direction
			edges = append(edges, Edge{
				ID:          fmt.Sprintf("edge-cl-dir-%s-%d-%d", wsID, now.UnixNano(), i),
				WorkspaceID: wsID,
				From:        nID,
				To:          dir.ID,
				Type:        EdgeJustifies,
			})
		}
	}
	return nodes, edges
}

func (p *DeterministicPlanner) generateDecision(session *ExplorationSession, wsID string, now time.Time) ([]Node, []Edge) {
	nID := fmt.Sprintf("dec-%s-%d", wsID, now.UnixNano())
	decision := Node{
		ID:          nID,
		WorkspaceID: wsID,
		Type:        NodeDecision,
		Title:       "Decision for: " + session.Topic,
		Summary:     "Resolved decision synthesizing all claims",
		Status:      NodeActive,
		Score:       0.85,
	}
	var edges []Edge
	for i, n := range session.Nodes {
		if n.Type == NodeClaim {
			edges = append(edges, Edge{
				ID:          fmt.Sprintf("edge-dec-%s-%d-%d", wsID, now.UnixNano(), i),
				WorkspaceID: wsID,
				From:        nID,
				To:          n.ID,
				Type:        EdgeJustifies,
			})
		}
	}
	return []Node{decision}, edges
}

func (p *DeterministicPlanner) generateUnknowns(dirs []Node, evidenceByDir map[string][]Node, wsID string, now time.Time) ([]Node, []Edge) {
	// Find direction with fewest evidence
	minDir := dirs[0]
	for _, d := range dirs[1:] {
		if len(evidenceByDir[d.ID]) < len(evidenceByDir[minDir.ID]) {
			minDir = d
		}
	}
	nID := fmt.Sprintf("unk-%s-%d", wsID, now.UnixNano())
	unknown := Node{
		ID:          nID,
		WorkspaceID: wsID,
		Type:        NodeUnknown,
		Title:       "Open question in: " + minDir.Title,
		Summary:     "Unresolved question that needs further exploration",
		Status:      NodeActive,
		Score:       0.4,
	}
	edge := Edge{
		ID:          fmt.Sprintf("edge-unk-%s-%d", wsID, now.UnixNano()),
		WorkspaceID: wsID,
		From:        nID,
		To:          minDir.ID,
		Type:        EdgeRaises,
	}
	return []Node{unknown}, []Edge{edge}
}

func (p *DeterministicPlanner) generateArtifact(wsID string, now time.Time, decision *Node, claim *Node) ([]Node, []Edge) {
	nID := fmt.Sprintf("art-%s-%d", wsID, now.UnixNano())
	artifact := Node{
		ID:          nID,
		WorkspaceID: wsID,
		Type:        NodeArtifact,
		Title:       "Output artifact",
		Summary:     "Synthesized output from exploration",
		Status:      NodeActive,
		Score:       0.9,
	}
	var edges []Edge
	if decision != nil {
		edges = append(edges, Edge{
			ID:          fmt.Sprintf("edge-art-%s-%d", wsID, now.UnixNano()),
			WorkspaceID: wsID,
			From:        nID,
			To:          decision.ID,
			Type:        EdgeResolves,
		})
	} else if claim != nil {
		edges = append(edges, Edge{
			ID:          fmt.Sprintf("edge-art-%s-%d", wsID, now.UnixNano()),
			WorkspaceID: wsID,
			From:        nID,
			To:          claim.ID,
			Type:        EdgeResolves,
		})
	}
	return []Node{artifact}, edges
}

func (p *DeterministicPlanner) latestClaim(session *ExplorationSession) *Node {
	for i := len(session.Nodes) - 1; i >= 0; i-- {
		if session.Nodes[i].Type == NodeClaim {
			n := session.Nodes[i]
			return &n
		}
	}
	return nil
}

// topicWords splits the topic into significant words, returning up to max.
func topicWords(topic string, max int) []string {
	stopwords := map[string]bool{
		"for": true, "the": true, "and": true, "a": true, "an": true,
		"of": true, "in": true, "to": true, "with": true, "on": true,
	}
	raw := strings.Fields(strings.ToLower(topic))
	var out []string
	for _, w := range raw {
		w = strings.Trim(w, ".,!?;:\"'")
		if len(w) < 3 || stopwords[w] {
			continue
		}
		out = append(out, w)
		if len(out) >= max {
			break
		}
	}
	return out
}

// buildInitialBalance returns the default BalanceState for a new run.
func buildInitialBalance(session ExplorationSession, runID string, now time.Time) BalanceState {
	return BalanceState{
		WorkspaceID: session.ID,
		RunID:       runID,
		Divergence:  0.6,
		Research:    0.7,
		Aggression:  0.45,
		Reason:      "bootstrap exploration state from initial workspace graph",
		UpdatedAt:   now.UnixMilli(),
	}
}
