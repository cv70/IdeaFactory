package exploration

import "testing"

func TestCreateWorkspaceSeedsOnlyTopicNode(t *testing.T) {
	session := createWorkspace("workspace-test", CreateWorkspaceReq{
		Topic:       "AI education",
		OutputGoal:  "research directions",
		Constraints: "low-cost",
	})

	if len(session.Nodes) != 1 {
		t.Fatalf("expected exactly one seeded node, got %d", len(session.Nodes))
	}
	if len(session.Edges) != 0 {
		t.Fatalf("expected no seeded edges, got %d", len(session.Edges))
	}

	topic := session.Nodes[0]
	if topic.Type != NodeTopic {
		t.Fatalf("expected only seeded node to be topic, got %s", topic.Type)
	}
	if session.ActiveOpportunityID != topic.ID {
		t.Fatalf("expected active opportunity id to point to topic node, got %q", session.ActiveOpportunityID)
	}
	if len(session.Runs) == 0 {
		t.Fatal("expected initial run note to exist")
	}
	if session.Runs[0].Summary != "Initialized workspace with topic anchor; waiting for agent-driven graph growth." {
		t.Fatalf("unexpected initial run summary: %q", session.Runs[0].Summary)
	}
}
