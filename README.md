# cloudfirewall

Cloudfirewall is organized around four product surfaces:

- `apps/engine`: policy validation, compilation, simulation, and artifact CLI
- `apps/agent`: host-side agent entrypoint that enrolls, heartbeats, and applies firewall configs
- `apps/api`: API server for enrollment, heartbeat tracking, and config delivery
- `apps/frontend`: React/Vite dashboard for fleet status

Engine-owned domain logic now lives under `apps/engine/internal/`. The other apps stay decoupled from those implementation details.

## Repository layout

```text
apps/
  agent/      Agent application code
  api/        API contracts and service-facing packages
  engine/     Engine CLI and engine-facing orchestration
  frontend/   React frontend workspace
```

## Included today

- policy authoring model and normalized IR
- normalization, resolution, validation, compilation, simulation, and artifact packages
- engine CLI
- API server for agent enrollment, heartbeat, config sync, and fleet listing
- agent enrollment and heartbeat loop with firewall config polling
- frontend dashboard for online status and nftables firewall versions
- JSON test fixtures and golden tests

## Quick start

```bash
make test
make cli
make api
make agent
./bin/engine-cli validate --policy apps/engine/testdata/policies/public-web-server.json
```

## Agent Flow

1. Agents enroll with `POST /api/v1/enroll` using a shared enrollment token.
2. The API returns an agent auth token plus suggested heartbeat and config poll intervals.
3. Agents pull `GET /api/v1/agents/self/config` to get the current nftables ruleset and config version.
4. Agents send `POST /api/v1/agents/self/heartbeat` so the API can mark them online and record the firewall version they are running.
5. The frontend reads `GET /api/v1/agents` to show fleet status.

## Run The Stack

Start the API:

```bash
./bin/api -enrollment-token dev-enrollment-token
```

Enroll and run an agent once in dry-run mode:

```bash
./bin/agent \
  -api-url http://localhost:8080 \
  -enrollment-token dev-enrollment-token \
  -name demo-agent \
  -once \
  -dry-run
```

Start the frontend:

```bash
cd apps/frontend
npm install
npm run dev
```

## App entrypoints

- Engine CLI: `./apps/engine/cmd/engine-cli`
- Engine internals: `./apps/engine/internal`
- Engine fixtures: `./apps/engine/testdata`
- Agent: `./apps/agent/cmd/agent`
- API server: `./apps/api/cmd/api`
- Frontend app: `./apps/frontend`

## Frontend

The frontend proxies `/api` requests to `http://localhost:8080` in development and renders the enrolled agents, their online status, and the firewall version reported in heartbeats.

The agent defaults to `-dry-run` so it can participate in the control-plane flow without needing root privileges or an installed `nft` binary. When you want it to apply the received ruleset for real, run it with `-dry-run=false`.

Dependencies are not installed automatically in this repo; when you're ready to work on the UI:

```bash
cd apps/frontend
npm install
npm run dev
```
