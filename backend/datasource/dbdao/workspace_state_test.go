package dbdao

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestDB creates an isolated in-memory SQLite database for a single test.
func newTestDB(t *testing.T) *DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&WorkspaceState{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return NewDB(gdb)
}

func TestPauseAndResumeWorkspaceState(t *testing.T) {
	db := newTestDB(t)

	if err := db.UpsertWorkspaceState(&WorkspaceState{WorkspaceID: "ws-pause-test", Topic: "t"}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// pause
	if err := db.PauseWorkspaceState("ws-pause-test"); err != nil {
		t.Fatalf("pause: %v", err)
	}
	state, err := db.GetWorkspaceState("ws-pause-test")
	if err != nil || state == nil {
		t.Fatalf("get after pause: %v", err)
	}
	if state.PausedAt == nil {
		t.Fatal("expected PausedAt to be set after pause")
	}

	// resume
	if err := db.ResumeWorkspaceState("ws-pause-test"); err != nil {
		t.Fatalf("resume: %v", err)
	}
	state, err = db.GetWorkspaceState("ws-pause-test")
	if err != nil || state == nil {
		t.Fatalf("get after resume: %v", err)
	}
	if state.PausedAt != nil {
		t.Fatal("expected PausedAt to be nil after resume")
	}
}

func TestListActiveWorkspaceStates(t *testing.T) {
	db := newTestDB(t)

	for _, id := range []string{"ws-b", "ws-c", "ws-a"} { // ws-a last → highest updated_at
		if err := db.UpsertWorkspaceState(&WorkspaceState{WorkspaceID: id, Topic: id}); err != nil {
			t.Fatalf("upsert %s: %v", id, err)
		}
	}
	if err := db.PauseWorkspaceState("ws-b"); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if err := db.ArchiveWorkspaceState("ws-c"); err != nil {
		t.Fatalf("archive: %v", err)
	}

	active, err := db.ListActiveWorkspaceStates(50)
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	// Use set-membership check — ordering may vary if timestamps are equal.
	found := map[string]bool{}
	for _, s := range active {
		found[s.WorkspaceID] = true
	}
	if !found["ws-a"] {
		t.Fatalf("expected ws-a in active results, got: %v", active)
	}
	if found["ws-b"] {
		t.Fatal("expected ws-b to be excluded (paused)")
	}
	if found["ws-c"] {
		t.Fatal("expected ws-c to be excluded (archived)")
	}
}

func TestUpsertWorkspaceStateByWorkspaceID(t *testing.T) {
	db := newTestDB(t)
	gdb := db.DB()

	first := &WorkspaceState{
		WorkspaceID: "ws-1",
		Topic:       "AI education",
		OutputGoal:  "Goal A",
		Snapshot:    `{"version":1}`,
	}
	if err := db.UpsertWorkspaceState(first); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	second := &WorkspaceState{
		WorkspaceID: "ws-1",
		Topic:       "AI education",
		OutputGoal:  "Goal B",
		Snapshot:    `{"version":2}`,
	}
	if err := db.UpsertWorkspaceState(second); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	var rows []WorkspaceState
	if err := gdb.Where("workspace_id = ?", "ws-1").Find(&rows).Error; err != nil {
		t.Fatalf("query workspace rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row for workspace_id ws-1, got %d", len(rows))
	}
	if rows[0].OutputGoal != "Goal B" {
		t.Fatalf("expected latest output goal Goal B, got %s", rows[0].OutputGoal)
	}
	if rows[0].Snapshot != `{"version":2}` {
		t.Fatalf("expected latest snapshot, got %s", rows[0].Snapshot)
	}
}
