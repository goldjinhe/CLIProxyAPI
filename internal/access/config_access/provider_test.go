package configaccess

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

func TestProviderAuthenticatesRestrictedRuleKey(t *testing.T) {
	rules := normalizeRules([]config.APIKeyRule{{
		APIKey:        " restricted-key ",
		AllowedModels: []string{" glm-* ", "glm-5", "glm-5"},
	}})
	provider := newProvider("test", nil, rules)
	req, errRequest := http.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", nil)
	if errRequest != nil {
		t.Fatalf("NewRequestWithContext() error = %v", errRequest)
	}
	req.Header.Set("Authorization", "Bearer restricted-key")

	result, authErr := provider.Authenticate(context.Background(), req)
	if authErr != nil {
		t.Fatalf("Authenticate() error = %v", authErr)
	}
	if result == nil {
		t.Fatal("Authenticate() result = nil")
	}
	if result.Principal != "restricted-key" {
		t.Fatalf("Principal = %q, want restricted-key", result.Principal)
	}
	if want := []string{"glm-*", "glm-5"}; !reflect.DeepEqual(result.AllowedModels, want) {
		t.Fatalf("AllowedModels = %#v, want %#v", result.AllowedModels, want)
	}
}

func TestProviderKeepsTopLevelAPIKeyUnrestricted(t *testing.T) {
	rules := normalizeRules([]config.APIKeyRule{{
		APIKey:        "shared-key",
		AllowedModels: []string{"glm-*"},
	}})
	provider := newProvider("test", []string{"shared-key"}, rules)
	req, errRequest := http.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/chat/completions", nil)
	if errRequest != nil {
		t.Fatalf("NewRequestWithContext() error = %v", errRequest)
	}
	req.Header.Set("X-Api-Key", "shared-key")

	result, authErr := provider.Authenticate(context.Background(), req)
	if authErr != nil {
		t.Fatalf("Authenticate() error = %v", authErr)
	}
	if result == nil {
		t.Fatal("Authenticate() result = nil")
	}
	if result.AllowedModels != nil {
		t.Fatalf("AllowedModels = %#v, want nil for unrestricted key", result.AllowedModels)
	}
}

func TestProviderPreservesEmptyRestrictedRule(t *testing.T) {
	provider := newProvider("test", nil, normalizeRules([]config.APIKeyRule{{APIKey: "deny-all"}}))
	req, errRequest := http.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/models", nil)
	if errRequest != nil {
		t.Fatalf("NewRequestWithContext() error = %v", errRequest)
	}
	req.Header.Set("Authorization", "Bearer deny-all")

	result, authErr := provider.Authenticate(context.Background(), req)
	if authErr != nil {
		t.Fatalf("Authenticate() error = %v", authErr)
	}
	if result.AllowedModels == nil || len(result.AllowedModels) != 0 {
		t.Fatalf("AllowedModels = %#v, want non-nil empty slice", result.AllowedModels)
	}
}
