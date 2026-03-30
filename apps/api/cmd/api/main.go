package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cloudfirewall/cloudfirewall/apps/api/internal/httpapi"
	"github.com/cloudfirewall/cloudfirewall/apps/api/internal/service"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	configPath := flag.String("config", "apps/engine/testdata/compiled/public-web-server.nft.golden", "path to nftables config file")
	adminUsername := flag.String("admin-username", envOrDefault("CLOUDFIREWALL_ADMIN_USERNAME", "admin"), "admin username for frontend login")
	adminPassword := flag.String("admin-password", envOrDefault("CLOUDFIREWALL_ADMIN_PASSWORD", "admin"), "admin password for frontend login")
	apiKey := flag.String("api-key", envOrDefault("CLOUDFIREWALL_API_KEY", "dev-api-key"), "API key for programmatic API access")
	heartbeatTimeout := flag.Duration("heartbeat-timeout", 30*time.Second, "duration before an agent is considered offline")
	heartbeatInterval := flag.Duration("heartbeat-interval", 10*time.Second, "suggested heartbeat interval returned to agents")
	configPollInterval := flag.Duration("config-poll-interval", 15*time.Second, "suggested config poll interval returned to agents")
	flag.Parse()

	config, err := loadFirewallConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	store := service.NewStore(
		service.SecurityConfig{
			AdminUsername: *adminUsername,
			AdminPassword: *adminPassword,
			APIKey:        *apiKey,
		},
		config,
		*heartbeatTimeout,
		*heartbeatInterval,
		*configPollInterval,
	)

	log.Printf("cloudfirewall api listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, httpapi.NewServer(store)))
}

func loadFirewallConfig(path string) (service.FirewallConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return service.FirewallConfig{}, err
	}

	sum := sha256.Sum256(content)
	version := "sha256-" + hex.EncodeToString(sum[:8])

	return service.FirewallConfig{
		Version:        version,
		NFTablesConfig: string(content),
		UpdatedAt:      time.Now().UTC(),
	}, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
