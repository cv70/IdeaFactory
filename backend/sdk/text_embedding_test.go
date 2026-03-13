package sdk

import (
	"context"
	"log/slog"
	"testing"

	"github.com/cv70/pkgo/gtime"
	"github.com/cv70/pkgo/mistake"
)

func TestTextEmbedding(t *testing.T) {
	defer gtime.LogTimeCost(0)()
	embeder := TextEmbeddingClient[float32]{
		URL: "http://10.5.55.55:18087/v1/embeddings",
	}
	ctx := context.Background()
	vectors, err := embeder.Embedding(ctx, "你好", "how are you")
	mistake.Unwrap(err)
	slog.Info("vectors", slog.Any("len", len(vectors[0])), slog.Any("vectors", vectors))
}
