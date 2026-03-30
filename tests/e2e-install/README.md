# E2E Install

This suite validates the installer-driven agent path inside Docker.

What it does:

- packages a local agent release archive
- starts the API and issues a one-time enrollment token
- boots an agent-host container that runs `scripts/install-agent.sh`
- verifies the installed binary, env file, and service unit exist
- starts the installed agent binary
- confirms the agent enrolls, heartbeats, applies nftables, and enforces the expected network policy

Run it with:

```bash
make test-e2e-install
```
