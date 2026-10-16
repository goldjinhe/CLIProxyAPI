package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBailianSubtitleExampleConfig(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "bailian-subtitles", "config.example.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example config: %v", err)
	}
	cfg, err := ParseConfigBytes(data)
	if err != nil {
		t.Fatalf("parse example config: %v", err)
	}
	if len(cfg.OpenAICompatibility) != 1 {
		t.Fatalf("providers = %d, want 1", len(cfg.OpenAICompatibility))
	}
	provider := cfg.OpenAICompatibility[0]
	if provider.Name != "bailian-subtitles" {
		t.Fatalf("provider name = %q", provider.Name)
	}
	if len(provider.Models) != 2 {
		t.Fatalf("models = %d, want 2", len(provider.Models))
	}
	aliases := map[string]bool{}
	for _, model := range provider.Models {
		if model.Name != "qwen-plus-2025-12-01" {
			t.Fatalf("upstream model = %q", model.Name)
		}
		aliases[model.Alias] = true
	}
	for _, alias := range []string{"drama-subtitle-semantic", "drama-subtitle-translation"} {
		if !aliases[alias] {
			t.Fatalf("missing alias %q", alias)
		}
	}
	if len(cfg.Payload.Override) != 1 {
		t.Fatalf("payload overrides = %d, want 1", len(cfg.Payload.Override))
	}
	if value, ok := cfg.Payload.Override[0].Params["enable_thinking"].(bool); !ok || value {
		t.Fatalf("enable_thinking override = %#v, want false", cfg.Payload.Override[0].Params["enable_thinking"])
	}
}
