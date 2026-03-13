package exploration

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.RouterGroup, domain *ExplorationDomain) {
	group := router.Group("/exploration")
	{
		group.GET("/ws", domain.ApiWebSocket)
		group.GET("/workspaces", domain.ApiListWorkspaces)
		group.POST("/workspaces", domain.ApiCreateWorkspace)
		group.GET("/workspaces/:workspaceID", domain.ApiGetWorkspace)
		group.GET("/workspaces/:workspaceID/mutations", domain.ApiReplayMutations)
		group.PUT("/workspaces/:workspaceID/strategy", domain.ApiUpdateStrategy)
		group.DELETE("/workspaces/:workspaceID", domain.ApiArchiveWorkspace)
		group.POST("/workspaces/:workspaceID/interventions", domain.ApiInterveneWorkspace)

		group.POST("/sessions", domain.ApiCreateSession)
		group.GET("/sessions/:workspaceID", domain.ApiGetSession)
		group.POST("/sessions/:workspaceID/opportunities/:opportunityID/expand", domain.ApiExpandOpportunity)
		group.POST("/sessions/:workspaceID/feedback", domain.ApiFeedback)
	}
}
