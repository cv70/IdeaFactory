package agentools

import (
	"encoding/json"
	"fmt"
)

type WrappedToolResponse struct {
	Tool    string         `json:"tool"`
	Payload map[string]any `json:"payload"`
	Summary string         `json:"summary"`
}

type RuntimeToolWrapper struct{}

func NewRuntimeToolWrapper() RuntimeToolWrapper {
	return RuntimeToolWrapper{}
}

func (RuntimeToolWrapper) NormalizeToolCall(toolName string, arguments string) (WrappedToolResponse, error) {
	if toolName == "" {
		return WrappedToolResponse{}, fmt.Errorf("tool name is required")
	}

	payload := map[string]any{}
	if err := json.Unmarshal([]byte(arguments), &payload); err != nil {
		return WrappedToolResponse{}, fmt.Errorf("decode tool args: %w", err)
	}

	return WrappedToolResponse{
		Tool:    toolName,
		Payload: payload,
		Summary: fmt.Sprintf("tool %s called with %d arguments", toolName, len(payload)),
	}, nil
}
