# Tool Calling

Twilight AI supports LLM tool calling (also known as function calling) with automatic multi-step execution. You define tools as Go structs with execution handlers, and the SDK manages the call-execute-respond loop.

## Defining a Tool

A `Tool` has a name, description, JSON Schema parameters, and an `Execute` function:

```go
weatherTool := sdk.Tool{
    Name:        "get_weather",
    Description: "Get the current weather for a given city",
    Parameters: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "city": map[string]any{
                "type":        "string",
                "description": "City name, e.g. 'Tokyo'",
            },
        },
        "required": []string{"city"},
    },
    Execute: func(ctx *sdk.ToolExecContext, input any) (any, error) {
        args := input.(map[string]any)
        city := args["city"].(string)
        // Call your weather API here...
        return map[string]any{
            "city":    city,
            "temp":    "22°C",
            "weather": "sunny",
        }, nil
    },
}
```

### Tool Fields

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Unique tool name passed to the LLM |
| `Description` | `string` | Human-readable description for the LLM |
| `Parameters` | `any` | JSON Schema describing expected input |
| `Execute` | `ToolExecuteFunc` | Go function that runs when the LLM calls this tool |
| `RequireApproval` | `bool` | If true, requires approval before execution |

### ToolExecContext

The execution function receives a `*ToolExecContext` that embeds `context.Context` and provides additional metadata:

```go
type ToolExecContext struct {
    context.Context
    ToolCallID   string           // unique ID for this call
    ToolName     string           // name of the tool being called
    SendProgress func(content any) // send progress updates (nil when not streaming)
}
```

## Single-Step Tool Calling

With `MaxSteps` at its default (`0`), the SDK returns the tool call without executing it:

```go
result, err := sdk.GenerateTextResult(ctx,
    sdk.WithModel(model),
    sdk.WithMessages([]sdk.Message{
        sdk.UserMessage("What's the weather in Tokyo?"),
    }),
    sdk.WithTools([]sdk.Tool{weatherTool}),
)

// result.ToolCalls contains the LLM's tool call request
// result.Text may be empty — the LLM chose to call a tool instead
for _, tc := range result.ToolCalls {
    fmt.Printf("Tool: %s, Input: %v\n", tc.ToolName, tc.Input)
}
```

## Multi-Step Execution

Set `WithMaxSteps` to enable automatic tool execution. The SDK will:

1. Send messages to the LLM
2. If the LLM returns tool calls, execute them
3. Append tool results to the conversation
4. Send updated messages back to the LLM
5. Repeat until the LLM stops calling tools or the step limit is reached

```go
result, err := sdk.GenerateTextResult(ctx,
    sdk.WithModel(model),
    sdk.WithMessages([]sdk.Message{
        sdk.UserMessage("What's the weather in Tokyo and Paris?"),
    }),
    sdk.WithTools([]sdk.Tool{weatherTool}),
    sdk.WithMaxSteps(10),
)

// result.Text contains the final response after all tool calls
// result.Steps contains each step's details
fmt.Println(result.Text)
fmt.Printf("Completed in %d steps\n", len(result.Steps))
```

### MaxSteps Values

| Value | Behavior |
|-------|----------|
| `0` (default) | Single LLM call, no tool auto-execution |
| `N` (N > 0) | Up to N LLM calls in the loop |
| `-1` | Unlimited — loops until the LLM stops requesting tools |

## Tool Choice

Control how the LLM decides whether to use tools:

```go
sdk.WithToolChoice("auto")     // LLM decides (default)
sdk.WithToolChoice("none")     // never call tools
sdk.WithToolChoice("required") // must call at least one tool
```

## Approval Flow

For sensitive operations, mark tools with `RequireApproval` and provide an approval handler:

```go
dangerousTool := sdk.Tool{
    Name:            "delete_file",
    Description:     "Delete a file from the filesystem",
    Parameters:      fileSchema,
    RequireApproval: true,
    Execute: func(ctx *sdk.ToolExecContext, input any) (any, error) {
        // This only runs if approved
        path := input.(map[string]any)["path"].(string)
        return os.Remove(path), nil
    },
}

result, err := sdk.GenerateTextResult(ctx,
    sdk.WithModel(model),
    sdk.WithMessages(msgs),
    sdk.WithTools([]sdk.Tool{dangerousTool}),
    sdk.WithMaxSteps(5),
    sdk.WithApprovalHandler(func(ctx context.Context, call sdk.ToolCall) (bool, error) {
        fmt.Printf("Allow %s with input %v? [y/n] ", call.ToolName, call.Input)
        var answer string
        fmt.Scanln(&answer)
        return answer == "y", nil
    }),
)
```

When a tool call is denied, a `ToolOutputDeniedPart` is sent in streaming mode, and the tool result is marked as an error.

## Streaming with Tools

Tool calling works seamlessly with `StreamText`. Progress updates from tool execution are delivered through the stream:

```go
sr, err := sdk.StreamText(ctx,
    sdk.WithModel(model),
    sdk.WithMessages([]sdk.Message{
        sdk.UserMessage("What's the weather in Tokyo?"),
    }),
    sdk.WithTools([]sdk.Tool{weatherTool}),
    sdk.WithMaxSteps(5),
)

for part := range sr.Stream {
    switch p := part.(type) {
    case *sdk.TextDeltaPart:
        fmt.Print(p.Text)
    case *sdk.StreamToolCallPart:
        fmt.Printf("\n[Calling tool: %s]\n", p.ToolName)
    case *sdk.StreamToolResultPart:
        fmt.Printf("[Tool result: %v]\n", p.Output)
    case *sdk.ToolProgressPart:
        fmt.Printf("[Progress: %v]\n", p.Content)
    case *sdk.ErrorPart:
        log.Fatal(p.Error)
    }
}
```

### Sending Progress from Tools

During streaming, tools can send progress updates via `SendProgress`:

```go
Execute: func(ctx *sdk.ToolExecContext, input any) (any, error) {
    if ctx.SendProgress != nil {
        ctx.SendProgress("Fetching data...")
    }
    // do work...
    if ctx.SendProgress != nil {
        ctx.SendProgress("Processing results...")
    }
    return result, nil
},
```

## Inspecting Steps

After a multi-step execution, inspect individual steps:

```go
for i, step := range result.Steps {
    fmt.Printf("Step %d: finish=%s, tokens=%d\n",
        i+1, step.FinishReason, step.Usage.TotalTokens)

    for _, tc := range step.ToolCalls {
        fmt.Printf("  Called: %s(%v)\n", tc.ToolName, tc.Input)
    }
    for _, tr := range step.ToolResults {
        fmt.Printf("  Result: %v\n", tr.Output)
    }
}
```

## Callbacks

### OnStep

Called after each step completes. Can override params for the next step:

```go
sdk.WithOnStep(func(step *sdk.StepResult) *sdk.GenerateParams {
    fmt.Printf("Step finished: %s\n", step.FinishReason)
    return nil // return non-nil to override next step's params
}),
```

### PrepareStep

Called before each step (starting from step 2). Allows modifying params:

```go
sdk.WithPrepareStep(func(params *sdk.GenerateParams) *sdk.GenerateParams {
    // Reduce temperature after first step
    t := 0.3
    params.Temperature = &t
    return params
}),
```

### OnFinish

Called once when all steps are complete:

```go
sdk.WithOnFinish(func(result *sdk.GenerateResult) {
    fmt.Printf("Done! Total tokens: %d\n", result.Usage.TotalTokens)
}),
```

## Next Steps

- [Streaming](streaming.md) — deep dive into StreamPart types
- [API Reference](api-reference.md) — complete type and function reference
