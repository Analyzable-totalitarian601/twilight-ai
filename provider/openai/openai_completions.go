package openai

import (
	"github.com/memohai/twilight-ai/types"
	"net/http"
)

type OpenAICompletionsProvider struct {
	apiKey string
	baseURL string
	httpClient *http.Client
}

type OpenAICompletionsProviderOption func(*OpenAICompletionsProvider)

func WithAPIKey(apiKey string) OpenAICompletionsProviderOption {
	return func(p *OpenAICompletionsProvider) {
		p.apiKey = apiKey
	}
}

func WithBaseURL(baseURL string) OpenAICompletionsProviderOption {
	return func(p *OpenAICompletionsProvider) {
		p.baseURL = baseURL
	}
}

func NewCompletions(options ...OpenAICompletionsProviderOption) *OpenAICompletionsProvider {
	provider := &OpenAICompletionsProvider{
		httpClient: &http.Client{},
	}
	for _, option := range options {
		option(provider)
	}
	return provider
}

func (p *OpenAICompletionsProvider) Name() string {
	return "openai-completions"
}

func (p *OpenAICompletionsProvider) GetModels() ([]types.ChatModel, error) {
	return nil, nil
}