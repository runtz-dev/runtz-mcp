// Command runtz-mcp is the offline-docs Model Context Protocol server for
// runtz. It is a hosted, docs-only HTTP server: it exposes an embedded
// snapshot of the runtz documentation to any MCP-capable AI client (Claude,
// Codex, Gemini, and others) over the MCP streamable HTTP transport. The
// protocol implementation and the docs are pure standard library; the only
// third-party dependency is the OpenTelemetry SDK, which stays dormant unless
// a collector endpoint is configured.
//
// runtz already runs this server for you — point your MCP client at
// https://mcp.runtz.dev/mcp, nothing to install. This binary is what the
// Helm chart in helm/runtz-mcp deploys; running it yourself is only useful
// for local development of the server itself.
//
//	RUNTZ_MCP_ADDR               Listen address (default: ":8080")
//	OTEL_EXPORTER_OTLP_ENDPOINT  OpenTelemetry collector; unset disables
//	                             telemetry entirely (see internal/telemetry)
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
	"github.com/runtz-dev/runtz-mcp/internal/telemetry"
	"github.com/runtz-dev/runtz-mcp/internal/tools"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
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

	shutdownTelemetry, err := telemetry.Setup(ctx, telemetry.Service{
		Name:    serverName,
		Version: version,
	}, logger.Printf)
	if err != nil {
		return fmt.Errorf("starting telemetry: %w", err)
	}
	defer func() {
		// Its own context: ctx is already cancelled by the time we get here,
		// and a cancelled context would drop the final batch of spans.
		flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(flushCtx); err != nil {
			logger.Printf("failed to flush telemetry: %v", err)
		}
	}()

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
		Addr: addr,
		Handler: otelhttp.NewHandler(withRoutePattern(mux), "runtz-mcp",
			otelhttp.WithFilter(func(r *http.Request) bool {
				// The kubelet hits /healthz on both probes every few seconds.
				// Tracing that says nothing and drowns out real traffic.
				return r.URL.Path != "/healthz"
			}),
		),
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

// withRoutePattern renames the request span after the ServeMux pattern that
// will handle it, so a trace reads "/mcp" rather than the generic handler name.
//
// It has to resolve the route itself: Go's ServeMux only fills Request.Pattern
// on the request it passes to the matched handler, which the middleware
// wrapping the mux never sees. Handler runs the same lookup ServeHTTP is about
// to run, so this costs one extra match per request and no allocation.
func withRoutePattern(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern != "" {
			span := trace.SpanFromContext(r.Context())
			span.SetName(pattern)
			span.SetAttributes(semconv.HTTPRoute(pattern))

			// The labeler feeds otelhttp's duration histogram, which would
			// otherwise carry no route at all.
			if labeler, ok := otelhttp.LabelerFromContext(r.Context()); ok {
				labeler.Add(semconv.HTTPRoute(pattern))
			}
		}

		mux.ServeHTTP(w, r)
	})
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
