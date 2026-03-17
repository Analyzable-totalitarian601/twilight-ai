package types

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
}
