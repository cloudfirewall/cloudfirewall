package agent

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cloudfirewall/cloudfirewall/apps/agent/internal/apiclient"
	"github.com/cloudfirewall/cloudfirewall/apps/agent/internal/firewall"
	"github.com/cloudfirewall/cloudfirewall/apps/api/types"
)

type Runner struct {
	client          *apiclient.Client
	applier         firewall.Applier
	enrollmentToken string
	name            string
	hostname        string
	version         string
}

func NewRunner(client *apiclient.Client, applier firewall.Applier, enrollmentToken, name, version string) *Runner {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host"
	}

	if strings.TrimSpace(name) == "" {
		name = hostname
	}

	return &Runner{
		client:          client,
		applier:         applier,
		enrollmentToken: enrollmentToken,
		name:            name,
		hostname:        hostname,
		version:         version,
	}
}

func (r *Runner) Run(ctx context.Context, once bool) error {
	enrollment, err := r.client.Enroll(ctx, types.EnrollAgentRequest{
		EnrollmentToken: r.enrollmentToken,
		AgentName:       r.name,
		Hostname:        r.hostname,
		AgentVersion:    r.version,
	})
	if err != nil {
		return err
	}

	log.Printf("agent enrolled: id=%s", enrollment.AgentID)

	if err := r.syncConfig(ctx); err != nil {
		log.Printf("config sync failed: %v", err)
	}
	if err := r.sendHeartbeat(ctx); err != nil {
		log.Printf("heartbeat failed: %v", err)
	}

	if once {
		return nil
	}

	heartbeatTicker := time.NewTicker(time.Duration(enrollment.HeartbeatIntervalSeconds) * time.Second)
	defer heartbeatTicker.Stop()

	configTicker := time.NewTicker(time.Duration(enrollment.ConfigPollIntervalSeconds) * time.Second)
	defer configTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-heartbeatTicker.C:
			if err := r.sendHeartbeat(ctx); err != nil {
				log.Printf("heartbeat failed: %v", err)
			}
		case <-configTicker.C:
			if err := r.syncConfig(ctx); err != nil {
				log.Printf("config sync failed: %v", err)
			}
		}
	}
}

func (r *Runner) syncConfig(ctx context.Context) error {
	config, err := r.client.Config(ctx)
	if err != nil {
		return err
	}

	if config.Version == r.applier.CurrentVersion() {
		return nil
	}

	if err := r.applier.Apply(ctx, config.Version, config.NFTablesConfig); err != nil {
		return err
	}

	log.Printf("applied firewall version=%s", config.Version)
	return nil
}

func (r *Runner) sendHeartbeat(ctx context.Context) error {
	_, err := r.client.Heartbeat(ctx, types.AgentHeartbeatRequest{
		Hostname:        r.hostname,
		AgentVersion:    r.version,
		FirewallVersion: r.applier.CurrentVersion(),
	})
	return err
}
