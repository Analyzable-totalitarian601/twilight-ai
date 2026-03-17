package types

type InputTokenDetail struct {
	noCacheTokens int
	cacheReadTokens int
}

type OutputTokenDetail struct {
	textTokens int
	reasoningTokens int
}

type ChatModelUsage struct {
	inputTokens int
	outputTokens int
	totalTokens int
	reasoningTokens int
	cachedInputTokens int
	inputTokenDetails InputTokenDetail
	outputTokenDetails OutputTokenDetail
}
