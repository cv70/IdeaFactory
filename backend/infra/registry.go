package infra

import (
	"backend/config"
	"backend/datasource/dbdao"
	"backend/datasource/scylladao"
	"backend/datasource/vectordao"
	"context"

	"backend/sdk"

	"github.com/cloudwego/eino/components/model"
	"github.com/redis/go-redis/v9"
)

type Registry struct {
	DB            *dbdao.DB
	Redis         *redis.Client
	Scylla        *scylladao.ScyllaDB
	VectorDB      *vectordao.VectorDB
	TextEmebdding sdk.EmbeddingClient
	LLM           model.ToolCallingChatModel
}

func NewRegistry(ctx context.Context, c *config.Config) (*Registry, error) {
	db, err := NewDB(ctx, c.Database)
	if err != nil {
		return nil, err
	}
	redis, err := NewRedis(ctx, c.Redis)
	if err != nil {
		return nil, err
	}
	scylla, err := NewScylla(ctx, c.Scylla)
	if err != nil {
		return nil, err
	}
	vectorDB, err := NewQdrant(ctx, c.Qdrant)
	if err != nil {
		return nil, err
	}
	textEmbedding, err := NewEmbeddingModel(ctx, c.TextEmbedding)
	if err != nil {
		return nil, err
	}
	llm, err := NewLLM(ctx, c.LLM)
	if err != nil {
		return nil, err
	}
	return &Registry{
		DB:            db,
		Redis:         redis,
		Scylla:        scylla,
		VectorDB:      vectorDB,
		TextEmebdding: textEmbedding,
		LLM:           llm,
	}, nil
}
