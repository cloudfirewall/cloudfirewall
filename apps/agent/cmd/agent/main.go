package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudfirewall/cloudfirewall/apps/agent/internal/agent"
	"github.com/cloudfirewall/cloudfirewall/apps/agent/internal/apiclient"
	"github.com/cloudfirewall/cloudfirewall/apps/agent/internal/firewall"
)

func main() {
	apiURL := flag.String("api-url", envOrDefault("CLOUDFIREWALL_API_URL", "http://localhost:8080"), "base URL of the cloudfirewall API")
	enrollmentToken := flag.String("enrollment-token", envOrDefault("CLOUDFIREWALL_ENROLLMENT_TOKEN", ""), "shared enrollment token")
	agentName := flag.String("name", "", "logical name for this agent")
	agentVersion := flag.String("agent-version", "0.1.0", "agent version reported to the API")
	once := flag.Bool("once", false, "run a single enroll/config/heartbeat cycle and exit")
	dryRun := flag.Bool("dry-run", true, "do not invoke the nft CLI when applying the config")
	flag.Parse()

	if *enrollmentToken == "" {
		log.Fatal("--enrollment-token is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runner := agent.NewRunner(
		apiclient.New(*apiURL),
		firewall.NewSystemApplier(*dryRun),
		*enrollmentToken,
		*agentName,
		*agentVersion,
	)

	if err := runner.Run(ctx, *once); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
