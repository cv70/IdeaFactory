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

	state := &WorkspaceState{Topic: "t"}
	if err := db.UpsertWorkspaceState(state); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	workspaceID := state.ID

	// pause
	if err := db.PauseWorkspaceState(workspaceID); err != nil {
		t.Fatalf("pause: %v", err)
	}
	state, err := db.GetWorkspaceState(workspaceID)
	if err != nil || state == nil {
		t.Fatalf("get after pause: %v", err)
	}
	if state.PausedAt == nil {
		t.Fatal("expected PausedAt to be set after pause")
	}

	// resume
	if err := db.ResumeWorkspaceState(workspaceID); err != nil {
		t.Fatalf("resume: %v", err)
	}
	state, err = db.GetWorkspaceState(workspaceID)
	if err != nil || state == nil {
		t.Fatalf("get after resume: %v", err)
	}
	if state.PausedAt != nil {
		t.Fatal("expected PausedAt to be nil after resume")
	}
}

func TestListActiveWorkspaceStates(t *testing.T) {
	db := newTestDB(t)

	ids := make(map[string]uint)
	for _, topic := range []string{"ws-b", "ws-c", "ws-a"} { // ws-a last → highest updated_at
		state := &WorkspaceState{Topic: topic}
		if err := db.UpsertWorkspaceState(state); err != nil {
			t.Fatalf("upsert %s: %v", topic, err)
		}
		ids[topic] = state.ID
	}
	if err := db.PauseWorkspaceState(ids["ws-b"]); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if err := db.ArchiveWorkspaceState(ids["ws-c"]); err != nil {
		t.Fatalf("archive: %v", err)
	}

	active, err := db.ListActiveWorkspaceStates(50)
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	// Use set-membership check — ordering may vary if timestamps are equal.
	found := map[string]bool{}
	for _, s := range active {
		found[s.Topic] = true
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

func TestUpsertWorkspaceStateByID(t *testing.T) {
	db := newTestDB(t)

	first := &WorkspaceState{
		Topic:      "AI education",
		OutputGoal: "Goal A",
	}
	if err := db.UpsertWorkspaceState(first); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	workspaceID := first.ID
	second := &WorkspaceState{
		Model:      first.Model,
		Topic:      "AI education",
		OutputGoal: "Goal B",
	}
	if err := db.UpsertWorkspaceState(second); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	row, err := db.GetWorkspaceState(workspaceID)
	if err != nil || row == nil {
		t.Fatalf("get workspace state: %v", err)
	}
	if row.OutputGoal != "Goal B" {
		t.Fatalf("expected latest output goal Goal B, got %s", row.OutputGoal)
	}
}
