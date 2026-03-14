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

type ExplorationDomain struct {
	DB        *dbdao.DB
	DeepAgent adk.ResumableAgent
	General   adk.Agent
	Model     model.ToolCallingChatModel
	store     *workspaceStore
	ws        *wsState
	runtime   *runtimeState
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

type runtimeState struct {
	mu           sync.Mutex
	running      map[string]bool
	cursor       map[string]int
	runs         map[string][]Run
	plans        map[string][]ExecutionPlan
	planSteps    map[string][]PlanStep
	agentTasks   map[string][]AgentTask
	results      map[string][]AgentTaskResultSummary
	balance      map[string]BalanceState
	mutations    map[string][]MutationEvent
	replanReason map[string]string
	intervention map[string]map[string]InterventionView
}

func NewExplorationDomain(db *dbdao.DB, lm model.ToolCallingChatModel) *ExplorationDomain {
	domain := &ExplorationDomain{
		DB:    db,
		Model: lm,
		store: &workspaceStore{
			workspaces: map[string]*ExplorationSession{},
		},
		ws: &wsState{
			subscribers: map[string]map[*wsClient]struct{}{},
		},
		runtime: &runtimeState{
			running:      map[string]bool{},
			cursor:       map[string]int{},
			runs:         map[string][]Run{},
			plans:        map[string][]ExecutionPlan{},
			planSteps:    map[string][]PlanStep{},
			agentTasks:   map[string][]AgentTask{},
			results:      map[string][]AgentTaskResultSummary{},
			balance:      map[string]BalanceState{},
			mutations:    map[string][]MutationEvent{},
			replanReason: map[string]string{},
			intervention: map[string]map[string]InterventionView{},
		},
	}
	if lm != nil {
		if agent, err := agents.BuildExplorationAgent(context.Background(), lm); err == nil {
			domain.DeepAgent = agent
		}
		if general, err := agents.NewGeneralAgent(context.Background(), lm); err == nil {
			domain.General = general
		}
	} else {
		if general, err := agents.NewGeneralAgent(context.Background(), nil); err == nil {
			domain.General = general
		}
	}
	return domain
}
