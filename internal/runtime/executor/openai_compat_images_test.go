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
