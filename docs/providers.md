# Providers

A **Provider** is the abstraction that connects the SDK to an AI backend. It handles HTTP communication, request/response mapping, and streaming protocol details.

## The Provider Interface

```go
type Provider interface {
    Name() string
    GetModels() ([]Model, error)
    DoGenerate(ctx context.Context, params GenerateParams) (*GenerateResult, error)
    DoStream(ctx context.Context, params GenerateParams) (*StreamResult, error)
}
```

| Method | Purpose |
|--------|---------|
| `Name()` | Returns a human-readable provider identifier (e.g. `"openai-completions"`) |
| `GetModels()` | Lists available models (optional, may return nil) |
| `DoGenerate()` | Performs a single non-streaming LLM call |
| `DoStream()` | Performs a streaming LLM call, returning a channel of `StreamPart` |

The SDK never calls a provider directly — it goes through the `Client` which adds orchestration (tool loop, callbacks, multi-step). The `Model` struct carries a reference to its provider:

```go
type Model struct {
    ID          string
    DisplayName string
    Provider    Provider
    Type        ModelType   // "chat"
    MaxTokens   int
}
```

## OpenAI Provider

The built-in `provider/openai` package implements the OpenAI Chat Completions API.

### Basic Usage

```go
import "github.com/memohai/twilight-ai/provider/openai"

provider := openai.NewCompletions(
    openai.WithAPIKey("sk-..."),
)
model := provider.ChatModel("gpt-4o-mini")
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithAPIKey(key)` | `""` | API key sent as `Authorization: Bearer <key>` |
| `WithBaseURL(url)` | `https://api.openai.com/v1` | Base URL for API requests |
| `WithHTTPClient(client)` | `&http.Client{}` | Custom HTTP client (for proxies, timeouts, etc.) |

### OpenAI-Compatible Providers

Any service that implements the OpenAI Chat Completions API works out of the box:

```go
// DeepSeek
provider := openai.NewCompletions(
    openai.WithAPIKey("your-deepseek-key"),
    openai.WithBaseURL("https://api.deepseek.com"),
)

// Groq
provider := openai.NewCompletions(
    openai.WithAPIKey("your-groq-key"),
    openai.WithBaseURL("https://api.groq.com/openai/v1"),
)

// Azure OpenAI
provider := openai.NewCompletions(
    openai.WithAPIKey("your-azure-key"),
    openai.WithBaseURL("https://your-resource.openai.azure.com/openai/deployments/gpt-4o"),
)

// Local (Ollama, vLLM, etc.)
provider := openai.NewCompletions(
    openai.WithBaseURL("http://localhost:11434/v1"),
)
```

### Supported Features

| Feature | Supported |
|---------|-----------|
| Chat completions | ✅ |
| Streaming (SSE) | ✅ |
| Tool/function calling | ✅ |
| Vision (image inputs) | ✅ |
| Reasoning content (o1, DeepSeek-R1) | ✅ |
| JSON mode / JSON Schema | ✅ |
| Token usage reporting | ✅ |
| Cached token details | ✅ |

### Custom HTTP Client

Use `WithHTTPClient` for custom timeouts, proxies, or TLS settings:

```go
provider := openai.NewCompletions(
    openai.WithAPIKey("sk-..."),
    openai.WithHTTPClient(&http.Client{
        Timeout: 120 * time.Second,
        Transport: &http.Transport{
            Proxy: http.ProxyFromEnvironment,
        },
    }),
)
```

## Implementing a Custom Provider

To add support for a new AI backend, implement the `sdk.Provider` interface:

```go
package myprovider

import (
    "context"
    "github.com/memohai/twilight-ai/sdk"
)

type MyProvider struct {
    apiKey string
}

func New(apiKey string) *MyProvider {
    return &MyProvider{apiKey: apiKey}
}

func (p *MyProvider) Name() string {
    return "my-provider"
}

func (p *MyProvider) GetModels() ([]sdk.Model, error) {
    return []sdk.Model{
        {ID: "my-model-v1", Provider: p, Type: sdk.ModelTypeChat},
    }, nil
}

func (p *MyProvider) ChatModel(id string) *sdk.Model {
    return &sdk.Model{ID: id, Provider: p, Type: sdk.ModelTypeChat}
}

func (p *MyProvider) DoGenerate(ctx context.Context, params sdk.GenerateParams) (*sdk.GenerateResult, error) {
    // Make HTTP request to your backend...
    // Map the response to *sdk.GenerateResult
    return &sdk.GenerateResult{
        Text:         "response text",
        FinishReason: sdk.FinishReasonStop,
    }, nil
}

func (p *MyProvider) DoStream(ctx context.Context, params sdk.GenerateParams) (*sdk.StreamResult, error) {
    ch := make(chan sdk.StreamPart, 64)

    go func() {
        defer close(ch)
        // Stream chunks from your backend...
        ch <- &sdk.StartPart{}
        ch <- &sdk.StartStepPart{}
        ch <- &sdk.TextStartPart{}
        ch <- &sdk.TextDeltaPart{Text: "Hello"}
        ch <- &sdk.TextEndPart{}
        ch <- &sdk.FinishStepPart{FinishReason: sdk.FinishReasonStop}
        ch <- &sdk.FinishPart{FinishReason: sdk.FinishReasonStop}
    }()

    return &sdk.StreamResult{Stream: ch}, nil
}
```

Then use it exactly like the built-in provider:

```go
provider := myprovider.New("my-key")
model := provider.ChatModel("my-model-v1")

text, err := sdk.GenerateText(ctx,
    sdk.WithModel(model),
    sdk.WithMessages([]sdk.Message{sdk.UserMessage("Hello")}),
)
```

## Next Steps

- [Tool Calling](tools.md) — define tools and enable multi-step execution
- [Streaming](streaming.md) — understand StreamPart types
- [API Reference](api-reference.md) — complete type and function reference
