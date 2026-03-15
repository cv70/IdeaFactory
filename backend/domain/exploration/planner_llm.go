package exploration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/kaptinlin/jsonrepair"
)

// LLMPlanner implements Planner by dispatching to specialist agents for node generation.
// GenerateNodesForCycle must be called OUTSIDE withWorkspaceState (it makes network calls).
// Any nil agent or parse failure falls back to DeterministicPlanner for that step.
type LLMPlanner struct {
	general  adk.Agent // Priority 1: Direction generation
	research adk.Agent // Priority 2: Evidence (ResearchAgent, has DuckDuckGo)
	graph    adk.Agent // Priority 3: Claim synthesis
	artifact adk.Agent // Priority 4: Decision + Artifact
	fallback *DeterministicPlanner
}

// NewLLMPlanner creates an LLMPlanner. Any agent may be nil — nil agents fall back to deterministic.
func NewLLMPlanner(general, research, graph, artifact adk.Agent) *LLMPlanner {
	return &LLMPlanner{
		general:  general,
		research: research,
		graph:    graph,
		artifact: artifact,
		fallback: NewDeterministicPlanner(),
	}
}

// Compile-time interface check
var _ Planner = &LLMPlanner{}

func (p *LLMPlanner) BuildInitialPlan(ctx context.Context, session *ExplorationSession, state *RuntimeWorkspaceState) (*ExecutionPlan, []PlanStep, error) {
	return p.fallback.BuildInitialPlan(ctx, session, state)
}

func (p *LLMPlanner) Replan(ctx context.Context, session *ExplorationSession, state *RuntimeWorkspaceState, trigger ReplanTrigger) (*ExecutionPlan, []PlanStep, error) {
	return p.fallback.Replan(ctx, session, state, trigger)
}

// GenerateNodesForCycle selects the highest-priority node type to generate,
// calls the appropriate agent, parses JSON output, and falls back to
// DeterministicPlanner on any error. Must be called outside withWorkspaceState.
func (p *LLMPlanner) GenerateNodesForCycle(ctx context.Context, session *ExplorationSession, state *RuntimeWorkspaceState) ([]Node, []Edge) {
	wsID := session.ID
	now := time.Now()
	balance := state.Balance

	// Collect graph state (mirrors DeterministicPlanner logic)
	var dirNodes []Node
	evidenceByDir := map[string][]Node{}
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

	// Priority 1: No directions
	if len(dirNodes) == 0 {
		return p.genDirections(ctx, session, wsID, now)
	}

	// Priority 2: Under-evidenced directions
	var underEvidenced []Node
	for _, dir := range dirNodes {
		if len(evidenceByDir[dir.ID]) < 2 {
			underEvidenced = append(underEvidenced, dir)
		}
	}
	if len(underEvidenced) > 0 && balance.Aggression <= 0.6 && balance.Research >= 0.5 {
		return p.genEvidence(ctx, session, underEvidenced, evidenceByDir, wsID, now)
	}

	// Priority 3: Directions missing claims
	var claimlessDirs []Node
	for _, dir := range dirNodes {
		if !claimByDir[dir.ID] {
			claimlessDirs = append(claimlessDirs, dir)
		}
	}
	if len(claimlessDirs) > 0 {
		return p.genClaims(ctx, session, claimlessDirs, evidenceByDir, wsID, now)
	}

	// Priority 4: Converge to decision + artifact
	if !hasDecision && balance.Divergence < 0.4 {
		return p.genArtifact(ctx, session, wsID, now)
	}

	return nil, nil
}

// genDirections calls the general agent to generate Direction nodes.
func (p *LLMPlanner) genDirections(ctx context.Context, session *ExplorationSession, wsID string, now time.Time) ([]Node, []Edge) {
	if p.general == nil {
		return p.fallback.generateDirections(session, wsID, now)
	}
	prompt := fmt.Sprintf(
		"Topic: %s\nOutput goal: %s\n\nGenerate 3-5 distinct exploration directions for this topic.\nReturn ONLY valid JSON: {\"directions\":[{\"title\":\"...\",\"summary\":\"...\"}]}",
		session.Topic, session.OutputGoal,
	)
	text, err := runAgent(ctx, p.general, prompt)
	if err != nil {
		return p.fallback.generateDirections(session, wsID, now)
	}
	var parsed struct {
		Directions []struct {
			Title   string `json:"title"`
			Summary string `json:"summary"`
		} `json:"directions"`
	}
	if err := parseJSON(text, &parsed); err != nil || len(parsed.Directions) == 0 {
		return p.fallback.generateDirections(session, wsID, now)
	}
	var nodes []Node
	for i, d := range parsed.Directions {
		if strings.TrimSpace(d.Title) == "" {
			continue
		}
		nodes = append(nodes, Node{
			ID:          fmt.Sprintf("dir-%s-%d-%d", wsID, now.UnixNano(), i),
			WorkspaceID: wsID,
			Type:        NodeDirection,
			Title:       strings.TrimSpace(d.Title),
			Summary:     strings.TrimSpace(d.Summary),
			Status:      NodeActive,
			Score:       0.5,
		})
	}
	if len(nodes) == 0 {
		return p.fallback.generateDirections(session, wsID, now)
	}
	return nodes, nil
}

// genEvidence calls the research agent to generate Evidence nodes.
func (p *LLMPlanner) genEvidence(ctx context.Context, session *ExplorationSession, underEvidenced []Node, evidenceByDir map[string][]Node, wsID string, now time.Time) ([]Node, []Edge) {
	if p.research == nil {
		return p.fallback.generateEvidence(underEvidenced, wsID, now)
	}
	var dirLines []string
	for _, dir := range underEvidenced {
		dirLines = append(dirLines, fmt.Sprintf("- %s: %s", dir.ID, dir.Title))
	}
	prompt := fmt.Sprintf(
		"Topic: %s\n\nDirections needing evidence (use these EXACT IDs in direction_id):\n%s\n\nGenerate 1-2 evidence items per direction. Return ONLY valid JSON:\n{\"evidence\":[{\"title\":\"...\",\"summary\":\"...\",\"edge_type\":\"supports\",\"direction_id\":\"<exact-id-from-above>\"}]}",
		session.Topic, strings.Join(dirLines, "\n"),
	)
	text, err := runAgent(ctx, p.research, prompt)
	if err != nil {
		return p.fallback.generateEvidence(underEvidenced, wsID, now)
	}
	var parsed struct {
		Evidence []struct {
			Title       string `json:"title"`
			Summary     string `json:"summary"`
			EdgeType    string `json:"edge_type"`
			DirectionID string `json:"direction_id"`
		} `json:"evidence"`
	}
	if err := parseJSON(text, &parsed); err != nil {
		return p.fallback.generateEvidence(underEvidenced, wsID, now)
	}
	// Validate direction_ids
	validDirIDs := map[string]bool{}
	for _, dir := range underEvidenced {
		validDirIDs[dir.ID] = true
	}
	var nodes []Node
	var edges []Edge
	for i, ev := range parsed.Evidence {
		if !validDirIDs[ev.DirectionID] || strings.TrimSpace(ev.Title) == "" {
			continue
		}
		et := EdgeSupports
		if strings.ToLower(ev.EdgeType) == "contradicts" {
			et = EdgeContradicts
		}
		nID := fmt.Sprintf("ev-%s-%d-%d", wsID, now.UnixNano(), i)
		nodes = append(nodes, Node{
			ID:          nID,
			WorkspaceID: wsID,
			Type:        NodeEvidence,
			Title:       strings.TrimSpace(ev.Title),
			Summary:     strings.TrimSpace(ev.Summary),
			Status:      NodeActive,
			Score:       0.6,
			Metadata:    NodeMetadata{BranchID: ev.DirectionID},
		})
		edges = append(edges, Edge{
			ID:          fmt.Sprintf("edge-%s-%d-%d", wsID, now.UnixNano(), i),
			WorkspaceID: wsID,
			From:        nID,
			To:          ev.DirectionID,
			Type:        et,
		})
	}
	if len(nodes) == 0 {
		return p.fallback.generateEvidence(underEvidenced, wsID, now)
	}
	return nodes, edges
}

// genClaims calls the graph agent to synthesize Claim nodes.
func (p *LLMPlanner) genClaims(ctx context.Context, session *ExplorationSession, claimlessDirs []Node, evidenceByDir map[string][]Node, wsID string, now time.Time) ([]Node, []Edge) {
	if p.graph == nil {
		return p.fallback.generateClaims(claimlessDirs, evidenceByDir, wsID, now)
	}
	var dirLines []string
	for _, dir := range claimlessDirs {
		evSummaries := []string{}
		for _, ev := range evidenceByDir[dir.ID] {
			evSummaries = append(evSummaries, ev.Summary)
		}
		line := fmt.Sprintf("- %s: %s", dir.ID, dir.Title)
		if len(evSummaries) > 0 {
			line += fmt.Sprintf(" [evidence: %s]", strings.Join(evSummaries, "; "))
		}
		dirLines = append(dirLines, line)
	}
	prompt := fmt.Sprintf(
		"Topic: %s\n\nDirections with evidence (use EXACT IDs in direction_id):\n%s\n\nSynthesize one core claim per direction. Return ONLY valid JSON:\n{\"claims\":[{\"title\":\"...\",\"summary\":\"...\",\"direction_id\":\"<exact-id>\"}]}",
		session.Topic, strings.Join(dirLines, "\n"),
	)
	text, err := runAgent(ctx, p.graph, prompt)
	if err != nil {
		return p.fallback.generateClaims(claimlessDirs, evidenceByDir, wsID, now)
	}
	var parsed struct {
		Claims []struct {
			Title       string `json:"title"`
			Summary     string `json:"summary"`
			DirectionID string `json:"direction_id"`
		} `json:"claims"`
	}
	if err := parseJSON(text, &parsed); err != nil {
		return p.fallback.generateClaims(claimlessDirs, evidenceByDir, wsID, now)
	}
	validDirIDs := map[string]bool{}
	for _, dir := range claimlessDirs {
		validDirIDs[dir.ID] = true
	}
	var nodes []Node
	var edges []Edge
	for i, cl := range parsed.Claims {
		if !validDirIDs[cl.DirectionID] || strings.TrimSpace(cl.Title) == "" {
			continue
		}
		nID := fmt.Sprintf("cl-%s-%d-%d", wsID, now.UnixNano(), i)
		nodes = append(nodes, Node{
			ID:          nID,
			WorkspaceID: wsID,
			Type:        NodeClaim,
			Title:       strings.TrimSpace(cl.Title),
			Summary:     strings.TrimSpace(cl.Summary),
			Status:      NodeActive,
			Score:       0.7,
			Metadata:    NodeMetadata{BranchID: cl.DirectionID},
		})
		evs := evidenceByDir[cl.DirectionID]
		if len(evs) > 0 {
			edges = append(edges, Edge{
				ID:          fmt.Sprintf("edge-cl-ev-%s-%d-%d", wsID, now.UnixNano(), i),
				WorkspaceID: wsID, From: nID, To: evs[len(evs)-1].ID, Type: EdgeJustifies,
			})
		} else {
			edges = append(edges, Edge{
				ID:          fmt.Sprintf("edge-cl-dir-%s-%d-%d", wsID, now.UnixNano(), i),
				WorkspaceID: wsID, From: nID, To: cl.DirectionID, Type: EdgeJustifies,
			})
		}
	}
	if len(nodes) == 0 {
		return p.fallback.generateClaims(claimlessDirs, evidenceByDir, wsID, now)
	}
	return nodes, edges
}

// genArtifact calls the artifact agent to generate Decision + Artifact nodes.
// Fallback (nil agent or error): calls generateDecision only; Artifact follows next cycle
// via DeterministicPlanner Rule 8 (since Decision node will now exist).
func (p *LLMPlanner) genArtifact(ctx context.Context, session *ExplorationSession, wsID string, now time.Time) ([]Node, []Edge) {
	var claimNodes []Node
	for _, n := range session.Nodes {
		if n.Type == NodeClaim {
			claimNodes = append(claimNodes, n)
		}
	}
	if p.artifact == nil {
		return p.fallback.generateDecision(session, wsID, now)
	}
	var claimLines []string
	for _, cl := range claimNodes {
		claimLines = append(claimLines, fmt.Sprintf("- %s: %s", cl.Title, cl.Summary))
	}
	prompt := fmt.Sprintf(
		"Topic: %s\nClaims:\n%s\n\nGenerate a decision and artifact. Return ONLY valid JSON:\n{\"decision\":{\"title\":\"...\",\"summary\":\"...\"},\"artifact\":{\"title\":\"...\",\"summary\":\"...\"}}",
		session.Topic, strings.Join(claimLines, "\n"),
	)
	text, err := runAgent(ctx, p.artifact, prompt)
	if err != nil {
		return p.fallback.generateDecision(session, wsID, now)
	}
	var parsed struct {
		Decision struct {
			Title   string `json:"title"`
			Summary string `json:"summary"`
		} `json:"decision"`
		Artifact struct {
			Title   string `json:"title"`
			Summary string `json:"summary"`
		} `json:"artifact"`
	}
	if err := parseJSON(text, &parsed); err != nil || strings.TrimSpace(parsed.Decision.Title) == "" {
		return p.fallback.generateDecision(session, wsID, now)
	}
	decID := fmt.Sprintf("dec-%s-%d", wsID, now.UnixNano())
	artID := fmt.Sprintf("art-%s-%d", wsID, now.UnixNano())
	var nodes []Node
	var edges []Edge
	nodes = append(nodes, Node{
		ID: decID, WorkspaceID: wsID, Type: NodeDecision,
		Title: strings.TrimSpace(parsed.Decision.Title), Summary: strings.TrimSpace(parsed.Decision.Summary),
		Status: NodeActive, Score: 0.85,
	})
	for i, cl := range claimNodes {
		edges = append(edges, Edge{
			ID:          fmt.Sprintf("edge-dec-%s-%d-%d", wsID, now.UnixNano(), i),
			WorkspaceID: wsID, From: decID, To: cl.ID, Type: EdgeJustifies,
		})
	}
	if strings.TrimSpace(parsed.Artifact.Title) != "" {
		nodes = append(nodes, Node{
			ID: artID, WorkspaceID: wsID, Type: NodeArtifact,
			Title: strings.TrimSpace(parsed.Artifact.Title), Summary: strings.TrimSpace(parsed.Artifact.Summary),
			Status: NodeActive, Score: 0.9,
		})
		edges = append(edges, Edge{
			ID:          fmt.Sprintf("edge-art-%s-%d", wsID, now.UnixNano()),
			WorkspaceID: wsID, From: artID, To: decID, Type: EdgeResolves,
		})
	}
	return nodes, edges
}

// runAgent calls an agent with a text prompt and extracts the final text response.
func runAgent(ctx context.Context, agent adk.Agent, prompt string) (string, error) {
	input := &adk.AgentInput{
		Messages: []*schema.Message{schema.UserMessage(prompt)},
	}
	iter := agent.Run(ctx, input)
	var lastContent string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			msg, err := event.Output.MessageOutput.GetMessage()
			if err == nil && msg != nil && strings.TrimSpace(msg.Content) != "" {
				lastContent = msg.Content
			}
		}
	}
	if lastContent == "" {
		return "", fmt.Errorf("agent returned empty content")
	}
	return lastContent, nil
}

// parseJSON repairs and unmarshals JSON agent output.
func parseJSON(text string, dest any) error {
	repaired, err := jsonrepair.Repair(text)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(repaired), dest)
}
