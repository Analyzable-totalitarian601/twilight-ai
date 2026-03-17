package chat

import (
	"context"
	"fmt"

	"github.com/memohai/twilight-ai/provider"
	"github.com/memohai/twilight-ai/types"
)

type TextGenerator struct {
	Provider         provider.Provider
	Model            *types.Model
	Messages         []types.Message
	Temperature      *float64
	TopP             *float64
	MaxTokens        *int
	StopSequences    []string
	FrequencyPenalty *float64
	PresencePenalty  *float64
	Seed             *int
	ReasoningEffort  *string
}

type TextGeneratorOption func(*TextGenerator)

func WithModel(model *types.Model) TextGeneratorOption {
	return func(g *TextGenerator) {
		g.Model = model
	}
}

func WithMessages(messages []types.Message) TextGeneratorOption {
	return func(g *TextGenerator) {
		g.Messages = messages
	}
}

func WithProvider(p provider.Provider) TextGeneratorOption {
	return func(g *TextGenerator) {
		g.Provider = p
	}
}

func WithTemperature(t float64) TextGeneratorOption {
	return func(g *TextGenerator) {
		g.Temperature = &t
	}
}

func WithTopP(p float64) TextGeneratorOption {
	return func(g *TextGenerator) {
		g.TopP = &p
	}
}

func WithMaxTokens(n int) TextGeneratorOption {
	return func(g *TextGenerator) {
		g.MaxTokens = &n
	}
}

func WithStopSequences(s []string) TextGeneratorOption {
	return func(g *TextGenerator) {
		g.StopSequences = s
	}
}

func WithFrequencyPenalty(p float64) TextGeneratorOption {
	return func(g *TextGenerator) {
		g.FrequencyPenalty = &p
	}
}

func WithPresencePenalty(p float64) TextGeneratorOption {
	return func(g *TextGenerator) {
		g.PresencePenalty = &p
	}
}

func WithSeed(s int) TextGeneratorOption {
	return func(g *TextGenerator) {
		g.Seed = &s
	}
}

func WithReasoningEffort(e string) TextGeneratorOption {
	return func(g *TextGenerator) {
		g.ReasoningEffort = &e
	}
}

func NewTextGenerator(options ...TextGeneratorOption) *TextGenerator {
	generator := &TextGenerator{}
	for _, option := range options {
		option(generator)
	}
	return generator
}

func (g *TextGenerator) params() types.GenerateParams {
	return types.GenerateParams{
		Model:            g.Model,
		Messages:         g.Messages,
		Temperature:      g.Temperature,
		TopP:             g.TopP,
		MaxTokens:        g.MaxTokens,
		StopSequences:    g.StopSequences,
		FrequencyPenalty: g.FrequencyPenalty,
		PresencePenalty:  g.PresencePenalty,
		Seed:             g.Seed,
		ReasoningEffort:  g.ReasoningEffort,
	}
}

func (g *TextGenerator) Generate(ctx context.Context) (string, error) {
	if g.Provider == nil {
		return "", fmt.Errorf("chat: provider is required")
	}

	result, err := g.Provider.DoGenerate(ctx, g.params())
	if err != nil {
		return "", err
	}

	return result.Text, nil
}

func (g *TextGenerator) GenerateResult(ctx context.Context) (*types.GenerateResult, error) {
	if g.Provider == nil {
		return nil, fmt.Errorf("chat: provider is required")
	}

	return g.Provider.DoGenerate(ctx, g.params())
}

func (g *TextGenerator) Stream(ctx context.Context) (*types.StreamResult, error) {
	if g.Provider == nil {
		return nil, fmt.Errorf("chat: provider is required")
	}

	return g.Provider.DoStream(ctx, g.params())
}
