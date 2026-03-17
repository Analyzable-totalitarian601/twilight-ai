package twilightai_test

import (
	"context"
	"os"
	"testing"

	twilightai "github.com/memohai/twilight-ai"
	"github.com/memohai/twilight-ai/internal/chat"
	"github.com/memohai/twilight-ai/internal/testutil"
	"github.com/memohai/twilight-ai/provider/openai"
	"github.com/memohai/twilight-ai/types"
)

func TestMain(m *testing.M) {
	testutil.LoadEnv()
	os.Exit(m.Run())
}

func envOrSkip(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Skipf("skipping: %s not set", key)
	}
	return v
}

func newClient(t *testing.T) *twilightai.Client {
	t.Helper()
	apiKey := envOrSkip(t, "OPENAI_API_KEY")
	opts := []openai.OpenAICompletionsProviderOption{openai.WithAPIKey(apiKey)}
	if base := os.Getenv("OPENAI_BASE_URL"); base != "" {
		opts = append(opts, openai.WithBaseURL(base))
	}
	return twilightai.NewClient(twilightai.WithProvider(openai.NewCompletions(opts...)))
}

func model(t *testing.T) *types.Model {
	t.Helper()
	m := os.Getenv("OPENAI_MODEL")
	if m == "" {
		m = "gpt-4o-mini"
	}
	return &types.Model{ID: m}
}

func TestClient_GenerateText(t *testing.T) {
	c := newClient(t)
	text, err := c.GenerateText(context.Background(),
		chat.WithModel(model(t)),
		chat.WithMessages([]types.Message{{
			Role:  types.MessageRoleUser,
			Parts: []types.MessagePart{types.TextPart{Text: "Say hi in one word."}},
		}}),
	)
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	t.Logf("response: %q", text)
	if text == "" {
		t.Error("expected non-empty response")
	}
}

func TestClient_GenerateTextResult(t *testing.T) {
	c := newClient(t)
	result, err := c.GenerateTextResult(context.Background(),
		chat.WithModel(model(t)),
		chat.WithMessages([]types.Message{{
			Role:  types.MessageRoleUser,
			Parts: []types.MessagePart{types.TextPart{Text: "Say hi in one word."}},
		}}),
	)
	if err != nil {
		t.Fatalf("GenerateTextResult: %v", err)
	}
	t.Logf("text=%q finish=%s input=%d output=%d",
		result.Text, result.FinishReason,
		result.Usage.InputTokens, result.Usage.OutputTokens)

	if result.Text == "" {
		t.Error("expected non-empty text")
	}
	if result.FinishReason != types.FinishReasonStop {
		t.Errorf("expected stop, got %s", result.FinishReason)
	}
}

func TestClient_StreamText(t *testing.T) {
	c := newClient(t)
	sr, err := c.StreamText(context.Background(),
		chat.WithModel(model(t)),
		chat.WithMessages([]types.Message{{
			Role:  types.MessageRoleUser,
			Parts: []types.MessagePart{types.TextPart{Text: "Count from 1 to 3."}},
		}}),
	)
	if err != nil {
		t.Fatalf("StreamText: %v", err)
	}

	var text string
	for part := range sr.Stream {
		switch p := part.(type) {
		case *types.TextDeltaPart:
			text += p.Text
		case *types.ErrorPart:
			t.Fatalf("stream error: %v", p.Error)
		case *types.FinishPart:
			t.Logf("finish=%s tokens=%d", p.FinishReason, p.TotalUsage.TotalTokens)
		}
	}
	t.Logf("streamed: %q", text)
	if text == "" {
		t.Error("expected non-empty streamed text")
	}
}
