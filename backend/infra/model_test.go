package infra

import (
	"backend/config"
	"context"
	"log/slog"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/cv70/pkgo/mistake"
)

func TestModel(t *testing.T) {
	ctx := context.Background()
	c, err := config.LoadConfig()
	mistake.Unwrap(err)
	r, err := NewRegistry(ctx, c)
	mistake.Unwrap(err)
	resp, err := r.Model.Generate(ctx, []*schema.Message{
		schema.UserMessage("hello"),
	})
	mistake.Unwrap(err)
	slog.Info(resp.ReasoningContent)
	slog.Info(resp.Content)
}
