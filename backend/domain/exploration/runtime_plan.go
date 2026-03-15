package exploration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/kaptinlin/jsonrepair"
)

func buildInitialPlan(domain *ExplorationDomain, session ExplorationSession, runID string, now time.Time) (ExecutionPlan, []PlanStep) {
	planID := fmt.Sprintf("plan-%s-%d", session.ID, now.UnixNano())
	plan := ExecutionPlan{
		ID:          planID,
		WorkspaceID: session.ID,
		RunID:       runID,
		Version:     1,
		CreatedAt:   now.UnixMilli(),
	}

	stepDescs := []string{
		"collect research signals for current opportunities",
		"structure research into graph mutations and decisions",
		"materialize high-confidence idea cards and summary",
	}
	if domain != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		if generated := domain.generatePlanStepsWithModel(ctx, session, 3); len(generated) > 0 {
			for i := 0; i < len(stepDescs) && i < len(generated); i++ {
				if strings.TrimSpace(generated[i]) != "" {
					stepDescs[i] = strings.TrimSpace(generated[i])
				}
			}
		}
	}

	steps := []PlanStep{
		{
			ID:          fmt.Sprintf("%s-step-1", planID),
			WorkspaceID: session.ID,
			RunID:       runID,
			PlanID:      planID,
			Index:       1,
			Desc:        stepDescs[0],
			Status:      PlanStepTodo,
			UpdatedAt:   now.UnixMilli(),
		},
		{
			ID:          fmt.Sprintf("%s-step-2", planID),
			WorkspaceID: session.ID,
			RunID:       runID,
			PlanID:      planID,
			Index:       2,
			Desc:        stepDescs[1],
			Status:      PlanStepTodo,
			UpdatedAt:   now.UnixMilli(),
		},
		{
			ID:          fmt.Sprintf("%s-step-3", planID),
			WorkspaceID: session.ID,
			RunID:       runID,
			PlanID:      planID,
			Index:       3,
			Desc:        stepDescs[2],
			Status:      PlanStepTodo,
			UpdatedAt:   now.UnixMilli(),
		},
	}
	return plan, steps
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
