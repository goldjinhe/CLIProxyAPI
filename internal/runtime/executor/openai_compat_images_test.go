package executor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
)

func TestOpenAICompatExecuteImagesGenerationsPassthrough(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotPayload map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"created":1,"data":[{"b64_json":"abc"}]}`))
	}))
	defer upstream.Close()

	exec := NewOpenAICompatExecutor("openai-compatibility", &config.Config{})
	auth := &cliproxyauth.Auth{
		ID:       "compat-image",
		Provider: "openai-compatibility",
		Attributes: map[string]string{
			"base_url": upstream.URL,
			"api_key":  "upstream-key",
		},
	}
	resp, err := exec.Execute(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "upstream-image-model",
		Payload: []byte(`{"model":"image-model","prompt":"draw"}`),
	}, cliproxyexecutor.Options{Alt: "images/generations"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if gotPath != "/images/generations" {
		t.Fatalf("path = %q, want %q", gotPath, "/images/generations")
	}
	if gotAuth != "Bearer upstream-key" {
		t.Fatalf("authorization = %q, want bearer upstream key", gotAuth)
	}
	if gotPayload["model"] != "upstream-image-model" {
		t.Fatalf("model = %v, want upstream-image-model; payload=%v", gotPayload["model"], gotPayload)
	}
	if string(resp.Payload) != `{"created":1,"data":[{"b64_json":"abc"}]}` {
		t.Fatalf("payload = %s", resp.Payload)
	}
}

func TestDashScopeImageGenerationEndpoint(t *testing.T) {
	got := dashScopeImageGenerationEndpoint("https://dashscope.aliyuncs.com/compatible-mode/v1")
	want := "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"
	if got != want {
		t.Fatalf("endpoint = %q, want %q", got, want)
	}
}

func TestOpenAICompatExecuteDashScopeImagesGenerations(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotPayload map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if gotPath != "/api/v1/services/aigc/multimodal-generation/generation" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":[{"image":"https://example.com/generated.png"}]}}]},"usage":{"image_count":1,"width":1024,"height":1024},"request_id":"req-1"}`))
	}))
	defer upstream.Close()

	exec := NewOpenAICompatExecutor("openai-compatibility", &config.Config{})
	auth := &cliproxyauth.Auth{
		ID:       "dashscope-image",
		Provider: "openai-compatibility",
		Attributes: map[string]string{
			"base_url": upstream.URL,
			"api_key":  "dashscope-key",
		},
	}
	resp, err := exec.executeDashScopeImageGeneration(context.Background(), auth, cliproxyexecutor.Request{
		Model:   "qwen-image-2.0-pro",
		Payload: []byte(`{"model":"gpt-image-1","prompt":"draw a cat","size":"1024x1024","n":1}`),
	}, "dashscope-key", "qwen-image-2.0-pro", upstream.URL+"/api/v1/services/aigc/multimodal-generation/generation")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if gotAuth != "Bearer dashscope-key" {
		t.Fatalf("authorization = %q, want bearer DashScope key", gotAuth)
	}
	if gotPayload["model"] != "qwen-image-2.0-pro" {
		t.Fatalf("model = %v, want qwen-image-2.0-pro", gotPayload["model"])
	}
	input, ok := gotPayload["input"].(map[string]any)
	if !ok {
		t.Fatalf("missing input object: %v", gotPayload)
	}
	messages, ok := input["messages"].([]any)
	if !ok || len(messages) != 1 {
		t.Fatalf("messages = %v, want one message", input["messages"])
	}
	parameters, ok := gotPayload["parameters"].(map[string]any)
	if !ok {
		t.Fatalf("missing parameters object: %v", gotPayload)
	}
	if parameters["size"] != "1024*1024" {
		t.Fatalf("size = %v, want 1024*1024", parameters["size"])
	}
	var out openAIImageResponse
	if err := json.Unmarshal(resp.Payload, &out); err != nil {
		t.Fatalf("decode OpenAI image response: %v; body=%s", err, resp.Payload)
	}
	if len(out.Data) != 1 || out.Data[0].URL != "https://example.com/generated.png" {
		t.Fatalf("unexpected response: %+v", out)
	}
}
