package messages

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/memohai/twilight-ai/internal/utils"
	"github.com/memohai/twilight-ai/sdk"
)

const (
	defaultBaseURL      = "https://api.anthropic.com"
	defaultAnthropicVer = "2023-06-01"

	// Content block types for Anthropic API
	blockTypeText      = "text"
	blockTypeThinking  = "thinking"
	blockTypeToolUse   = "tool_use"
)

// ThinkingConfig controls extended thinking for Anthropic models.
type ThinkingConfig struct {
	Type         string // "enabled", "adaptive", or "disabled"
	BudgetTokens int    // required when Type is "enabled"
}

type Provider struct {
	apiKey     string
	authToken  string
	baseURL    string
	httpClient *http.Client
	headers    map[string]string
	thinking   *ThinkingConfig
}

type Option func(*Provider)

func WithAPIKey(apiKey string) Option {
	return func(p *Provider) {
		p.apiKey = apiKey
	}
}

// WithAuthToken sets a Bearer token for authentication instead of x-api-key.
// Useful for proxies like OpenRouter that require Authorization: Bearer.
func WithAuthToken(token string) Option {
	return func(p *Provider) {
		p.authToken = token
	}
}

func WithBaseURL(baseURL string) Option {
	return func(p *Provider) {
		p.baseURL = baseURL
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(p *Provider) {
		p.httpClient = client
	}
}

// WithHeaders sets additional HTTP headers for requests.
func WithHeaders(headers map[string]string) Option {
	return func(p *Provider) {
		p.headers = headers
	}
}

// WithThinking enables extended thinking for all requests made by this provider.
func WithThinking(cfg ThinkingConfig) Option {
	return func(p *Provider) {
		p.thinking = &cfg
	}
}

func New(options ...Option) *Provider {
	provider := &Provider{
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{},
	}
	for _, option := range options {
		option(provider)
	}
	return provider
}

func (p *Provider) Name() string {
	return "anthropic-messages"
}

func (p *Provider) GetModels() ([]sdk.Model, error) {
	return nil, nil
}

func (p *Provider) ChatModel(id string) *sdk.Model {
	return &sdk.Model{
		ID:       id,
		Provider: p,
		Type:     sdk.ModelTypeChat,
	}
}

func (p *Provider) requestHeaders() map[string]string {
	h := map[string]string{
		"anthropic-version": defaultAnthropicVer,
		"Content-Type":      "application/json",
	}
	if p.authToken != "" {
		h["Authorization"] = "Bearer " + p.authToken
	} else if p.apiKey != "" {
		h["x-api-key"] = p.apiKey
	}
	for k, v := range p.headers {
		h[k] = v
	}
	return h
}

// ---------- DoGenerate ----------

func (p *Provider) DoGenerate(ctx context.Context, params sdk.GenerateParams) (*sdk.GenerateResult, error) { //nolint:gocritic // interface method
	if params.Model == nil {
		return nil, fmt.Errorf("anthropic: model is required")
	}

	req := p.buildRequest(&params)

	resp, err := utils.FetchJSON[messagesResponse](ctx, p.httpClient, &utils.RequestOptions{
		Method:  http.MethodPost,
		BaseURL: p.baseURL,
		Path:    "/v1/messages",
		Headers: p.requestHeaders(),
		Body:    req,
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic: messages request failed: %w", err)
	}

	return p.parseResponse(resp)
}

// ---------- buildRequest ----------

func (p *Provider) buildRequest(params *sdk.GenerateParams) *messagesRequest {
	system, messages := convertMessages(params)

	req := &messagesRequest{
		Model:       params.Model.ID,
		System:      system,
		Messages:    messages,
		MaxTokens:   params.MaxTokens,
		Temperature: params.Temperature,
		TopP:        params.TopP,
	}

	if len(params.StopSequences) > 0 {
		req.StopSequences = params.StopSequences
	}
	if len(params.Tools) > 0 {
		req.Tools = convertTools(params.Tools)
		req.ToolChoice = convertToolChoice(params.ToolChoice)
	}

	if p.thinking != nil && p.thinking.Type != "" && p.thinking.Type != "disabled" {
		req.Thinking = &anthropicThinking{
			Type:         p.thinking.Type,
			BudgetTokens: p.thinking.BudgetTokens,
		}
	}

	return req
}

func convertTools(tools []sdk.Tool) []anthropicTool {
	out := make([]anthropicTool, 0, len(tools))
	for _, t := range tools {
		out = append(out, anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}
	return out
}

func convertToolChoice(choice any) *anthropicToolChoice {
	if choice == nil {
		return nil
	}
	switch v := choice.(type) {
	case string:
		switch v {
		case "auto":
			return &anthropicToolChoice{Type: "auto"}
		case "required":
			return &anthropicToolChoice{Type: "any"}
		case "none":
			return nil
		default:
			return &anthropicToolChoice{Type: "auto"}
		}
	case map[string]any:
		tc := &anthropicToolChoice{Type: "tool"}
		if fn, ok := v["function"].(map[string]any); ok {
			if name, ok := fn["name"].(string); ok {
				tc.Name = name
			}
		}
		return tc
	default:
		return nil
	}
}

// ---------- message conversion ----------

// convertMessages splits SDK messages into Anthropic's system blocks and
// alternating user/assistant messages. Tool result messages are merged into
// user messages, as required by the Anthropic API.
func convertMessages(params *sdk.GenerateParams) ([]contentBlock, []anthropicMessage) {
	var system []contentBlock
	var out []anthropicMessage

	if params.System != "" {
		system = append(system, contentBlock{Type: blockTypeText, Text: params.System})
	}

	for _, msg := range params.Messages {
		switch msg.Role {
		case sdk.MessageRoleSystem:
			for _, part := range msg.Content {
				if tp, ok := part.(sdk.TextPart); ok {
					system = append(system, contentBlock{Type: blockTypeText, Text: tp.Text})
				}
			}

		case sdk.MessageRoleUser:
			blocks := convertUserContent(msg.Content)
			out = appendUserBlocks(out, blocks)

		case sdk.MessageRoleAssistant:
			out = append(out, convertAssistantMessage(msg))

		case sdk.MessageRoleTool:
			blocks := convertToolResults(msg.Content)
			out = appendUserBlocks(out, blocks)
		}
	}

	return system, out
}

// appendUserBlocks appends content blocks to the last user message if it exists,
// or creates a new user message.
func appendUserBlocks(messages []anthropicMessage, blocks []contentBlock) []anthropicMessage {
	if len(messages) > 0 && messages[len(messages)-1].Role == "user" {
		messages[len(messages)-1].Content = append(messages[len(messages)-1].Content, blocks...)
		return messages
	}
	return append(messages, anthropicMessage{
		Role:    "user",
		Content: blocks,
	})
}

func convertUserContent(parts []sdk.MessagePart) []contentBlock {
	var blocks []contentBlock
	for _, part := range parts {
		switch p := part.(type) {
		case sdk.TextPart:
			blocks = append(blocks, contentBlock{Type: blockTypeText, Text: p.Text})
		case sdk.ImagePart:
			blocks = append(blocks, contentBlock{
				Type: "image",
				Source: &imageSource{
					Type:      "base64",
					MediaType: p.MediaType,
					Data:      p.Image,
				},
			})
		case sdk.FilePart:
			blocks = append(blocks, contentBlock{Type: blockTypeText, Text: p.Data})
		}
	}
	return blocks
}

func convertAssistantMessage(msg sdk.Message) anthropicMessage {
	var blocks []contentBlock

	for _, part := range msg.Content {
		switch p := part.(type) {
		case sdk.TextPart:
			blocks = append(blocks, contentBlock{Type: blockTypeText, Text: p.Text})
		case sdk.ReasoningPart:
			blocks = append(blocks, contentBlock{
				Type:      blockTypeThinking,
				Thinking:  p.Text,
				Signature: p.Signature,
			})
		case sdk.ToolCallPart:
			id := p.ToolCallID
			if id == "" {
				id = generateID()
			}
			blocks = append(blocks, contentBlock{
				Type:  blockTypeToolUse,
				ID:    id,
				Name:  p.ToolName,
				Input: p.Input,
			})
		}
	}

	return anthropicMessage{Role: "assistant", Content: blocks}
}

func convertToolResults(parts []sdk.MessagePart) []contentBlock {
	var blocks []contentBlock
	for _, part := range parts {
		if trp, ok := part.(sdk.ToolResultPart); ok {
			content, _ := json.Marshal(trp.Result)
			block := contentBlock{
				Type:      "tool_result",
				ToolUseID: trp.ToolCallID,
				Content:   string(content),
				IsError:   trp.IsError,
			}
			blocks = append(blocks, block)
		}
	}
	return blocks
}

// ---------- parseResponse ----------

func (p *Provider) parseResponse(resp *messagesResponse) (*sdk.GenerateResult, error) {
	result := &sdk.GenerateResult{
		Usage:           convertUsage(&resp.Usage),
		FinishReason:    mapFinishReason(resp.StopReason),
		RawFinishReason: resp.StopReason,
		Response: sdk.ResponseMetadata{
			ID:      resp.ID,
			ModelID: resp.Model,
		},
	}

	for i := range resp.Content {
		block := &resp.Content[i]
		switch block.Type {
		case blockTypeText:
			result.Text += block.Text
		case blockTypeThinking:
			result.Reasoning += block.Thinking
		case "redacted_thinking":
			// Redacted thinking blocks don't contain readable text
		case blockTypeToolUse:
			result.ToolCalls = append(result.ToolCalls, sdk.ToolCall{
				ToolCallID: block.ID,
				ToolName:   block.Name,
				Input:      block.Input,
			})
		}
	}

	return result, nil
}

// ---------- DoStream ----------

func (p *Provider) DoStream(ctx context.Context, params sdk.GenerateParams) (*sdk.StreamResult, error) { //nolint:gocritic // interface method
	if params.Model == nil {
		return nil, fmt.Errorf("anthropic: model is required")
	}

	req := p.buildRequest(&params)
	req.Stream = true

	ch := make(chan sdk.StreamPart, 64)

	go func() {
		defer close(ch)

		var (
			rawFinishReason string
			finishReason    sdk.FinishReason
			usage           sdk.Usage
			messageID       string
			messageModel    string

			// Track active content blocks by index for proper end handling.
			activeBlocks = map[int]*streamingBlock{}
		)

		send := func(part sdk.StreamPart) bool {
			select {
			case ch <- part:
				return true
			case <-ctx.Done():
				return false
			}
		}

		if !send(&sdk.StartPart{}) {
			return
		}
		if !send(&sdk.StartStepPart{}) {
			return
		}

		err := utils.FetchSSE(ctx, p.httpClient, &utils.RequestOptions{
			Method:  http.MethodPost,
			BaseURL: p.baseURL,
			Path:    "/v1/messages",
			Headers: p.requestHeaders(),
			Body:    req,
		}, func(ev *utils.SSEEvent) error {
			var event streamEvent
			if err := json.Unmarshal([]byte(ev.Data), &event); err != nil {
				send(&sdk.ErrorPart{Error: fmt.Errorf("anthropic: unmarshal event: %w", err)})
				return err
			}

			switch event.Type {
			case "message_start":
				if event.Message != nil {
					messageID = event.Message.ID
					messageModel = event.Message.Model
					usage = convertUsage(&event.Message.Usage)
				}

			case "content_block_start":
				if event.Index == nil || event.ContentBlock == nil {
					return nil
				}
				idx := *event.Index
				cb := event.ContentBlock
				switch cb.Type {
				case blockTypeText:
					activeBlocks[idx] = &streamingBlock{blockType: blockTypeText}
					send(&sdk.TextStartPart{ID: messageID})
				case blockTypeThinking:
					activeBlocks[idx] = &streamingBlock{blockType: blockTypeThinking}
					send(&sdk.ReasoningStartPart{ID: messageID})
				case blockTypeToolUse:
					activeBlocks[idx] = &streamingBlock{
						blockType:  blockTypeToolUse,
						toolID:     cb.ID,
						toolName:   cb.Name,
					}
					send(&sdk.ToolInputStartPart{
						ID:       cb.ID,
						ToolName: cb.Name,
					})
				}

			case "content_block_delta":
				if event.Index == nil || event.Delta == nil {
					return nil
				}
				idx := *event.Index
				delta := event.Delta
				sb := activeBlocks[idx]

				switch delta.Type {
				case "text_delta":
					send(&sdk.TextDeltaPart{ID: messageID, Text: delta.Text})
				case "thinking_delta":
					send(&sdk.ReasoningDeltaPart{ID: messageID, Text: delta.Thinking})
				case "input_json_delta":
					if sb != nil {
						sb.args += delta.PartialJSON
						send(&sdk.ToolInputDeltaPart{
							ID:    sb.toolID,
							Delta: delta.PartialJSON,
						})
					}
				case "signature_delta":
					// signature is part of thinking blocks; no SDK stream part needed
				}

			case "content_block_stop":
				if event.Index == nil {
					return nil
				}
				idx := *event.Index
				sb, ok := activeBlocks[idx]
				if !ok {
					return nil
				}
				delete(activeBlocks, idx)

				switch sb.blockType {
				case blockTypeText:
					send(&sdk.TextEndPart{ID: messageID})
				case blockTypeThinking:
					send(&sdk.ReasoningEndPart{ID: messageID})
				case blockTypeToolUse:
					send(&sdk.ToolInputEndPart{ID: sb.toolID})
					var input any
					if sb.args != "" {
						if err := json.Unmarshal([]byte(sb.args), &input); err != nil {
							send(&sdk.ErrorPart{Error: fmt.Errorf("anthropic: unmarshal tool args for %q: %w", sb.toolName, err)})
						}
					}
					send(&sdk.StreamToolCallPart{
						ToolCallID: sb.toolID,
						ToolName:   sb.toolName,
						Input:      input,
					})
				}

			case "message_delta":
				if event.Delta != nil {
					rawFinishReason = event.Delta.StopReason
					finishReason = mapFinishReason(rawFinishReason)
				}
				if event.Usage != nil {
					usage.OutputTokens = event.Usage.OutputTokens
					usage.TotalTokens = usage.InputTokens + usage.OutputTokens
				}
				send(&sdk.FinishStepPart{
					FinishReason:    finishReason,
					RawFinishReason: rawFinishReason,
					Usage:           usage,
					Response: sdk.ResponseMetadata{
						ID:      messageID,
						ModelID: messageModel,
					},
				})

			case "message_stop":
				return utils.ErrStreamDone

			case "ping":
				// ignore

			case "error":
				errMsg := "unknown error"
				if event.Delta != nil && event.Delta.Text != "" {
					errMsg = event.Delta.Text
				}
				send(&sdk.ErrorPart{Error: fmt.Errorf("anthropic: stream error: %s", errMsg)})
			}

			return nil
		})

		if err != nil {
			send(&sdk.ErrorPart{Error: fmt.Errorf("anthropic: stream failed: %w", err)})
		}

		send(&sdk.FinishPart{
			FinishReason:    finishReason,
			RawFinishReason: rawFinishReason,
			TotalUsage:      usage,
		})
	}()

	return &sdk.StreamResult{Stream: ch}, nil
}

type streamingBlock struct {
	blockType string
	toolID    string
	toolName  string
	args      string
}

// ---------- helpers ----------

func generateID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return fmt.Sprintf("toolu_%x", b)
}

func convertUsage(u *messagesUsage) sdk.Usage {
	total := u.InputTokens + u.OutputTokens
	return sdk.Usage{
		InputTokens:       u.InputTokens,
		OutputTokens:      u.OutputTokens,
		TotalTokens:       total,
		CachedInputTokens: u.CacheReadInputTokens,
		InputTokenDetails: sdk.InputTokenDetail{
			CacheReadTokens:  u.CacheReadInputTokens,
			CacheWriteTokens: u.CacheCreationInputTokens,
		},
	}
}

func mapFinishReason(reason string) sdk.FinishReason {
	switch reason {
	case "end_turn", "stop_sequence":
		return sdk.FinishReasonStop
	case "tool_use":
		return sdk.FinishReasonToolCalls
	case "max_tokens":
		return sdk.FinishReasonLength
	default:
		return sdk.FinishReasonUnknown
	}
}
