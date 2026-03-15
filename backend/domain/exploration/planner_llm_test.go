package exploration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// stubAgent returns a fixed string response for testing.
type stubAgent struct {
	response string
	err      error
}

func (s *stubAgent) Name(_ context.Context) string        { return "stub" }
func (s *stubAgent) Description(_ context.Context) string { return "stub" }
func (s *stubAgent) Run(_ context.Context, _ *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	if s.err != nil {
		gen.Send(&adk.AgentEvent{Err: s.err})
	} else {
		gen.Send(&adk.AgentEvent{
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					IsStreaming: false,
					Message:     schema.AssistantMessage(s.response, nil),
				},
			},
		})
	}
	gen.Close()
	return iter
}

func TestLLMPlanner_NilAgentFallsBackToDeterministic(t *testing.T) {
	p := NewLLMPlanner(nil, nil, nil, nil)
	session := &ExplorationSession{
		ID:    "ws-test",
		Topic: "quantum computing",
	}
	state := &RuntimeWorkspaceState{
		Balance: BalanceState{Divergence: 0.6, Research: 0.7, Aggression: 0.45},
	}
	nodes, edges := p.GenerateNodesForCycle(context.Background(), session, state)
	// With no directions yet, should generate Direction nodes via deterministic fallback
	if len(nodes) == 0 {
		t.Fatal("expected fallback to generate direction nodes")
	}
	for _, n := range nodes {
		if n.Type != NodeDirection {
			t.Errorf("expected NodeDirection, got %s", n.Type)
		}
	}
	_ = edges
}

func TestLLMPlanner_DirectionGenerationFromAgent(t *testing.T) {
	response := `{"directions":[{"title":"Quantum ML","summary":"Intersection of quantum and ML"},{"title":"Error Correction","summary":"Quantum error correction techniques"}]}`
	agent := &stubAgent{response: response}
	p := NewLLMPlanner(agent, nil, nil, nil)
	session := &ExplorationSession{ID: "ws-test", Topic: "quantum computing"}
	state := &RuntimeWorkspaceState{
		Balance: BalanceState{Divergence: 0.6, Research: 0.7, Aggression: 0.45},
	}
	nodes, _ := p.GenerateNodesForCycle(context.Background(), session, state)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 direction nodes, got %d", len(nodes))
	}
	if nodes[0].Type != NodeDirection {
		t.Errorf("expected NodeDirection, got %s", nodes[0].Type)
	}
	if nodes[0].Title != "Quantum ML" {
		t.Errorf("unexpected title: %s", nodes[0].Title)
	}
	if nodes[0].WorkspaceID != "ws-test" {
		t.Errorf("WorkspaceID not set: %s", nodes[0].WorkspaceID)
	}
}

func TestLLMPlanner_InvalidDirectionIDDropped(t *testing.T) {
	// Evidence agent returns an item with an unrecognized direction_id
	response := `{"evidence":[{"title":"E1","summary":"s1","edge_type":"supports","direction_id":"nonexistent-id"}]}`
	researchAgent := &stubAgent{response: response}
	p := NewLLMPlanner(nil, researchAgent, nil, nil)

	dirNode := Node{ID: "dir-real-1", WorkspaceID: "ws-test", Type: NodeDirection, Title: "Real Dir"}
	session := &ExplorationSession{
		ID:    "ws-test",
		Topic: "quantum",
		Nodes: []Node{dirNode},
	}
	state := &RuntimeWorkspaceState{
		Balance: BalanceState{Divergence: 0.6, Research: 0.7, Aggression: 0.4},
	}
	// All items dropped → deterministic fallback should produce Evidence nodes
	nodes, _ := p.GenerateNodesForCycle(context.Background(), session, state)
	if len(nodes) == 0 {
		t.Fatal("expected fallback evidence nodes after direction_id validation failure")
	}
	for _, n := range nodes {
		if n.Type != NodeEvidence {
			t.Errorf("expected NodeEvidence from fallback, got %s", n.Type)
		}
	}
}

func TestLLMPlanner_AgentErrorFallsBack(t *testing.T) {
	errAgent := &stubAgent{err: fmt.Errorf("agent unavailable")}
	p := NewLLMPlanner(errAgent, nil, nil, nil)
	session := &ExplorationSession{ID: "ws-test", Topic: "AI"}
	state := &RuntimeWorkspaceState{
		Balance: BalanceState{Divergence: 0.6, Research: 0.7, Aggression: 0.4},
	}
	nodes, _ := p.GenerateNodesForCycle(context.Background(), session, state)
	// Should fall back to deterministic and still produce Direction nodes
	if len(nodes) == 0 {
		t.Fatal("expected fallback nodes on agent error")
	}
}

func TestLLMPlanner_ContextCancelledFallsBack(t *testing.T) {
	// stubAgent returns an error event (context.Canceled) → runAgent returns error → fallback
	blockingAgent := &stubAgent{err: context.Canceled}
	p := NewLLMPlanner(blockingAgent, nil, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	session := &ExplorationSession{ID: "ws-test", Topic: "AI"}
	state := &RuntimeWorkspaceState{
		Balance: BalanceState{Divergence: 0.6, Research: 0.7, Aggression: 0.4},
	}
	nodes, _ := p.GenerateNodesForCycle(ctx, session, state)
	if len(nodes) == 0 {
		t.Fatal("expected fallback Direction nodes when agent errors with context.Canceled")
	}
}

func TestRunAgent_EmptyContentError(t *testing.T) {
	agent := &stubAgent{response: "   "}
	_, err := runAgent(context.Background(), agent, "prompt")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("unexpected error: %v", err)
	}
}
