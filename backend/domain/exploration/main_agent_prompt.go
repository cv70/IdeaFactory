package exploration

import (
	"fmt"
	"strings"
)

func (d *ExplorationDomain) buildMainAgentGraphPrompt(session *ExplorationSession, state *RuntimeWorkspaceState) string {
	return fmt.Sprintf(
		`You are the MainAgent for IdeaFactory.

Your job is to decide the next append-only graph mutation for the workspace.
The backend will execute %q exactly once with your batch. Do not rewrite existing nodes or edges.

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
1. Return JSON only.
2. Use append-only mutations.
3. Node types must be one of: %s
4. Edge types must be one of: %s
5. Node status must be one of: %s
6. Edge endpoints must reference an existing node id or a node created in the same batch.
7. If no change is warranted, return {"summary":"no changes","nodes":[],"edges":[]}

Return this exact shape:
{
  "summary": "short reason for this append batch",
  "nodes": [
    {
      "id": "string",
      "type": "string",
      "title": "string",
      "summary": "string",
      "status": "active",
      "score": 0.0,
      "depth": 0,
      "parent_context": "string",
      "metadata": {
        "branchId": "string",
        "slot": "string",
        "cluster": "string"
      },
      "evidence_summary": "string"
    }
  ],
  "edges": [
    {
      "id": "string",
      "from": "string",
      "to": "string",
      "type": "string"
    }
  ]
}`,
		MainAgentGraphGoal,
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
