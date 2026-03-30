package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudfirewall/cloudfirewall/apps/api/types"
)

var (
	ErrInvalidEnrollmentToken = errors.New("invalid enrollment token")
	ErrUnauthorized           = errors.New("unauthorized")
)

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

type Store struct {
	mu                 sync.RWMutex
	enrollmentTokens   map[string]struct{}
	agents             map[string]*AgentRecord
	agentIDsByToken    map[string]string
	firewallConfig     FirewallConfig
	heartbeatTimeout   time.Duration
	heartbeatInterval  time.Duration
	configPollInterval time.Duration
}

func NewStore(tokens []string, config FirewallConfig, heartbeatTimeout, heartbeatInterval, configPollInterval time.Duration) *Store {
	tokenSet := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		if trimmed := strings.TrimSpace(token); trimmed != "" {
			tokenSet[trimmed] = struct{}{}
		}
	}

	return &Store{
		enrollmentTokens:   tokenSet,
		agents:             make(map[string]*AgentRecord),
		agentIDsByToken:    make(map[string]string),
		firewallConfig:     config,
		heartbeatTimeout:   heartbeatTimeout,
		heartbeatInterval:  heartbeatInterval,
		configPollInterval: configPollInterval,
	}
}

func (s *Store) Enroll(req types.EnrollAgentRequest) (types.EnrollAgentResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.enrollmentTokens[strings.TrimSpace(req.EnrollmentToken)]; !ok {
		return types.EnrollAgentResponse{}, ErrInvalidEnrollmentToken
	}

	now := time.Now().UTC()
	agentID := "agt-" + randomHex(8)
	authToken := "cfw_" + randomHex(24)
	name := strings.TrimSpace(req.AgentName)
	if name == "" {
		name = strings.TrimSpace(req.Hostname)
	}
	if name == "" {
		name = agentID
	}

	record := &AgentRecord{
		ID:           agentID,
		AuthToken:    authToken,
		Name:         name,
		Hostname:     strings.TrimSpace(req.Hostname),
		AgentVersion: strings.TrimSpace(req.AgentVersion),
		EnrolledAt:   now,
	}

	s.agents[agentID] = record
	s.agentIDsByToken[authToken] = agentID

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
