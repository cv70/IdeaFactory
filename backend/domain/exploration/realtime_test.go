package exploration

import (
	"testing"
	"time"
)

func TestRuntimeContinuouslyExpandsWorkspace(t *testing.T) {
	domain := NewExplorationDomain(nil)
	created := domain.CreateWorkspace(CreateWorkspaceReq{
		Topic:       "AI education",
		OutputGoal:  "Research directions",
		Constraints: "Low-cost",
	})
	initialRuns := len(created.Exploration.Runs)

	time.Sleep(4500 * time.Millisecond)

	updated, ok := domain.GetWorkspace(created.Exploration.ID)
	if !ok {
		t.Fatal("workspace should exist")
	}
	if len(updated.Exploration.Runs) <= initialRuns {
		t.Fatalf("expected runtime to append runs, before=%d after=%d", initialRuns, len(updated.Exploration.Runs))
	}
}

