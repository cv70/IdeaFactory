package exploration

import (
	"backend/agentools"
	"backend/agents"
	"backend/datasource/dbdao"
	"backend/infra"
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/gorilla/websocket"
)

// RuntimeWorkspaceState holds all per-workspace runtime data.
// Access exclusively via withWorkspaceState.
type RuntimeWorkspaceState struct {
	Runs            []Run
	AgentTasks      []AgentTask
	Results         []AgentTaskResultSummary
	Balance         BalanceState
	Mutations       []MutationEvent
	ReplanReason    string
	Interventions   map[string]InterventionView // keyed by intervention ID
	AgentRunning    bool                        // true while runAgentCycle goroutine is active
	cancelScheduler context.CancelFunc          // non-nil while a scheduler goroutine is pending the next run
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
	DB              *dbdao.DB
	DeepAgent       adk.ResumableAgent
	Model           model.ToolCallingChatModel
	GraphAppendTool tool.InvokableTool
	store           *workspaceStore
	ws              *wsState
	runtime         *runtimeWorkspaces
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

func BuildExplorationDomain(registry *infra.Registry) (*ExplorationDomain, error) {
	ctx := context.Background()
	domain := &ExplorationDomain{
		DB:    registry.DB,
		Model: registry.Model,
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

	var err error
	domain.DeepAgent, err = agents.NewExplorationAgent(ctx, registry.Model, domain.GraphAppendTool)
	return domain, err
}

// Start loads all active (non-paused, non-archived) workspaces from the DB and
// schedules a run for each. It is non-fatal per workspace: errors are logged and skipped.
// Call this after RegisterRoutes so WS subscribers can receive mutation events.
// Run in a goroutine from main.go to avoid blocking HTTP server startup.
func (d *ExplorationDomain) Start(ctx context.Context) error {
	if d.DB == nil {
		return nil
	}
	states, err := d.DB.ListActiveWorkspaceStates(200)
	if err != nil {
		return fmt.Errorf("Start: list active workspaces: %w", err)
	}
	for _, state := range states {
		wsID := formatWorkspaceID(state.ID)
		session, ok := d.loadWorkspace(wsID)
		if !ok || session == nil {
			continue
		}
		d.store.mu.Lock()
		d.store.workspaces[wsID] = session
		d.store.mu.Unlock()

		d.restoreRuntimeState(wsID)
		d.scheduleNextRun(wsID)
	}
	return nil
}
