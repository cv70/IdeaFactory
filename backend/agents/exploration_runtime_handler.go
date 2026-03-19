package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
)

const (
	RuntimeEventAgentStart    = "agent_start"
	RuntimeEventAgentDelegate = "agent_delegate"
	RuntimeEventToolCall      = "tool_call"
)

type RuntimeEvent struct {
	EventType string         `json:"event_type"`
	Actor     string         `json:"actor"`
	Target    string         `json:"target,omitempty"`
	Summary   string         `json:"summary"`
	Payload   map[string]any `json:"payload,omitempty"`
}

type runtimeEventHandler struct {
	*adk.BaseChatModelAgentMiddleware
	agentName string
	rootName  string
}

func newRuntimeEventHandler(agentName, rootName string) adk.ChatModelAgentMiddleware {
	return &runtimeEventHandler{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		agentName:                    agentName,
		rootName:                     rootName,
	}
}

func (h *runtimeEventHandler) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	event := RuntimeEvent{
		Actor: h.agentName,
	}
	if h.agentName == h.rootName {
		event.EventType = RuntimeEventAgentStart
		event.Summary = fmt.Sprintf("%s started a new exploration run", h.agentName)
	} else {
		event.EventType = RuntimeEventAgentDelegate
		event.Actor = h.rootName
		event.Target = h.agentName
		event.Summary = fmt.Sprintf("%s delegated work to %s", h.rootName, h.agentName)
	}
	_ = adk.SendEvent(ctx, &adk.AgentEvent{
		AgentName: h.agentName,
		Output:    &adk.AgentOutput{CustomizedOutput: event},
	})
	return ctx, runCtx, nil
}

func (h *runtimeEventHandler) WrapModel(ctx context.Context, m model.BaseChatModel, mc *adk.ModelContext) (model.BaseChatModel, error) {
	return m, nil
}

func (h *runtimeEventHandler) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	return func(callCtx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		_ = adk.SendEvent(callCtx, &adk.AgentEvent{
			AgentName: h.agentName,
			Output: &adk.AgentOutput{
				CustomizedOutput: RuntimeEvent{
					EventType: RuntimeEventToolCall,
					Actor:     h.agentName,
					Target:    tCtx.Name,
					Summary:   fmt.Sprintf("%s called %s", h.agentName, tCtx.Name),
					Payload: map[string]any{
						"args_summary": summarizeToolArguments(tCtx.Name, argumentsInJSON),
					},
				},
			},
		})
		return endpoint(callCtx, argumentsInJSON, opts...)
	}, nil
}

func summarizeToolArguments(toolName string, raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		if len(trimmed) > 160 {
			return trimmed[:160]
		}
		return trimmed
	}
	switch toolName {
	case "append_graph_batch":
		nodes, _ := payload["nodes"].([]any)
		edges, _ := payload["edges"].([]any)
		workspaceID, _ := payload["workspace_id"].(string)
		return fmt.Sprintf("workspace=%s nodes=%d edges=%d", workspaceID, len(nodes), len(edges))
	default:
		keys := make([]string, 0, len(payload))
		for key := range payload {
			keys = append(keys, key)
		}
		return fmt.Sprintf("keys=%s", strings.Join(keys, ","))
	}
}
