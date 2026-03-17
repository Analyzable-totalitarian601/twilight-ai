package types

type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonContentFilter FinishReason = "content-filter"
	FinishReasonToolCalls     FinishReason = "tool-calls"
	FinishReasonError         FinishReason = "error"
	FinishReasonOther         FinishReason = "other"
	FinishReasonUnknown       FinishReason = "unknown"
)

type GenerateParams struct {
	Model    *Model
	Messages []Message

	Temperature      *float64
	TopP             *float64
	MaxTokens        *int
	StopSequences    []string
	FrequencyPenalty *float64
	PresencePenalty  *float64
	Seed             *int
	ReasoningEffort  *string
}

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
}
