package exploration

import (
	"fmt"
	"strings"
	"time"
)

func buildTraceSummary(workspaceID string, runID string, state RuntimeStateSnapshot) TraceSummaryResponse {
	resp := TraceSummaryResponse{WorkspaceID: workspaceID, RunID: runID, Items: []TraceSummaryItem{}}
	for _, run := range state.Runs {
		resp.Items = append(resp.Items, TraceSummaryItem{
			ID:        "run-" + run.ID,
			Timestamp: toRFC3339(run.StartedAt),
			Level:     "info",
			Category:  "run",
			Message:   fmt.Sprintf("run %s started with source %s", run.ID, run.Source),
			RelatedIDs: []string{
				run.ID,
			},
		})
	}
	for _, event := range state.Events {
		category := "tool"
		switch event.EventType {
		case "agent_start", "run_summary", "run_error":
			category = "run"
		case "turn_started", "turn_completed", "turn_failed":
			category = "turn"
		case "agent_delegate":
			category = "tool"
		case "tool_call":
			category = "tool"
		case "control_action_received", "control_action_absorbed", "control_action_reflected":
			category = "control_action"
		}
		resp.Items = append(resp.Items, TraceSummaryItem{
			ID:        "event-" + event.ID,
			Timestamp: toRFC3339(event.CreatedAt),
			Level:     "info",
			Category:  category,
			Message:   event.Summary,
			RelatedIDs: []string{
				event.RunID,
			},
		})
	}
	for _, result := range state.Results {
		message := result.Summary
		if len(result.Timeline) > 0 {
			message = fmt.Sprintf("%s [timeline: %s]", message, strings.Join(result.Timeline, " -> "))
		}
		resp.Items = append(resp.Items, TraceSummaryItem{
			ID:        "tool-" + result.TaskID,
			Timestamp: toRFC3339(result.UpdatedAt),
			Level:     "info",
			Category:  "tool",
			Message:   message,
			RelatedIDs: []string{
				result.TaskID,
			},
		})
	}
	for _, mutation := range state.Mutations {
		resp.Items = append(resp.Items, TraceSummaryItem{
			ID:        mutation.ID,
			Timestamp: toRFC3339(mutation.CreatedAt),
			Level:     "info",
			Category:  "mutation",
			Message:   mutation.Kind,
			RelatedIDs: []string{
				mutation.WorkspaceID,
			},
		})
	}
	for _, action := range state.ControlActions {
		resp.Items = append(resp.Items, TraceSummaryItem{
			ID:        "control-action-" + action.ID,
			Timestamp: action.UpdatedAt,
			Level:     "info",
			Category:  "control_action",
			Message:   fmt.Sprintf("%s %s", action.Kind, action.Status),
			RelatedIDs: []string{
				action.ID,
			},
		})
	}
	if state.LatestReplanReason != "" {
		refID := ""
		if state.Balance.RunID != "" {
			refID = state.Balance.RunID
		} else {
			refID = "latest"
		}
		resp.Items = append(resp.Items, TraceSummaryItem{
			ID:        "intervention-" + refID,
			Timestamp: toRFC3339(state.Balance.UpdatedAt),
			Level:     "info",
			Category:  "intervention",
			Message:   state.LatestReplanReason,
		})
	}
	if len(resp.Items) == 0 {
		resp.Items = append(resp.Items, TraceSummaryItem{
			ID:        "projection-empty",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Level:     "info",
			Category:  "projection",
			Message:   "no trace events yet",
		})
	}
	return resp
}

func applyTracePagination(items []TraceSummaryItem, cursor string, limit int) ([]TraceSummaryItem, string, bool) {
	if len(items) == 0 {
		return items, "", false
	}
	start := 0
	if cursor != "" {
		cTime, cID, ok := parseOrderedCursor(cursor)
		if ok {
			start = len(items)
			for i := range items {
				ts, err := time.Parse(time.RFC3339, items[i].Timestamp)
				if err != nil {
					continue
				}
				if ts.After(cTime) || (ts.Equal(cTime) && items[i].ID > cID) {
					start = i
					break
				}
			}
		}
	}
	if start >= len(items) {
		return []TraceSummaryItem{}, "", false
	}
	filtered := items[start:]
	if len(filtered) <= limit {
		return filtered, "", false
	}
	page := filtered[:limit]
	last := page[len(page)-1]
	ts, err := time.Parse(time.RFC3339, last.Timestamp)
	if err != nil {
		return page, "", true
	}
	return page, buildOrderedCursor(ts, last.ID), true
}

func isValidTraceCategory(category string) bool {
	switch category {
	case "run", "turn", "tool", "approval", "mutation", "projection", "control_action", "intervention", "memory", "skill", "balance":
		return true
	default:
		return false
	}
}

func isValidTraceLevel(level string) bool {
	switch level {
	case "info", "warn", "error":
		return true
	default:
		return false
	}
}
