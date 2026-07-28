BINARY := runtz-mcp
PKG := ./cmd/runtz-mcp
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build run test vet fmt clean

build: ## Build the server binary into ./bin
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) $(PKG)

run: ## Run the HTTP server locally on :8080 (smoke test; runtz.dev runs this for you in prod)
	RUNTZ_MCP_ADDR=:8080 go run $(PKG)

test: ## Run the test suite
	go test ./...

vet: ## Static checks
	go vet ./...

fmt: ## Format the code
	gofmt -w .

clean:
	rm -rf bin
