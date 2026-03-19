package exploration

import (
	"fmt"
	"strings"
)

func (d *ExplorationDomain) buildMainAgentGraphPrompt(session *ExplorationSession, state *RuntimeWorkspaceState) string {
	return fmt.Sprintf(
		`You are the exploration runtime coordinator for IdeaFactory.

Run the agent system for this workspace now. You may coordinate specialist subagents, but the final responsibility is yours.
Grow the graph only when the current workspace state justifies it, and use %q for every graph mutation. Do not rewrite existing nodes or edges outside that tool.

Workspace:
- workspace_id: %s
- topic: %s
- output_goal: %s
- constraints: %s
- active_opportunity_id: %s
- latest_replan_reason: %s
- balance: divergence=%.2f research=%.2f aggression=%.2f

Current graph summary:
- node_count: %d
- edge_count: %d
- recent_nodes: %s

Rules:
1. Use append-only mutations.
2. Node types must be one of: %s
3. Edge types must be one of: %s
4. Node status must be one of: %s
5. Edge endpoints must reference an existing node id or a node created in the same batch.
6. Prefer GraphAgent for graph growth and graph-structure decisions.
7. Use ResearchAgent only when an evidence gap blocks a concrete graph mutation right now.
8. Use ArtifactAgent only when packaging a concise result helps complete this run.
9. Keep work tight: make only the smallest useful graph growth for this run.
10. If no change is warranted, do not force a tool call.
11. When the run is complete, your final assistant message must be exactly one line in this format:
   SUMMARY: <brief result of this run>

Examples:
- SUMMARY: append_graph_batch added 1 node and 1 edge for the active branch
- SUMMARY: no graph changes were needed this run`,
		MainAgentCycleGoal,
		session.ID,
		session.Topic,
		firstNonEmpty(session.OutputGoal, "none"),
		firstNonEmpty(session.Constraints, "none"),
		session.ActiveOpportunityID,
		firstNonEmpty(state.ReplanReason, "none"),
		state.Balance.Divergence,
		state.Balance.Research,
		state.Balance.Aggression,
		len(session.Nodes),
		len(session.Edges),
		strings.Join(recentNodeIDs(session.Nodes, 8), ", "),
		joinNodeTypes(validNodeTypes()),
		joinEdgeTypes(validEdgeTypes()),
		joinNodeStatuses(validNodeStatuses()),
	)
}

func recentNodeIDs(nodes []Node, limit int) []string {
	if len(nodes) == 0 {
		return []string{"none"}
	}
	if limit <= 0 || len(nodes) <= limit {
		return collectNodeIDs(nodes)
	}
	return collectNodeIDs(nodes[len(nodes)-limit:])
}

func collectNodeIDs(nodes []Node) []string {
	out := make([]string, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, node.ID)
	}
	return out
}

func joinNodeTypes(values []NodeType) string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}
	return strings.Join(out, ", ")
}

func joinEdgeTypes(values []EdgeType) string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}
	return strings.Join(out, ", ")
}

func joinNodeStatuses(values []NodeStatus) string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}
	return strings.Join(out, ", ")
}
