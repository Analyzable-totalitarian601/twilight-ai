package types

type ModelType string

const (
	ModelTypeChat ModelType = "chat"
)

type Model struct {
	ID          string
	DisplayName string
	Type        ModelType
	MaxTokens   int
}

type ModelOption func(*Model)

func WithID(id string) ModelOption {
	return func(model *Model) {
		model.ID = id
	}
}

func WithDisplayName(displayName string) ModelOption {
	return func(model *Model) {
		model.DisplayName = displayName
	}
}

func WithMaxTokens(maxTokens int) ModelOption {
	return func(model *Model) {
		model.MaxTokens = maxTokens
	}
}

func NewModel(options ...ModelOption) *Model {
	model := &Model{
		Type: ModelTypeChat,
	}
	for _, option := range options {
		option(model)
	}
	return model
}
