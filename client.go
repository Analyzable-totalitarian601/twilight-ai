package twilightai

import (
	"context"

	"github.com/memohai/twilight-ai/internal/chat"
	"github.com/memohai/twilight-ai/provider"
	"github.com/memohai/twilight-ai/types"
)

type Client struct {
	provider provider.Provider
}

type ClientOption func(*Client)

func WithProvider(p provider.Provider) ClientOption {
	return func(c *Client) {
		c.provider = p
	}
}

func NewClient(options ...ClientOption) *Client {
	client := &Client{}
	for _, option := range options {
		option(client)
	}
	return client
}

func (c *Client) GenerateText(ctx context.Context, options ...chat.TextGeneratorOption) (string, error) {
	generator := chat.NewTextGenerator(append(options, chat.WithProvider(c.provider))...)
	return generator.Generate(ctx)
}

func (c *Client) GenerateTextResult(ctx context.Context, options ...chat.TextGeneratorOption) (*types.GenerateResult, error) {
	generator := chat.NewTextGenerator(append(options, chat.WithProvider(c.provider))...)
	return generator.GenerateResult(ctx)
}

func (c *Client) StreamText(ctx context.Context, options ...chat.TextGeneratorOption) (*types.StreamResult, error) {
	generator := chat.NewTextGenerator(append(options, chat.WithProvider(c.provider))...)
	return generator.Stream(ctx)
}
