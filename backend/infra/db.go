package infra

import (
	"backend/config"
	"backend/datasource/dbdao"
	"context"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDB(ctx context.Context, c *config.DatabaseConfig) (*dbdao.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True", c.User, c.Password, c.Host, c.DBName)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
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
