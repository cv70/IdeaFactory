package exploration

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

func (d *ExplorationDomain) SummarizeTask(ctx context.Context, subAgent string, goal string) (string, error) {
	if d == nil || d.Model == nil {
		return "", fmt.Errorf("runtime model is nil")
	}

	prompt := fmt.Sprintf(
		"You are sub-agent %s. Task goal: %s. Return one sentence summary.",
		subAgent,
		goal,
	)
	msg, err := d.Model.Generate(ctx, []*schema.Message{schema.UserMessage(prompt)})
	if err != nil {
		return "", err
	}
	if msg != nil && strings.TrimSpace(msg.Content) != "" {
		return strings.TrimSpace(msg.Content), nil
	}
	return fmt.Sprintf("%s llm adapter executed task: %s", subAgent, strings.TrimSpace(goal)), nil
}
