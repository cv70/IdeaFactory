package exploration

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool { return true },
}

type wsRequest struct {
	RequestID   string          `json:"request_id"`
	Action      string          `json:"action"`
	WorkspaceID string          `json:"workspace_id,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
}

type wsEnvelope struct {
	Type        string               `json:"type"`
	RequestID   string               `json:"request_id,omitempty"`
	WorkspaceID string               `json:"workspace_id,omitempty"`
	Code        int                  `json:"code,omitempty"`
	Msg         string               `json:"msg,omitempty"`
	Data        *WorkspaceSnapshot   `json:"data,omitempty"`
	Runtime     RuntimeStateSnapshot `json:"runtime,omitempty"`
	Mutations   []MutationEvent      `json:"mutations,omitempty"`
	NextCursor  string               `json:"next_cursor,omitempty"`
	HasMore     bool                 `json:"has_more,omitempty"`
}

func (d *ExplorationDomain) addSubscriber(workspaceID string, client *wsClient) {
	d.ws.mu.Lock()
	defer d.ws.mu.Unlock()
	if d.ws.subscribers[workspaceID] == nil {
		d.ws.subscribers[workspaceID] = map[*wsClient]struct{}{}
	}
	d.ws.subscribers[workspaceID][client] = struct{}{}
}

func (d *ExplorationDomain) removeSubscriber(workspaceID string, client *wsClient) {
	d.ws.mu.Lock()
	defer d.ws.mu.Unlock()
	subs := d.ws.subscribers[workspaceID]
	if subs == nil {
		return
	}
	delete(subs, client)
	if len(subs) == 0 {
		delete(d.ws.subscribers, workspaceID)
	}
}

func (d *ExplorationDomain) removeClientFromAll(client *wsClient) {
	d.ws.mu.Lock()
	defer d.ws.mu.Unlock()
	for workspaceID, subs := range d.ws.subscribers {
		delete(subs, client)
		if len(subs) == 0 {
			delete(d.ws.subscribers, workspaceID)
		}
	}
}

func (d *ExplorationDomain) writeEnvelope(client *wsClient, msg wsEnvelope) error {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.conn.WriteJSON(msg)
}

func (d *ExplorationDomain) broadcastSnapshot(workspaceID string, snapshot *WorkspaceSnapshot) {
	d.ws.mu.RLock()
	subs := d.ws.subscribers[workspaceID]
	clients := make([]*wsClient, 0, len(subs))
	for c := range subs {
		clients = append(clients, c)
	}
	d.ws.mu.RUnlock()

	msg := wsEnvelope{
		Type:        "snapshot",
		WorkspaceID: workspaceID,
		Code:        http.StatusOK,
		Data:        snapshot,
	}
	for _, client := range clients {
		if err := d.writeEnvelope(client, msg); err != nil {
			d.removeSubscriber(workspaceID, client)
		}
	}
}

func (d *ExplorationDomain) broadcastMutations(workspaceID string, mutations []MutationEvent) {
	if len(mutations) == 0 {
		return
	}

	d.ws.mu.RLock()
	subs := d.ws.subscribers[workspaceID]
	clients := make([]*wsClient, 0, len(subs))
	for c := range subs {
		clients = append(clients, c)
	}
	d.ws.mu.RUnlock()

	msg := wsEnvelope{
		Type:        "mutation",
		WorkspaceID: workspaceID,
		Code:        http.StatusOK,
		Mutations:   mutations,
	}

	for _, client := range clients {
		if err := d.writeEnvelope(client, msg); err != nil {
			d.removeSubscriber(workspaceID, client)
		}
	}
}
