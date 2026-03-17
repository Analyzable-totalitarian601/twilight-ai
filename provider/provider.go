package provider

import (
	"context"

	"github.com/memohai/twilight-ai/types"
)

type Provider interface {
	Name() string
	GetModels() ([]types.Model, error)
	DoGenerate(ctx context.Context, params types.GenerateParams) (*types.GenerateResult, error)
	DoStream(ctx context.Context, params types.GenerateParams) (*types.StreamResult, error)
}
