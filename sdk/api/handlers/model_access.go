package handlers

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/interfaces"
)

const accessAllowedModelsContextKey = "accessAllowedModels"

// SetAccessAllowedModels stores the model patterns granted to the authenticated client.
// A non-nil empty slice represents a credential that cannot access any model.
func SetAccessAllowedModels(c *gin.Context, patterns []string) {
	if c == nil || patterns == nil {
		return
	}
	cloned := make([]string, len(patterns))
	copy(cloned, patterns)
	c.Set(accessAllowedModelsContextKey, cloned)
}

// IsModelAllowedForAccess reports whether a model is visible to the authenticated client.
func IsModelAllowedForAccess(c *gin.Context, model string) bool {
	patterns, restricted := accessAllowedModelsFromGin(c)
	return !restricted || modelMatchesAccessPatterns(model, patterns)
}

// FilterModelsForAccess filters model metadata using the authenticated client's model rules.
// The first matching string field in modelFields is treated as the model identifier.
func FilterModelsForAccess(c *gin.Context, models []map[string]any, modelFields ...string) []map[string]any {
	patterns, restricted := accessAllowedModelsFromGin(c)
	if !restricted {
		return models
	}
	filtered := make([]map[string]any, 0, len(models))
	for _, model := range models {
		modelName := ""
		for _, field := range modelFields {
			if value, ok := model[field].(string); ok && strings.TrimSpace(value) != "" {
				modelName = value
				break
			}
		}
		if modelMatchesAccessPatterns(modelName, patterns) {
			filtered = append(filtered, model)
		}
	}
	return filtered
}

func validateModelAccess(ctx context.Context, model string) *interfaces.ErrorMessage {
	patterns, restricted := accessAllowedModelsFromContext(ctx)
	if !restricted || modelMatchesAccessPatterns(model, patterns) {
		return nil
	}
	modelName := strings.TrimSpace(model)
	if modelName == "" {
		modelName = "<unspecified>"
	}
	return &interfaces.ErrorMessage{
		StatusCode: http.StatusForbidden,
		Error:      fmt.Errorf("model %q is not allowed for this API key", modelName),
	}
}

func accessAllowedModelsFromContext(ctx context.Context) ([]string, bool) {
	if ctx == nil {
		return nil, false
	}
	c, ok := ctx.Value("gin").(*gin.Context)
	if !ok {
		return nil, false
	}
	return accessAllowedModelsFromGin(c)
}

func accessAllowedModelsFromGin(c *gin.Context) ([]string, bool) {
	if c == nil {
		return nil, false
	}
	value, exists := c.Get(accessAllowedModelsContextKey)
	if !exists {
		return nil, false
	}
	patterns, ok := value.([]string)
	return patterns, ok
}

func modelMatchesAccessPatterns(model string, patterns []string) bool {
	candidates := modelAccessCandidates(model)
	if len(candidates) == 0 {
		return false
	}
	for _, rawPattern := range patterns {
		pattern := strings.ToLower(strings.TrimSpace(rawPattern))
		if pattern == "" {
			continue
		}
		for _, candidate := range candidates {
			matched, errMatch := path.Match(pattern, candidate)
			if errMatch == nil && matched {
				return true
			}
		}
	}
	return false
}

func modelAccessCandidates(model string) []string {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		return nil
	}
	candidates := make([]string, 0, 4)
	appendCandidate := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		for _, existing := range candidates {
			if existing == candidate {
				return
			}
		}
		candidates = append(candidates, candidate)
	}
	appendCandidate(model)
	model = strings.TrimPrefix(model, "models/")
	appendCandidate(model)
	if open := strings.LastIndex(model, "("); open >= 0 && strings.HasSuffix(model, ")") {
		model = model[:open]
		appendCandidate(model)
	}
	if slash := strings.LastIndex(model, "/"); slash >= 0 && slash+1 < len(model) {
		appendCandidate(model[slash+1:])
	}
	return candidates
}
