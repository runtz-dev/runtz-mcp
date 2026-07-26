// Command runtz-mcp is an offline Model Context Protocol server for runtz.
//
// It exposes the runtz scanners (sca, sast, host, container, k8s) and an
// embedded snapshot of the runtz documentation to any MCP-capable AI client
// (Claude, Codex, Gemini, and others). It speaks JSON-RPC 2.0 over stdio by
// default, or over a streamable HTTP transport when RUNTZ_MCP_HTTP is set (for
// hosted deployments, where it runs docs-only). It has no third-party
// dependencies, so it runs anywhere the Go standard library does.
//
// Configuration comes entirely from the environment, which the MCP client
// injects through its server config so the workspace token never appears in
// prompts or tool arguments:
//
//	RUNTZ_ENDPOINT        Runtz backend endpoint (default: hosted SaaS engine)
//	RUNTZ_TOKEN           Workspace token generated in the platform
//	RUNTZ_MCP_BIN         runtz CLI binary or launcher (default: "runtz")
//	RUNTZ_MCP_WORKDIR     Working directory for scans (default: process cwd)
//	RUNTZ_MCP_ALLOW_SCANS Set to false for a docs-only server (default: true)
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
var version = "1.0.0-rc1"

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

	// stdout is reserved for the JSON-RPC stream; all diagnostics go to stderr.
	logger := log.New(os.Stderr, "[runtz-mcp] ", log.LstdFlags)

	if err := run(logger); err != nil {
		logger.Fatalf("fatal: %v", err)
	}
}

func run(logger *log.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := runtz.LoadConfig()

	docs, err := runtz.LoadDocs()
	if err != nil {
		return fmt.Errorf("loading embedded docs: %w", err)
	}

	// The hosted server (RUNTZ_MCP_HTTP set) is docs-only: scans need the user's
	// code on the machine running the CLI, which the shared HTTP server does not
	// have, so scan tools are never exposed there regardless of the env.
	httpAddr := os.Getenv("RUNTZ_MCP_HTTP")
	if httpAddr != "" {
		cfg.AllowScans = false
	}

	server := mcp.NewServer(serverName, version)
	server.SetLogger(logger.Printf)
	server.SetInstructions(instructions)

	runner := runtz.NewRunner(cfg)
	tools.Register(server, runner, docs)

	logger.Printf("ready: %d docs, scans=%t, endpoint=%s", len(docs.List()), runner.Allowed(), cfg.Endpoint)

	if httpAddr != "" {
		return serveHTTP(ctx, logger, server, httpAddr)
	}

	return server.Serve(ctx, os.Stdin, os.Stdout)
}

// serveHTTP runs the streamable HTTP transport used by the hosted deployment. It
// exposes the JSON-RPC endpoint at POST /mcp and a liveness probe at /healthz,
// and shuts down gracefully when the context is cancelled.
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
		logger.Printf("serving MCP over HTTP on %s (docs-only)", addr)
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

const instructions = `runtz MCP server. Use the runtz_docs_* tools to read the offline runtz documentation, and the runtz_sca / runtz_sast / runtz_host / runtz_container / runtz_k8s tools to run DevSecOps scans. Authentication (endpoint and token) is configured via environment variables, so do not ask the user for a token.`

const usage = `runtz-mcp - offline Model Context Protocol server for runtz

Usage:
  runtz-mcp            Start the stdio MCP server (used by MCP clients)
  runtz-mcp --version  Print version
  runtz-mcp --help     Print this help

Environment:
  RUNTZ_ENDPOINT        Runtz backend endpoint (default: hosted SaaS engine)
  RUNTZ_TOKEN           Workspace token generated in the platform
  RUNTZ_MCP_BIN         runtz CLI binary or launcher (default: "runtz")
  RUNTZ_MCP_WORKDIR     Working directory for scans (default: process cwd)
  RUNTZ_MCP_ALLOW_SCANS Set to false for a docs-only server (default: true)
  RUNTZ_MCP_HTTP        Listen address (e.g. ":8080") to serve the streamable
                        HTTP transport instead of stdio. Always docs-only.

Over stdio the server is meant to be launched by an MCP client (Claude, Codex,
Gemini, ...). With RUNTZ_MCP_HTTP set it serves JSON-RPC over HTTP at POST /mcp
(with a /healthz probe) for hosted deployments. See README.md for client setup.
`
