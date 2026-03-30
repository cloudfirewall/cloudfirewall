.PHONY: fmt vet test build cli agent api frontend

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

api:
	mkdir -p bin && go build -o bin/api ./apps/api/cmd/api

frontend:
	cd apps/frontend && npm install && npm run build
