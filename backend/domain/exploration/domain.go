package exploration

import (
	"backend/datasource/dbdao"
	"sync"

	"github.com/cloudwego/eino/components/model"
	"github.com/gorilla/websocket"
)

type ExplorationDomain struct {
	DB    *dbdao.DB
	LLM   model.ToolCallingChatModel
	store *workspaceStore
	ws    *wsState
	runtime *runtimeState
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
	mu      sync.Mutex
	running map[string]bool
	cursor  map[string]int
}

func NewExplorationDomain(db *dbdao.DB) *ExplorationDomain {
	return &ExplorationDomain{
		DB: db,
		store: &workspaceStore{
			workspaces: map[string]*ExplorationSession{},
		},
		ws: &wsState{
			subscribers: map[string]map[*wsClient]struct{}{},
		},
		runtime: &runtimeState{
			running: map[string]bool{},
			cursor:  map[string]int{},
		},
	}
}
