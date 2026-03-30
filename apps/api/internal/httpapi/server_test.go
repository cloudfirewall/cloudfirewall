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

	listResp := doJSON[types.ListAgentsResponse](t, server, http.MethodGet, "/api/v1/agents", "", nil, http.StatusOK)
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
