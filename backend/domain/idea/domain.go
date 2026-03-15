package idea

import (
	"backend/datasource/dbdao"
	"backend/infra"

	"github.com/cloudwego/eino/components/model"
)

type IdeaDomain struct {
	DB    *dbdao.DB
	Model model.ToolCallingChatModel
}

func BuildIdeaDomain(registry *infra.Registry) (*IdeaDomain, error) {
	return &IdeaDomain{
		DB: registry.DB,
		Model: registry.Model,
	}, nil
}
