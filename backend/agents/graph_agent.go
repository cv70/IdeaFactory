package agents

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

func NewGraphAgent(ctx context.Context, cm model.ToolCallingChatModel) (adk.Agent, error) {
	if cm == nil {
		return NewStaticAgent("GraphAgent", "Graph structuring fallback without model"), nil
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "GraphAgent",
		Description:   "Transform evidence into graph-oriented structure and decisions.",
		Instruction:   "Structure findings into clear nodes, relations, and decision candidates.",
		Model:         cm,
		MaxIterations: 6,
	})
}
