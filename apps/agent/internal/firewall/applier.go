package firewall

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
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

		if family, tableName, ok := managedTableSpec(config); ok {
			if err := deleteExistingTable(ctx, nftPath, family, tableName); err != nil {
				return err
			}
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

func managedTableSpec(config string) (family string, tableName string, ok bool) {
	scanner := bufio.NewScanner(strings.NewReader(config))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "table" {
			return fields[1], fields[2], true
		}
		return "", "", false
	}

	return "", "", false
}

func deleteExistingTable(ctx context.Context, nftPath, family, tableName string) error {
	listCmd := exec.CommandContext(ctx, nftPath, "list", "table", family, tableName)
	if err := listCmd.Run(); err != nil {
		return nil
	}

	deleteCmd := exec.CommandContext(ctx, nftPath, "delete", "table", family, tableName)
	output, err := deleteCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("delete nftables table %s %s failed: %w: %s", family, tableName, err, string(output))
	}

	return nil
}
