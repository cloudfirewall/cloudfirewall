package firewall

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/google/nftables"
)

type Applier interface {
	Apply(ctx context.Context, version, config string) error
	CurrentVersion() string
}

type SystemApplier struct {
	mu             sync.RWMutex
	currentVersion string
	dryRun         bool
}

func NewSystemApplier(dryRun bool) *SystemApplier {
	return &SystemApplier{dryRun: dryRun}
}

func (a *SystemApplier) Apply(ctx context.Context, version, config string) error {
	if err := probeNFTables(); err != nil {
		return err
	}

	if !a.dryRun {
		nftPath, err := exec.LookPath("nft")
		if err != nil {
			return fmt.Errorf("nft executable not available: %w", err)
		}

		cmd := exec.CommandContext(ctx, nftPath, "-f", "-")
		cmd.Stdin = bytes.NewBufferString(config)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("nft apply failed: %w: %s", err, string(output))
		}
	}

	a.mu.Lock()
	a.currentVersion = version
	a.mu.Unlock()
	return nil
}

func (a *SystemApplier) CurrentVersion() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentVersion
}

func probeNFTables() error {
	conn, err := nftables.New()
	if err != nil {
		return fmt.Errorf("open nftables connection: %w", err)
	}

	if _, err := conn.ListTables(); err != nil {
		return fmt.Errorf("list nftables tables: %w", err)
	}
	return nil
}
