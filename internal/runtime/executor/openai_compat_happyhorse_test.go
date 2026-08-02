package executor

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor/helps"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
	"github.com/tidwall/gjson"
)

func TestOpenAICompatExecutorHappyHorseAllModes(t *testing.T) {
	type testCase struct {
		name    string
		model   string
		payload string
	}
	tests := []testCase{
		{
			name:    "text-to-video",
			model:   helps.HappyHorseT2VModel,
			payload: "{\"model\":\"happyhorse-1.1-t2v\",\"prompt\":\"a paper city at night\"}",
		},
		{
			name:    "image-to-video",
			model:   helps.HappyHorseI2VModel,
			payload: "{\"model\":\"happyhorse-1.1-i2v\",\"prompt\":\"make it move\",\"image_url\":\"https://example.com/first.png\"}",
		},
		{
			name:    "reference-to-video",
			model:   helps.HappyHorseR2VModel,
			payload: "{\"model\":\"happyhorse-1.1-r2v\",\"prompt\":\"animate [Image 1]\",\"reference_images\":[\"https://example.com/reference.png\"]}",
		},
	}

	var requests []*http.Request
	var bodies [][]byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r)
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"output\":{\"task_id\":\"task-123\",\"task_status\":\"PENDING\"},\"request_id\":\"request-123\"}"))
	}))
	defer server.Close()

	auth := &cliproxyauth.Auth{ID: "auth-1", Attributes: map[string]string{
		"base_url": server.URL + "/compatible-mode/v1",
		"api_key":  "sk-test",
	}}
	executor := NewOpenAICompatExecutor("bailian", nil)
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			resp, err := executor.Execute(
				t.Context(),
				auth,
				cliproxyexecutor.Request{Model: testCase.model, Payload: []byte(testCase.payload)},
				cliproxyexecutor.Options{
					SourceFormat: sdktranslator.FromString("openai-video"),
					Alt:          "happyhorse/create",
				},
			)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if got := gjson.GetBytes(resp.Payload, "output.task_id").String(); got != "task-123" {
				t.Fatalf("task ID = %q", got)
			}
			index := len(requests) - 1
			request := requests[index]
			if request.Method != http.MethodPost || request.URL.Path != "/api/v1/services/aigc/video-generation/video-synthesis" {
				t.Fatalf("request = %s %s", request.Method, request.URL.Path)
			}
			if request.Header.Get("Authorization") != "Bearer sk-test" {
				t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
			}
			if request.Header.Get("X-DashScope-Async") != "enable" {
				t.Fatalf("async header = %q", request.Header.Get("X-DashScope-Async"))
			}
			if got := gjson.GetBytes(bodies[index], "model").String(); got != testCase.model {
				t.Fatalf("native model = %q, want %q", got, testCase.model)
			}
		})
	}
}

func TestOpenAICompatExecutorHappyHorseRetrieve(t *testing.T) {
	var gotMethod string
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"output\":{\"task_id\":\"task-456\",\"task_status\":\"SUCCEEDED\",\"video_url\":\"https://example.com/video.mp4\"}}"))
	}))
	defer server.Close()

	auth := &cliproxyauth.Auth{Attributes: map[string]string{
		"base_url": server.URL + "/compatible-mode/v1",
		"api_key":  "sk-test",
	}}
	executor := NewOpenAICompatExecutor("bailian", nil)
	resp, err := executor.Execute(
		t.Context(),
		auth,
		cliproxyexecutor.Request{Model: helps.HappyHorseR2VModel, Payload: []byte("{\"request_id\":\"task-456\"}")},
		cliproxyexecutor.Options{
			SourceFormat: sdktranslator.FromString("openai-video"),
			Alt:          "happyhorse/retrieve",
		},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/api/v1/tasks/task-456" {
		t.Fatalf("request = %s %s", gotMethod, gotPath)
	}
	if got := gjson.GetBytes(resp.Payload, "output.video_url").String(); got != "https://example.com/video.mp4" {
		t.Fatalf("video URL = %q", got)
	}
}

func TestOpenAICompatExecutorHappyHorseRejectsStreaming(t *testing.T) {
	executor := NewOpenAICompatExecutor("bailian", nil)
	_, err := executor.ExecuteStream(
		t.Context(),
		nil,
		cliproxyexecutor.Request{Model: helps.HappyHorseT2VModel, Payload: []byte("{\"prompt\":\"x\"}")},
		cliproxyexecutor.Options{SourceFormat: sdktranslator.FromString("openai-video")},
	)
	if err == nil {
		t.Fatal("expected streaming error")
	}
	if got := err.(statusErr).StatusCode(); got != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", got, http.StatusBadRequest)
	}
}
