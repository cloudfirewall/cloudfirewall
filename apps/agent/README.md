# Agent

The agent app is the host-side runtime entrypoint.

- enrolls with the API using a one-time signed enrollment token
- polls the API for the latest nftables ruleset
- sends periodic heartbeats with its current firewall version
- can run in dry-run mode for local development
