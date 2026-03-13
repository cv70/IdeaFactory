package sdk

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/cv70/pkgo/gtime"
	"github.com/cv70/pkgo/mistake"
)

func TestImageEmbedding(t *testing.T) {
	defer gtime.LogTimeCost(0)()
	embeder := ImageEmbeddingClient[float32]{
		URL: "http://10.5.55.55:18086/image/embedding",
	}
	ctx := context.Background()
	imageData, err := os.ReadFile("/home/x/space/training_cv/models/pp_structure_v3/img_question.jpg")
	mistake.Unwrap(err)
	imageReader := bytes.NewReader(imageData)
	vectors, err := embeder.Embedding(ctx, imageReader)
	mistake.Unwrap(err)
	slog.Info("vectors", slog.Any("len", len(vectors[0])), slog.Any("vectors", vectors))
}
