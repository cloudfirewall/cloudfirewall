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
	bucketAgents              = []byte("agents")
	bucketEnrollmentTokens    = []byte("enrollment_tokens")
	bucketFirewallConfigs     = []byte("firewall_configs")
	bucketMetadata            = []byte("metadata")
	keyFirewallConfig         = []byte("firewall_config")
	keyActiveFirewallConfigID = []byte("active_firewall_config_id")
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
		for _, bucket := range [][]byte{bucketAgents, bucketEnrollmentTokens, bucketFirewallConfigs, bucketMetadata} {
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
		if err := loadBucketRecords(tx.Bucket(bucketFirewallConfigs), &s.firewallConfigs, nil); err != nil {
			return err
		}

		metadata := tx.Bucket(bucketMetadata)
		if metadata == nil {
			return errors.New("metadata bucket missing")
		}

		activeID := metadata.Get(keyActiveFirewallConfigID)
		if len(activeID) != 0 {
			s.activeConfigID = string(activeID)
			return nil
		}

		payload := metadata.Get(keyFirewallConfig)
		if len(payload) != 0 {
			var migrated FirewallConfig
			if err := json.Unmarshal(payload, &migrated); err != nil {
				return err
			}
			if migrated.ID == "" {
				migrated.ID = "cfg_default"
			}
			if migrated.Name == "" {
				migrated.Name = "Default Firewall"
			}
			s.firewallConfigs[migrated.ID] = &migrated
			s.activeConfigID = migrated.ID
			if err := putJSON(tx.Bucket(bucketFirewallConfigs), []byte(migrated.ID), migrated); err != nil {
				return err
			}
			return metadata.Put(keyActiveFirewallConfigID, []byte(migrated.ID))
		}

		initial := initialConfig
		if initial.ID == "" {
			initial.ID = "cfg_default"
		}
		if initial.Name == "" {
			initial.Name = "Default Firewall"
		}
		s.firewallConfigs[initial.ID] = &initial
		s.activeConfigID = initial.ID
		if err := putJSON(tx.Bucket(bucketFirewallConfigs), []byte(initial.ID), initial); err != nil {
			return err
		}
		return metadata.Put(keyActiveFirewallConfigID, []byte(initial.ID))
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
		return putJSON(tx.Bucket(bucketFirewallConfigs), []byte(config.ID), config)
	})
}

func (s *Store) saveActiveFirewallConfigID(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketMetadata).Put(keyActiveFirewallConfigID, []byte(id))
	})
}

func (s *Store) deleteFirewallConfig(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketFirewallConfigs).Delete([]byte(id))
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
