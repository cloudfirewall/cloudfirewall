# E2E

This directory contains a Docker-based end-to-end test scaffold for cloudfirewall agents.

The stack includes:

- `api`: the control-plane API
- `enrollment-token`: a bootstrap container that creates a one-time enrollment token through the API
- `agent`: a containerized agent with `CAP_NET_ADMIN` so nftables can be applied inside the container namespace
- `probe`: a placeholder network peer container for future traffic validation scenarios

Run it with:

```bash
make test-e2e
```

Current validation:

- `public-web-server` scenario:
  - API starts
  - frontend-style enrollment token issuance path works
  - agent enrolls with the generated token
  - agent fetches config, applies nftables in-container, heartbeats, and shows as online
  - probe-to-agent traffic on port `443` succeeds
  - probe-to-agent traffic on port `22` is blocked by the nftables policy
- `risky-public-ssh` scenario:
  - probe-to-agent traffic on port `22` succeeds
  - probe-to-agent traffic on port `443` is blocked by the nftables policy

Planned next steps:

- add multiple agent scenarios
- add CI gating behind an opt-in privileged job
