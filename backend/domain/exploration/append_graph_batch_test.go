package exploration

import (
	"backend/agentools"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type scriptedToolCallingModel struct {
	replies []string
	index   int
}

func (m *scriptedToolCallingModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	if m.index >= len(m.replies) {
		return schema.AssistantMessage(`{"summary":"no changes","nodes":[],"edges":[]}`, nil), nil
	}
	reply := m.replies[m.index]
	m.index++
	return schema.AssistantMessage(reply, nil), nil
}

func (m *scriptedToolCallingModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return schema.StreamReaderFromArray([]*schema.Message{schema.AssistantMessage(`{"summary":"no changes","nodes":[],"edges":[]}`, nil)}), nil
}

func (m *scriptedToolCallingModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

func newScriptedExplorationDomain(replies ...string) *ExplorationDomain {
	domain := &ExplorationDomain{
		Model: &scriptedToolCallingModel{replies: replies},
		store: &workspaceStore{
			workspaces: map[string]*ExplorationSession{},
		},
		ws: &wsState{
			subscribers: map[string]map[*wsClient]struct{}{},
		},
		runtime: &runtimeWorkspaces{
			workspaces: map[string]*RuntimeWorkspaceState{},
		},
	}
	domain.GraphAppendTool = agentools.NewAppendGraphBatchTool(domain)
	domain.DeepAgent = &scriptedResumableAgent{
		runFn: func(ctx context.Context, input *adk.AgentInput) *adk.AsyncIterator[*adk.AgentEvent] {
			iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
			go func() {
				defer gen.Close()

				msg, err := domain.Model.Generate(ctx, []*schema.Message{schema.UserMessage("run exploration cycle")})
				if err != nil {
					gen.Send(&adk.AgentEvent{Err: err})
					return
				}

				output := struct {
					Summary string                           `json:"summary"`
					Nodes   []agentools.AppendGraphBatchNode `json:"nodes"`
					Edges   []agentools.AppendGraphBatchEdge `json:"edges"`
				}{}
				if msg != nil && strings.TrimSpace(msg.Content) != "" {
					if err := json.Unmarshal([]byte(msg.Content), &output); err != nil {
						gen.Send(&adk.AgentEvent{Err: fmt.Errorf("decode scripted graph batch: %w", err)})
						return
					}
				}

				workspaceID := ""
				if len(input.Messages) > 0 {
					workspaceID = extractWorkspaceIDFromPrompt(input.Messages[0].Content)
				}
				if workspaceID == "" && len(domain.store.workspaces) == 1 {
					for id := range domain.store.workspaces {
						workspaceID = id
					}
				}

				if workspaceID != "" && (len(output.Nodes) > 0 || len(output.Edges) > 0) {
					result, err := domain.AppendGraphBatch(ctx, agentools.AppendGraphBatchParams{
						WorkspaceID: workspaceID,
						Nodes:       output.Nodes,
						Edges:       output.Edges,
					})
					if err != nil {
						gen.Send(&adk.AgentEvent{Err: err})
						return
					}
					output.Summary = result.Summary
				}
				summary := strings.TrimSpace(output.Summary)
				if summary == "" {
					summary = "no graph changes were needed this run"
				}
				gen.Send(adk.EventFromMessage(schema.AssistantMessage("SUMMARY: "+summary, nil), nil, schema.Assistant, ""))
			}()
			return iter
		},
	}
	return domain
}

func extractWorkspaceIDFromPrompt(prompt string) string {
	for _, line := range strings.Split(prompt, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- workspace_id:") {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(line, "- workspace_id:"))
	}
	return ""
}

func TestAppendGraphBatchToolRejectsInvalidEdgeEndpoint(t *testing.T) {
	domain := newScriptedExplorationDomain(`{"summary":"no changes","nodes":[],"edges":[]}`)
	snapshot, err := domain.CreateWorkspace(CreateWorkspaceReq{Topic: "agent graph", OutputGoal: "validate"})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	tool := agentools.NewAppendGraphBatchTool(domain)
	args := map[string]any{
		"workspace_id": snapshot.Exploration.ID,
		"nodes": []map[string]any{
			{
				"id":      "idea-agent-invalid",
				"type":    string(NodeIdea),
				"title":   "Broken idea",
				"summary": "Should fail because the edge endpoint is missing",
			},
		},
		"edges": []map[string]any{
			{
				"id":   "edge-agent-invalid",
				"from": "idea-agent-invalid",
				"to":   "missing-node",
				"type": string(EdgeSupports),
			},
		},
	}
	raw, _ := json.Marshal(args)

	if _, err := tool.InvokableRun(context.Background(), string(raw)); err == nil {
		t.Fatal("expected invalid edge endpoint to fail")
	}
}

func TestAppendGraphBatchToolAppendsNodesEdgesAndMutations(t *testing.T) {
	domain := newScriptedExplorationDomain(`{"summary":"no changes","nodes":[],"edges":[]}`)
	snapshot, err := domain.CreateWorkspace(CreateWorkspaceReq{Topic: "agent graph", OutputGoal: "append"})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	beforeRuntime, ok := domain.GetRuntimeState(snapshot.Exploration.ID)
	if !ok {
		t.Fatal("expected runtime state")
	}
	beforeMutations := len(beforeRuntime.Mutations)
	targetID := snapshot.Exploration.ActiveOpportunityID

	tool := agentools.NewAppendGraphBatchTool(domain)
	args := map[string]any{
		"workspace_id": snapshot.Exploration.ID,
		"nodes": []map[string]any{
			{
				"id":      "idea-agent-success",
				"type":    string(NodeIdea),
				"title":   "Agent-created idea",
				"summary": "Append-only idea proposed by the graph agent",
				"status":  string(NodeActive),
				"depth":   5,
			},
		},
		"edges": []map[string]any{
			{
				"id":   "edge-agent-success",
				"from": targetID,
				"to":   "idea-agent-success",
				"type": string(EdgeLeadsTo),
			},
		},
	}
	raw, _ := json.Marshal(args)

	resp, err := tool.InvokableRun(context.Background(), string(raw))
	if err != nil {
		t.Fatalf("append_graph_batch failed: %v", err)
	}

	var result agentools.AppendGraphBatchResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		t.Fatalf("decode tool response: %v", err)
	}
	if result.AppliedNodes != 1 || result.AppliedEdges != 1 {
		t.Fatalf("unexpected append counts: %+v", result)
	}

	updated, ok := domain.GetWorkspace(snapshot.Exploration.ID)
	if !ok {
		t.Fatal("expected workspace")
	}
	if !hasNode(updated.DirectionMap.Nodes, "idea-agent-success") {
		t.Fatal("expected appended node to be persisted in workspace graph")
	}
	if !hasEdge(updated.DirectionMap.Edges, "edge-agent-success") {
		t.Fatal("expected appended edge to be persisted in workspace graph")
	}

	afterRuntime, ok := domain.GetRuntimeState(snapshot.Exploration.ID)
	if !ok {
		t.Fatal("expected runtime state after append")
	}
	if len(afterRuntime.Mutations) <= beforeMutations {
		t.Fatal("expected append_graph_batch to record runtime mutations")
	}
}

func TestRunSingleAgentPassUsesAppendGraphBatchTool(t *testing.T) {
	domain := newScriptedExplorationDomain(
		`{"summary":"workspace bootstrap noop","nodes":[],"edges":[]}`,
		`{"summary":"expand the active branch","nodes":[{"id":"idea-agent-runtime","type":"idea","title":"Agent runtime idea","summary":"Generated by the main agent through append_graph_batch","status":"active","depth":5}],"edges":[{"id":"edge-agent-runtime","from":"%s","to":"idea-agent-runtime","type":"leads_to"}]}`,
	)

	snapshot, err := domain.CreateWorkspace(CreateWorkspaceReq{Topic: "agent graph", OutputGoal: "runtime"})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	activeID := snapshot.Exploration.ActiveOpportunityID
	model := domain.Model.(*scriptedToolCallingModel)
	model.replies[1] = strings.ReplaceAll(model.replies[1], "%s", activeID)

	runID, launched := domain.triggerRun(context.Background(), snapshot.Exploration.ID, string(RunSourceManual))
	if !launched {
		t.Fatal("expected manual trigger to launch")
	}
	if runID == "" {
		t.Fatal("expected run id")
	}

	time.Sleep(150 * time.Millisecond)

	updated, ok := domain.GetWorkspace(snapshot.Exploration.ID)
	if !ok {
		t.Fatal("expected workspace after run")
	}
	if !hasNode(updated.DirectionMap.Nodes, "idea-agent-runtime") {
		t.Fatal("expected main agent run to append graph node through tool")
	}

	state, ok := domain.GetRuntimeState(snapshot.Exploration.ID)
	if !ok {
		t.Fatal("expected runtime state")
	}
	if len(state.Results) == 0 {
		t.Fatal("expected runtime results")
	}
	lastResult := state.Results[len(state.Results)-1]
	if !strings.Contains(lastResult.Summary, agentools.ToolAppendGraphBatch) {
		t.Fatalf("expected tool-driven summary, got %q", lastResult.Summary)
	}
}

func hasNode(nodes []Node, id string) bool {
	for _, node := range nodes {
		if node.ID == id {
			return true
		}
	}
	return false
}

func hasEdge(edges []Edge, id string) bool {
	for _, edge := range edges {
		if edge.ID == id {
			return true
		}
	}
	return false
}
