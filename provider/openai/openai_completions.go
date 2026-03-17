package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/memohai/twilight-ai/internal/utils"
	"github.com/memohai/twilight-ai/types"
)

const defaultBaseURL = "https://api.openai.com/v1"

type OpenAICompletionsProvider struct {
	apiKey     string
	baseURL    string
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
		baseURL:    defaultBaseURL,
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

func (p *OpenAICompletionsProvider) GetModels() ([]types.Model, error) {
	return nil, nil
}

func (p *OpenAICompletionsProvider) DoGenerate(ctx context.Context, params types.GenerateParams) (*types.GenerateResult, error) {
	if params.Model == nil {
		return nil, fmt.Errorf("openai: model is required")
	}

	req := p.buildRequest(params)

	resp, err := utils.FetchJSON[chatResponse](ctx, p.httpClient, utils.RequestOptions{
		Method:  http.MethodPost,
		BaseURL: p.baseURL,
		Path:    "/chat/completions",
		Headers: utils.AuthHeader(p.apiKey),
		Body:    req,
	})
	if err != nil {
		return nil, fmt.Errorf("openai: chat completions request failed: %w", err)
	}

	return p.parseResponse(resp), nil
}

func (p *OpenAICompletionsProvider) buildRequest(params types.GenerateParams) *chatRequest {
	req := &chatRequest{
		Model:               params.Model.ID,
		Messages:            convertMessages(params.Messages),
		Temperature:         params.Temperature,
		TopP:                params.TopP,
		MaxCompletionTokens: params.MaxTokens,
		FrequencyPenalty:    params.FrequencyPenalty,
		PresencePenalty:     params.PresencePenalty,
		Seed:                params.Seed,
		ReasoningEffort:     params.ReasoningEffort,
	}
	if len(params.StopSequences) > 0 {
		req.Stop = params.StopSequences
	}
	return req
}

func convertMessages(messages []types.Message) []chatMessage {
	out := make([]chatMessage, 0, len(messages))
	for _, msg := range messages {
		out = append(out, chatMessage{
			Role:    string(msg.Role),
			Content: convertContent(msg.Parts),
		})
	}
	return out
}

func convertContent(parts []types.MessagePart) any {
	if len(parts) == 1 {
		if tp, ok := parts[0].(types.TextPart); ok {
			return tp.Text
		}
	}

	out := make([]any, 0, len(parts))
	for _, part := range parts {
		switch p := part.(type) {
		case types.TextPart:
			out = append(out, chatContentPartText{Type: "text", Text: p.Text})
		case types.ImagePart:
			out = append(out, chatContentPartImage{
				Type:     "image_url",
				ImageURL: chatImageURL{URL: p.Image},
			})
		}
	}
	return out
}

func (p *OpenAICompletionsProvider) parseResponse(resp *chatResponse) *types.GenerateResult {
	result := &types.GenerateResult{
		Usage: convertUsage(&resp.Usage),
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		result.Text = choice.Message.Content
		result.FinishReason = mapFinishReason(choice.FinishReason)
	}

	return result
}

func (p *OpenAICompletionsProvider) DoStream(ctx context.Context, params types.GenerateParams) (*types.StreamResult, error) {
	if params.Model == nil {
		return nil, fmt.Errorf("openai: model is required")
	}

	req := p.buildRequest(params)
	req.Stream = true
	req.StreamOptions = &chatStreamOptions{IncludeUsage: true}

	ch := make(chan types.StreamPart, 64)

	go func() {
		defer close(ch)

		var (
			textStartSent bool
			finishReason  types.FinishReason
			usage         types.Usage
		)

		send := func(part types.StreamPart) bool {
			select {
			case ch <- part:
				return true
			case <-ctx.Done():
				return false
			}
		}

		if !send(&types.StartPart{}) {
			return
		}

		err := utils.FetchSSE(ctx, p.httpClient, utils.RequestOptions{
			Method:  http.MethodPost,
			BaseURL: p.baseURL,
			Path:    "/chat/completions",
			Headers: utils.AuthHeader(p.apiKey),
			Body:    req,
		}, func(ev *utils.SSEEvent) error {
			if ev.Data == "[DONE]" {
				return utils.ErrStreamDone
			}

			var chunk chatChunkResponse
			if err := json.Unmarshal([]byte(ev.Data), &chunk); err != nil {
				send(&types.ErrorPart{Error: fmt.Errorf("openai: unmarshal chunk: %w", err)})
				return err
			}

			if chunk.Usage != nil {
				usage = convertUsage(chunk.Usage)
			}

			if len(chunk.Choices) == 0 {
				return nil
			}
			choice := chunk.Choices[0]

			if choice.Delta.Content != "" {
				if !textStartSent {
					send(&types.TextStartPart{ID: chunk.ID})
					textStartSent = true
				}
				send(&types.TextDeltaPart{ID: chunk.ID, Text: choice.Delta.Content})
			}

			if choice.FinishReason != nil && *choice.FinishReason != "" {
				finishReason = mapFinishReason(*choice.FinishReason)

				if textStartSent {
					send(&types.TextEndPart{ID: chunk.ID})
				}

				send(&types.FinishStepPart{
					FinishReason: finishReason,
					Usage:        usage,
				})
			}

			return nil
		})

		if err != nil {
			send(&types.ErrorPart{Error: fmt.Errorf("openai: stream failed: %w", err)})
		}

		send(&types.FinishPart{
			FinishReason: finishReason,
			TotalUsage:   usage,
		})
	}()

	return &types.StreamResult{Stream: ch}, nil
}

func convertUsage(u *chatUsage) types.Usage {
	usage := types.Usage{
		InputTokens:  u.PromptTokens,
		OutputTokens: u.CompletionTokens,
		TotalTokens:  u.TotalTokens,
	}
	if u.PromptTokensDetails != nil {
		usage.CachedInputTokens = u.PromptTokensDetails.CachedTokens
		usage.InputTokenDetails.CacheReadTokens = u.PromptTokensDetails.CachedTokens
	}
	if u.CompletionTokensDetails != nil {
		usage.ReasoningTokens = u.CompletionTokensDetails.ReasoningTokens
		usage.OutputTokenDetails.ReasoningTokens = u.CompletionTokensDetails.ReasoningTokens
		usage.OutputTokenDetails.TextTokens = u.CompletionTokensDetails.TextTokens
	}
	return usage
}

func mapFinishReason(reason string) types.FinishReason {
	switch reason {
	case "stop":
		return types.FinishReasonStop
	case "length":
		return types.FinishReasonLength
	case "content_filter":
		return types.FinishReasonContentFilter
	case "tool_calls":
		return types.FinishReasonToolCalls
	default:
		return types.FinishReasonUnknown
	}
}
