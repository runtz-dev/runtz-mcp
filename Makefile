BINARY := runtz-mcp
PKG := ./cmd/runtz-mcp
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build install run test vet fmt clean

build: ## Build the server binary into ./bin
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) $(PKG)

install: ## Install the server into GOBIN / GOPATH/bin
	go install -ldflags "$(LDFLAGS)" $(PKG)

run: ## Run the server on stdio (mostly useful for a quick smoke test)
	go run $(PKG)

test: ## Run the test suite
	go test ./...

vet: ## Static checks
	go vet ./...

fmt: ## Format the code
	gofmt -w .

clean:
	rm -rf bin
