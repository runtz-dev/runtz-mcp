package runtz

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ScanResult captures everything an agent needs to reason about a run.
type ScanResult struct {
	Command  string
	ExitCode int
	Stdout   string
	Stderr   string
}

// String renders the result as a single readable block for tool output.
func (r ScanResult) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "$ %s\n", r.Command)
	fmt.Fprintf(&b, "exit code: %d\n", r.ExitCode)
	if out := strings.TrimRight(r.Stdout, "\n"); out != "" {
		b.WriteString("\n--- output ---\n")
		b.WriteString(out)
		b.WriteString("\n")
	}
	if errOut := strings.TrimRight(r.Stderr, "\n"); errOut != "" {
		b.WriteString("\n--- progress / errors ---\n")
		b.WriteString(errOut)
		b.WriteString("\n")
	}
	return b.String()
}

// Runner executes runtz scans by invoking the CLI.
type Runner struct {
	cfg Config
}

// NewRunner builds a Runner from configuration.
func NewRunner(cfg Config) *Runner { return &Runner{cfg: cfg} }

// Allowed reports whether scan tools should be exposed.
func (r *Runner) Allowed() bool { return r.cfg.AllowScans }

// Run executes `runtz <subcommand>` with the provided flags. Authentication
// flags (--endpoint / --token) are injected from config unless the caller has
// already supplied them, so the model never needs to know the secret.
func (r *Runner) Run(ctx context.Context, subcommand string, flags []string) (ScanResult, error) {
	if !r.cfg.AllowScans {
		return ScanResult{}, errors.New("scans are disabled (RUNTZ_MCP_ALLOW_SCANS=false)")
	}

	argv := r.cfg.BinArgv()
	args := append([]string{}, argv[1:]...)
	args = append(args, subcommand)
	args = append(args, flags...)

	if !hasFlag(flags, "--endpoint") && r.cfg.Endpoint != "" {
		args = append(args, "--endpoint", r.cfg.Endpoint)
	}
	if !hasFlag(flags, "--token") && r.cfg.Token != "" {
		args = append(args, "--token", r.cfg.Token)
	}

	cmd := exec.CommandContext(ctx, argv[0], args...)
	if r.cfg.WorkDir != "" {
		cmd.Dir = r.cfg.WorkDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	result := ScanResult{
		Command:  redactedCommand(argv[0], args),
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}

	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			// A non-zero exit (e.g. findings present, auth missing) is still a
			// useful result, not a transport error.
			return result, nil
		}
		// The binary could not be launched at all.
		return result, fmt.Errorf("could not run %q: %w (set RUNTZ_MCP_BIN to the runtz binary or 'go run ./cmd/runtz')", argv[0], runErr)
	}

	return result, nil
}

func hasFlag(flags []string, name string) bool {
	for _, f := range flags {
		if f == name || strings.HasPrefix(f, name+"=") {
			return true
		}
	}
	return false
}

// redactedCommand rebuilds the command line for display, masking the token value.
func redactedCommand(bin string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, bin)
	for i := 0; i < len(args); i++ {
		parts = append(parts, args[i])
		if args[i] == "--token" && i+1 < len(args) {
			parts = append(parts, "rtz_***")
			i++
		}
	}
	return strings.Join(parts, " ")
}
