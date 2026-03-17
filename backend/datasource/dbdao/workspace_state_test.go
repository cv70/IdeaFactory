package dbdao

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUpsertWorkspaceStateByWorkspaceID(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&WorkspaceState{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	db := NewDB(gdb)

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
