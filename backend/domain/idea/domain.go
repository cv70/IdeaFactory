package idea

import (
	"backend/datasource/dbdao"

	"github.com/cloudwego/eino/components/model"
)

type IdeaDomain struct {
	DB    *dbdao.DB
	LLM   model.ToolCallingChatModel
}
