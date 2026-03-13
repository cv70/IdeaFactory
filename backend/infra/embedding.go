package infra

import (
	"backend/config"
	"context"

	"backend/sdk"
)

func NewEmbeddingModel(ctx context.Context, c *config.EmbeddingConfig) (sdk.EmbeddingClient, error) {
	embeddingClient := &sdk.AnythingEmbeddingClient{
		URL: c.BaseURL,
	}
	return embeddingClient, nil
}
