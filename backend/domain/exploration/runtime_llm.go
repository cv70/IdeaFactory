package exploration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/kaptinlin/jsonrepair"
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

func (d *ExplorationDomain) generatePlanStepsWithModel(ctx context.Context, session ExplorationSession, limit int) []string {
	if d == nil || d.Model == nil || limit <= 0 {
		return nil
	}

	prompt := fmt.Sprintf(
		"Generate %d concise execution steps for idea exploration topic '%s' with output goal '%s'. "+
			"Return JSON only: {\"steps\":[\"step1\",\"step2\",\"step3\"]}.",
		limit,
		strings.TrimSpace(session.Topic),
		strings.TrimSpace(session.OutputGoal),
	)
	msg, err := d.Model.Generate(ctx, []*schema.Message{schema.UserMessage(prompt)})
	if err != nil || msg == nil || strings.TrimSpace(msg.Content) == "" {
		return nil
	}

	repaired, err := jsonrepair.Repair(msg.Content)
	if err != nil {
		return nil
	}
	var parsed struct {
		Steps []string `json:"steps"`
	}
	if err := json.Unmarshal([]byte(repaired), &parsed); err != nil {
		return nil
	}

	out := make([]string, 0, limit)
	for _, item := range parsed.Steps {
		if strings.TrimSpace(item) == "" {
			continue
		}
		out = append(out, strings.TrimSpace(item))
		if len(out) >= limit {
			break
		}
	}
	return out
}
