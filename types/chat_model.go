package types

type ChatModel struct {
	id string
	displayName string
	modelType ModelType
	maxTokens int
}

type ChatModelOption func(*ChatModel)

func WithID(id string) ChatModelOption {
	return func(model *ChatModel) {
		model.id = id
	}
}

func WithDisplayName(displayName string) ChatModelOption {
	return func(model *ChatModel) {
		model.displayName = displayName
	}
}

func WithMaxTokens(maxTokens int) ChatModelOption {
	return func(model *ChatModel) {
		model.maxTokens = maxTokens
	}
}

func New(options ...ChatModelOption) *ChatModel {
	model := &ChatModel{
		modelType: ModelTypeChat,
	}
	for _, option := range options {
		option(model)
	}
	return model
}
