package exploration

import (
	"fmt"
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
	for _, result := range state.Results {
		resp.Items = append(resp.Items, TraceSummaryItem{
			ID:        "tool-" + result.TaskID,
			Timestamp: toRFC3339(result.UpdatedAt),
			Level:     "info",
			Category:  "tool",
			Message:   result.Summary,
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
	case "run", "tool", "mutation", "projection", "intervention", "balance":
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
