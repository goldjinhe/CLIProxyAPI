package openai

import (
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	"github.com/tidwall/gjson"
)

func TestHappyHorseVideoModelValidation(t *testing.T) {
	if !isSupportedVideosModel("happyhorse-1.1-t2v") {
		t.Fatal("text-to-video model should be supported")
	}
	if !isSupportedVideosModel("happyhorse-1.1-i2v") {
		t.Fatal("image-to-video model should be supported")
	}
	if !isSupportedVideosModel("happyhorse-1.1-r2v") {
		t.Fatal("reference-to-video model should be supported")
	}

	const clientID = "happyhorse-video-model-test"
	registry.GetGlobalRegistry().RegisterClient(clientID, "bailian", []*registry.ModelInfo{{
		ID:   "happyhorse-t2v-alias",
		Type: registry.OpenAIVideoModelType,
	}})
	defer registry.GetGlobalRegistry().UnregisterClient(clientID)
	if !isSupportedVideosModel("happyhorse-t2v-alias") {
		t.Fatal("configured OpenAI video alias should be supported")
	}
	if !isSupportedVideosModel("bailian/happyhorse-t2v-alias") {
		t.Fatal("provider-prefixed OpenAI video alias should be supported")
	}
	if got := canonicalVideoBindingModel("happyhorse-t2v-alias"); got != "happyhorse-t2v-alias" {
		t.Fatalf("canonical video binding model = %q", got)
	}
}

func TestBuildHappyHorseCreateAPIResponse(t *testing.T) {
	payload := []byte("{\"output\":{\"task_id\":\"task-123\",\"task_status\":\"PENDING\"},\"request_id\":\"request-123\"}")
	request := []byte("{\"model\":\"happyhorse-1.1-t2v\",\"prompt\":\"a paper city\"}")
	out, err := buildHappyHorseCreateAPIResponse(payload, "happyhorse-1.1-t2v", request)
	if err != nil {
		t.Fatalf("buildHappyHorseCreateAPIResponse() error = %v", err)
	}
	if got := gjson.GetBytes(out, "id").String(); got != "task-123" {
		t.Fatalf("id = %q", got)
	}
	if got := gjson.GetBytes(out, "status").String(); got != "queued" {
		t.Fatalf("status = %q, want queued", got)
	}
	if got := gjson.GetBytes(out, "model").String(); got != "happyhorse-1.1-t2v" {
		t.Fatalf("model = %q", got)
	}
	if got := gjson.GetBytes(out, "seconds").String(); got != "5" {
		t.Fatalf("seconds = %q, want 5", got)
	}
}

func TestBuildHappyHorseRetrieveAPIResponse(t *testing.T) {
	payload := []byte("{\"output\":{\"task_id\":\"task-123\",\"task_status\":\"SUCCEEDED\",\"orig_prompt\":\"animate the references\",\"video_url\":\"https://example.com/video.mp4\"}}")
	out, err := buildHappyHorseRetrieveAPIResponse("task-123", payload, "happyhorse-1.1-r2v")
	if err != nil {
		t.Fatalf("buildHappyHorseRetrieveAPIResponse() error = %v", err)
	}
	if got := gjson.GetBytes(out, "status").String(); got != "completed" {
		t.Fatalf("status = %q, want completed", got)
	}
	if got := gjson.GetBytes(out, "progress").Int(); got != 100 {
		t.Fatalf("progress = %d, want 100", got)
	}
	if got := gjson.GetBytes(out, "prompt").String(); got != "animate the references" {
		t.Fatalf("prompt = %q", got)
	}
	if got := gjson.GetBytes(out, "video_url").String(); got != "https://example.com/video.mp4" {
		t.Fatalf("video URL = %q", got)
	}
}

func TestBuildHappyHorseRetrieveAPIResponseFailure(t *testing.T) {
	payload := []byte("{\"output\":{\"task_id\":\"task-123\",\"task_status\":\"FAILED\",\"code\":\"InvalidParameter\",\"message\":\"bad image\"}}")
	out, err := buildHappyHorseRetrieveAPIResponse("task-123", payload, "happyhorse-1.1-i2v")
	if err != nil {
		t.Fatalf("buildHappyHorseRetrieveAPIResponse() error = %v", err)
	}
	if got := gjson.GetBytes(out, "status").String(); got != "failed" {
		t.Fatalf("status = %q, want failed", got)
	}
	if got := gjson.GetBytes(out, "error.code").String(); got != "InvalidParameter" {
		t.Fatalf("error code = %q", got)
	}
	if got := gjson.GetBytes(out, "error.message").String(); got != "bad image" {
		t.Fatalf("error message = %q", got)
	}
}

func TestHappyHorseVideoContentURLFromPayload(t *testing.T) {
	got, err := happyHorseVideoContentURLFromPayload([]byte("{\"output\":{\"video_url\":\"https://example.com/video.mp4?Expires=1\"}}"))
	if err != nil {
		t.Fatalf("happyHorseVideoContentURLFromPayload() error = %v", err)
	}
	if got != "https://example.com/video.mp4?Expires=1" {
		t.Fatalf("video URL = %q", got)
	}
	if _, err := happyHorseVideoContentURLFromPayload([]byte("{\"output\":{\"video_url\":\"file:///tmp/video.mp4\"}}")); err == nil {
		t.Fatal("expected invalid URL error")
	}
}
