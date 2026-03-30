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
	store := newTestStore(t,
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
	)
	server := httpapi.NewServer(store)

	tokenResp := doJSON[types.CreateEnrollmentTokenResponse](t, server, http.MethodPost, "/api/v1/enrollment-tokens", "", types.CreateEnrollmentTokenRequest{
		TTLSeconds: 300,
	}, http.StatusCreated, withAPIKey())

	enrollReq := types.EnrollAgentRequest{
		EnrollmentToken: tokenResp.Token,
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
	store := newTestStore(t,
		service.SecurityConfig{
			AdminUsername: "admin",
			AdminPassword: "secret",
			APIKey:        "dev-api-key",
		},
		service.FirewallConfig{},
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
	if _, ok := paths["/api/v1/enrollment-tokens"]; !ok {
		t.Fatalf("enrollment token path missing from spec")
	}
	if _, ok := paths["/api/v1/firewall-config"]; !ok {
		t.Fatalf("firewall config path missing from spec")
	}
}

func TestListAgentsAcceptsAPIKey(t *testing.T) {
	store := newTestStore(t,
		service.SecurityConfig{
			AdminUsername: "admin",
			AdminPassword: "secret",
			APIKey:        "dev-api-key",
		},
		service.FirewallConfig{},
	)
	server := httpapi.NewServer(store)

	doJSON[types.ListAgentsResponse](t, server, http.MethodGet, "/api/v1/agents", "", nil, http.StatusOK, withAPIKey())
}

func TestUpdateFirewallConfig(t *testing.T) {
	store := newTestStore(t,
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
	)
	server := httpapi.NewServer(store)

	updateResp := doJSON[types.UpdateFirewallConfigResponse](t, server, http.MethodPost, "/api/v1/firewall-config", "", types.UpdateFirewallConfigRequest{
		Version:        "cfg-2",
		NFTablesConfig: "table inet cloudfirewall { chain input { type filter hook input priority 0; policy accept; drop } }",
	}, http.StatusOK, withAPIKey())
	if updateResp.Version != "cfg-2" {
		t.Fatalf("unexpected updated version: %s", updateResp.Version)
	}

	tokenResp := doJSON[types.CreateEnrollmentTokenResponse](t, server, http.MethodPost, "/api/v1/enrollment-tokens", "", nil, http.StatusCreated, withAPIKey())
	enrollResp := doJSON[types.EnrollAgentResponse](t, server, http.MethodPost, "/api/v1/enroll", "", types.EnrollAgentRequest{
		EnrollmentToken: tokenResp.Token,
		AgentName:       "edge-01",
		Hostname:        "edge-01.local",
		AgentVersion:    "1.0.0",
	}, http.StatusCreated)

	configResp := doJSON[types.AgentConfigResponse](t, server, http.MethodGet, "/api/v1/agents/self/config", enrollResp.AuthToken, nil, http.StatusOK)
	if configResp.Version != "cfg-2" {
		t.Fatalf("unexpected config version after update: %s", configResp.Version)
	}
}

func TestEnrollmentTokenIsOneTimeUse(t *testing.T) {
	store := newTestStore(t,
		service.SecurityConfig{
			AdminUsername: "admin",
			AdminPassword: "secret",
			APIKey:        "dev-api-key",
		},
		service.FirewallConfig{},
	)
	server := httpapi.NewServer(store)

	tokenResp := doJSON[types.CreateEnrollmentTokenResponse](t, server, http.MethodPost, "/api/v1/enrollment-tokens", "", nil, http.StatusCreated, withAPIKey())

	doJSON[types.EnrollAgentResponse](t, server, http.MethodPost, "/api/v1/enroll", "", types.EnrollAgentRequest{
		EnrollmentToken: tokenResp.Token,
		AgentName:       "edge-01",
		Hostname:        "edge-01.local",
		AgentVersion:    "1.0.0",
	}, http.StatusCreated)

	doJSON[map[string]string](t, server, http.MethodPost, "/api/v1/enroll", "", types.EnrollAgentRequest{
		EnrollmentToken: tokenResp.Token,
		AgentName:       "edge-02",
		Hostname:        "edge-02.local",
		AgentVersion:    "1.0.0",
	}, http.StatusUnauthorized)
}

func TestAgentStatePersistsAcrossStoreRestart(t *testing.T) {
	dbPath := t.TempDir() + "/api.db"
	security := service.SecurityConfig{
		AdminUsername: "admin",
		AdminPassword: "secret",
		APIKey:        "dev-api-key",
	}
	config := service.FirewallConfig{
		Version:        "cfg-1",
		NFTablesConfig: "table inet cloudfirewall {}",
		UpdatedAt:      time.Unix(1700000000, 0).UTC(),
	}

	store, err := service.NewStore(security, config, dbPath, 30*time.Second, 10*time.Second, 15*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	server := httpapi.NewServer(store)

	tokenResp := doJSON[types.CreateEnrollmentTokenResponse](t, server, http.MethodPost, "/api/v1/enrollment-tokens", "", nil, http.StatusCreated, withAPIKey())
	enrollResp := doJSON[types.EnrollAgentResponse](t, server, http.MethodPost, "/api/v1/enroll", "", types.EnrollAgentRequest{
		EnrollmentToken: tokenResp.Token,
		AgentName:       "edge-01",
		Hostname:        "edge-01.local",
		AgentVersion:    "1.0.0",
	}, http.StatusCreated)
	doJSON[types.AgentHeartbeatResponse](t, server, http.MethodPost, "/api/v1/agents/self/heartbeat", enrollResp.AuthToken, types.AgentHeartbeatRequest{
		Hostname:        "edge-01.local",
		AgentVersion:    "1.0.0",
		FirewallVersion: "cfg-1",
	}, http.StatusOK)

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	restartedStore, err := service.NewStore(security, service.FirewallConfig{
		Version:        "cfg-bootstrap",
		NFTablesConfig: "table inet cloudfirewall { chain input { type filter hook input priority 0; policy drop; } }",
		UpdatedAt:      time.Unix(1700000100, 0).UTC(),
	}, dbPath, 30*time.Second, 10*time.Second, 15*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := restartedStore.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	restartedServer := httpapi.NewServer(restartedStore)
	listResp := doJSON[types.ListAgentsResponse](t, restartedServer, http.MethodGet, "/api/v1/agents", "", nil, http.StatusOK, withAPIKey())
	if len(listResp.Agents) != 1 {
		t.Fatalf("expected 1 persisted agent, got %d", len(listResp.Agents))
	}
	if listResp.Agents[0].Name != "edge-01" {
		t.Fatalf("unexpected persisted agent name: %s", listResp.Agents[0].Name)
	}

	configResp := doJSON[types.AgentConfigResponse](t, restartedServer, http.MethodGet, "/api/v1/agents/self/config", enrollResp.AuthToken, nil, http.StatusOK)
	if configResp.Version != "cfg-1" {
		t.Fatalf("expected persisted config version, got %s", configResp.Version)
	}
}

func newTestStore(t *testing.T, security service.SecurityConfig, config service.FirewallConfig) *service.Store {
	t.Helper()

	store, err := service.NewStore(security, config, t.TempDir()+"/api.db", 30*time.Second, 10*time.Second, 15*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
	})

	return store
}

type requestOption func(*http.Request)

func withAPIKey() requestOption {
	return func(req *http.Request) {
		req.Header.Set("X-API-Key", "dev-api-key")
	}
}

func doJSON[T any](t *testing.T, handler http.Handler, method, path, authToken string, payload any, wantStatus int, opts ...requestOption) T {
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
	for _, opt := range opts {
		opt(req)
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
