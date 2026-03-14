package agents

import (
	"context"

	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

func NewResearchAgent(ctx context.Context, cm model.ToolCallingChatModel) (adk.Agent, error) {
	if cm == nil {
		return NewStaticAgent("ResearchAgent", "Research agent fallback without model"), nil
	}

	searchTool, err := duckduckgo.NewTextSearchTool(ctx, &duckduckgo.Config{})
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ResearchAgent",
		Description: "Collect evidence and external signals for current exploration directions.",
		Instruction: "Focus on collecting credible evidence and summarize key findings briefly.",
		Model:       cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{searchTool},
			},
		},
		MaxIterations: 8,
	})
}
