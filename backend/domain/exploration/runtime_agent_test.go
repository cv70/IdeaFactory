package exploration

import (
	"testing"
	"time"
)

func TestSnapshotForCycle_ReturnsTrueWhenTodoStepExists(t *testing.T) {
	domain := NewExplorationDomain(nil, nil)
	wsID := "ws-snap-test"
	// Manually set up state with one Todo step
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		s.Plans = []ExecutionPlan{{ID: "plan-1"}}
		s.PlanSteps = []PlanStep{
			{ID: "step-1", PlanID: "plan-1", Status: PlanStepTodo},
		}
		s.Balance = BalanceState{Divergence: 0.6, Research: 0.7, Aggression: 0.4}
	})
	domain.store.mu.Lock()
	domain.store.workspaces[wsID] = &ExplorationSession{ID: wsID, Topic: "test"}
	domain.store.mu.Unlock()

	_, _, hasTodo := domain.snapshotForCycle(wsID)
	if !hasTodo {
		t.Fatal("expected hasTodo=true when Todo step exists")
	}
}

func TestSnapshotForCycle_ReturnsFalseWhenNoTodoSteps(t *testing.T) {
	domain := NewExplorationDomain(nil, nil)
	wsID := "ws-snap-done"
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		s.Plans = []ExecutionPlan{{ID: "plan-1"}}
		s.PlanSteps = []PlanStep{
			{ID: "step-1", PlanID: "plan-1", Status: PlanStepDone},
		}
	})
	domain.store.mu.Lock()
	domain.store.workspaces[wsID] = &ExplorationSession{ID: wsID, Topic: "test"}
	domain.store.mu.Unlock()

	_, _, hasTodo := domain.snapshotForCycle(wsID)
	if hasTodo {
		t.Fatal("expected hasTodo=false when no Todo steps")
	}
}

func TestMarkStepDoneAndCheck_MarksFirstTodoDone(t *testing.T) {
	domain := NewExplorationDomain(nil, nil)
	wsID := "ws-mark-test"
	now := time.Now()
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		s.Plans = []ExecutionPlan{{ID: "plan-1", RunID: "run-1"}}
		s.PlanSteps = []PlanStep{
			{ID: "step-1", PlanID: "plan-1", RunID: "run-1", Status: PlanStepTodo, Index: 1},
			{ID: "step-2", PlanID: "plan-1", RunID: "run-1", Status: PlanStepTodo, Index: 2},
		}
		s.Runs = []Run{{ID: "run-1", Status: RunStatusRunning, StartedAt: now.UnixMilli()}}
	})
	done := domain.markStepDoneAndCheck(wsID)
	// Two steps remain after marking one done, so not fully done yet
	if done {
		t.Fatal("expected done=false with steps remaining")
	}
	var step1Status PlanStepStatus
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		step1Status = s.PlanSteps[0].Status
	})
	if step1Status != PlanStepDone {
		t.Errorf("expected step-1 to be Done, got %s", step1Status)
	}
}
