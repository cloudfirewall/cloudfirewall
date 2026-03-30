package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketAgents           = []byte("agents")
	bucketEnrollmentTokens = []byte("enrollment_tokens")
	bucketMetadata         = []byte("metadata")
	keyFirewallConfig      = []byte("firewall_config")
)

func openDB(path string) (*bolt.DB, error) {
	if path == "" {
		path = defaultDBPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range [][]byte{bucketAgents, bucketEnrollmentTokens, bucketMetadata} {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func defaultDBPath() string {
	return filepath.Join("var", "api", "cloudfirewall.db")
}

func (s *Store) loadPersistedState(initialConfig FirewallConfig) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		if err := loadBucketRecords(tx.Bucket(bucketAgents), &s.agents, func(agent *AgentRecord) {
			s.agentIDsByToken[agent.AuthToken] = agent.ID
		}); err != nil {
			return err
		}

		if err := loadBucketRecords(tx.Bucket(bucketEnrollmentTokens), &s.enrollmentTokens, nil); err != nil {
			return err
		}

		metadata := tx.Bucket(bucketMetadata)
		if metadata == nil {
			return errors.New("metadata bucket missing")
		}

		payload := metadata.Get(keyFirewallConfig)
		if len(payload) == 0 {
			s.firewallConfig = initialConfig
			return putJSON(metadata, keyFirewallConfig, initialConfig)
		}

		return json.Unmarshal(payload, &s.firewallConfig)
	})
}

func loadBucketRecords[T any](bucket *bolt.Bucket, target *map[string]*T, afterLoad func(*T)) error {
	if bucket == nil {
		return nil
	}

	return bucket.ForEach(func(k, v []byte) error {
		record := new(T)
		if err := json.Unmarshal(v, record); err != nil {
			return err
		}
		(*target)[string(k)] = record
		if afterLoad != nil {
			afterLoad(record)
		}
		return nil
	})
}

func (s *Store) saveEnrollmentToken(record *EnrollmentTokenRecord) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bucketEnrollmentTokens), []byte(record.ID), record)
	})
}

func (s *Store) saveAgent(record *AgentRecord) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bucketAgents), []byte(record.ID), record)
	})
}

func (s *Store) saveFirewallConfig(config FirewallConfig) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return putJSON(tx.Bucket(bucketMetadata), keyFirewallConfig, config)
	})
}

func putJSON(bucket *bolt.Bucket, key []byte, value any) error {
	if bucket == nil {
		return errors.New("bucket missing")
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return bucket.Put(key, payload)
}
