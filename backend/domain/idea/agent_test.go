package idea

import (
	"backend/config"
	"backend/infra"
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cv70/pkgo/mistake"
)

func newTestIdeaDomain() *IdeaDomain {
	ctx := context.Background()
	c, err := config.LoadConfig()
	mistake.Unwrap(err)
	r, err := infra.NewRegistry(ctx, c)
	mistake.Unwrap(err)
	id, err := BuildIdeaDomain(r)
	mistake.Unwrap(err)
	return id
}

type fakeToolCallingModel struct {
	replies []string
	idx     int
	fail    bool
}

func (f *fakeToolCallingModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if f.fail {
		return nil, errors.New("forced failure")
	}
	if f.idx >= len(f.replies) {
		return schema.AssistantMessage(`{"clusters":[]}`, nil), nil
	}
	out := f.replies[f.idx]
	f.idx++
	return schema.AssistantMessage(out, nil), nil
}

func (f *fakeToolCallingModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("not implemented")
}

func (f *fakeToolCallingModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return f, nil
}

func TestGenerateIdeasUsesAgentWhenLLMAvailable(t *testing.T) {
	d := newTestIdeaDomain()

	resp, err := d.GenerateIdeas(GenerateIdeasReq{
		Topic: "creator economy",
		Count: 6,
		Angle: "niche-first",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(resp.Clusters) == 0 {
		t.Fatalf("expected non-empty clusters")
	}
	found := false
	for _, c := range resp.Clusters {
		for _, idea := range c.Ideas {
			if idea.Name == "Agent Idea A1" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("expected agent-generated content, got %+v", resp.Clusters)
	}
	if resp.Meta.Source != "agent" {
		t.Fatalf("expected agent source, got %s", resp.Meta.Source)
	}
	if resp.Meta.Rounds < 1 || resp.Meta.Rounds > 3 {
		t.Fatalf("unexpected rounds: %d", resp.Meta.Rounds)
	}
}

func TestGenerateIdeasFallbackWhenAgentFails(t *testing.T) {
	d := newTestIdeaDomain()

	resp, err := d.GenerateIdeas(GenerateIdeasReq{
		Topic: "pet economy",
		Count: 8,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(resp.Clusters) == 0 {
		t.Fatalf("expected fallback clusters")
	}
	if resp.Meta.Source != "fallback" {
		t.Fatalf("expected fallback source, got %s", resp.Meta.Source)
	}
}
