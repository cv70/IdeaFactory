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
	db.AutoMigrate(
		&dbdao.ExplorationSession{},
		&dbdao.GraphNode{},
		&dbdao.GraphEdge{},
		&dbdao.WorkspaceState{},
		&dbdao.WorkspaceRuntimeState{},
		&dbdao.RuntimeRunRecord{},
		&dbdao.RuntimePlanRecord{},
		&dbdao.RuntimePlanStepRecord{},
		&dbdao.RuntimeAgentTaskRecord{},
		&dbdao.RuntimeTaskResultRecord{},
		&dbdao.RuntimeBalanceRecord{},
		&dbdao.InterventionEvent{},
		&dbdao.MutationLog{},
	)
	return dbdao.NewDB(db), nil
}
