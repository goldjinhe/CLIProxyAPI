package openai

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor/helps"
	apihandlers "github.com/router-for-me/CLIProxyAPI/v7/sdk/api/handlers"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	coreexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
	"github.com/tidwall/gjson"
)

type happyHorseVideoCaptureExecutor struct {
	mu      sync.Mutex
	authIDs []string
	alts    []string
}

func (e *happyHorseVideoCaptureExecutor) Identifier() string { return "bailian" }

func (e *happyHorseVideoCaptureExecutor) Execute(_ context.Context, auth *coreauth.Auth, req coreexecutor.Request, opts coreexecutor.Options) (coreexecutor.Response, error) {
	authID := ""
	if auth != nil {
		authID = auth.ID
	}
	e.mu.Lock()
	e.authIDs = append(e.authIDs, authID)
	e.alts = append(e.alts, opts.Alt)
	e.mu.Unlock()
	if opts.Alt == happyHorseRetrieveOperation {
		return coreexecutor.Response{Payload: []byte("{\"output\":{\"task_id\":\"hh-task\",\"task_status\":\"SUCCEEDED\",\"video_url\":\"https://example.com/video.mp4\"}}")}, nil
	}
	return coreexecutor.Response{Payload: []byte("{\"output\":{\"task_id\":\"hh-task\",\"task_status\":\"PENDING\"},\"request_id\":\"hh-request\"}")}, nil
}

func (e *happyHorseVideoCaptureExecutor) ExecuteStream(context.Context, *coreauth.Auth, coreexecutor.Request, coreexecutor.Options) (*coreexecutor.StreamResult, error) {
	return nil, &coreauth.Error{Code: "not_implemented", Message: "ExecuteStream not implemented"}
}

func (e *happyHorseVideoCaptureExecutor) Refresh(_ context.Context, auth *coreauth.Auth) (*coreauth.Auth, error) {
	return auth, nil
}

func (e *happyHorseVideoCaptureExecutor) CountTokens(context.Context, *coreauth.Auth, coreexecutor.Request, coreexecutor.Options) (coreexecutor.Response, error) {
	return coreexecutor.Response{}, &coreauth.Error{Code: "not_implemented", Message: "CountTokens not implemented"}
}

func (e *happyHorseVideoCaptureExecutor) HttpRequest(context.Context, *coreauth.Auth, *http.Request) (*http.Response, error) {
	return nil, &coreauth.Error{Code: "not_implemented", Message: "HttpRequest not implemented"}
}

func (e *happyHorseVideoCaptureExecutor) calls() (authIDs, alts []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.authIDs...), append([]string(nil), e.alts...)
}

func TestHappyHorseVideosCreateAndRetrieveUseBoundAuth(t *testing.T) {
	resetVideoAuthBindingsForTest(t)
	executor := &happyHorseVideoCaptureExecutor{}
	manager := coreauth.NewManager(nil, &coreauth.RoundRobinSelector{}, nil)
	manager.RegisterExecutor(executor)
	auth := &coreauth.Auth{ID: "happyhorse-auth", Provider: "bailian", Status: coreauth.StatusActive}
	if _, err := manager.Register(context.Background(), auth); err != nil {
		t.Fatalf("manager.Register() error = %v", err)
	}
	registry.GetGlobalRegistry().RegisterClient(auth.ID, auth.Provider, []*registry.ModelInfo{{
		ID:   helps.HappyHorseT2VModel,
		Type: registry.OpenAIVideoModelType,
	}})
	t.Cleanup(func() {
		registry.GetGlobalRegistry().UnregisterClient(auth.ID)
	})

	handler := NewOpenAIAPIHandler(apihandlers.NewBaseAPIHandlers(&sdkconfig.SDKConfig{}, manager))
	createResp := performVideosEndpointRequest(t, http.MethodPost, openAIVideosPath, "application/json", strings.NewReader("{\"model\":\"happyhorse-1.1-t2v\",\"prompt\":\"make a video\"}"), handler.VideosCreate)
	if createResp.Code != http.StatusOK {
		t.Fatalf("create status = %d, want %d: %s", createResp.Code, http.StatusOK, createResp.Body.String())
	}
	if got := gjson.GetBytes(createResp.Body.Bytes(), "id").String(); got != "hh-task" {
		t.Fatalf("create ID = %q", got)
	}
	if got := gjson.GetBytes(createResp.Body.Bytes(), "status").String(); got != "queued" {
		t.Fatalf("create status = %q", got)
	}

	retrieveResp := performVideosRouteRequest(t, http.MethodGet, openAIVideosPath+"/:video_id", openAIVideosPath+"/hh-task", "", nil, handler.VideosRetrieve)
	if retrieveResp.Code != http.StatusOK {
		t.Fatalf("retrieve status = %d, want %d: %s", retrieveResp.Code, http.StatusOK, retrieveResp.Body.String())
	}
	if got := gjson.GetBytes(retrieveResp.Body.Bytes(), "status").String(); got != "completed" {
		t.Fatalf("retrieve status = %q", got)
	}

	authIDs, alts := executor.calls()
	if len(authIDs) != 2 || authIDs[0] != auth.ID || authIDs[1] != auth.ID {
		t.Fatalf("auth IDs = %v, want two calls with %s", authIDs, auth.ID)
	}
	if len(alts) != 2 || alts[0] != happyHorseCreateOperation || alts[1] != happyHorseRetrieveOperation {
		t.Fatalf("operations = %v, want create/retrieve", alts)
	}
}
