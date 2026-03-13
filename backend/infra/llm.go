package infra

import (
	"backend/config"
	"context"
	"net/http"
	"time"

	"github.com/cv70/pkgo/conv"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/components/model"
)

func NewLLM(ctx context.Context, config *config.LLMConfig, options ...func(*qwen.ChatModelConfig)) (model.ToolCallingChatModel, error) {
	qwenConfig := &qwen.ChatModelConfig{}
	buildQwenModelOptionalConfig(qwenConfig, config)
	for _, opt := range options {
		opt(qwenConfig)
	}
	chatModel, err := qwen.NewChatModel(ctx, qwenConfig)
	return chatModel, err
}

func LLMWithModel(model string) func(*qwen.ChatModelConfig) {
	return func(c *qwen.ChatModelConfig) {
		c.Model = model
	}
}

func LLMWithTimeout(timeout time.Duration) func(*qwen.ChatModelConfig) {
	return func(c *qwen.ChatModelConfig) {
		c.Timeout = timeout
	}
}

func LLMWithHTTPClient(httpClient *http.Client) func(*qwen.ChatModelConfig) {
	return func(c *qwen.ChatModelConfig) {
		c.HTTPClient = httpClient
	}
}

func LLMWithMaxTokens(maxTokens int) func(*qwen.ChatModelConfig) {
	return func(c *qwen.ChatModelConfig) {
		c.MaxTokens = conv.Ptr(maxTokens)
	}
}

func LLMWithTemperature(temperature float32) func(*qwen.ChatModelConfig) {
	return func(c *qwen.ChatModelConfig) {
		c.Temperature = conv.Ptr(temperature)
	}
}

func LLMWithTopP(topP float32) func(*qwen.ChatModelConfig) {
	return func(c *qwen.ChatModelConfig) {
		c.TopP = conv.Ptr(topP)
	}
}

func LLMWithPresencePenalty(presencePenalty float32) func(*qwen.ChatModelConfig) {
	return func(c *qwen.ChatModelConfig) {
		c.PresencePenalty = conv.Ptr(presencePenalty)
	}
}

func LLMWithFrequencyPenalty(frequencyPenalty float32) func(*qwen.ChatModelConfig) {
	return func(c *qwen.ChatModelConfig) {
		c.FrequencyPenalty = conv.Ptr(frequencyPenalty)
	}
}

func LLMWithResponseFormat(responseFormat *openai.ChatCompletionResponseFormat) func(*qwen.ChatModelConfig) {
	return func(c *qwen.ChatModelConfig) {
		c.ResponseFormat = responseFormat
	}
}

func buildQwenModelOptionalConfig(dstConfig *qwen.ChatModelConfig, srcConfig *config.LLMConfig) {
	if dstConfig == nil || srcConfig == nil {
		return
	}
	dstConfig.BaseURL = srcConfig.BaseURL
	dstConfig.Model = srcConfig.Model
	dstConfig.Timeout = srcConfig.Timeout
	dstConfig.MaxTokens = srcConfig.MaxTokens
	dstConfig.Temperature = srcConfig.Temperature
	dstConfig.TopP = srcConfig.TopP
	dstConfig.PresencePenalty = srcConfig.PresencePenalty
	dstConfig.FrequencyPenalty = srcConfig.FrequencyPenalty
}
