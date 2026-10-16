package executor

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
	"github.com/tidwall/gjson"
)

func TestOpenAICompatExecutorBailianSubtitleJSONContract(t *testing.T) {
	var gotBody []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/compatible-mode/v1/chat/completions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_subtitle","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"{\"segments\":[{\"segmentId\":\"s1\",\"text\":\"Hello\"}]}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":10,"total_tokens":30}}`))
	}))
	defer upstream.Close()

	cfg := &config.Config{Payload: config.PayloadConfig{Override: []config.PayloadRule{{
		Models: []config.PayloadModelRule{
			{Name: "drama-subtitle-semantic", Protocol: "openai"},
			{Name: "drama-subtitle-translation", Protocol: "openai"},
		},
		Params: map[string]any{"enable_thinking": false},
	}}}}
	executor := NewOpenAICompatExecutor("bailian-subtitles", cfg)
	auth := &cliproxyauth.Auth{Attributes: map[string]string{
		"base_url": upstream.URL + "/compatible-mode/v1",
		"api_key":  "test-dashscope-key",
	}}
	payload := []byte(`{"model":"drama-subtitle-translation","messages":[{"role":"system","content":"Return JSON only"},{"role":"user","content":"{\"segments\":[{\"segmentId\":\"s1\",\"text\":\"你好\"}]}"}],"response_format":{"type":"json_object"},"max_completion_tokens":8000}`)

	response, err := executor.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "qwen-plus-2025-12-01",
		Payload: payload,
	}, cliproxyexecutor.Options{
		SourceFormat: sdktranslator.FromString("openai"),
		Stream:       false,
		Metadata: map[string]any{
			cliproxyexecutor.RequestedModelMetadataKey: "drama-subtitle-translation",
		},
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if got := gjson.GetBytes(gotBody, "model").String(); got != "qwen-plus-2025-12-01" {
		t.Fatalf("model = %q; body=%s", got, string(gotBody))
	}
	if got := gjson.GetBytes(gotBody, "response_format.type").String(); got != "json_object" {
		t.Fatalf("response format = %q; body=%s", got, string(gotBody))
	}
	if got := gjson.GetBytes(gotBody, "max_completion_tokens").Int(); got != 8000 {
		t.Fatalf("max_completion_tokens = %d; body=%s", got, string(gotBody))
	}
	if got := gjson.GetBytes(gotBody, "enable_thinking"); !got.Exists() || got.Bool() {
		t.Fatalf("enable_thinking = %s; body=%s", got.Raw, string(gotBody))
	}
	if got := gjson.GetBytes(response.Payload, "choices.0.message.content").String(); got == "" {
		t.Fatalf("missing subtitle JSON response: %s", string(response.Payload))
	}
}
