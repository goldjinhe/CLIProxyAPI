package helps

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestBuildHappyHorseVideoPayloadTextToVideo(t *testing.T) {
	payload := []byte(`{"model":"happyhorse-1.1-t2v","prompt":"a paper city at night","size":"1280x720","duration":7}`)
	out, err := BuildHappyHorseVideoPayload(payload, HappyHorseT2VModel)
	if err != nil {
		t.Fatalf("BuildHappyHorseVideoPayload() error = %v", err)
	}
	if got := gjson.GetBytes(out, "model").String(); got != HappyHorseT2VModel {
		t.Fatalf("model = %q, want %q", got, HappyHorseT2VModel)
	}
	if got := gjson.GetBytes(out, "input.prompt").String(); got != "a paper city at night" {
		t.Fatalf("prompt = %q", got)
	}
	if got := gjson.GetBytes(out, "parameters.resolution").String(); got != "720P" {
		t.Fatalf("resolution = %q, want 720P", got)
	}
	if got := gjson.GetBytes(out, "parameters.ratio").String(); got != "16:9" {
		t.Fatalf("ratio = %q, want 16:9", got)
	}
	if got := gjson.GetBytes(out, "parameters.duration").Int(); got != 7 {
		t.Fatalf("duration = %d, want 7", got)
	}
}

func TestBuildHappyHorseVideoPayloadImageToVideo(t *testing.T) {
	payload := []byte(`{"prompt":"make it move","image_url":"data:image/png;base64,AAAA","resolution":"720P","duration":5}`)
	out, err := BuildHappyHorseVideoPayload(payload, HappyHorseI2VModel)
	if err != nil {
		t.Fatalf("BuildHappyHorseVideoPayload() error = %v", err)
	}
	if got := gjson.GetBytes(out, "input.media.0.type").String(); got != "first_frame" {
		t.Fatalf("media type = %q", got)
	}
	if got := gjson.GetBytes(out, "input.media.0.url").String(); got != "data:image/png;base64,AAAA" {
		t.Fatalf("media URL = %q", got)
	}
	if gjson.GetBytes(out, "parameters.ratio").Exists() {
		t.Fatal("image-to-video payload should not include ratio")
	}
}

func TestBuildHappyHorseVideoPayloadReferenceToVideo(t *testing.T) {
	payload := []byte(`{"prompt":"animate [Image 1] and [Image 2]","reference_images":[{"image_url":"https://example.com/one.png"},"https://example.com/two.png"]}`)
	out, err := BuildHappyHorseVideoPayload(payload, HappyHorseR2VModel)
	if err != nil {
		t.Fatalf("BuildHappyHorseVideoPayload() error = %v", err)
	}
	if got := gjson.GetBytes(out, "input.media.#").Int(); got != 2 {
		t.Fatalf("media count = %d, want 2", got)
	}
	if got := gjson.GetBytes(out, "input.media.0.type").String(); got != "reference_image" {
		t.Fatalf("first media type = %q", got)
	}
	if got := gjson.GetBytes(out, "input.media.1.url").String(); got != "https://example.com/two.png" {
		t.Fatalf("second media URL = %q", got)
	}
}

func TestBuildHappyHorseVideoPayloadRejectsInvalidModeInputs(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		payload string
	}{
		{"text-to-video image", HappyHorseT2VModel, "{\"prompt\":\"x\",\"image_url\":\"https://example.com/a.png\"}"},
		{"image-to-video missing image", HappyHorseI2VModel, "{\"prompt\":\"x\"}"},
		{"reference-to-video missing prompt", HappyHorseR2VModel, "{\"reference_images\":[\"https://example.com/a.png\"]}"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := BuildHappyHorseVideoPayload([]byte(testCase.payload), testCase.model); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestHappyHorseTaskHelpers(t *testing.T) {
	payload := []byte(`{"output":{"task_id":"task-123","task_status":"SUCCEEDED","video_url":"https://example.com/video.mp4"}}`)
	if got := HappyHorseTaskID(payload); got != "task-123" {
		t.Fatalf("task ID = %q", got)
	}
	if got := HappyHorseTaskStatus(payload); got != "SUCCEEDED" {
		t.Fatalf("task status = %q", got)
	}
	if got := HappyHorseVideoURL(payload); got != "https://example.com/video.mp4" {
		t.Fatalf("video URL = %q", got)
	}
}
