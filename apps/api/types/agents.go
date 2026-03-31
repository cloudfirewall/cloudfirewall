package types

import "github.com/cloudfirewall/cloudfirewall/apps/engine/policybuilder"

type EnrollAgentRequest struct {
	EnrollmentToken string `json:"enrollmentToken"`
	AgentName       string `json:"agentName"`
	Hostname        string `json:"hostname"`
	AgentVersion    string `json:"agentVersion"`
}

type CreateEnrollmentTokenRequest struct {
	TTLSeconds int `json:"ttlSeconds,omitempty"`
}

type CreateEnrollmentTokenResponse struct {
	Token     string `json:"token"`
	TokenID   string `json:"tokenId"`
	ExpiresAt string `json:"expiresAt"`
}

type UpdateFirewallConfigRequest struct {
	Name           string                     `json:"name,omitempty"`
	Version        string                     `json:"version,omitempty"`
	NFTablesConfig string                     `json:"nftablesConfig,omitempty"`
	Policy         *policybuilder.PolicyDraft `json:"policy,omitempty"`
}

type UpdateFirewallConfigResponse struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Version   string `json:"version"`
	UpdatedAt string `json:"updatedAt"`
}

type FirewallConfigSummary struct {
	ID             string                     `json:"id"`
	Name           string                     `json:"name"`
	Version        string                     `json:"version"`
	UpdatedAt      string                     `json:"updatedAt"`
	IsActive       bool                       `json:"isActive"`
	NFTablesConfig string                     `json:"nftablesConfig,omitempty"`
	Policy         *policybuilder.PolicyDraft `json:"policy,omitempty"`
}

type CreateFirewallConfigRequest struct {
	Name           string                     `json:"name"`
	Version        string                     `json:"version,omitempty"`
	NFTablesConfig string                     `json:"nftablesConfig,omitempty"`
	Policy         *policybuilder.PolicyDraft `json:"policy,omitempty"`
}

type CreateFirewallConfigResponse = FirewallConfigSummary

type GetFirewallConfigResponse = FirewallConfigSummary

type ListFirewallConfigsResponse struct {
	Configs []FirewallConfigSummary `json:"configs"`
}

type ApplyFirewallConfigResponse struct {
	Config FirewallConfigSummary `json:"config"`
}

type AdminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AdminLoginResponse struct {
	AuthToken string `json:"authToken"`
}

type EnrollAgentResponse struct {
	AgentID                   string `json:"agentId"`
	AuthToken                 string `json:"authToken"`
	HeartbeatIntervalSeconds  int    `json:"heartbeatIntervalSeconds"`
	ConfigPollIntervalSeconds int    `json:"configPollIntervalSeconds"`
}

type AgentHeartbeatRequest struct {
	Hostname        string `json:"hostname"`
	AgentVersion    string `json:"agentVersion"`
	FirewallVersion string `json:"firewallVersion"`
}

type AgentHeartbeatResponse struct {
	ReceivedAt string `json:"receivedAt"`
	Online     bool   `json:"online"`
}

type AgentConfigResponse struct {
	Version        string `json:"version"`
	NFTablesConfig string `json:"nftablesConfig"`
	UpdatedAt      string `json:"updatedAt"`
}

type AgentSummary struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Hostname        string `json:"hostname"`
	AgentVersion    string `json:"agentVersion"`
	FirewallVersion string `json:"firewallVersion"`
	EnrolledAt      string `json:"enrolledAt"`
	LastSeenAt      string `json:"lastSeenAt,omitempty"`
	Online          bool   `json:"online"`
}

type ListAgentsResponse struct {
	Agents []AgentSummary `json:"agents"`
}
