package agents

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

func NewArtifactAgent(ctx context.Context, cm model.ToolCallingChatModel) (adk.Agent, error) {
	if cm == nil {
		return NewStaticAgent("ArtifactAgent", "Artifact generation fallback without model"), nil
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "ArtifactAgent",
		Description:   "Materialize outputs into concise, usable artifacts.",
		Instruction:   "Produce compact high-signal artifact output for idea exploration progress.",
		Model:         cm,
		MaxIterations: 6,
	})
}
