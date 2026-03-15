package infra

import (
	"backend/config"
	"backend/datasource/dbdao"
	"context"

	"github.com/cloudwego/eino/components/model"
)

type Registry struct {
	DB            *dbdao.DB
	Model         model.ToolCallingChatModel
}

func NewRegistry(ctx context.Context, c *config.Config) (*Registry, error) {
	db, err := NewDB(ctx, c.Database)
	if err != nil {
		return nil, err
	}
	model, err := NewModel(ctx, c.Model)
	if err != nil {
		return nil, err
	}
	return &Registry{
		DB:            db,
		Model:         model,
	}, nil
}
