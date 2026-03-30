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

1. Agents enroll with `POST /api/v1/enroll` using a one-time signed enrollment token.
2. Admin users log into the frontend and create one-time signed enrollment tokens with `POST /api/v1/enrollment-tokens`.
3. The API verifies the enrollment token signature and one-time-use status, then returns an agent auth token plus suggested heartbeat and config poll intervals.
4. Agents pull `GET /api/v1/agents/self/config` to get the current nftables ruleset and config version.
5. Agents send `POST /api/v1/agents/self/heartbeat` so the API can mark them online and record the firewall version they are running.
6. The frontend reads `GET /api/v1/agents` to show fleet status.

## Run The Stack

Start the API:

```bash
./bin/api \
  -db-path ./var/api/cloudfirewall.db \
  -admin-username admin \
  -admin-password admin \
  -api-key dev-api-key
```

Open the API docs:

```text
http://localhost:8080/swagger
```

Frontend login uses the configured admin username and password. Programmatic API access can use the configured `X-API-Key` header. Enrollment tokens are now created from the frontend after login and are signed, expiring, and single-use.

The API persists enrolled agents, one-time enrollment token state, and the active firewall configuration in an embedded BoltDB database. Point `-db-path` or `CLOUDFIREWALL_API_DB_PATH` at a stable file location if you want agent/auth state to survive API restarts. Admin login sessions are intentionally ephemeral and are not retained across restarts.

Log into the frontend, generate an enrollment token, then run an agent once in dry-run mode:

```bash
./bin/agent \
  -api-url http://localhost:8080 \
  -enrollment-token <generated-enrollment-token> \
  -name demo-agent \
  -once \
  -dry-run
```

To install an agent on a server as a managed service in one line:

```bash
curl -fsSL https://raw.githubusercontent.com/cloudfirewall/cloudfirewall/main/scripts/install-agent.sh | sudo sh -s -- \
  --api-url http://YOUR-API:8080 \
  --enrollment-token <generated-enrollment-token> \
  --name edge-01
```

The installer installs `cloudfirewall-agent` under `/usr/local/bin`, writes `/etc/cloudfirewall/agent.env`, and creates `cloudfirewall-agent.service` for systemd.

For tagged releases, use the prebuilt binary flow so the server does not need Go installed:

```bash
curl -fsSL https://raw.githubusercontent.com/cloudfirewall/cloudfirewall/main/scripts/install-agent.sh | sudo sh -s -- \
  --api-url http://YOUR-API:8080 \
  --enrollment-token <generated-enrollment-token> \
  --name edge-01 \
  --release-version v0.1.0
```

If `--release-version` is set, the installer downloads `cloudfirewall-agent_<version>_linux_<arch>.tar.gz` from the GitHub Releases download URL. If neither `--release-version` nor `--binary-url` is set, it falls back to building from the GitHub source archive.

To prepare release artifacts locally:

```bash
make agent-release VERSION=v0.1.0
```

That writes Linux `amd64` and `arm64` agent archives into `dist/releases/`, ready to upload to a GitHub release.

Tagged pushes such as `v0.1.0` also publish those two archives automatically through [release-agent.yml](/home/user/github/sireto/cloudfirewall/.github/workflows/release-agent.yml). After that, the installer can use `--release-version v0.1.0` without requiring Go on the target server.

Start the frontend:

```bash
cd apps/frontend
npm install
npm run dev
```

## E2E Testing

For containerized agent validation without touching the host firewall:

```bash
make test-e2e
make test-install-agent
```

The Docker-based e2e stack runs the API, generates a one-time enrollment token, starts an agent container with `CAP_NET_ADMIN`, applies nftables inside the container namespace, and verifies the agent shows as online through the API. The scaffold lives in `tests/e2e/`.

`make test-install-agent` separately validates the one-line installer in a disposable Docker container by packaging a local release archive, running the real install script, and checking the installed binary, environment file, and systemd unit output.

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
