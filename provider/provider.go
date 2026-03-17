package provider

import (
	"github.com/memohai/twilight-ai/types"
)

type Provider interface {
	Name() string
	GetModels() ([]types.ChatModel, error)
}
