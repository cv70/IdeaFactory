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
	rootName := "exploration-main-agent"

	// 创建研究代理，用于收集外部证据和信号
	researchAgent, err := NewResearchAgent(ctx, cm, []adk.ChatModelAgentMiddleware{
		newRuntimeEventHandler("ResearchAgent", rootName),
	})
	if err != nil {
		return nil, err
	}
	// 创建图代理，用于处理图相关操作
	graphAgent, err := NewGraphAgent(ctx, cm, []adk.ChatModelAgentMiddleware{
		newRuntimeEventHandler("GraphAgent", rootName),
	}, graphTools...)
	if err != nil {
		return nil, err
	}
	// 创建制品代理，用于将输出物化为简洁可用的制品
	artifactAgent, err := NewArtifactAgent(ctx, cm, []adk.ChatModelAgentMiddleware{
		newRuntimeEventHandler("ArtifactAgent", rootName),
	})
	if err != nil {
		return nil, err
	}
	// 创建通用代理，用于处理通用任务
	generalAgent, err := NewGeneralAgent(ctx, cm, []adk.ChatModelAgentMiddleware{
		newRuntimeEventHandler("GeneralAgent", rootName),
	})
	if err != nil {
		return nil, err
	}
	// 使用深度代理构建器创建主探索代理，组合所有子代理
	explorationAgent, err := deep.New(ctx, &deep.Config{
		Name:        rootName,
		Description: "Main deep agent for exploration runtime tasks",
		ChatModel:   cm,
		Instruction: `You coordinate IdeaFactory exploration runs.

Decide whether the workspace graph should grow during this run, delegate only when it helps finish the current run, and keep every graph mutation append-only through append_graph_batch.

Prefer GraphAgent for graph growth and graph-structure decisions.
Use ResearchAgent only when missing evidence blocks a concrete next graph mutation.
Use ArtifactAgent only when a concise packaged result helps close the current run.
Use GeneralAgent only for coordination work that no specialist agent fits.
Avoid open-ended exploration. Make the smallest high-value graph progress possible for each run.

When the run is complete, your final assistant message must be exactly one line:
SUMMARY: <brief result of this run>

Do not end with JSON. Do not omit the SUMMARY line.`,
		SubAgents: []adk.Agent{
			researchAgent,
			graphAgent,
			artifactAgent,
			generalAgent,
		},
		MaxIteration: 100,
		Handlers: []adk.ChatModelAgentMiddleware{
			newRuntimeEventHandler(rootName, rootName),
		},
	})
	return explorationAgent, err
}
