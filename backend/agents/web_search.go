package agents

import (
	"context"

	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// NewWebSearchAgent 创建一个网络搜索代理，使用DuckDuckGo工具和ReAct模型分析信息并完成任务。
// 参数:
//   - ctx: 上下文，用于控制取消和传递请求范围值。
//   - model: 用于工具调用的聊天模型。
//
// 返回:
//   - adk.Agent: 网络搜索代理实例。
//   - error: 如果创建过程中发生错误（例如工具初始化失败）。
func NewWebSearchAgent(ctx context.Context, model model.ToolCallingChatModel) (adk.Agent, error) {
	// 创建DuckDuckGo文本搜索工具
	searchTool, err := duckduckgo.NewTextSearchTool(ctx, &duckduckgo.Config{})
	if err != nil {
		return nil, err
	}

	// 使用聊天模型和搜索工具创建网络搜索代理
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "WebSearchAgent",
		Description: "WebSearchAgent utilizes the ReAct model to analyze input information and accomplish tasks using web search tools.",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{searchTool},
			},
		},
		MaxIterations: 10,
	})
}
