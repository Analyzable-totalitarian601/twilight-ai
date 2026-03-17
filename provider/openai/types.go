package openai

// --- Request types ---

type chatRequest struct {
	Model               string              `json:"model"`
	Messages            []chatMessage       `json:"messages"`
	Temperature         *float64            `json:"temperature,omitempty"`
	TopP                *float64            `json:"top_p,omitempty"`
	MaxCompletionTokens *int                `json:"max_completion_tokens,omitempty"`
	Stop                []string            `json:"stop,omitempty"`
	FrequencyPenalty    *float64            `json:"frequency_penalty,omitempty"`
	PresencePenalty     *float64            `json:"presence_penalty,omitempty"`
	Seed                *int                `json:"seed,omitempty"`
	ReasoningEffort     *string             `json:"reasoning_effort,omitempty"`
	Stream              bool                `json:"stream,omitempty"`
	StreamOptions       *chatStreamOptions  `json:"stream_options,omitempty"`
}

type chatStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type chatContentPartText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type chatContentPartImage struct {
	Type     string        `json:"type"`
	ImageURL chatImageURL  `json:"image_url"`
}

type chatImageURL struct {
	URL string `json:"url"`
}

// --- Response types ---

type chatResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []chatChoice   `json:"choices"`
	Usage   chatUsage      `json:"usage"`
}

type chatChoice struct {
	Index        int             `json:"index"`
	Message      chatRespMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

type chatRespMessage struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Refusal   string `json:"refusal,omitempty"`
}

type chatUsage struct {
	PromptTokens            int                      `json:"prompt_tokens"`
	CompletionTokens        int                      `json:"completion_tokens"`
	TotalTokens             int                      `json:"total_tokens"`
	PromptTokensDetails     *chatPromptTokenDetails  `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *chatCompletionTokenDetails `json:"completion_tokens_details,omitempty"`
}

type chatPromptTokenDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type chatCompletionTokenDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
	TextTokens      int `json:"text_tokens"`
}

// --- Streaming chunk types (chat.completion.chunk) ---

type chatChunkResponse struct {
	ID      string            `json:"id"`
	Object  string            `json:"object"`
	Created int64             `json:"created"`
	Model   string            `json:"model"`
	Choices []chatChunkChoice `json:"choices"`
	Usage   *chatUsage        `json:"usage,omitempty"`
}

type chatChunkChoice struct {
	Index        int            `json:"index"`
	Delta        chatChunkDelta `json:"delta"`
	FinishReason *string        `json:"finish_reason"`
}

type chatChunkDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
	Refusal string `json:"refusal,omitempty"`
}
