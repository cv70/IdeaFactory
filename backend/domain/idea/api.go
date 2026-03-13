package idea

import (
	"backend/utils"
	"log/slog"

	"github.com/gin-gonic/gin"
)

func (d *IdeaDomain) ApiGenerateIdeas(c *gin.Context) {
	var req GenerateIdeasReq
	err := c.ShouldBind(&req)
	if err != nil {
		slog.Error("failed to parse generate ideas request", slog.Any("e", err))
		utils.RespError(c, 400, "failed to parse request")
		return
	}

	resp, err := d.GenerateIdeas(req)
	if err != nil {
		slog.Error("failed to generate ideas", slog.Any("e", err))
		utils.RespError(c, 500, "failed to generate ideas")
		return
	}

	utils.RespSuccess(c, resp)
}

func (d *IdeaDomain) ApiExpandIdeas(c *gin.Context) {
	var req ExpandIdeasReq
	err := c.ShouldBind(&req)
	if err != nil {
		slog.Error("failed to parse expand ideas request", slog.Any("e", err))
		utils.RespError(c, 400, "failed to parse request")
		return
	}

	resp, err := d.ExpandIdeas(req)
	if err != nil {
		slog.Error("failed to expand ideas", slog.Any("e", err))
		utils.RespError(c, 500, "failed to expand ideas")
		return
	}

	utils.RespSuccess(c, resp)
}

func (d *IdeaDomain) ApiRegenerateCluster(c *gin.Context) {
	var req RegenerateClusterReq
	err := c.ShouldBind(&req)
	if err != nil {
		slog.Error("failed to parse regenerate cluster request", slog.Any("e", err))
		utils.RespError(c, 400, "failed to parse request")
		return
	}

	resp, err := d.RegenerateCluster(req)
	if err != nil {
		slog.Error("failed to regenerate cluster", slog.Any("e", err))
		utils.RespError(c, 500, "failed to regenerate cluster")
		return
	}
	utils.RespSuccess(c, resp)
}

func (d *IdeaDomain) ApiSaveFavorite(c *gin.Context) {
	var req SaveFavoriteReq
	err := c.ShouldBind(&req)
	if err != nil {
		slog.Error("failed to parse save favorite request", slog.Any("e", err))
		utils.RespError(c, 400, "failed to parse request")
		return
	}
	err = d.SaveFavorite(req)
	if err != nil {
		slog.Error("failed to save favorite", slog.Any("e", err))
		utils.RespError(c, 400, "failed to save favorite")
		return
	}
	utils.RespSuccess(c, gin.H{"saved": true})
}

func (d *IdeaDomain) ApiListFavorites(c *gin.Context) {
	utils.RespSuccess(c, d.ListFavorites())
}

func (d *IdeaDomain) ApiRemoveFavorite(c *gin.Context) {
	id := c.Param("id")
	err := d.RemoveFavorite(id)
	if err != nil {
		slog.Error("failed to remove favorite", slog.Any("e", err))
		utils.RespError(c, 400, "failed to remove favorite")
		return
	}
	utils.RespSuccess(c, gin.H{"removed": true})
}
