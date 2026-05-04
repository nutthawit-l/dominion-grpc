.PHONY: generate test test-short lint server bot install-tools tidy

generate:
	buf generate

test:
	go test ./...

test-short:
	go test -short ./...

lint:
	buf lint
	golangci-lint run ./...

server:
	go run ./cmd/server

bot:
	go run ./cmd/bot $(ARGS)

tidy:
	go mod tidy

GOPATH := $(shell go env GOPATH)
GOBIN  := $(GOPATH)/bin

install-tools:
	GOBIN=$(GOBIN) go install github.com/bufbuild/buf/cmd/buf@v1.67.0
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(GOBIN) v2.11.4

install-figma-mcp:
	claude plugin install figma@claude-plugins-official

install-superpower:
	claude plugin install superpowers@claude-plugins-official