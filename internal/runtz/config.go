// Package runtz holds the runtz-specific behaviour exposed by the MCP server:
// configuration, scan execution (by shelling out to the runtz CLI) and access to
// the embedded offline documentation snapshot.
package runtz

import (
	"os"
	"strings"
)

// SaaSEndpoint is the hosted Runtz engine, mirrored from the CLI default so the
// model can fall back to it when no endpoint is configured.
const SaaSEndpoint = "https://engine.runtz.dev"

// Config is resolved once at startup from the environment. The MCP client is
// expected to inject these through the server's `env` block (see the example
// configs), which keeps the token out of prompts, logs and tool arguments.
type Config struct {
	// Bin is the runtz CLI binary or launcher. It is split on spaces so values
	// like "go run ./cmd/runtz" work as well as a plain "runtz".
	Bin string
	// WorkDir is the working directory scans run from (defaults to the process
	// working directory). Useful when Bin is "go run ./cmd/runtz".
	WorkDir string
	// Endpoint and Token are injected into every scan that does not override
	// them, so the model never has to handle the secret.
	Endpoint string
	Token    string
	// AllowScans gates the scan tools. Set RUNTZ_MCP_ALLOW_SCANS=false to run a
	// docs-only server (for example in untrusted or read-only contexts).
	AllowScans bool
}

// LoadConfig reads configuration from the environment.
func LoadConfig() Config {
	cfg := Config{
		Bin:        envOr("RUNTZ_MCP_BIN", "runtz"),
		WorkDir:    os.Getenv("RUNTZ_MCP_WORKDIR"),
		Endpoint:   envOr("RUNTZ_ENDPOINT", ""),
		Token:      firstNonEmpty(os.Getenv("RUNTZ_TOKEN"), os.Getenv("RUNTZ_API_KEY")),
		AllowScans: envBoolDefault("RUNTZ_MCP_ALLOW_SCANS", true),
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = SaaSEndpoint
	}
	return cfg
}

// BinArgv splits Bin into a command and its leading arguments.
func (c Config) BinArgv() []string {
	fields := strings.Fields(c.Bin)
	if len(fields) == 0 {
		return []string{"runtz"}
	}
	return fields
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func envBoolDefault(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch v {
	case "":
		return fallback
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
