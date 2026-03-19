package exploration

import (
	"backend/agentools"
	"backend/agents"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type scriptedResumableAgent struct {
	runFn   func(ctx context.Context, input *adk.AgentInput) *adk.AsyncIterator[*adk.AgentEvent]
	called  bool
	inputs  []*adk.AgentInput
	resumed bool
}

func (f *scriptedResumableAgent) Name(_ context.Context) string {
	return "scripted-deep-agent"
}

func (f *scriptedResumableAgent) Description(_ context.Context) string {
	return "scripted deep agent"
}

func (f *scriptedResumableAgent) Run(ctx context.Context, input *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	f.called = true
	f.inputs = append(f.inputs, input)
	if f.runFn != nil {
		return f.runFn(ctx, input)
	}
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		defer gen.Close()
		gen.Send(adk.EventFromMessage(schema.AssistantMessage("SUMMARY: no graph changes were needed this run", nil), nil, schema.Assistant, ""))
	}()
	return iter
}

func (f *scriptedResumableAgent) Resume(_ context.Context, _ *adk.ResumeInfo, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	f.resumed = true
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Close()
	return iter
}

func TestRunMainAgentCycleUsesDeepAgentSummary(t *testing.T) {
	domain := newScriptedExplorationDomain()
	agent := &scriptedResumableAgent{
		runFn: func(_ context.Context, _ *adk.AgentInput) *adk.AsyncIterator[*adk.AgentEvent] {
			iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
			go func() {
				defer gen.Close()
				gen.Send(adk.EventFromMessage(schema.AssistantMessage("SUMMARY: expanded the active branch through append_graph_batch", nil), nil, schema.Assistant, ""))
			}()
			return iter
		},
	}
	domain.DeepAgent = agent

	session := &ExplorationSession{
		ID:                  "ws-deep-summary",
		Topic:               "agent graph",
		OutputGoal:          "grow graph",
		ActiveOpportunityID: "node-root",
		Nodes:               []Node{{ID: "node-root", Type: NodeOpportunity, Title: "Root"}},
	}
	state := &RuntimeWorkspaceState{
		Balance: BalanceState{Divergence: 0.4, Research: 0.5, Aggression: 0.6},
	}

	result, err := domain.runMainAgentCycle(context.Background(), session.ID, session, state)
	if err != nil {
		t.Fatalf("runMainAgentCycle failed: %v", err)
	}
	if !agent.called {
		t.Fatal("expected DeepAgent to be called")
	}
	if result.Summary != "expanded the active branch through append_graph_batch" {
		t.Fatalf("unexpected summary: %q", result.Summary)
	}
	if len(agent.inputs) != 1 || len(agent.inputs[0].Messages) != 1 {
		t.Fatalf("expected a single user message, got %+v", agent.inputs)
	}
	prompt := agent.inputs[0].Messages[0].Content
	if !strings.Contains(prompt, "SUMMARY:") {
		t.Fatalf("expected prompt to require fixed summary output, got %q", prompt)
	}
	if !strings.Contains(prompt, "Prefer GraphAgent") {
		t.Fatalf("expected prompt to steer graph growth through GraphAgent, got %q", prompt)
	}
	if !strings.Contains(prompt, session.ActiveOpportunityID) {
		t.Fatalf("expected prompt to include active opportunity id, got %q", prompt)
	}
}

func TestRunMainAgentCycleCapturesHandlerRuntimeEvents(t *testing.T) {
	domain := newScriptedExplorationDomain()
	domain.DeepAgent = &scriptedResumableAgent{
		runFn: func(_ context.Context, _ *adk.AgentInput) *adk.AsyncIterator[*adk.AgentEvent] {
			iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
			go func() {
				defer gen.Close()
				gen.Send(&adk.AgentEvent{
					AgentName: "exploration-main-agent",
					Output: &adk.AgentOutput{
						CustomizedOutput: agents.RuntimeEvent{
							EventType: agents.RuntimeEventAgentStart,
							Actor:     "exploration-main-agent",
							Summary:   "exploration-main-agent started a new exploration run",
							Payload: map[string]any{
								"source": string(RunSourceManual),
							},
						},
					},
				})
				gen.Send(&adk.AgentEvent{
					AgentName: "GraphAgent",
					Output: &adk.AgentOutput{
						CustomizedOutput: agents.RuntimeEvent{
							EventType: agents.RuntimeEventAgentDelegate,
							Actor:     "exploration-main-agent",
							Target:    "GraphAgent",
							Summary:   "exploration-main-agent delegated work to GraphAgent",
						},
					},
				})
				gen.Send(&adk.AgentEvent{
					AgentName: "GraphAgent",
					Output: &adk.AgentOutput{
						CustomizedOutput: agents.RuntimeEvent{
							EventType: agents.RuntimeEventToolCall,
							Actor:     "GraphAgent",
							Target:    "append_graph_batch",
							Summary:   "GraphAgent called append_graph_batch",
							Payload: map[string]any{
								"args_summary": "workspace=ws-deep-summary nodes=1 edges=1",
							},
						},
					},
				})
				gen.Send(adk.EventFromMessage(schema.AssistantMessage("SUMMARY: append_graph_batch added 1 nodes and 1 edges", nil), nil, schema.Assistant, ""))
			}()
			return iter
		},
	}

	session := &ExplorationSession{
		ID:                  "ws-deep-summary",
		Topic:               "agent graph",
		OutputGoal:          "grow graph",
		ActiveOpportunityID: "node-root",
		Nodes:               []Node{{ID: "node-root", Type: NodeOpportunity, Title: "Root"}},
	}
	state := &RuntimeWorkspaceState{
		Runs:    []Run{{ID: "run-1", WorkspaceID: "ws-deep-summary", Source: string(RunSourceManual)}},
		Balance: BalanceState{Divergence: 0.4, Research: 0.5, Aggression: 0.6},
	}

	result, err := domain.runMainAgentCycle(context.Background(), session.ID, session, state)
	if err != nil {
		t.Fatalf("runMainAgentCycle failed: %v", err)
	}
	if len(result.Events) != 4 {
		t.Fatalf("expected 4 runtime events, got %+v", result.Events)
	}
	if result.Events[0].EventType != agents.RuntimeEventAgentStart {
		t.Fatalf("expected agent_start event, got %+v", result.Events[0])
	}
	if result.Events[1].Target != "GraphAgent" {
		t.Fatalf("expected delegation target GraphAgent, got %+v", result.Events[1])
	}
	if result.Events[2].Target != "append_graph_batch" {
		t.Fatalf("expected tool call event, got %+v", result.Events[2])
	}
	if result.Events[3].EventType != "run_summary" {
		t.Fatalf("expected final run_summary event, got %+v", result.Events[3])
	}
}

func TestRunSingleAgentPassUsesDeepAgentToGrowGraph(t *testing.T) {
	domain := newScriptedExplorationDomain()
	snapshot, err := domain.CreateWorkspace(CreateWorkspaceReq{Topic: "agent graph", OutputGoal: "runtime"})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	activeID := snapshot.Exploration.ActiveOpportunityID
	domain.DeepAgent = &scriptedResumableAgent{
		runFn: func(ctx context.Context, input *adk.AgentInput) *adk.AsyncIterator[*adk.AgentEvent] {
			iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
			go func() {
				defer gen.Close()
				_, err := domain.AppendGraphBatch(ctx, agentools.AppendGraphBatchParams{
					WorkspaceID: snapshot.Exploration.ID,
					Nodes: []agentools.AppendGraphBatchNode{{
						ID:      "idea-deep-runtime",
						Type:    string(NodeIdea),
						Title:   "Deep runtime idea",
						Summary: "Generated by the deep exploration agent",
						Status:  string(NodeActive),
						Depth:   5,
					}},
					Edges: []agentools.AppendGraphBatchEdge{{
						ID:   "edge-deep-runtime",
						From: activeID,
						To:   "idea-deep-runtime",
						Type: string(EdgeLeadsTo),
					}},
				})
				if err != nil {
					gen.Send(&adk.AgentEvent{Err: err})
					return
				}
				gen.Send(&adk.AgentEvent{
					AgentName: "GraphAgent",
					Output: &adk.AgentOutput{
						MessageOutput: &adk.MessageVariant{
							Message: schema.AssistantMessage("SUMMARY: append_graph_batch added 1 nodes and 1 edges", nil),
							Role:    schema.Assistant,
						},
					},
				})
			}()
			return iter
		},
	}

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
	if !hasNode(updated.DirectionMap.Nodes, "idea-deep-runtime") {
		t.Fatal("expected deep agent run to append graph node through tool")
	}

	state, ok := domain.GetRuntimeState(snapshot.Exploration.ID)
	if !ok {
		t.Fatal("expected runtime state")
	}
	if len(state.Results) == 0 {
		t.Fatal("expected runtime results")
	}
	lastResult := state.Results[len(state.Results)-1]
	if lastResult.Summary != "GraphAgent led this run: append_graph_batch added 1 nodes and 1 edges" {
		t.Fatalf("expected fixed-format summary, got %q", lastResult.Summary)
	}
	if strings.Join(lastResult.Timeline, " -> ") != "GraphAgent -> SUMMARY" {
		t.Fatalf("expected brief timeline, got %+v", lastResult.Timeline)
	}
	lastTask := state.AgentTasks[len(state.AgentTasks)-1]
	if lastTask.SubAgent != "GraphAgent" {
		t.Fatalf("expected lead sub-agent to be GraphAgent, got %q", lastTask.SubAgent)
	}
}

func TestRunSingleAgentPassCapturesLeadAgentInRuntimeTrace(t *testing.T) {
	domain := newScriptedExplorationDomain()
	snapshot, err := domain.CreateWorkspace(CreateWorkspaceReq{Topic: "agent graph", OutputGoal: "runtime"})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	domain.DeepAgent = &scriptedResumableAgent{
		runFn: func(_ context.Context, _ *adk.AgentInput) *adk.AsyncIterator[*adk.AgentEvent] {
			iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
			go func() {
				defer gen.Close()
				gen.Send(&adk.AgentEvent{
					AgentName: "ResearchAgent",
					Output: &adk.AgentOutput{
						MessageOutput: &adk.MessageVariant{
							Message: schema.AssistantMessage("gathering evidence", nil),
							Role:    schema.Assistant,
						},
					},
				})
				gen.Send(&adk.AgentEvent{
					AgentName: "GraphAgent",
					Output: &adk.AgentOutput{
						MessageOutput: &adk.MessageVariant{
							Message: schema.AssistantMessage("proposing graph growth", nil),
							Role:    schema.Assistant,
						},
					},
				})
				gen.Send(&adk.AgentEvent{
					AgentName: "GraphAgent",
					Output: &adk.AgentOutput{
						MessageOutput: &adk.MessageVariant{
							Message: schema.AssistantMessage("SUMMARY: append_graph_batch added 1 nodes and 1 edges", nil),
							Role:    schema.Assistant,
						},
					},
				})
			}()
			return iter
		},
	}

	_, launched := domain.triggerRun(context.Background(), snapshot.Exploration.ID, string(RunSourceManual))
	if !launched {
		t.Fatal("expected manual trigger to launch")
	}

	time.Sleep(150 * time.Millisecond)

	state, ok := domain.GetRuntimeState(snapshot.Exploration.ID)
	if !ok {
		t.Fatal("expected runtime state")
	}
	lastTask := state.AgentTasks[len(state.AgentTasks)-1]
	if lastTask.SubAgent != "GraphAgent" {
		t.Fatalf("expected GraphAgent to be recorded as lead agent, got %q", lastTask.SubAgent)
	}
	lastResult := state.Results[len(state.Results)-1]
	if !strings.Contains(lastResult.Summary, "GraphAgent") {
		t.Fatalf("expected runtime summary to include lead agent, got %q", lastResult.Summary)
	}
	if strings.Join(lastResult.Timeline, " -> ") != "ResearchAgent -> GraphAgent -> SUMMARY" {
		t.Fatalf("expected ordered timeline, got %+v", lastResult.Timeline)
	}

	trace := buildTraceSummary(snapshot.Exploration.ID, latestRunID(state.Runs), state)
	foundLeadTrace := false
	for _, item := range trace.Items {
		if item.Category == "tool" && strings.Contains(item.Message, "GraphAgent") && strings.Contains(item.Message, "ResearchAgent -> GraphAgent -> SUMMARY") {
			foundLeadTrace = true
			break
		}
	}
	if !foundLeadTrace {
		t.Fatal("expected trace summary to surface the lead agent")
	}
}
