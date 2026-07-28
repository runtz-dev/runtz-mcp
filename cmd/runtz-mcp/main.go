// Command runtz-mcp is the offline-docs Model Context Protocol server for
// runtz. It is a hosted, docs-only HTTP server: it exposes an embedded
// snapshot of the runtz documentation to any MCP-capable AI client (Claude,
// Codex, Gemini, and others) over the MCP streamable HTTP transport. It has
// no third-party dependencies, so it runs anywhere the Go standard library
// does.
//
// runtz already runs this server for you — point your MCP client at
// https://mcp.runtz.dev/mcp, nothing to install. This binary is what the
// Helm chart in helm/runtz-mcp deploys; running it yourself is only useful
// for local development of the server itself.
//
//	RUNTZ_MCP_ADDR  Listen address (default: ":8080")
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/runtz-dev/runtz-mcp/internal/mcp"
	"github.com/runtz-dev/runtz-mcp/internal/runtz"
	"github.com/runtz-dev/runtz-mcp/internal/tools"
)

// version is overridable at build time with -ldflags "-X main.version=...".
var version = "1.0.0-rc2"

const serverName = "runtz-mcp"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "--version", "version":
			fmt.Printf("%s %s\n", serverName, version)
			return
		case "-h", "--help", "help":
			fmt.Fprint(os.Stderr, usage)
			return
		}
	}

	logger := log.New(os.Stderr, "[runtz-mcp] ", log.LstdFlags)

	if err := run(logger); err != nil {
		logger.Fatalf("fatal: %v", err)
	}
}

func run(logger *log.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	docs, err := runtz.LoadDocs()
	if err != nil {
		return fmt.Errorf("loading embedded docs: %w", err)
	}

	addr := envOr("RUNTZ_MCP_ADDR", ":8080")

	server := mcp.NewServer(serverName, version)
	server.SetLogger(logger.Printf)
	server.SetInstructions(instructions)

	tools.Register(server, docs)

	logger.Printf("ready: %d docs, serving HTTP on %s", len(docs.List()), addr)

	return serveHTTP(ctx, logger, server, addr)
}

// serveHTTP runs the streamable HTTP transport. It exposes the JSON-RPC
// endpoint at POST /mcp and a liveness probe at /healthz, and shuts down
// gracefully when the context is cancelled.
func serveHTTP(ctx context.Context, logger *log.Logger, server *mcp.Server, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/mcp", server.MessageHandler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	})

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		errc <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

const instructions = `runtz MCP server. Use the runtz_docs_* tools to read the offline runtz documentation. This server is docs-only; run DevSecOps scans with the runtz CLI directly (see runtz_docs_read "cli").`

const usage = `runtz-mcp - hosted, docs-only Model Context Protocol server for runtz

Usage:
  runtz-mcp            Start the HTTP server (used by the Helm chart)
  runtz-mcp --version  Print version
  runtz-mcp --help     Print this help

Environment:
  RUNTZ_MCP_ADDR  Listen address (default: ":8080")

Serves JSON-RPC over HTTP at POST /mcp (with a /healthz probe). runtz already
runs this for you at https://mcp.runtz.dev/mcp — most users never run this
binary themselves. See README.md for client setup.
`
