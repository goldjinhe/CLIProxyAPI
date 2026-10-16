package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestModelMatchesAccessPatterns(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		patterns []string
		want     bool
	}{
		{name: "glob", model: "glm-5.3-flash", patterns: []string{"glm-*"}, want: true},
		{name: "case insensitive", model: "GLM-5", patterns: []string{"glm-*"}, want: true},
		{name: "reasoning suffix", model: "glm-5.3-flash(low)", patterns: []string{"glm-*"}, want: true},
		{name: "provider prefix", model: "zhipu/glm-5.3", patterns: []string{"glm-*"}, want: true},
		{name: "gemini resource prefix", model: "models/glm-5", patterns: []string{"glm-*"}, want: true},
		{name: "denied", model: "gpt-5.4", patterns: []string{"glm-*"}, want: false},
		{name: "missing model", model: "", patterns: []string{"glm-*"}, want: false},
		{name: "invalid pattern", model: "glm-5", patterns: []string{"["}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := modelMatchesAccessPatterns(tt.model, tt.patterns); got != tt.want {
				t.Fatalf("modelMatchesAccessPatterns(%q, %#v) = %t, want %t", tt.model, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestValidateModelAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	SetAccessAllowedModels(c, []string{"glm-*"})
	ctx := context.WithValue(context.Background(), "gin", c)

	if errMsg := validateModelAccess(ctx, "glm-5.3"); errMsg != nil {
		t.Fatalf("validateModelAccess() allowed error = %v", errMsg.Error)
	}
	errMsg := validateModelAccess(ctx, "gpt-5.4")
	if errMsg == nil || errMsg.StatusCode != http.StatusForbidden {
		t.Fatalf("validateModelAccess() denied = %#v, want HTTP 403", errMsg)
	}
	if errMsg := validateModelAccess(context.Background(), "gpt-5.4"); errMsg != nil {
		t.Fatalf("validateModelAccess() unrestricted error = %v", errMsg.Error)
	}
}

func TestFilterModelsForAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	SetAccessAllowedModels(c, []string{"glm-*"})
	models := []map[string]any{
		{"id": "glm-5.3"},
		{"id": "gpt-5.4"},
		{"name": "models/glm-4.7"},
	}

	filtered := FilterModelsForAccess(c, models, "id", "name")
	want := []map[string]any{{"id": "glm-5.3"}, {"name": "models/glm-4.7"}}
	if !reflect.DeepEqual(filtered, want) {
		t.Fatalf("FilterModelsForAccess() = %#v, want %#v", filtered, want)
	}
}

func TestExecutionPathsRejectDisallowedModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	SetAccessAllowedModels(c, []string{"glm-*"})
	ctx := context.WithValue(context.Background(), "gin", c)
	handler := &BaseAPIHandler{}

	_, _, errMsg := handler.ExecuteWithAuthManager(ctx, "openai", "gpt-5.4", nil, "")
	if errMsg == nil || errMsg.StatusCode != http.StatusForbidden {
		t.Fatalf("ExecuteWithAuthManager() error = %#v, want HTTP 403", errMsg)
	}

	_, _, errMsg = handler.ExecuteCountWithAuthManager(ctx, "claude", "claude-opus-4-6", nil, "")
	if errMsg == nil || errMsg.StatusCode != http.StatusForbidden {
		t.Fatalf("ExecuteCountWithAuthManager() error = %#v, want HTTP 403", errMsg)
	}

	data, _, errChan := handler.executeStreamWithAuthManager(ctx, "openai", "gemini-3-pro", nil, "", false)
	if data != nil {
		t.Fatalf("executeStreamWithAuthManager() data channel = %#v, want nil", data)
	}
	errMsg = <-errChan
	if errMsg == nil || errMsg.StatusCode != http.StatusForbidden {
		t.Fatalf("executeStreamWithAuthManager() error = %#v, want HTTP 403", errMsg)
	}
}
