.PHONY: fmt vet test build cli agent

fmt:
	gofmt -w ./cmd ./internal ./api

vet:
	go vet ./...

test:
	go test ./...

build:
	go build ./...

cli:
	mkdir -p bin && go build -o bin/engine-cli ./cmd/engine-cli

agent:
	mkdir -p bin && go build -o bin/agent ./cmd/agent
