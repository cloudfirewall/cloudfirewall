.PHONY: fmt vet test build cli agent api frontend agent-release test-install-agent test-e2e e2e-down

fmt:
	gofmt -w ./apps

vet:
	go vet ./...

test:
	GOCACHE=/tmp/go-build-cache go test ./...

build:
	go build ./...

cli:
	mkdir -p bin && go build -o bin/engine-cli ./apps/engine/cmd/engine-cli

agent:
	mkdir -p bin && go build -o bin/agent ./apps/agent/cmd/agent

agent-release:
	test -n "$(VERSION)"
	sh ./scripts/package-agent-release.sh

test-install-agent:
	tests/install-agent/run.sh

api:
	mkdir -p bin && go build -o bin/api ./apps/api/cmd/api

frontend:
	cd apps/frontend && npm install && npm run build

test-e2e:
	tests/e2e/run.sh

e2e-down:
	docker compose -f tests/e2e/docker-compose.yml down -v --remove-orphans
