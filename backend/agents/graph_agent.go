package agents

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// NewGraphAgent 创建一个图代理，用于将证据转换为图结构和决策。
// 参数:
//   - ctx: 上下文，用于控制取消和传递请求范围值。
//   - cm: 用于工具调用的聊天模型，如果为nil则返回一个静态代理作为后备。
//
// 返回:
//   - adk.Agent: 图代理实例。
//   - error: 如果创建过程中发生错误。
func NewGraphAgent(ctx context.Context, cm model.ToolCallingChatModel) (adk.Agent, error) {
	// 使用聊天模型创建图代理
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "GraphAgent",
		Description:   "Transform evidence into graph-oriented structure and decisions.",
		Instruction:   "Structure findings into clear nodes, relations, and decision candidates.",
		Model:         cm,
		MaxIterations: 6,
	})
}
