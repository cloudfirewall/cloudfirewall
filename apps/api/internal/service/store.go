package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudfirewall/cloudfirewall/apps/api/types"
	bolt "go.etcd.io/bbolt"
)

var (
	ErrInvalidEnrollmentToken = errors.New("invalid enrollment token")
	ErrUnauthorized           = errors.New("unauthorized")
)

type SecurityConfig struct {
	AdminUsername string
	AdminPassword string
	APIKey        string
}

type FirewallConfig struct {
	Version        string
	NFTablesConfig string
	UpdatedAt      time.Time
}

type AgentRecord struct {
	ID              string
	AuthToken       string
	Name            string
	Hostname        string
	AgentVersion    string
	FirewallVersion string
	EnrolledAt      time.Time
	LastSeenAt      time.Time
}

type EnrollmentTokenClaims struct {
	ID  string `json:"id"`
	Exp int64  `json:"exp"`
	Iat int64  `json:"iat"`
}

type EnrollmentTokenRecord struct {
	ID        string
	IssuedAt  time.Time
	ExpiresAt time.Time
	UsedAt    time.Time
}

type Store struct {
	mu                 sync.RWMutex
	db                 *bolt.DB
	enrollmentTokens   map[string]*EnrollmentTokenRecord
	agents             map[string]*AgentRecord
	agentIDsByToken    map[string]string
	adminSessions      map[string]time.Time
	firewallConfig     FirewallConfig
	security           SecurityConfig
	heartbeatTimeout   time.Duration
	heartbeatInterval  time.Duration
	configPollInterval time.Duration
}

func NewStore(security SecurityConfig, config FirewallConfig, dbPath string, heartbeatTimeout, heartbeatInterval, configPollInterval time.Duration) (*Store, error) {
	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}

	store := &Store{
		db:                 db,
		enrollmentTokens:   make(map[string]*EnrollmentTokenRecord),
		agents:             make(map[string]*AgentRecord),
		agentIDsByToken:    make(map[string]string),
		adminSessions:      make(map[string]time.Time),
		firewallConfig:     config,
		security:           security,
		heartbeatTimeout:   heartbeatTimeout,
		heartbeatInterval:  heartbeatInterval,
		configPollInterval: configPollInterval,
	}

	if err := store.loadPersistedState(config); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) CreateEnrollmentToken(req types.CreateEnrollmentTokenRequest) (types.CreateEnrollmentTokenResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ttl := time.Duration(req.TTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	if ttl > 24*time.Hour {
		ttl = 24 * time.Hour
	}

	now := time.Now().UTC()
	record := &EnrollmentTokenRecord{
		ID:        "enr_" + randomHex(8),
		IssuedAt:  now,
		ExpiresAt: now.Add(ttl),
	}
	s.enrollmentTokens[record.ID] = record
	if err := s.saveEnrollmentToken(record); err != nil {
		return types.CreateEnrollmentTokenResponse{}, err
	}

	token, err := s.signEnrollmentToken(EnrollmentTokenClaims{
		ID:  record.ID,
		Exp: record.ExpiresAt.Unix(),
		Iat: record.IssuedAt.Unix(),
	})
	if err != nil {
		return types.CreateEnrollmentTokenResponse{}, err
	}

	return types.CreateEnrollmentTokenResponse{
		Token:     token,
		TokenID:   record.ID,
		ExpiresAt: record.ExpiresAt.Format(time.RFC3339),
	}, nil
}

func (s *Store) AdminLogin(req types.AdminLoginRequest) (types.AdminLoginResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(req.Username) != s.security.AdminUsername || req.Password != s.security.AdminPassword {
		return types.AdminLoginResponse{}, ErrUnauthorized
	}

	token := "adm_" + randomHex(24)
	s.adminSessions[token] = time.Now().UTC()
	return types.AdminLoginResponse{AuthToken: token}, nil
}

func (s *Store) Enroll(req types.EnrollAgentRequest) (types.EnrollAgentResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	claims, err := s.verifyEnrollmentToken(strings.TrimSpace(req.EnrollmentToken))
	if err != nil {
		return types.EnrollAgentResponse{}, ErrInvalidEnrollmentToken
	}

	tokenRecord, ok := s.enrollmentTokens[claims.ID]
	if !ok || tokenRecord.ExpiresAt.Before(now) || !tokenRecord.UsedAt.IsZero() {
		return types.EnrollAgentResponse{}, ErrInvalidEnrollmentToken
	}
	tokenRecord.UsedAt = now
	if err := s.saveEnrollmentToken(tokenRecord); err != nil {
		return types.EnrollAgentResponse{}, err
	}

	agentID := "agt-" + randomHex(8)
	authToken := "cfw_" + randomHex(24)
	name := strings.TrimSpace(req.AgentName)
	if name == "" {
		name = strings.TrimSpace(req.Hostname)
	}
	if name == "" {
		name = agentID
	}

	agentRecord := &AgentRecord{
		ID:           agentID,
		AuthToken:    authToken,
		Name:         name,
		Hostname:     strings.TrimSpace(req.Hostname),
		AgentVersion: strings.TrimSpace(req.AgentVersion),
		EnrolledAt:   now,
	}

	s.agents[agentID] = agentRecord
	s.agentIDsByToken[authToken] = agentID
	if err := s.saveAgent(agentRecord); err != nil {
		return types.EnrollAgentResponse{}, err
	}

	return types.EnrollAgentResponse{
		AgentID:                   agentID,
		AuthToken:                 authToken,
		HeartbeatIntervalSeconds:  int(s.heartbeatInterval.Seconds()),
		ConfigPollIntervalSeconds: int(s.configPollInterval.Seconds()),
	}, nil
}

func (s *Store) Heartbeat(authToken string, req types.AgentHeartbeatRequest) (types.AgentHeartbeatResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, err := s.lookupAgent(authToken)
	if err != nil {
		return types.AgentHeartbeatResponse{}, err
	}

	now := time.Now().UTC()
	record.LastSeenAt = now
	if hostname := strings.TrimSpace(req.Hostname); hostname != "" {
		record.Hostname = hostname
	}
	if version := strings.TrimSpace(req.AgentVersion); version != "" {
		record.AgentVersion = version
	}
	record.FirewallVersion = strings.TrimSpace(req.FirewallVersion)
	if err := s.saveAgent(record); err != nil {
		return types.AgentHeartbeatResponse{}, err
	}

	return types.AgentHeartbeatResponse{
		ReceivedAt: now.Format(time.RFC3339),
		Online:     true,
	}, nil
}

func (s *Store) Config(authToken string) (types.AgentConfigResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := s.lookupAgent(authToken); err != nil {
		return types.AgentConfigResponse{}, err
	}

	return types.AgentConfigResponse{
		Version:        s.firewallConfig.Version,
		NFTablesConfig: s.firewallConfig.NFTablesConfig,
		UpdatedAt:      s.firewallConfig.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Store) UpdateFirewallConfig(req types.UpdateFirewallConfigRequest) (types.UpdateFirewallConfigResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	content := strings.TrimSpace(req.NFTablesConfig)
	if content == "" {
		return types.UpdateFirewallConfigResponse{}, errors.New("nftablesConfig is required")
	}

	now := time.Now().UTC()
	version := strings.TrimSpace(req.Version)
	if version == "" {
		sum := sha256.Sum256([]byte(content))
		version = "sha256-" + hex.EncodeToString(sum[:8])
	}

	s.firewallConfig = FirewallConfig{
		Version:        version,
		NFTablesConfig: req.NFTablesConfig,
		UpdatedAt:      now,
	}
	if err := s.saveFirewallConfig(s.firewallConfig); err != nil {
		return types.UpdateFirewallConfigResponse{}, err
	}

	return types.UpdateFirewallConfigResponse{
		Version:   version,
		UpdatedAt: now.Format(time.RFC3339),
	}, nil
}

func (s *Store) AuthorizeAPIKey(apiKey string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(apiKey) != s.security.APIKey {
		return ErrUnauthorized
	}
	return nil
}

func (s *Store) AuthorizeAdminSession(authToken string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.adminSessions[strings.TrimSpace(authToken)]; !ok {
		return ErrUnauthorized
	}
	return nil
}

func (s *Store) signEnrollmentToken(claims EnrollmentTokenClaims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(s.security.APIKey))
	_, _ = mac.Write([]byte(encodedPayload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encodedPayload + "." + signature, nil
}

func (s *Store) verifyEnrollmentToken(token string) (EnrollmentTokenClaims, error) {
	var claims EnrollmentTokenClaims

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return claims, ErrInvalidEnrollmentToken
	}

	mac := hmac.New(sha256.New, []byte(s.security.APIKey))
	_, _ = mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)

	provided, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(expected, provided) {
		return claims, ErrInvalidEnrollmentToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return claims, ErrInvalidEnrollmentToken
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return claims, ErrInvalidEnrollmentToken
	}

	if claims.ID == "" || claims.Exp <= time.Now().UTC().Unix() {
		return claims, ErrInvalidEnrollmentToken
	}

	return claims, nil
}

func (s *Store) ListAgents() types.ListAgentsResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	response := types.ListAgentsResponse{
		Agents: make([]types.AgentSummary, 0, len(s.agents)),
	}

	for _, agent := range s.agents {
		response.Agents = append(response.Agents, types.AgentSummary{
			ID:              agent.ID,
			Name:            agent.Name,
			Hostname:        agent.Hostname,
			AgentVersion:    agent.AgentVersion,
			FirewallVersion: agent.FirewallVersion,
			EnrolledAt:      agent.EnrolledAt.Format(time.RFC3339),
			LastSeenAt:      formatTime(agent.LastSeenAt),
			Online:          s.isOnline(now, agent.LastSeenAt),
		})
	}

	return response
}

func (s *Store) lookupAgent(authToken string) (*AgentRecord, error) {
	agentID, ok := s.agentIDsByToken[strings.TrimSpace(authToken)]
	if !ok {
		return nil, ErrUnauthorized
	}

	record, ok := s.agents[agentID]
	if !ok {
		return nil, ErrUnauthorized
	}

	return record, nil
}

func (s *Store) isOnline(now, lastSeen time.Time) bool {
	if lastSeen.IsZero() {
		return false
	}
	return now.Sub(lastSeen) <= s.heartbeatTimeout
}

func formatTime(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.Format(time.RFC3339)
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("rand.Read: %v", err))
	}
	return hex.EncodeToString(buf)
}
