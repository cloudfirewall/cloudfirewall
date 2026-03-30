# cloudfirewall

Cloudfirewall is organized around four product surfaces:

- `apps/engine`: policy validation, compilation, simulation, and artifact CLI
- `apps/agent`: host-side agent entrypoint
- `apps/api`: shared API-facing response types and future service code
- `apps/frontend`: React/Vite frontend workspace scaffold

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
- agent stub
- API response types
- React-ready frontend scaffold
- JSON test fixtures and golden tests

## Quick start

```bash
make test
make cli
./bin/engine-cli validate --policy apps/engine/testdata/policies/public-web-server.json
```

## App entrypoints

- Engine CLI: `./apps/engine/cmd/engine-cli`
- Engine internals: `./apps/engine/internal`
- Engine fixtures: `./apps/engine/testdata`
- Agent: `./apps/agent/cmd/agent`
- API package root: `./apps/api`
- Frontend app: `./apps/frontend`

## Frontend

The frontend workspace is prepared for a React app using Vite and TypeScript. Dependencies are not installed automatically in this repo; when you're ready to work on the UI:

```bash
cd apps/frontend
npm install
npm run dev
```
