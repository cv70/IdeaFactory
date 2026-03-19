package exploration

import (
	"context"
	"testing"
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

func TestStart_DoesNotPanicWithNilDB(t *testing.T) {
	// The test domain has no DB — Start should handle this gracefully (no-op for DB path).
	domain := newTestExplorationDomain()
	ctx := context.Background()

	// Should not panic or error even with nil DB.
	if err := domain.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
}
