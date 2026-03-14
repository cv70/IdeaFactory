package agents

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/model"
)

func BuildExplorationAgent(ctx context.Context, cm model.ToolCallingChatModel) (adk.ResumableAgent, error) {
	if cm == nil {
		return nil, fmt.Errorf("chat model is nil")
	}

	researchAgent, err := NewResearchAgent(ctx, cm)
	if err != nil {
		return nil, err
	}
	graphAgent, err := NewGraphAgent(ctx, cm)
	if err != nil {
		return nil, err
	}
	artifactAgent, err := NewArtifactAgent(ctx, cm)
	if err != nil {
		return nil, err
	}
	generalAgent, err := NewGeneralAgent(ctx, cm)
	if err != nil {
		return nil, err
	}
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
		MaxIteration: 6,
	})
	return explorationAgent, err
}
