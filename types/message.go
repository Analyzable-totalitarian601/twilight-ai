package types

type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
	MessageRoleTool      MessageRole = "tool"
)

type MessagePartType string

const (
	MessagePartTypeText       MessagePartType = "text"
	MessagePartTypeReasoning  MessagePartType = "reasoning"
	MessagePartTypeImage      MessagePartType = "image"
	MessagePartTypeFile       MessagePartType = "file"
	MessagePartTypeToolCall   MessagePartType = "tool-call"
	MessagePartTypeToolResult MessagePartType = "tool-result"
)

type MessagePart interface {
	PartType() MessagePartType
}

// --- Text ---

type TextPart struct {
	Text string
}

func (p TextPart) PartType() MessagePartType { return MessagePartTypeText }

// --- Reasoning ---

type ReasoningPart struct {
	Text      string
	Signature string
}

func (p ReasoningPart) PartType() MessagePartType { return MessagePartTypeReasoning }

// --- Image ---

type ImagePart struct {
	Image     string // URL or base64 encoded data
	MediaType string
}

func (p ImagePart) PartType() MessagePartType { return MessagePartTypeImage }

// --- File ---

type FilePart struct {
	Data      string // base64 encoded data or URL
	MediaType string
	Filename  string
}

func (p FilePart) PartType() MessagePartType { return MessagePartTypeFile }

// --- Tool Call (in assistant messages) ---

type ToolCallPart struct {
	ToolCallID string
	ToolName   string
	Input      any
}

func (p ToolCallPart) PartType() MessagePartType { return MessagePartTypeToolCall }

// --- Tool Result (in tool messages) ---

type ToolResultPart struct {
	ToolCallID string
	ToolName   string
	Result     any
	IsError    bool
}

func (p ToolResultPart) PartType() MessagePartType { return MessagePartTypeToolResult }

// --- Message ---

type Message struct {
	Role  MessageRole
	Parts []MessagePart
}
