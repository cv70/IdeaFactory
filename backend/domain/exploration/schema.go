package exploration

type CreateSessionReq struct {
	WorkspaceID string `json:"workspace_id" binding:"required"`
	Topic       string `json:"topic" binding:"required"`
}
