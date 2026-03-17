# Getting Started

This guide walks you through installing Twilight AI and making your first LLM request.

## Prerequisites

- Go 1.25 or later
- An API key from OpenAI (or any OpenAI-compatible provider)

## Installation

```bash
go get github.com/memohai/twilight-ai
```

## Setup

### 1. Create a Provider

A **Provider** is the bridge between the SDK and a specific AI backend. The SDK ships with an OpenAI provider that works with OpenAI, Azure OpenAI, DeepSeek, Groq, and any other OpenAI-compatible API.

```go
import "github.com/memohai/twilight-ai/provider/openai"

provider := openai.NewCompletions(
    openai.WithAPIKey("sk-..."),
)
```

For OpenAI-compatible endpoints, add `WithBaseURL`:

```go
provider := openai.NewCompletions(
    openai.WithAPIKey("your-key"),
    openai.WithBaseURL("https://api.deepseek.com/v1"),
)
```

### 2. Get a Model

```go
model := provider.ChatModel("gpt-4o-mini")
```

The model carries a reference to its provider, so the SDK knows which backend to call.

### 3. Generate Text

The simplest way to get a response:

```go
import "github.com/memohai/twilight-ai/sdk"

text, err := sdk.GenerateText(ctx,
    sdk.WithModel(model),
    sdk.WithMessages([]sdk.Message{
        sdk.UserMessage("What is the capital of France?"),
    }),
)
// text == "The capital of France is Paris."
```

### 4. Get the Full Result

If you need token usage, finish reason, or other metadata:

```go
result, err := sdk.GenerateTextResult(ctx,
    sdk.WithModel(model),
    sdk.WithSystem("You are a helpful assistant."),
    sdk.WithMessages([]sdk.Message{
        sdk.UserMessage("What is the capital of France?"),
    }),
)

fmt.Println(result.Text)                      // response text
fmt.Println(result.Usage.TotalTokens)         // token count
fmt.Println(result.FinishReason)              // "stop"
fmt.Println(result.Response.ModelID)          // "gpt-4o-mini"
```

### 5. Stream the Response

For real-time output, use `StreamText`:

```go
sr, err := sdk.StreamText(ctx,
    sdk.WithModel(model),
    sdk.WithMessages([]sdk.Message{
        sdk.UserMessage("Tell me a short story."),
    }),
)
if err != nil {
    log.Fatal(err)
}

for part := range sr.Stream {
    switch p := part.(type) {
    case *sdk.TextDeltaPart:
        fmt.Print(p.Text)
    case *sdk.ErrorPart:
        log.Fatal(p.Error)
    }
}
fmt.Println() // newline after streaming
```

Or use the convenience helper to collect all text at once:

```go
sr, err := sdk.StreamText(ctx, sdk.WithModel(model), ...)
text, err := sr.Text()
```

## Using a Client Instance

The package-level functions (`sdk.GenerateText`, `sdk.StreamText`) use a default client. You can also create your own:

```go
client := sdk.NewClient()
text, err := client.GenerateText(ctx, sdk.WithModel(model), ...)
```

This is useful when you need multiple clients with different configurations.

## Environment Variables

The SDK itself does not read environment variables, but the test suite supports a `.env` file with:

```
OPENAI_API_KEY=sk-...
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=gpt-4o-mini
```

## Next Steps

- [Providers](providers.md) — learn about the Provider interface and OpenAI options
- [Tool Calling](tools.md) — define tools and enable multi-step execution
- [Streaming](streaming.md) — understand StreamPart types and advanced patterns
- [API Reference](api-reference.md) — complete type and function reference
