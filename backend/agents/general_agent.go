package agents

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

func NewGeneralAgent(ctx context.Context, cm model.ToolCallingChatModel) (adk.Agent, error) {
	if cm == nil {
		return NewStaticAgent("GeneralAgent", "General fallback agent without model"), nil
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "GeneralAgent",
		Description:   "Handle generic exploration subtasks when no specialist agent fits.",
		Instruction:   "Provide pragmatic, concise completion for generic exploration tasks.",
		Model:         cm,
		MaxIterations: 6,
	})
}
