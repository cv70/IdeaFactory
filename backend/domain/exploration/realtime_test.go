package exploration

import (
	"backend/agentools"
	"backend/config"
	"backend/infra"
	"context"
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
	ctx := context.Background()
	c, err := config.LoadConfig()
	mistake.Unwrap(err)
	r, err := infra.NewRegistry(ctx, c)
	mistake.Unwrap(err)
	ed, err := BuildExplorationDomain(r)
	mistake.Unwrap(err)
	return ed
}

func TestRuntimeContinuouslyExpandsWorkspace(t *testing.T) {
	domain := newTestExplorationDomain()
	created, err := domain.CreateWorkspace(CreateWorkspaceReq{
		Topic:       "AI education",
		OutputGoal:  "Research directions",
		Constraints: "Low-cost",
	})
	mistake.Unwrap(err)
	initialRuns := len(created.Exploration.Runs)

	time.Sleep(4500 * time.Millisecond)

	updated, ok := domain.GetWorkspace(created.Exploration.ID)
	if !ok {
		t.Fatal("workspace should exist")
	}
	if len(updated.Exploration.Runs) <= initialRuns {
		t.Fatalf("expected runtime to append runs, before=%d after=%d", initialRuns, len(updated.Exploration.Runs))
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
	if len(state.Plans) == 0 {
		t.Fatal("expected at least one plan")
	}
	if len(state.PlanSteps) == 0 {
		t.Fatal("expected at least one plan step")
	}
	if len(state.AgentTasks) == 0 {
		t.Fatal("expected at least one agent task")
	}
	if state.Balance.WorkspaceID == "" {
		t.Fatal("expected balance state")
	}
}

func TestRuntimeContext(t *testing.T) {
	ctx := context.Background()
	ctx = InitRuntimeContext(ctx, RuntimeContextData{
		WorkspaceID: "ws-1",
		RunID:       "run-1",
		PlanID:      "plan-1",
		WorkDir:     "/tmp/idea-factory/ws-1/run-1",
		InputDigest: "topic=ai",
	})

	got, ok := RuntimeContextFrom(ctx)
	if !ok {
		t.Fatal("expected runtime context to be available")
	}
	if got.RunID != "run-1" {
		t.Fatalf("unexpected run id: %s", got.RunID)
	}

	abs, err := ResolveWorkFile(ctx, "notes/result.md")
	if err != nil {
		t.Fatalf("resolve work file: %v", err)
	}
	expected := filepath.Clean("/tmp/idea-factory/ws-1/run-1/notes/result.md")
	if abs != expected {
		t.Fatalf("unexpected absolute path: %s", abs)
	}
}

func TestRunCreatesExplicitPlan(t *testing.T) {
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
	if len(state.Plans) == 0 {
		t.Fatal("expected plan to be created")
	}
	if state.Balance.WorkspaceID == "" {
		t.Fatal("expected balance state to be computed")
	}
}

func TestPlanStepTransitions(t *testing.T) {
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

	doneOrFailed := false
	for _, step := range state.PlanSteps {
		if step.Status == PlanStepDone || step.Status == PlanStepFailed {
			doneOrFailed = true
			break
		}
	}
	if !doneOrFailed {
		t.Fatal("expected at least one transitioned plan step")
	}

	for _, task := range state.AgentTasks {
		if task.SubAgent == "" {
			t.Fatal("expected sub-agent to be recorded")
		}
	}
}

func TestInterventionTriggersReplanning(t *testing.T) {
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
	if len(before.Plans) == 0 {
		t.Fatal("expected initial plan")
	}
	beforePlanCount := len(before.Plans)
	beforeVersion := before.Plans[len(before.Plans)-1].Version

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
	if len(after.Plans) <= beforePlanCount {
		t.Fatal("expected new plan version after intervention")
	}
	lastPlan := after.Plans[len(after.Plans)-1]
	if lastPlan.Version <= beforeVersion {
		t.Fatalf("expected plan version to increase, before=%d after=%d", beforeVersion, lastPlan.Version)
	}

	skipped := 0
	for _, step := range after.PlanSteps {
		if step.Status == PlanStepSkipped {
			skipped++
		}
	}
	if skipped == 0 {
		t.Fatal("expected pending steps from previous plan to be marked skipped")
	}
	if after.LatestReplanReason == "" {
		t.Fatal("expected replan reason to be recorded")
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
	domain := newTestExplorationDomain()
	created, err := domain.CreateWorkspace(CreateWorkspaceReq{
		Topic:       "AI education",
		OutputGoal:  "Research directions",
		Constraints: "Low-cost",
	})
	mistake.Unwrap(err)

	initialNodes := len(created.DirectionMap.Nodes)

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
	if len(state.Plans) == 0 {
		t.Fatal("expected at least one execution plan")
	}
	if len(state.AgentTasks) == 0 {
		t.Fatal("expected delegated agent tasks")
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
