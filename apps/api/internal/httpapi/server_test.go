package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudfirewall/cloudfirewall/apps/api/internal/httpapi"
	"github.com/cloudfirewall/cloudfirewall/apps/api/internal/service"
	"github.com/cloudfirewall/cloudfirewall/apps/api/types"
)

func TestEnrollHeartbeatListAndConfig(t *testing.T) {
	store := service.NewStore(
		[]string{"dev-enrollment-token"},
		service.SecurityConfig{
			AdminUsername: "admin",
			AdminPassword: "secret",
			APIKey:        "dev-api-key",
		},
		service.FirewallConfig{
			Version:        "cfg-1",
			NFTablesConfig: "table inet cloudfirewall {}",
			UpdatedAt:      time.Unix(1700000000, 0).UTC(),
		},
		30*time.Second,
		10*time.Second,
		15*time.Second,
	)
	server := httpapi.NewServer(store)

	enrollReq := types.EnrollAgentRequest{
		EnrollmentToken: "dev-enrollment-token",
		AgentName:       "edge-01",
		Hostname:        "edge-01.local",
		AgentVersion:    "1.0.0",
	}
	enrollResp := doJSON[types.EnrollAgentResponse](t, server, http.MethodPost, "/api/v1/enroll", "", enrollReq, http.StatusCreated)
	if enrollResp.AgentID == "" || enrollResp.AuthToken == "" {
		t.Fatalf("expected enrollment identifiers, got %#v", enrollResp)
	}

	heartbeatReq := types.AgentHeartbeatRequest{
		Hostname:        "edge-01.local",
		AgentVersion:    "1.0.0",
		FirewallVersion: "cfg-1",
	}
	doJSON[types.AgentHeartbeatResponse](t, server, http.MethodPost, "/api/v1/agents/self/heartbeat", enrollResp.AuthToken, heartbeatReq, http.StatusOK)

	configResp := doJSON[types.AgentConfigResponse](t, server, http.MethodGet, "/api/v1/agents/self/config", enrollResp.AuthToken, nil, http.StatusOK)
	if configResp.Version != "cfg-1" {
		t.Fatalf("unexpected config version: %s", configResp.Version)
	}

	loginResp := doJSON[types.AdminLoginResponse](t, server, http.MethodPost, "/api/v1/admin/login", "", types.AdminLoginRequest{
		Username: "admin",
		Password: "secret",
	}, http.StatusOK)

	listResp := doJSON[types.ListAgentsResponse](t, server, http.MethodGet, "/api/v1/agents", loginResp.AuthToken, nil, http.StatusOK)
	if len(listResp.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(listResp.Agents))
	}
	if !listResp.Agents[0].Online {
		t.Fatalf("expected agent to be online")
	}
	if listResp.Agents[0].FirewallVersion != "cfg-1" {
		t.Fatalf("unexpected firewall version: %s", listResp.Agents[0].FirewallVersion)
	}
}

func TestSwaggerAndOpenAPIEndpoints(t *testing.T) {
	store := service.NewStore(
		[]string{"dev-enrollment-token"},
		service.SecurityConfig{
			AdminUsername: "admin",
			AdminPassword: "secret",
			APIKey:        "dev-api-key",
		},
		service.FirewallConfig{},
		30*time.Second,
		10*time.Second,
		15*time.Second,
	)
	server := httpapi.NewServer(store)

	swaggerReq, err := http.NewRequest(http.MethodGet, "/swagger", nil)
	if err != nil {
		t.Fatal(err)
	}
	swaggerRecorder := httptest.NewRecorder()
	server.ServeHTTP(swaggerRecorder, swaggerReq)
	if swaggerRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected swagger status %d", swaggerRecorder.Code)
	}
	if got := swaggerRecorder.Body.String(); !bytes.Contains([]byte(got), []byte("SwaggerUIBundle")) {
		t.Fatalf("swagger html did not include SwaggerUI bootstrap")
	}

	spec := doJSON[map[string]any](t, server, http.MethodGet, "/openapi.json", "", nil, http.StatusOK)
	if spec["openapi"] != "3.0.3" {
		t.Fatalf("unexpected openapi version: %#v", spec["openapi"])
	}
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths missing from spec")
	}
	if _, ok := paths["/api/v1/enroll"]; !ok {
		t.Fatalf("enroll path missing from spec")
	}
	if _, ok := paths["/api/v1/admin/login"]; !ok {
		t.Fatalf("admin login path missing from spec")
	}
}

func TestListAgentsAcceptsAPIKey(t *testing.T) {
	store := service.NewStore(
		[]string{"dev-enrollment-token"},
		service.SecurityConfig{
			AdminUsername: "admin",
			AdminPassword: "secret",
			APIKey:        "dev-api-key",
		},
		service.FirewallConfig{},
		30*time.Second,
		10*time.Second,
		15*time.Second,
	)
	server := httpapi.NewServer(store)

	req, err := http.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-API-Key", "dev-api-key")

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", recorder.Code)
	}
}

func doJSON[T any](t *testing.T, handler http.Handler, method, path, authToken string, payload any, wantStatus int) T {
	t.Helper()

	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
	}

	req, err := http.NewRequest(method, path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != wantStatus {
		t.Fatalf("unexpected status %d", recorder.Code)
	}

	var decoded T
	if err := json.NewDecoder(recorder.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}
