package openai_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/memohai/twilight-ai/internal/testutil"
	"github.com/memohai/twilight-ai/provider/openai"
	"github.com/memohai/twilight-ai/types"
)

// ---------- unit tests (mock server) ----------

func TestDoGenerate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "gpt-4o-mini" {
			t.Errorf("unexpected model: %v", body["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 1700000000,
			"model":   "gpt-4o-mini",
			"choices": []map[string]any{{
				"index":         0,
				"finish_reason": "stop",
				"message":       map[string]any{"role": "assistant", "content": "Hello!"},
			}},
			"usage": map[string]any{
				"prompt_tokens":     5,
				"completion_tokens": 2,
				"total_tokens":      7,
			},
		})
	}))
	defer srv.Close()

	p := openai.NewCompletions(
		openai.WithAPIKey("test-key"),
		openai.WithBaseURL(srv.URL),
	)

	model := &types.Model{ID: "gpt-4o-mini"}
	result, err := p.DoGenerate(context.Background(), types.GenerateParams{
		Model: model,
		Messages: []types.Message{{
			Role:  types.MessageRoleUser,
			Parts: []types.MessagePart{types.TextPart{Text: "Hi"}},
		}},
	})
	if err != nil {
		t.Fatalf("DoGenerate failed: %v", err)
	}

	if result.Text != "Hello!" {
		t.Errorf("expected 'Hello!', got %q", result.Text)
	}
	if result.FinishReason != types.FinishReasonStop {
		t.Errorf("expected finish reason 'stop', got %q", result.FinishReason)
	}
	if result.Usage.InputTokens != 5 {
		t.Errorf("expected 5 input tokens, got %d", result.Usage.InputTokens)
	}
	if result.Usage.OutputTokens != 2 {
		t.Errorf("expected 2 output tokens, got %d", result.Usage.OutputTokens)
	}
}

func TestDoStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("server does not support flushing")
		}

		chunks := []string{
			`{"id":"chunk-1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
			`{"id":"chunk-1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
			`{"id":"chunk-1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`,
		}
		for _, c := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", c)
			flusher.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	p := openai.NewCompletions(
		openai.WithAPIKey("test-key"),
		openai.WithBaseURL(srv.URL),
	)

	model := &types.Model{ID: "gpt-4o-mini"}
	sr, err := p.DoStream(context.Background(), types.GenerateParams{
		Model: model,
		Messages: []types.Message{{
			Role:  types.MessageRoleUser,
			Parts: []types.MessagePart{types.TextPart{Text: "Hi"}},
		}},
	})
	if err != nil {
		t.Fatalf("DoStream failed: %v", err)
	}

	var collected string
	var gotStart, gotFinish bool
	for part := range sr.Stream {
		switch p := part.(type) {
		case *types.StartPart:
			gotStart = true
		case *types.TextDeltaPart:
			collected += p.Text
		case *types.FinishPart:
			gotFinish = true
			if p.FinishReason != types.FinishReasonStop {
				t.Errorf("expected stop, got %q", p.FinishReason)
			}
		}
	}

	if !gotStart {
		t.Error("missing StartPart")
	}
	if !gotFinish {
		t.Error("missing FinishPart")
	}
	if collected != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", collected)
	}
}

func TestDoGenerate_NoModel(t *testing.T) {
	p := openai.NewCompletions(openai.WithAPIKey("k"))
	_, err := p.DoGenerate(context.Background(), types.GenerateParams{})
	if err == nil {
		t.Fatal("expected error for nil model")
	}
}

func TestDoStream_NoModel(t *testing.T) {
	p := openai.NewCompletions(openai.WithAPIKey("k"))
	_, err := p.DoStream(context.Background(), types.GenerateParams{})
	if err == nil {
		t.Fatal("expected error for nil model")
	}
}

// ---------- integration tests (real API, skipped without env) ----------

func envOrSkip(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Skipf("skipping: %s not set", key)
	}
	return v
}

func newIntegrationProvider(t *testing.T) *openai.OpenAICompletionsProvider {
	t.Helper()
	apiKey := envOrSkip(t, "OPENAI_API_KEY")
	opts := []openai.OpenAICompletionsProviderOption{openai.WithAPIKey(apiKey)}
	if base := os.Getenv("OPENAI_BASE_URL"); base != "" {
		opts = append(opts, openai.WithBaseURL(base))
	}
	return openai.NewCompletions(opts...)
}

func integrationModel(t *testing.T) *types.Model {
	t.Helper()
	m := os.Getenv("OPENAI_MODEL")
	if m == "" {
		m = "gpt-4o-mini"
	}
	return &types.Model{ID: m}
}

func TestIntegration_DoGenerate(t *testing.T) {
	p := newIntegrationProvider(t)
	result, err := p.DoGenerate(context.Background(), types.GenerateParams{
		Model: integrationModel(t),
		Messages: []types.Message{{
			Role:  types.MessageRoleUser,
			Parts: []types.MessagePart{types.TextPart{Text: "Say hello in one word."}},
		}},
	})
	if err != nil {
		t.Fatalf("DoGenerate: %v", err)
	}
	t.Logf("text=%q finish=%s tokens=%d/%d", result.Text, result.FinishReason,
		result.Usage.InputTokens, result.Usage.OutputTokens)

	if result.Text == "" {
		t.Error("expected non-empty text")
	}
}

func TestIntegration_DoStream(t *testing.T) {
	p := newIntegrationProvider(t)
	sr, err := p.DoStream(context.Background(), types.GenerateParams{
		Model: integrationModel(t),
		Messages: []types.Message{{
			Role:  types.MessageRoleUser,
			Parts: []types.MessagePart{types.TextPart{Text: "Count from 1 to 5."}},
		}},
	})
	if err != nil {
		t.Fatalf("DoStream: %v", err)
	}

	var text string
	for part := range sr.Stream {
		switch p := part.(type) {
		case *types.TextDeltaPart:
			text += p.Text
			t.Logf("text delta: %q", p.Text)
		case *types.ErrorPart:
			t.Fatalf("stream error: %v", p.Error)
		case *types.FinishPart:
			t.Logf("finish=%s", p.FinishReason)
		}
	}
	t.Logf("streamed text: %q", text)
	if text == "" {
		t.Error("expected non-empty streamed text")
	}
}

func TestMain(m *testing.M) {
	testutil.LoadEnv()
	os.Exit(m.Run())
}
