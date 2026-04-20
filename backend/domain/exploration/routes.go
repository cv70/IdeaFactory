package exploration

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.RouterGroup, domain *ExplorationDomain) {
	group := router.Group("/exploration")
	{
		group.GET("/ws", domain.ApiWebSocket)
		group.GET("/workspaces", domain.ApiListWorkspaces)
		group.POST("/workspaces", domain.ApiCreateWorkspace)
		group.GET("/workspaces/:workspaceID", domain.ApiGetWorkspace)
		group.GET("/workspaces/:workspaceID/runtime", domain.ApiGetRuntimeState)
		group.GET("/workspaces/:workspaceID/mutations", domain.ApiReplayMutations)
		group.PUT("/workspaces/:workspaceID/strategy", domain.ApiUpdateStrategy)
		group.DELETE("/workspaces/:workspaceID", domain.ApiArchiveWorkspace)
		group.POST("/workspaces/:workspaceID/interventions", domain.ApiInterveneWorkspace)

		group.POST("/sessions", domain.ApiCreateSession)
		group.GET("/sessions/:workspaceID", domain.ApiGetSession)
		group.POST("/sessions/:workspaceID/opportunities/:opportunityID/expand", domain.ApiExpandOpportunity)
		group.POST("/sessions/:workspaceID/feedback", domain.ApiFeedback)
	}

	// Target-state contract routes aligned with docs/superpowers/specs/idea-factory-openapi.yaml
	v1 := router.Group("")
	{
		v1.POST("/workspaces", domain.ApiV1CreateWorkspace)
		v1.GET("/workspaces/:workspaceID", domain.ApiV1GetWorkspace)
		v1.PATCH("/workspaces/:workspaceID", domain.ApiV1PatchWorkspace)
		v1.POST("/workspaces/:workspaceID/runs", domain.ApiV1CreateRun)
		v1.GET("/workspaces/:workspaceID/runs/:runID", domain.ApiV1GetRun)
		v1.GET("/workspaces/:workspaceID/projection", domain.ApiV1GetProjection)
		v1.POST("/workspaces/:workspaceID/control-actions", domain.ApiV1CreateControlAction)
		v1.GET("/workspaces/:workspaceID/control-actions/:controlActionID", domain.ApiV1GetControlAction)
		v1.GET("/workspaces/:workspaceID/control-actions/:controlActionID/events", domain.ApiV1ListControlActionEvents)
		v1.POST("/workspaces/:workspaceID/interventions", domain.ApiV1CreateIntervention)
		v1.GET("/workspaces/:workspaceID/interventions/:interventionID", domain.ApiV1GetIntervention)
		v1.GET("/workspaces/:workspaceID/interventions/:interventionID/events", domain.ApiV1ListInterventionEvents)
		v1.GET("/workspaces/:workspaceID/trace/summary", domain.ApiV1GetTraceSummary)
		v1.GET("/workspaces/:workspaceID/trace/events", domain.ApiV1ListTraceEvents)
	}
}
