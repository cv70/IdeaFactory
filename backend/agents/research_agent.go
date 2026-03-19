package agents

import (
	"context"

	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// NewResearchAgent 创建一个研究代理，用于收集外部证据和信号。
// 参数:
//   - ctx: 上下文，用于控制取消和传递请求范围值。
//   - cm: 用于工具调用的聊天模型，如果为nil则返回一个静态代理作为后备。
//
// 返回:
//   - adk.Agent: 研究代理实例。
//   - error: 如果创建过程中发生错误。
func NewResearchAgent(ctx context.Context, cm model.ToolCallingChatModel, handlers []adk.ChatModelAgentMiddleware) (adk.Agent, error) {
	// 创建DuckDuckGo文本搜索工具
	searchTool, err := duckduckgo.NewTextSearchTool(ctx, &duckduckgo.Config{})
	if err != nil {
		return nil, err
	}

	// 使用聊天模型和搜索工具创建研究代理
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ResearchAgent",
		Description: "Collect evidence and external signals for current exploration directions.",
		Instruction: "Focus on collecting credible evidence and summarize key findings briefly.",
		Model:       cm,
		Handlers:    handlers,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{searchTool},
			},
		},
		MaxIterations: 8,
	})
}
