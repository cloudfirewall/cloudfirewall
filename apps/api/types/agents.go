package types

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
	Version        string `json:"version,omitempty"`
	NFTablesConfig string `json:"nftablesConfig"`
}

type UpdateFirewallConfigResponse struct {
	Version   string `json:"version"`
	UpdatedAt string `json:"updatedAt"`
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
