package agents

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type staticAgent struct {
	name        string
	description string
}

func NewStaticAgent(name string, description string) adk.Agent {
	return staticAgent{name: name, description: description}
}

func (s staticAgent) Name(_ context.Context) string {
	return s.name
}

func (s staticAgent) Description(_ context.Context) string {
	return s.description
}

func (s staticAgent) Run(_ context.Context, _ *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(&adk.AgentEvent{
		AgentName: s.name,
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				IsStreaming: false,
				Message:     schema.AssistantMessage(s.name+" completed the task.", nil),
			},
		},
	})
	gen.Close()
	return iter
}
