package config

import (
	"reflect"
	"testing"
)

func TestParseConfigBytesAPIKeyRules(t *testing.T) {
	cfg, errParse := ParseConfigBytes([]byte(`api-keys:
  - unrestricted-key
api-key-rules:
  - api-key: glm-key
    allowed-models:
      - glm-*
      - glm-5.3-flash
`))
	if errParse != nil {
		t.Fatalf("ParseConfigBytes() error = %v", errParse)
	}
	if len(cfg.APIKeyRules) != 1 {
		t.Fatalf("APIKeyRules length = %d, want 1", len(cfg.APIKeyRules))
	}
	if want := []string{"glm-*", "glm-5.3-flash"}; !reflect.DeepEqual(cfg.APIKeyRules[0].AllowedModels, want) {
		t.Fatalf("AllowedModels = %#v, want %#v", cfg.APIKeyRules[0].AllowedModels, want)
	}
	if want := []string{"unrestricted-key", "glm-key"}; !reflect.DeepEqual(cfg.ClientAPIKeys(), want) {
		t.Fatalf("ClientAPIKeys() = %#v, want %#v", cfg.ClientAPIKeys(), want)
	}
}
