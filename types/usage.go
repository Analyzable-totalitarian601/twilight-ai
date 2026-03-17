package types

type InputTokenDetail struct {
	NoCacheTokens    int
	CacheReadTokens  int
	CacheWriteTokens int
}

type OutputTokenDetail struct {
	TextTokens      int
	ReasoningTokens int
}

type Usage struct {
	InputTokens        int
	OutputTokens       int
	TotalTokens        int
	ReasoningTokens    int
	CachedInputTokens  int
	InputTokenDetails  InputTokenDetail
	OutputTokenDetails OutputTokenDetail
}
