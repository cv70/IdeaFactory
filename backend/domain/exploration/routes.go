package exploration

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.RouterGroup, domain *ExplorationDomain) {
	group := router.Group("/exploration")
	{
		group.POST("/sessions", domain.ApiCreateSession)
	}
}
