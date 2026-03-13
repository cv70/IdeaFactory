package idea

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

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
	m := &fakeToolCallingModel{
		replies: []string{
			`{"round":1,"focuses":["audience","scenario"],"cluster_targets":{"audience":3,"scenario":3}}`,
			`{"clusters":[{"cluster_id":"audience","title":"Audience Slice","ideas":[{"id":"a1","name":"Agent Idea A1","one_liner":"A1","target_audience":"founders","core_scenario":"planning","value_point":"speed","business_tags":["tool"],"opportunity_tags":["niche"]},{"id":"a2","name":"Agent Idea A2","one_liner":"A2","target_audience":"operators","core_scenario":"planning","value_point":"speed","business_tags":["tool"],"opportunity_tags":["niche"]},{"id":"a3","name":"Agent Idea A3","one_liner":"A3","target_audience":"teams","core_scenario":"planning","value_point":"speed","business_tags":["tool"],"opportunity_tags":["niche"]}]},{"cluster_id":"scenario","title":"Scenario Slice","ideas":[{"id":"s1","name":"Agent Idea S1","one_liner":"S1","target_audience":"founders","core_scenario":"launch","value_point":"clarity","business_tags":["service"],"opportunity_tags":["efficiency"]},{"id":"s2","name":"Agent Idea S2","one_liner":"S2","target_audience":"operators","core_scenario":"retention","value_point":"clarity","business_tags":["service"],"opportunity_tags":["efficiency"]},{"id":"s3","name":"Agent Idea S3","one_liner":"S3","target_audience":"teams","core_scenario":"growth","value_point":"clarity","business_tags":["service"],"opportunity_tags":["efficiency"]}]}]}`,
			`{"reject_names":[],"next_focuses":["motivation"],"quality_score":4.2,"duplicate_rate":0.0}`,
		},
	}
	d := IdeaDomain{LLM: m}

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
	d := IdeaDomain{LLM: &fakeToolCallingModel{fail: true}}
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
