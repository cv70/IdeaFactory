package exploration

import (
	"fmt"
	"time"
)

func subAgentForStep(index int) string {
	switch index {
	case 1:
		return "research"
	case 2:
		return "graph"
	case 3:
		return "artifact"
	default:
		return "general"
	}
}

func dispatchPlanSteps(session ExplorationSession, runID string, plan ExecutionPlan, steps []PlanStep, now time.Time) ([]PlanStep, []AgentTask, []AgentTaskResultSummary) {
	nextSteps := make([]PlanStep, 0, len(steps))
	tasks := make([]AgentTask, 0, len(steps))
	results := make([]AgentTaskResultSummary, 0, len(steps))

	for i := range steps {
		step := steps[i]
		if i > 0 {
			step.Status = PlanStepTodo
			step.UpdatedAt = now.UnixMilli()
			nextSteps = append(nextSteps, step)
			continue
		}

		taskID := fmt.Sprintf("task-%s-%d", plan.ID, step.Index)
		subAgent := subAgentForStep(step.Index)

		step.Status = PlanStepDoing
		step.UpdatedAt = now.UnixMilli()
		step.Status = PlanStepDone
		step.UpdatedAt = now.UnixMilli()
		nextSteps = append(nextSteps, step)

		task := AgentTask{
			ID:          taskID,
			WorkspaceID: session.ID,
			RunID:       runID,
			PlanID:      plan.ID,
			PlanStepID:  step.ID,
			SubAgent:    subAgent,
			Goal:        step.Desc,
			Status:      PlanStepDone,
			UpdatedAt:   now.UnixMilli(),
		}
		tasks = append(tasks, task)

		results = append(results, AgentTaskResultSummary{
			TaskID:    taskID,
			Summary:   fmt.Sprintf("%s step completed", subAgent),
			IsSuccess: true,
			UpdatedAt: now.UnixMilli(),
		})
	}

	return nextSteps, tasks, results
}
