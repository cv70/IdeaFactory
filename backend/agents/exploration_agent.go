package agents

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
)

// NewExplorationAgent 创建一个探索主代理，它是一个深度代理，包含多个子代理。
// 这个函数组装了研究代理、图代理、制品代理和通用代理作为子代理，
// 并使用Eino的deep包创建一个主探索代理。
// 参数:
//   - ctx: 上下文，用于控制取消和传递请求范围值。
//   - cm: 用于工具调用的聊天模型，如果为nil则返回错误。
//
// 返回:
//   - adk.ResumableAgent: 可恢复的探索主代理实例。
//   - error: 如果创建过程中发生错误（例如模型为nil或子代理创建失败）。
func NewExplorationAgent(ctx context.Context, cm model.ToolCallingChatModel, graphTools ...tool.BaseTool) (adk.ResumableAgent, error) {
	// 创建研究代理，用于收集外部证据和信号
	researchAgent, err := NewResearchAgent(ctx, cm)
	if err != nil {
		return nil, err
	}
	// 创建图代理，用于处理图相关操作
	graphAgent, err := NewGraphAgent(ctx, cm, graphTools...)
	if err != nil {
		return nil, err
	}
	// 创建制品代理，用于将输出物化为简洁可用的制品
	artifactAgent, err := NewArtifactAgent(ctx, cm)
	if err != nil {
		return nil, err
	}
	// 创建通用代理，用于处理通用任务
	generalAgent, err := NewGeneralAgent(ctx, cm)
	if err != nil {
		return nil, err
	}
	// 使用深度代理构建器创建主探索代理，组合所有子代理
	explorationAgent, err := deep.New(ctx, &deep.Config{
		Name:        "exploration-main-agent",
		Description: "Main deep agent for exploration runtime tasks",
		ChatModel:   cm,
		SubAgents: []adk.Agent{
			researchAgent,
			graphAgent,
			artifactAgent,
			generalAgent,
		},
		MaxIteration: 100,
	})
	return explorationAgent, err
}
