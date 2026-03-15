package exploration

import (
	"backend/agents"
	"backend/datasource/dbdao"
	"context"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/gorilla/websocket"
)

// RuntimeWorkspaceState holds all per-workspace runtime data.
// Access exclusively via withWorkspaceState.
type RuntimeWorkspaceState struct {
	Runs          []Run
	Plans         []ExecutionPlan
	PlanSteps     []PlanStep
	AgentTasks    []AgentTask
	Results       []AgentTaskResultSummary
	Balance       BalanceState
	Mutations     []MutationEvent
	ReplanReason  string
	Interventions map[string]InterventionView // keyed by intervention ID
	Running       bool
	AgentRunning  bool // true while runAgentCycle goroutine is active
	Cursor        int
}

type workspaceStore struct {
	mu         sync.RWMutex
	workspaces map[string]*ExplorationSession
}

type wsClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

type wsState struct {
	mu          sync.RWMutex
	subscribers map[string]map[*wsClient]struct{}
}

type runtimeWorkspaces struct {
	mu         sync.Mutex
	workspaces map[string]*RuntimeWorkspaceState // keyed by workspace ID
}

type ExplorationDomain struct {
	DB        *dbdao.DB
	DeepAgent adk.ResumableAgent
	General   adk.Agent
	Model     model.ToolCallingChatModel
	store     *workspaceStore
	ws        *wsState
	planner   Planner
	runtime   *runtimeWorkspaces
}

// getWorkspaceState returns the state for workspaceID, initializing it if absent.
// Callers MUST hold runtime.mu before calling.
func (d *ExplorationDomain) getWorkspaceState(workspaceID string) *RuntimeWorkspaceState {
	state, ok := d.runtime.workspaces[workspaceID]
	if !ok {
		state = &RuntimeWorkspaceState{
			Interventions: map[string]InterventionView{},
		}
		d.runtime.workspaces[workspaceID] = state
	}
	return state
}

// withWorkspaceState locks runtime.mu, fetches or initializes the state, calls fn, then unlocks.
// fn MUST NOT call withWorkspaceState (no re-entry).
func (d *ExplorationDomain) withWorkspaceState(workspaceID string, fn func(*RuntimeWorkspaceState)) {
	d.runtime.mu.Lock()
	state := d.getWorkspaceState(workspaceID)
	fn(state)
	d.runtime.mu.Unlock()
}

func NewExplorationDomain(db *dbdao.DB, lm model.ToolCallingChatModel) *ExplorationDomain {
	ctx := context.Background()
	domain := &ExplorationDomain{
		DB:    db,
		Model: lm,
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

	if lm != nil {
		g, _ := agents.NewGeneralAgent(ctx, lm)
		r, _ := agents.NewResearchAgent(ctx, lm) // nil if DuckDuckGo init fails
		gr, _ := agents.NewGraphAgent(ctx, lm)
		a, _ := agents.NewArtifactAgent(ctx, lm)
		domain.planner = NewLLMPlanner(g, r, gr, a)
		domain.General = g
		domain.DeepAgent, _ = agents.NewExplorationAgent(ctx, lm)
	} else {
		domain.planner = NewDeterministicPlanner()
		domain.General, _ = agents.NewGeneralAgent(ctx, nil)
	}
	return domain
}
