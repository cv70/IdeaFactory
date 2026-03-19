package infra

import (
	"backend/config"
	"backend/datasource/dbdao"
	"context"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewDB(ctx context.Context, c *config.DatabaseConfig) (*dbdao.DB, error) {
	db, err := gorm.Open(sqlite.Open(c.DB), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	// One-time destructive migration for the snapshot era schema. When the legacy
	// snapshot column exists, we reset exploration persistence tables and rebuild
	// the normalized schema from scratch.
	if db.Migrator().HasColumn(&dbdao.WorkspaceState{}, "snapshot") {
		_ = db.Migrator().DropTable(
			"workspace_runtime_states",
			&dbdao.RuntimeBalanceRecord{},
			&dbdao.RuntimeTaskResultRecord{},
			&dbdao.RuntimeAgentTaskRecord{},
			&dbdao.RuntimeRunRecord{},
			&dbdao.GraphEdge{},
			&dbdao.GraphNode{},
			&dbdao.WorkspaceState{},
		)
	}
	db.AutoMigrate(
		&dbdao.ExplorationSession{},
		&dbdao.GraphNode{},
		&dbdao.GraphEdge{},
		&dbdao.WorkspaceState{},
		&dbdao.RuntimeRunRecord{},
		&dbdao.RuntimeAgentTaskRecord{},
		&dbdao.RuntimeTaskResultRecord{},
		&dbdao.RuntimeBalanceRecord{},
		&dbdao.InterventionEvent{},
		&dbdao.MutationLog{},
	)
	return dbdao.NewDB(db), nil
}
