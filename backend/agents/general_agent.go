package agents

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// NewGeneralAgent 创建一个通用代理，用于处理没有专门代理能处理的通用探索子任务。
// 参数:
//   - ctx: 上下文，用于控制取消和传递请求范围值。
//   - cm: 用于工具调用的聊天模型，如果为nil则返回一个静态代理作为后备。
//
// 返回:
//   - adk.Agent: 通用代理实例。
//   - error: 如果创建过程中发生错误。
func NewGeneralAgent(ctx context.Context, cm model.ToolCallingChatModel) (adk.Agent, error) {
	// 使用聊天模型创建通用代理
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "GeneralAgent",
		Description:   "Handle generic exploration subtasks when no specialist agent fits.",
		Instruction:   "Provide pragmatic, concise completion for generic exploration tasks.",
		Model:         cm,
		MaxIterations: 6,
	})
}
