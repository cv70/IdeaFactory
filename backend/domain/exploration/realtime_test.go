package exploration

import (
	"backend/agentools"
	"backend/config"
	"backend/infra"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cv70/pkgo/mistake"
)

func newTestExplorationDomain() *ExplorationDomain {
	dir, err := os.MkdirTemp("", "idea-factory-test-*")
	mistake.Unwrap(err)
	db, err := infra.NewDB(context.Background(), &config.DatabaseConfig{
		DB: filepath.Join(dir, "idea-factory.sqlite"),
	})
	mistake.Unwrap(err)
	domain := newScriptedExplorationDomain()
	domain.DB = db
	return domain
}

func TestRuntimeCanBeTriggeredForAdditionalRuns(t *testing.T) {
	domain := newTestExplorationDomain()
	created, err := domain.CreateWorkspace(CreateWorkspaceReq{
		Topic:       "AI education",
		OutputGoal:  "Research directions",
		Constraints: "Low-cost",
	})
	mistake.Unwrap(err)
	initialState, ok := domain.GetRuntimeState(created.Exploration.ID)
	if !ok {
		t.Fatal("expected runtime state")
	}
	initialRuns := len(initialState.Runs)

	runID, launched := domain.triggerRun(context.Background(), created.Exploration.ID, "manual")
	if !launched {
		t.Fatal("expected manual trigger to launch a run")
	}
	if runID == "" {
		t.Fatal("expected run id")
	}
	time.Sleep(150 * time.Millisecond)

	updated, ok := domain.GetRuntimeState(created.Exploration.ID)
	if !ok {
		t.Fatal("expected runtime state after trigger")
	}
	if len(updated.Runs) <= initialRuns {
		t.Fatalf("expected trigger to append runs, before=%d after=%d", initialRuns, len(updated.Runs))
	}
}

func TestRunPlanTaskPersistence(t *testing.T) {
	domain := newTestExplorationDomain()
	created, err := domain.CreateWorkspace(CreateWorkspaceReq{
		Topic:       "AI education",
		OutputGoal:  "Research directions",
		Constraints: "Low-cost",
	})
	mistake.Unwrap(err)

	state, ok := domain.GetRuntimeState(created.Exploration.ID)
	if !ok {
		t.Fatal("expected runtime state to exist")
	}
	if len(state.Runs) == 0 {
		t.Fatal("expected at least one run")
	}
	if len(state.AgentTasks) == 0 {
		t.Fatal("expected at least one runtime activity record")
	}
	if state.Balance.WorkspaceID == "" {
		t.Fatal("expected balance state")
	}
}

func TestRunCreatesInitialMainAgentActivity(t *testing.T) {
	domain := newTestExplorationDomain()
	created, err := domain.CreateWorkspace(CreateWorkspaceReq{
		Topic:       "AI education",
		OutputGoal:  "Research directions",
		Constraints: "Low-cost",
	})
	mistake.Unwrap(err)

	state, ok := domain.GetRuntimeState(created.Exploration.ID)
	if !ok {
		t.Fatal("expected runtime state")
	}
	if len(state.AgentTasks) == 0 {
		t.Fatal("expected main-agent activity to be recorded")
	}
	if state.Balance.WorkspaceID == "" {
		t.Fatal("expected balance state to be computed")
	}
}

func TestMainAgentActivityTransitions(t *testing.T) {
	domain := newTestExplorationDomain()
	created, err := domain.CreateWorkspace(CreateWorkspaceReq{
		Topic:       "AI education",
		OutputGoal:  "Research directions",
		Constraints: "Low-cost",
	})
	mistake.Unwrap(err)

	state, ok := domain.GetRuntimeState(created.Exploration.ID)
	if !ok {
		t.Fatal("expected runtime state")
	}

	for _, task := range state.AgentTasks {
		if task.SubAgent == "" {
			t.Fatal("expected actor to be recorded")
		}
	}
	if len(state.Results) == 0 {
		t.Fatal("expected runtime result summaries")
	}
}

func TestInterventionTriggersRuntimeReflection(t *testing.T) {
	domain := newTestExplorationDomain()
	created, err := domain.CreateWorkspace(CreateWorkspaceReq{
		Topic:       "AI education",
		OutputGoal:  "Research directions",
		Constraints: "Low-cost",
	})
	mistake.Unwrap(err)

	before, ok := domain.GetRuntimeState(created.Exploration.ID)
	if !ok {
		t.Fatal("expected runtime state")
	}
	beforeMutationCount := len(before.Mutations)

	_, _, ok = domain.ApplyIntervention(created.Exploration.ID, InterventionReq{
		Type:     InterventionShiftFocus,
		TargetID: created.Exploration.ActiveOpportunityID,
		Note:     "focus on measurable outcomes",
	})
	if !ok {
		t.Fatal("expected intervention to succeed")
	}

	after, ok := domain.GetRuntimeState(created.Exploration.ID)
	if !ok {
		t.Fatal("expected runtime state after intervention")
	}
	if len(after.Mutations) <= beforeMutationCount {
		t.Fatal("expected intervention to produce additional runtime mutations")
	}
	if after.LatestReplanReason == "" {
		t.Fatal("expected intervention reason to be recorded")
	}
	if after.Balance.UpdatedAt == 0 {
		t.Fatal("expected balance state to be recomputed")
	}
}

func TestWrappedToolResponses(t *testing.T) {
	wrapper := agentools.NewRuntimeToolWrapper()

	resp, err := wrapper.NormalizeToolCall("read_file", "{path:'/tmp/a',}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Tool != "read_file" {
		t.Fatalf("unexpected tool: %s", resp.Tool)
	}
	if !strings.Contains(resp.Summary, "read_file") {
		t.Fatalf("unexpected summary: %s", resp.Summary)
	}

	if _, err := wrapper.NormalizeToolCall("", "{bad"); err == nil {
		t.Fatal("expected invalid payload to fail")
	}
}

func TestDeepAgentRunE2E(t *testing.T) {
	domain := newScriptedExplorationDomain(
		`{"summary":"workspace bootstrap noop","nodes":[],"edges":[]}`,
		`{"summary":"append an agent runtime idea","nodes":[{"id":"idea-deep-agent","type":"idea","title":"Deep agent idea","summary":"Generated by the main agent run","status":"active","depth":4}],"edges":[{"id":"edge-deep-agent","from":"%s","to":"idea-deep-agent","type":"leads_to"}]}`,
	)
	created, err := domain.CreateWorkspace(CreateWorkspaceReq{
		Topic:       "AI education",
		OutputGoal:  "Research directions",
		Constraints: "Low-cost",
	})
	mistake.Unwrap(err)

	initialNodes := len(created.DirectionMap.Nodes)
	model := domain.Model.(*scriptedToolCallingModel)
	model.replies[1] = strings.ReplaceAll(model.replies[1], "%s", created.Exploration.ActiveOpportunityID)

	_, launched := domain.triggerRun(context.Background(), created.Exploration.ID, string(RunSourceManual))
	if !launched {
		t.Fatal("expected manual trigger to launch a run")
	}

	time.Sleep(200 * time.Millisecond)

	updated, ok := domain.GetWorkspace(created.Exploration.ID)
	if !ok {
		t.Fatal("workspace should exist")
	}
	state, ok := domain.GetRuntimeState(created.Exploration.ID)
	if !ok {
		t.Fatal("expected runtime state")
	}
	if len(state.Runs) == 0 {
		t.Fatal("expected at least one run in runtime state")
	}
	if len(state.AgentTasks) == 0 {
		t.Fatal("expected main-agent activity records")
	}
	if len(updated.DirectionMap.Nodes) <= initialNodes {
		t.Fatalf("expected graph to grow, before=%d after=%d", initialNodes, len(updated.DirectionMap.Nodes))
	}
}

type fakeToolCallingModel struct{}

func (fakeToolCallingModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	return schema.AssistantMessage("ok", nil), nil
}

func (fakeToolCallingModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return schema.StreamReaderFromArray([]*schema.Message{schema.AssistantMessage("ok", nil)}), nil
}

func (fakeToolCallingModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return fakeToolCallingModel{}, nil
}

type fakePlanningModel struct{}

func (fakePlanningModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	return schema.AssistantMessage(`{"steps":["scan evidence","update direction graph","package top idea"]}`, nil), nil
}

func (fakePlanningModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return schema.StreamReaderFromArray([]*schema.Message{schema.AssistantMessage(`{"steps":["scan evidence","update direction graph","package top idea"]}`, nil)}), nil
}

func (fakePlanningModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return fakePlanningModel{}, nil
}

type fakeResumableAgent struct {
	called bool
}

func (f *fakeResumableAgent) Name(_ context.Context) string {
	return "fake-deep-agent"
}

func (f *fakeResumableAgent) Description(_ context.Context) string {
	return "fake"
}

func (f *fakeResumableAgent) Run(_ context.Context, _ *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	f.called = true
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Close()
	return iter
}

func (f *fakeResumableAgent) Resume(_ context.Context, _ *adk.ResumeInfo, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Close()
	return iter
}
