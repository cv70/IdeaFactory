package exploration

import (
	"context"
	"testing"
	"time"
)

func TestScheduleNextRun_DoesNotScheduleWhenMaxRunsReached(t *testing.T) {
	// With nil DB, GetWorkspaceState returns nil → treated as "not paused" (safe degradation).
	// So we test the "MaxRuns reached" guard instead as a proxy for the early-exit paths.
	domain := newTestExplorationDomain()
	wsID := "ws-sched-maxruns"

	// Add workspace to store
	domain.store.mu.Lock()
	domain.store.workspaces[wsID] = &ExplorationSession{
		ID:       wsID,
		Topic:    "test",
		Strategy: RuntimeStrategy{MaxRuns: 1, IntervalMs: 0},
	}
	domain.store.mu.Unlock()

	// Seed one completed run so run count == MaxRuns
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		s.Runs = []Run{{ID: "run-1", WorkspaceID: wsID, Status: RunStatusCompleted}}
	})

	domain.scheduleNextRun(wsID)

	// cancelScheduler should NOT be set (no scheduler was launched)
	var cancelSet bool
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		cancelSet = s.cancelScheduler != nil
	})
	if cancelSet {
		t.Fatal("expected no scheduler to be launched when MaxRuns reached")
	}
}

func TestScheduleNextRun_DoesNotScheduleWhenAgentRunning(t *testing.T) {
	domain := newTestExplorationDomain()
	wsID := "ws-sched-running"

	domain.store.mu.Lock()
	domain.store.workspaces[wsID] = &ExplorationSession{
		ID:       wsID,
		Topic:    "test",
		Strategy: RuntimeStrategy{MaxRuns: 0, IntervalMs: 0},
	}
	domain.store.mu.Unlock()

	// Mark agent as already running
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		s.AgentRunning = true
	})

	domain.scheduleNextRun(wsID)

	var cancelSet bool
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		cancelSet = s.cancelScheduler != nil
	})
	if cancelSet {
		t.Fatal("expected no scheduler to be launched when AgentRunning=true")
	}
}

func TestPauseScheduler_CancelsWaitingScheduler(t *testing.T) {
	domain := newTestExplorationDomain()
	wsID := "ws-pause-sched"

	domain.store.mu.Lock()
	domain.store.workspaces[wsID] = &ExplorationSession{
		ID:       wsID,
		Topic:    "test",
		Strategy: RuntimeStrategy{MaxRuns: 0, IntervalMs: 60000}, // 60s — won't fire during test
	}
	domain.store.mu.Unlock()

	domain.scheduleNextRun(wsID)

	// cancelScheduler should be set now
	var cancelSetBefore bool
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		cancelSetBefore = s.cancelScheduler != nil
	})
	if !cancelSetBefore {
		t.Fatal("expected scheduler goroutine to be waiting (cancelScheduler set)")
	}

	domain.pauseScheduler(wsID)

	// cancelScheduler should be nil after pause
	var cancelSetAfter bool
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		cancelSetAfter = s.cancelScheduler != nil
	})
	if cancelSetAfter {
		t.Fatal("expected cancelScheduler to be nil after pauseScheduler")
	}
}

func TestSnapshotForCycle_ReturnsTrueWhenTodoStepExists(t *testing.T) {
	domain := newTestExplorationDomain()
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
	domain := newTestExplorationDomain()
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

func TestStart_DoesNotPanicWithNilDB(t *testing.T) {
	// The test domain has no DB — Start should handle this gracefully (no-op for DB path).
	domain := newTestExplorationDomain()
	ctx := context.Background()

	// Should not panic or error even with nil DB.
	if err := domain.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
}

func TestMarkStepDoneAndCheck_MarksFirstTodoDone(t *testing.T) {
	domain := newTestExplorationDomain()
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
	var agentTaskCount, resultCount int
	domain.withWorkspaceState(wsID, func(s *RuntimeWorkspaceState) {
		step1Status = s.PlanSteps[0].Status
		agentTaskCount = len(s.AgentTasks)
		resultCount = len(s.Results)
	})
	if step1Status != PlanStepDone {
		t.Errorf("expected step-1 to be Done, got %s", step1Status)
	}
	if agentTaskCount != 1 {
		t.Errorf("expected 1 AgentTask appended, got %d", agentTaskCount)
	}
	if resultCount != 1 {
		t.Errorf("expected 1 Result appended, got %d", resultCount)
	}
}
