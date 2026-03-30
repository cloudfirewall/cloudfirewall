package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cloudfirewall/cloudfirewall/apps/api/types"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	authToken  string
}

func New(baseURL string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{},
	}
}

func (c *Client) Enroll(ctx context.Context, req types.EnrollAgentRequest) (types.EnrollAgentResponse, error) {
	resp, err := c.doJSON(ctx, http.MethodPost, "/api/v1/enroll", "", req)
	if err != nil {
		return types.EnrollAgentResponse{}, err
	}

	var decoded types.EnrollAgentResponse
	if err := json.Unmarshal(resp, &decoded); err != nil {
		return types.EnrollAgentResponse{}, err
	}

	c.authToken = decoded.AuthToken
	return decoded, nil
}

func (c *Client) Heartbeat(ctx context.Context, req types.AgentHeartbeatRequest) (types.AgentHeartbeatResponse, error) {
	resp, err := c.doJSON(ctx, http.MethodPost, "/api/v1/agents/self/heartbeat", c.authToken, req)
	if err != nil {
		return types.AgentHeartbeatResponse{}, err
	}

	var decoded types.AgentHeartbeatResponse
	if err := json.Unmarshal(resp, &decoded); err != nil {
		return types.AgentHeartbeatResponse{}, err
	}
	return decoded, nil
}

func (c *Client) Config(ctx context.Context) (types.AgentConfigResponse, error) {
	resp, err := c.doJSON(ctx, http.MethodGet, "/api/v1/agents/self/config", c.authToken, nil)
	if err != nil {
		return types.AgentConfigResponse{}, err
	}

	var decoded types.AgentConfigResponse
	if err := json.Unmarshal(resp, &decoded); err != nil {
		return types.AgentConfigResponse{}, err
	}
	return decoded, nil
}

func (c *Client) doJSON(ctx context.Context, method, path, authToken string, payload any) ([]byte, error) {
	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var apiErr map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr["error"] != "" {
			return nil, fmt.Errorf("%s", apiErr["error"])
		}
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
