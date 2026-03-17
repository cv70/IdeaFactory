package agents

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// NewArtifactAgent 创建一个制品代理，用于将输出物化为简洁可用的制品。
// 参数:
//   - ctx: 上下文，用于控制取消和传递请求范围值。
//   - cm: 用于工具调用的聊天模型，如果为nil则返回一个静态代理作为后备。
//
// 返回:
//   - adk.Agent: 制品代理实例。
//   - error: 如果创建过程中发生错误。
func NewArtifactAgent(ctx context.Context, cm model.ToolCallingChatModel) (adk.Agent, error) {
	// 使用聊天模型创建制品代理
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "ArtifactAgent",
		Description:   "Materialize outputs into concise, usable artifacts.",
		Instruction:   "Produce compact high-signal artifact output for idea exploration progress.",
		Model:         cm,
		MaxIterations: 6,
	})
}
