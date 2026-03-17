# API Reference

Complete reference for all exported types and functions in the Twilight AI SDK.

## Package `sdk`

### Client

```go
type Client struct{}

func NewClient() *Client
```

A `Client` provides text generation methods. The provider is resolved from the `Model` passed via `WithModel`.

#### Methods

```go
func (c *Client) GenerateText(ctx context.Context, options ...GenerateOption) (string, error)
```

Generates text and returns only the response string.

```go
func (c *Client) GenerateTextResult(ctx context.Context, options ...GenerateOption) (*GenerateResult, error)
```

Generates text and returns the full result including usage, steps, and metadata.

```go
func (c *Client) StreamText(ctx context.Context, options ...GenerateOption) (*StreamResult, error)
```

Returns a streaming result with a channel of `StreamPart` chunks.

#### Package-Level Functions

These use a default client instance:

```go
func GenerateText(ctx context.Context, options ...GenerateOption) (string, error)
func GenerateTextResult(ctx context.Context, options ...GenerateOption) (*GenerateResult, error)
func StreamText(ctx context.Context, options ...GenerateOption) (*StreamResult, error)
```

---

### Provider

```go
type Provider interface {
    Name() string
    GetModels() ([]Model, error)
    DoGenerate(ctx context.Context, params GenerateParams) (*GenerateResult, error)
    DoStream(ctx context.Context, params GenerateParams) (*StreamResult, error)
}
```

### Model

```go
type Model struct {
    ID          string
    DisplayName string
    Provider    Provider
    Type        ModelType
    MaxTokens   int
}

type ModelType string
const ModelTypeChat ModelType = "chat"
```

---

### Messages

```go
type Message struct {
    Role    MessageRole
    Content []MessagePart
}
```

#### MessageRole

| Constant | Value |
|----------|-------|
| `MessageRoleUser` | `"user"` |
| `MessageRoleAssistant` | `"assistant"` |
| `MessageRoleSystem` | `"system"` |
| `MessageRoleTool` | `"tool"` |

#### Message Constructors

```go
func UserMessage(text string, extra ...MessagePart) Message
func SystemMessage(text string) Message
func AssistantMessage(text string) Message
func ToolMessage(results ...ToolResultPart) Message
```

`UserMessage` accepts optional extra parts (e.g. `ImagePart`) after the text.

#### MessagePart Interface

```go
type MessagePart interface {
    PartType() MessagePartType
}
```

#### Part Types

```go
type TextPart struct {
    Text string
}

type ReasoningPart struct {
    Text      string
    Signature string  // optional
}

type ImagePart struct {
    Image     string  // URL or base64
    MediaType string  // optional, e.g. "image/png"
}

type FilePart struct {
    Data      string
    MediaType string  // optional
    Filename  string  // optional
}

type ToolCallPart struct {
    ToolCallID string
    ToolName   string
    Input      any
}

type ToolResultPart struct {
    ToolCallID string
    ToolName   string
    Result     any
    IsError    bool   // optional
}
```

`Message` supports full JSON serialization with automatic type discrimination.

---

### Generation

#### GenerateParams

```go
type GenerateParams struct {
    Model            *Model
    System           string
    Messages         []Message
    Tools            []Tool
    ToolChoice       any              // "auto", "none", "required"
    ResponseFormat   *ResponseFormat
    Temperature      *float64
    TopP             *float64
    MaxTokens        *int
    StopSequences    []string
    FrequencyPenalty *float64
    PresencePenalty  *float64
    Seed             *int
    ReasoningEffort  *string
}
```

#### GenerateResult

```go
type GenerateResult struct {
    Text            string
    Reasoning       string
    FinishReason    FinishReason
    RawFinishReason string
    Usage           Usage
    Sources         []Source
    Files           []GeneratedFile
    ToolCalls       []ToolCall
    ToolResults     []ToolResult
    Response        ResponseMetadata
    Steps           []StepResult
    Messages        []Message
}
```

#### StepResult

```go
type StepResult struct {
    Text            string
    Reasoning       string
    FinishReason    FinishReason
    RawFinishReason string
    Usage           Usage
    ToolCalls       []ToolCall
    ToolResults     []ToolResult
    Response        ResponseMetadata
    Messages        []Message
}
```

#### FinishReason

| Constant | Value | Description |
|----------|-------|-------------|
| `FinishReasonStop` | `"stop"` | Normal completion |
| `FinishReasonLength` | `"length"` | Max tokens reached |
| `FinishReasonContentFilter` | `"content-filter"` | Content filter triggered |
| `FinishReasonToolCalls` | `"tool-calls"` | Model wants to call tools |
| `FinishReasonError` | `"error"` | An error occurred |
| `FinishReasonOther` | `"other"` | Provider-specific reason |
| `FinishReasonUnknown` | `"unknown"` | Unknown reason |

#### ResponseFormat

```go
type ResponseFormat struct {
    Type       ResponseFormatType
    JSONSchema any                 // required when Type is json_schema
}

type ResponseFormatType string
const (
    ResponseFormatText       ResponseFormatType = "text"
    ResponseFormatJSONObject ResponseFormatType = "json_object"
    ResponseFormatJSONSchema ResponseFormatType = "json_schema"
)
```

---

### Options

All options are of type `GenerateOption` (`func(*generateConfig)`).

#### Provider-Level Options

| Function | Description |
|----------|-------------|
| `WithModel(model *Model)` | **Required.** The model to use |
| `WithMessages(msgs []Message)` | Chat messages |
| `WithSystem(text string)` | System prompt |
| `WithTools(tools []Tool)` | Tool definitions |
| `WithToolChoice(choice any)` | `"auto"`, `"none"`, `"required"` |
| `WithResponseFormat(rf ResponseFormat)` | Response format constraint |
| `WithTemperature(t float64)` | Sampling temperature |
| `WithTopP(topP float64)` | Nucleus sampling |
| `WithMaxTokens(n int)` | Maximum output tokens |
| `WithStopSequences(s []string)` | Stop sequences |
| `WithFrequencyPenalty(p float64)` | Frequency penalty |
| `WithPresencePenalty(p float64)` | Presence penalty |
| `WithSeed(s int)` | Random seed for reproducibility |
| `WithReasoningEffort(effort string)` | Reasoning effort level |

#### Orchestration Options

| Function | Description |
|----------|-------------|
| `WithMaxSteps(n int)` | `0` = single call (default), `N` = up to N calls, `-1` = unlimited |
| `WithOnFinish(fn func(*GenerateResult))` | Called when all steps complete |
| `WithOnStep(fn func(*StepResult) *GenerateParams)` | Called after each step; return non-nil to override next step |
| `WithPrepareStep(fn func(*GenerateParams) *GenerateParams)` | Called before each step (from step 2); can modify params |
| `WithApprovalHandler(fn func(ctx, ToolCall) (bool, error))` | Approval for tools with `RequireApproval` |

---

### Tools

```go
type Tool struct {
    Name            string
    Description     string
    Parameters      any              // JSON Schema
    Execute         ToolExecuteFunc
    RequireApproval bool
}

type ToolExecuteFunc func(ctx *ToolExecContext, input any) (any, error)

type ToolExecContext struct {
    context.Context
    ToolCallID   string
    ToolName     string
    SendProgress func(content any) // nil outside streaming mode
}
```

#### ToolCall & ToolResult

```go
type ToolCall struct {
    ToolCallID string
    ToolName   string
    Input      any
}

type ToolResult struct {
    ToolCallID string
    ToolName   string
    Input      any
    Output     any
    IsError    bool
}
```

---

### Streaming

#### StreamResult

```go
type StreamResult struct {
    Stream   <-chan StreamPart
    Steps    []StepResult  // populated after stream consumed
    Messages []Message     // populated after stream consumed
}

func (sr *StreamResult) Text() (string, error)
func (sr *StreamResult) ToResult() (*GenerateResult, error)
```

#### StreamPart Interface

```go
type StreamPart interface {
    Type() StreamPartType
}
```

#### All StreamPart Types

**Text:**

| Type | Key Fields |
|------|-----------|
| `*TextStartPart` | `ID` |
| `*TextDeltaPart` | `ID`, `Text` |
| `*TextEndPart` | `ID` |

**Reasoning:**

| Type | Key Fields |
|------|-----------|
| `*ReasoningStartPart` | `ID` |
| `*ReasoningDeltaPart` | `ID`, `Text` |
| `*ReasoningEndPart` | `ID` |

**Tool Input:**

| Type | Key Fields |
|------|-----------|
| `*ToolInputStartPart` | `ID`, `ToolName` |
| `*ToolInputDeltaPart` | `ID`, `Delta` |
| `*ToolInputEndPart` | `ID` |

**Tool Execution:**

| Type | Key Fields |
|------|-----------|
| `*StreamToolCallPart` | `ToolCallID`, `ToolName`, `Input` |
| `*StreamToolResultPart` | `ToolCallID`, `ToolName`, `Input`, `Output` |
| `*StreamToolErrorPart` | `ToolCallID`, `ToolName`, `Error` |
| `*ToolOutputDeniedPart` | `ToolCallID`, `ToolName` |
| `*ToolApprovalRequestPart` | `ApprovalID`, `ToolCallID`, `ToolName`, `Input` |
| `*ToolProgressPart` | `ToolCallID`, `ToolName`, `Content` |

**Sources & Files:**

| Type | Key Fields |
|------|-----------|
| `*StreamSourcePart` | `Source` |
| `*StreamFilePart` | `File` |

**Lifecycle:**

| Type | Key Fields |
|------|-----------|
| `*StartPart` | — |
| `*FinishPart` | `FinishReason`, `RawFinishReason`, `TotalUsage` |
| `*StartStepPart` | — |
| `*FinishStepPart` | `FinishReason`, `RawFinishReason`, `Usage`, `Response` |
| `*ErrorPart` | `Error` |
| `*AbortPart` | `Reason` |
| `*RawPart` | `RawValue` |

---

### Usage

```go
type Usage struct {
    InputTokens         int
    OutputTokens        int
    TotalTokens         int
    ReasoningTokens     int
    CachedInputTokens   int
    InputTokenDetails   InputTokenDetail
    OutputTokenDetails  OutputTokenDetail
}

type InputTokenDetail struct {
    CacheReadTokens    int
    CacheCreationTokens int
}

type OutputTokenDetail struct {
    TextTokens      int
    ReasoningTokens int
    AudioTokens     int
}
```

### Source

```go
type Source struct {
    SourceType       string
    ID               string
    URL              string
    Title            string
    ProviderMetadata map[string]any
}
```

### GeneratedFile

```go
type GeneratedFile struct {
    Data      string
    MediaType string
}
```

### ResponseMetadata

```go
type ResponseMetadata struct {
    ID        string
    ModelID   string
    Timestamp time.Time
    Headers   map[string]string
}
```

---

## Package `provider/openai`

### OpenAICompletionsProvider

```go
type OpenAICompletionsProvider struct { /* unexported */ }

func NewCompletions(options ...OpenAICompletionsProviderOption) *OpenAICompletionsProvider
```

Implements `sdk.Provider`. Uses the OpenAI Chat Completions API.

#### Options

```go
func WithAPIKey(apiKey string) OpenAICompletionsProviderOption
func WithBaseURL(baseURL string) OpenAICompletionsProviderOption
func WithHTTPClient(client *http.Client) OpenAICompletionsProviderOption
```

#### Methods

```go
func (p *OpenAICompletionsProvider) Name() string                  // "openai-completions"
func (p *OpenAICompletionsProvider) GetModels() ([]sdk.Model, error)
func (p *OpenAICompletionsProvider) ChatModel(id string) *sdk.Model
func (p *OpenAICompletionsProvider) DoGenerate(ctx, params) (*sdk.GenerateResult, error)
func (p *OpenAICompletionsProvider) DoStream(ctx, params) (*sdk.StreamResult, error)
```
