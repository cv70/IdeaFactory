package idea

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.RouterGroup, domain *IdeaDomain) {
	group := router.Group("/ideas")
	{
		group.POST("/generate", domain.ApiGenerateIdeas)
		group.POST("/expand", domain.ApiExpandIdeas)
		group.POST("/regenerate-cluster", domain.ApiRegenerateCluster)
		group.POST("/favorites", domain.ApiSaveFavorite)
		group.GET("/favorites", domain.ApiListFavorites)
		group.DELETE("/favorites/:id", domain.ApiRemoveFavorite)
	}
}
