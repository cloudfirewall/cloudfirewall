# API

This app owns the control plane for agents.

- `cmd/api`: HTTP server entrypoint
- `internal/httpapi`: route handling for enrollment, heartbeat, config, and fleet listing
- `internal/service`: in-memory agent registry and config store
- `types/`: transport DTOs shared with the agent and frontend-facing endpoints
