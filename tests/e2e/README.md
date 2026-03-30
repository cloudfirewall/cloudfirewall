# E2E

This directory contains a Docker-based end-to-end test scaffold for cloudfirewall agents.

The stack includes:

- `api`: the control-plane API
- `enrollment-token`: a bootstrap container that creates a one-time enrollment token through the API
- `agent`: a containerized agent with `CAP_NET_ADMIN` so nftables can be applied inside the container namespace
- `probe`: a network peer container that performs real TCP reachability checks against the agents

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
  - probe-to-agent TCP on port `443` succeeds
  - probe-to-agent TCP on port `22` is blocked by the nftables policy
- `risky-public-ssh` scenario:
  - probe-to-agent TCP on port `22` succeeds
  - probe-to-agent TCP on port `443` is blocked by the nftables policy
- `rollout-two-agents` scenario:
  - two agents enroll independently with separate one-time enrollment tokens
  - both agents converge on the initial `public-web-server` firewall version
  - the API updates the active firewall config in place
  - both agents poll, apply, and report the new `risky-public-ssh` firewall version
  - probe checks confirm both agents changed TCP reachability behavior after rollout

Planned next steps:

- add CI gating behind an opt-in privileged job
