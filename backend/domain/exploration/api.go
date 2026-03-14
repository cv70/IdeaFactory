package exploration

import (
	"backend/utils"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func (d *ExplorationDomain) ApiCreateWorkspace(c *gin.Context) {
	var req CreateWorkspaceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespError(c, 400, "failed to parse create workspace request")
		return
	}
	utils.RespSuccess(c, d.CreateWorkspace(req))
}

func (d *ExplorationDomain) ApiListWorkspaces(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	items, err := d.ListWorkspaces(limit)
	if err != nil {
		utils.RespError(c, 500, "failed to list workspaces")
		return
	}
	utils.RespSuccess(c, gin.H{
		"workspaces": items,
	})
}

func (d *ExplorationDomain) ApiGetWorkspace(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	snapshot, ok := d.GetWorkspace(workspaceID)
	if !ok {
		utils.RespError(c, 404, "workspace not found")
		return
	}
	utils.RespSuccess(c, snapshot)
}

func (d *ExplorationDomain) ApiGetRuntimeState(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	// Ensure workspace exists and runtime has been initialized for loaded sessions.
	if _, ok := d.GetWorkspace(workspaceID); !ok {
		utils.RespError(c, 404, "workspace not found")
		return
	}
	query := RuntimeStateQuery{
		RunID: strings.TrimSpace(c.Query("run_id")),
	}
	if latestRuns := strings.TrimSpace(c.Query("latest_runs")); latestRuns != "" {
		value, err := strconv.Atoi(latestRuns)
		if err != nil || value <= 0 {
			utils.RespError(c, 400, "invalid latest_runs")
			return
		}
		query.LatestRuns = value
	}

	state, ok := d.QueryRuntimeState(workspaceID, query)
	if !ok {
		utils.RespError(c, 404, "runtime state not found")
		return
	}
	utils.RespSuccess(c, state)
}

func (d *ExplorationDomain) ApiReplayMutations(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	cursor := c.Query("cursor")
	if cursor == "" {
		since, _ := strconv.ParseInt(c.Query("since"), 10, 64)
		if since > 0 {
			cursor = strconv.FormatInt(since, 10) + "|"
		}
	}
	limit, _ := strconv.Atoi(c.Query("limit"))

	page, err := d.replayMutations(workspaceID, cursor, limit)
	if err != nil {
		utils.RespError(c, 500, "failed to replay mutations")
		return
	}
	utils.RespSuccess(c, page)
}

func (d *ExplorationDomain) ApiUpdateStrategy(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	var req UpdateStrategyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespError(c, 400, "failed to parse strategy request")
		return
	}
	snapshot, mutations, ok := d.UpdateStrategy(workspaceID, req)
	if !ok {
		utils.RespError(c, 404, "workspace not found")
		return
	}
	d.broadcastMutations(workspaceID, mutations)
	utils.RespSuccess(c, snapshot)
}

func (d *ExplorationDomain) ApiArchiveWorkspace(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	if ok := d.ArchiveWorkspace(workspaceID); !ok {
		utils.RespError(c, 404, "workspace not found")
		return
	}
	utils.RespSuccess(c, gin.H{"archived": true})
}

func (d *ExplorationDomain) ApiInterveneWorkspace(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	var req InterventionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespError(c, 400, "failed to parse intervention request")
		return
	}

	snapshot, mutations, ok := d.ApplyIntervention(workspaceID, req)
	if !ok {
		utils.RespError(c, 404, "workspace not found or invalid intervention type")
		return
	}
	d.broadcastMutations(workspaceID, mutations)
	utils.RespSuccess(c, snapshot)
}

func (d *ExplorationDomain) ApiCreateSession(c *gin.Context) {
	var req CreateSessionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespError(c, 400, "failed to parse create session request")
		return
	}

	session, err := d.CreateSession(&req)
	if err != nil {
		utils.RespError(c, 500, "failed to create session")
		return
	}
	utils.RespSuccess(c, session)
}

func (d *ExplorationDomain) ApiGetSession(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	snapshot, ok := d.GetWorkspace(workspaceID)
	if !ok {
		utils.RespError(c, 404, "exploration not found")
		return
	}
	utils.RespSuccess(c, snapshot)
}

func (d *ExplorationDomain) ApiExpandOpportunity(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	opportunityID := c.Param("opportunityID")
	snapshot, mutations, ok := d.ApplyIntervention(workspaceID, InterventionReq{
		Type:     InterventionExpandOpportunity,
		TargetID: opportunityID,
	})
	if !ok {
		utils.RespError(c, 404, "exploration not found")
		return
	}
	d.broadcastMutations(workspaceID, mutations)
	utils.RespSuccess(c, snapshot)
}

func (d *ExplorationDomain) ApiFeedback(c *gin.Context) {
	workspaceID := c.Param("workspaceID")
	var req FeedbackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespError(c, 400, "failed to parse feedback request")
		return
	}

	switch req.Type {
	case "toggle_favorite":
		snapshot, mutations, ok := d.ApplyIntervention(workspaceID, InterventionReq{
			Type:     InterventionToggleFavorite,
			TargetID: req.NodeID,
		})
		if !ok {
			utils.RespError(c, 404, "exploration not found")
			return
		}
		d.broadcastMutations(workspaceID, mutations)
		utils.RespSuccess(c, snapshot)
	default:
		utils.RespError(c, 400, "unsupported feedback type")
	}
}

func (d *ExplorationDomain) ApiWebSocket(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &wsClient{conn: conn}
	defer func() {
		d.removeClientFromAll(client)
		_ = conn.Close()
	}()

	for {
		var req wsRequest
		if err := conn.ReadJSON(&req); err != nil {
			return
		}

		switch req.Action {
		case "create_workspace":
			var payload CreateWorkspaceReq
			if err := json.Unmarshal(req.Payload, &payload); err != nil {
				_ = d.writeEnvelope(client, wsEnvelope{
					Type:      "response",
					RequestID: req.RequestID,
					Code:      http.StatusBadRequest,
					Msg:       "invalid create workspace payload",
				})
				continue
			}
			snapshot := d.CreateWorkspace(payload)
			_ = d.writeEnvelope(client, wsEnvelope{
				Type:      "response",
				RequestID: req.RequestID,
				Code:      http.StatusOK,
				Data:      snapshot,
			})
		case "get_workspace":
			snapshot, ok := d.GetWorkspace(req.WorkspaceID)
			if !ok {
				_ = d.writeEnvelope(client, wsEnvelope{
					Type:      "response",
					RequestID: req.RequestID,
					Code:      http.StatusNotFound,
					Msg:       "workspace not found",
				})
				continue
			}
			_ = d.writeEnvelope(client, wsEnvelope{
				Type:      "response",
				RequestID: req.RequestID,
				Code:      http.StatusOK,
				Data:      snapshot,
			})
		case "get_runtime_state":
			var payload struct {
				RunID      string `json:"run_id"`
				LatestRuns int    `json:"latest_runs"`
			}
			if len(req.Payload) > 0 {
				if err := json.Unmarshal(req.Payload, &payload); err != nil {
					_ = d.writeEnvelope(client, wsEnvelope{
						Type:      "response",
						RequestID: req.RequestID,
						Code:      http.StatusBadRequest,
						Msg:       "invalid runtime query payload",
					})
					continue
				}
			}
			if _, ok := d.GetWorkspace(req.WorkspaceID); !ok {
				_ = d.writeEnvelope(client, wsEnvelope{
					Type:      "response",
					RequestID: req.RequestID,
					Code:      http.StatusNotFound,
					Msg:       "workspace not found",
				})
				continue
			}
			state, ok := d.QueryRuntimeState(req.WorkspaceID, RuntimeStateQuery{
				RunID:      strings.TrimSpace(payload.RunID),
				LatestRuns: payload.LatestRuns,
			})
			if !ok {
				_ = d.writeEnvelope(client, wsEnvelope{
					Type:      "response",
					RequestID: req.RequestID,
					Code:      http.StatusNotFound,
					Msg:       "runtime state not found",
				})
				continue
			}
			_ = d.writeEnvelope(client, wsEnvelope{
				Type:      "response",
				RequestID: req.RequestID,
				Code:      http.StatusOK,
				Runtime:   state,
			})
		case "subscribe":
			d.addSubscriber(req.WorkspaceID, client)
			snapshot, ok := d.GetWorkspace(req.WorkspaceID)
			if ok {
				_ = d.writeEnvelope(client, wsEnvelope{
					Type:        "snapshot",
					WorkspaceID: req.WorkspaceID,
					Code:        http.StatusOK,
					Data:        snapshot,
				})
			}
			_ = d.writeEnvelope(client, wsEnvelope{
				Type:      "response",
				RequestID: req.RequestID,
				Code:      http.StatusOK,
			})
		case "unsubscribe":
			d.removeSubscriber(req.WorkspaceID, client)
			_ = d.writeEnvelope(client, wsEnvelope{
				Type:      "response",
				RequestID: req.RequestID,
				Code:      http.StatusOK,
			})
		case "intervention":
			var payload InterventionReq
			if err := json.Unmarshal(req.Payload, &payload); err != nil {
				_ = d.writeEnvelope(client, wsEnvelope{
					Type:      "response",
					RequestID: req.RequestID,
					Code:      http.StatusBadRequest,
					Msg:       "invalid intervention payload",
				})
				continue
			}
			snapshot, mutations, ok := d.ApplyIntervention(req.WorkspaceID, payload)
			if !ok {
				_ = d.writeEnvelope(client, wsEnvelope{
					Type:      "response",
					RequestID: req.RequestID,
					Code:      http.StatusNotFound,
					Msg:       "workspace not found",
				})
				continue
			}
			d.broadcastMutations(req.WorkspaceID, mutations)
			_ = d.writeEnvelope(client, wsEnvelope{
				Type:      "response",
				RequestID: req.RequestID,
				Code:      http.StatusOK,
				Data:      snapshot,
			})
		case "update_strategy":
			var payload UpdateStrategyReq
			if len(req.Payload) > 0 {
				if err := json.Unmarshal(req.Payload, &payload); err != nil {
					_ = d.writeEnvelope(client, wsEnvelope{
						Type:      "response",
						RequestID: req.RequestID,
						Code:      http.StatusBadRequest,
						Msg:       "invalid strategy payload",
					})
					continue
				}
			}
			snapshot, mutations, ok := d.UpdateStrategy(req.WorkspaceID, payload)
			if !ok {
				_ = d.writeEnvelope(client, wsEnvelope{
					Type:      "response",
					RequestID: req.RequestID,
					Code:      http.StatusNotFound,
					Msg:       "workspace not found",
				})
				continue
			}
			d.broadcastMutations(req.WorkspaceID, mutations)
			_ = d.writeEnvelope(client, wsEnvelope{
				Type:      "response",
				RequestID: req.RequestID,
				Code:      http.StatusOK,
				Data:      snapshot,
			})
		case "archive_workspace":
			if ok := d.ArchiveWorkspace(req.WorkspaceID); !ok {
				_ = d.writeEnvelope(client, wsEnvelope{
					Type:      "response",
					RequestID: req.RequestID,
					Code:      http.StatusNotFound,
					Msg:       "workspace not found",
				})
				continue
			}
			_ = d.writeEnvelope(client, wsEnvelope{
				Type:      "response",
				RequestID: req.RequestID,
				Code:      http.StatusOK,
			})
		case "replay_mutations":
			var payload struct {
				Since  int64  `json:"since"`
				Cursor string `json:"cursor"`
				Limit  int    `json:"limit"`
			}
			if len(req.Payload) > 0 {
				if err := json.Unmarshal(req.Payload, &payload); err != nil {
					_ = d.writeEnvelope(client, wsEnvelope{
						Type:      "response",
						RequestID: req.RequestID,
						Code:      http.StatusBadRequest,
						Msg:       "invalid replay payload",
					})
					continue
				}
			}
			cursor := payload.Cursor
			if cursor == "" && payload.Since > 0 {
				cursor = strconv.FormatInt(payload.Since, 10) + "|"
			}

			page, err := d.replayMutations(req.WorkspaceID, cursor, payload.Limit)
			if err != nil {
				_ = d.writeEnvelope(client, wsEnvelope{
					Type:      "response",
					RequestID: req.RequestID,
					Code:      http.StatusInternalServerError,
					Msg:       "failed to replay mutations",
				})
				continue
			}
			_ = d.writeEnvelope(client, wsEnvelope{
				Type:       "response",
				RequestID:  req.RequestID,
				Code:       http.StatusOK,
				Mutations:  page.Mutations,
				NextCursor: page.NextCursor,
				HasMore:    page.HasMore,
			})
		default:
			_ = d.writeEnvelope(client, wsEnvelope{
				Type:      "response",
				RequestID: req.RequestID,
				Code:      http.StatusBadRequest,
				Msg:       "unsupported action",
			})
		}
	}
}
