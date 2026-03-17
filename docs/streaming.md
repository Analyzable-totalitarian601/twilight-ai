# Streaming

Twilight AI uses Go channels for streaming, giving you type-safe, idiomatic control over real-time LLM output.

## Basic Streaming

```go
sr, err := sdk.StreamText(ctx,
    sdk.WithModel(model),
    sdk.WithMessages([]sdk.Message{
        sdk.UserMessage("Count from 1 to 10."),
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
```

## StreamResult

`StreamText` returns a `*StreamResult`:

```go
type StreamResult struct {
    Stream   <-chan StreamPart  // channel that yields stream parts
    Steps    []StepResult       // populated after stream is consumed
    Messages []Message          // populated after stream is consumed
}
```

`Steps` and `Messages` are filled as you consume the stream and are safe to read after the `for range` loop exits.

### Convenience Methods

**`Text()`** — consumes the stream and returns concatenated text:

```go
sr, _ := sdk.StreamText(ctx, ...)
text, err := sr.Text()
```

**`ToResult()`** — consumes the stream and assembles a full `GenerateResult`:

```go
sr, _ := sdk.StreamText(ctx, ...)
result, err := sr.ToResult()
fmt.Println(result.Text, result.Usage.TotalTokens)
```

## StreamPart Types

Every chunk from the stream implements the `StreamPart` interface:

```go
type StreamPart interface {
    Type() StreamPartType
}
```

Use a type switch to handle specific parts. Here is the complete list:

### Text Parts

| Type | Fields | Description |
|------|--------|-------------|
| `*TextStartPart` | `ID` | Text generation has started |
| `*TextDeltaPart` | `ID`, `Text` | A chunk of generated text |
| `*TextEndPart` | `ID` | Text generation has ended |

### Reasoning Parts

For models that emit reasoning content (e.g. o1, DeepSeek-R1):

| Type | Fields | Description |
|------|--------|-------------|
| `*ReasoningStartPart` | `ID` | Reasoning started |
| `*ReasoningDeltaPart` | `ID`, `Text` | A chunk of reasoning text |
| `*ReasoningEndPart` | `ID` | Reasoning ended |

### Tool Input Parts

Streamed as the LLM constructs tool call arguments:

| Type | Fields | Description |
|------|--------|-------------|
| `*ToolInputStartPart` | `ID`, `ToolName` | LLM started building tool arguments |
| `*ToolInputDeltaPart` | `ID`, `Delta` | A chunk of tool argument JSON |
| `*ToolInputEndPart` | `ID` | Tool argument construction complete |

### Tool Execution Parts

Emitted during the tool execution phase (multi-step mode):

| Type | Fields | Description |
|------|--------|-------------|
| `*StreamToolCallPart` | `ToolCallID`, `ToolName`, `Input` | Complete tool call (parsed input) |
| `*StreamToolResultPart` | `ToolCallID`, `ToolName`, `Input`, `Output` | Tool execution result |
| `*StreamToolErrorPart` | `ToolCallID`, `ToolName`, `Error` | Tool execution failed |
| `*ToolOutputDeniedPart` | `ToolCallID`, `ToolName` | Tool call denied by approval handler |
| `*ToolApprovalRequestPart` | `ApprovalID`, `ToolCallID`, `ToolName`, `Input` | Approval requested |
| `*ToolProgressPart` | `ToolCallID`, `ToolName`, `Content` | Progress update from tool execution |

### Source & File Parts

| Type | Fields | Description |
|------|--------|-------------|
| `*StreamSourcePart` | `Source` | A source reference (RAG) |
| `*StreamFilePart` | `File` | A generated file |

### Lifecycle Parts

| Type | Fields | Description |
|------|--------|-------------|
| `*StartPart` | — | Stream started |
| `*FinishPart` | `FinishReason`, `RawFinishReason`, `TotalUsage` | Stream finished |
| `*StartStepPart` | — | A new step started |
| `*FinishStepPart` | `FinishReason`, `RawFinishReason`, `Usage`, `Response` | Step finished |
| `*ErrorPart` | `Error` | An error occurred |
| `*AbortPart` | `Reason` | Stream was aborted |
| `*RawPart` | `RawValue` | Raw provider-specific data |

## Stream Lifecycle

A typical single-step stream produces parts in this order:

```
StartPart
  StartStepPart
    TextStartPart
    TextDeltaPart (repeated)
    TextEndPart
  FinishStepPart
FinishPart
```

A multi-step stream with tool calls:

```
StartPart
  StartStepPart                 ← Step 1
    ToolInputStartPart
    ToolInputDeltaPart (repeated)
    ToolInputEndPart
    StreamToolCallPart
  FinishStepPart
  StreamToolResultPart          ← Tool execution
  StartStepPart                 ← Step 2
    TextStartPart
    TextDeltaPart (repeated)
    TextEndPart
  FinishStepPart
FinishPart
```

## Handling Reasoning Content

Models like o1 or DeepSeek-R1 emit reasoning before the final answer:

```go
for part := range sr.Stream {
    switch p := part.(type) {
    case *sdk.ReasoningDeltaPart:
        fmt.Fprintf(os.Stderr, "[thinking] %s", p.Text)
    case *sdk.TextDeltaPart:
        fmt.Print(p.Text)
    }
}
```

## Full Example: Rich Stream Handler

```go
sr, err := sdk.StreamText(ctx,
    sdk.WithModel(model),
    sdk.WithMessages(msgs),
    sdk.WithTools(tools),
    sdk.WithMaxSteps(10),
)
if err != nil {
    log.Fatal(err)
}

for part := range sr.Stream {
    switch p := part.(type) {
    case *sdk.StartPart:
        fmt.Println("--- Stream started ---")

    case *sdk.TextDeltaPart:
        fmt.Print(p.Text)

    case *sdk.ReasoningDeltaPart:
        // optionally display reasoning
        fmt.Fprintf(os.Stderr, "%s", p.Text)

    case *sdk.StreamToolCallPart:
        fmt.Printf("\n🔧 Calling %s\n", p.ToolName)

    case *sdk.StreamToolResultPart:
        fmt.Printf("✅ %s returned: %v\n", p.ToolName, p.Output)

    case *sdk.StreamToolErrorPart:
        fmt.Printf("❌ %s error: %v\n", p.ToolName, p.Error)

    case *sdk.ToolProgressPart:
        fmt.Printf("⏳ %s: %v\n", p.ToolName, p.Content)

    case *sdk.FinishPart:
        fmt.Printf("\n--- Done (reason: %s, tokens: %d) ---\n",
            p.FinishReason, p.TotalUsage.TotalTokens)

    case *sdk.ErrorPart:
        log.Printf("Error: %v", p.Error)
    }
}

// Safe to access after stream is consumed
fmt.Printf("Total steps: %d\n", len(sr.Steps))
```

## Next Steps

- [Tool Calling](tools.md) — tool definitions and multi-step execution
- [API Reference](api-reference.md) — complete type and function reference
