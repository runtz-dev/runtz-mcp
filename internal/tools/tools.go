// Package tools wires runtz behaviour into MCP tools and resources.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runtz-dev/runtz-mcp/internal/mcp"
	"github.com/runtz-dev/runtz-mcp/internal/runtz"
)

// Register adds every runtz tool and documentation resource to the server.
func Register(s *mcp.Server, runner *runtz.Runner, docs *runtz.Docs) {
	registerDocTools(s, docs)
	registerDocResources(s, docs)
	if runner.Allowed() {
		registerScanTools(s, runner)
	}
}

// --- scan tools ---

// scanSpec declares a scan tool and how its arguments map onto CLI flags.
type scanSpec struct {
	name        string
	subcommand  string
	description string
	// properties is the JSON Schema for arguments.
	properties map[string]any
	// positionalArg names the argument passed as the leading positional CLI
	// argument (e.g. the scan path or image reference).
	positionalArg string
	// stringFlags maps an argument name to its CLI flag (value passed through).
	stringFlags map[string]string
	// boolFlags maps an argument name to a CLI flag added when the value is true.
	boolFlags map[string]string
}

func registerScanTools(s *mcp.Server, runner *runtz.Runner) {
	specs := []scanSpec{
		{
			name:        "runtz_sca",
			subcommand:  "sca",
			description: "Run a Software Composition Analysis scan over a repository or a single dependency manifest (package.json, requirements.txt, go.mod, pom.xml, Gemfile.lock, composer.json, Cargo.toml, *.csproj) and report vulnerable dependencies. Sends the report to the configured Runtz workspace.",
			properties: map[string]any{
				"path":         strProp("Repository directory or single manifest file to scan.", "."),
				"project":      strProp("Project name override shown in the platform.", ""),
				"source":       strProp("Project source path, repository or URL.", ""),
				"github_token": strProp("Optional GitHub token for higher advisory API limits.", ""),
			},
			positionalArg: "path",
			stringFlags: map[string]string{
				"project":      "--project",
				"source":       "--source",
				"github_token": "--github-token",
			},
		},
		{
			name:        "runtz_sast",
			subcommand:  "sast",
			description: "Run a Static Application Security Testing scan over source code. Detects high-signal issues such as committed secrets, dynamic code execution, disabled TLS verification and weak hashing.",
			properties: map[string]any{
				"path":    strProp("Source file or directory to scan.", "."),
				"project": strProp("Project name override shown in the platform.", ""),
				"source":  strProp("Project source path, repository or URL.", ""),
			},
			positionalArg: "path",
			stringFlags: map[string]string{
				"project": "--project",
				"source":  "--source",
			},
		},
		{
			name:        "runtz_host",
			subcommand:  "host",
			description: "Scan installed packages on the current Linux host (Debian/Ubuntu, RPM, Alpine or Arch based) and report package CVEs from OSV.",
			properties: map[string]any{
				"rootfs":   strProp("Root filesystem to scan when not /.", "/"),
				"hostname": strProp("Hostname override shown in the platform.", ""),
				"osv_url":  strProp("Optional OSV API base URL.", ""),
			},
			stringFlags: map[string]string{
				"rootfs":   "--rootfs",
				"hostname": "--hostname",
				"osv_url":  "--osv-url",
			},
		},
		{
			name:        "runtz_container",
			subcommand:  "container",
			description: "Scan OS packages inside a container image (Debian/Ubuntu, RPM, Alpine or Arch based) and report CVEs. Pulls from a registry by default; set local=true to read from the local Docker daemon.",
			properties: map[string]any{
				"image":   strProp("Container image reference, e.g. ubuntu:22.04 or alpine:3.19.", ""),
				"local":   boolProp("Read the image from the local Docker daemon instead of a registry."),
				"osv_url": strProp("Optional OSV API base URL.", ""),
			},
			positionalArg: "image",
			stringFlags: map[string]string{
				"osv_url": "--osv-url",
			},
			boolFlags: map[string]string{"local": "--local"},
		},
		{
			name:        "runtz_k8s",
			subcommand:  "k8s",
			description: "Scan a Kubernetes cluster (through kubectl) or a directory of manifests for workload and RBAC posture findings.",
			properties: map[string]any{
				"path":           strProp("Optional manifest file or directory to scan instead of the live cluster.", ""),
				"target":         strProp("Target name shown in the platform.", ""),
				"namespace":      strProp("Namespace to scan instead of all namespaces.", ""),
				"context":        strProp("Kubernetes context override.", ""),
				"kubeconfig":     strProp("Kubeconfig path.", ""),
				"all_namespaces": boolProp("Scan all namespaces when namespace is not set (default true)."),
			},
			positionalArg: "path",
			stringFlags: map[string]string{
				"target":     "--target",
				"namespace":  "--namespace",
				"context":    "--context",
				"kubeconfig": "--kubeconfig",
			},
			boolFlags: map[string]string{"all_namespaces": "--all-namespaces"},
		},
	}

	for _, spec := range specs {
		spec := spec
		s.AddTool(&mcp.Tool{
			Name:        spec.name,
			Description: spec.description,
			InputSchema: objectSchema(spec.properties),
			Handler: func(ctx context.Context, arguments json.RawMessage) (string, error) {
				flags, err := spec.flags(arguments)
				if err != nil {
					return "", err
				}
				res, err := runner.Run(ctx, spec.subcommand, flags)
				if err != nil {
					return "", err
				}
				return res.String(), nil
			},
		})
	}
}

// flags translates the tool arguments into ordered CLI flags.
func (spec scanSpec) flags(arguments json.RawMessage) ([]string, error) {
	args := map[string]any{}
	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
	}

	var flags []string
	if spec.positionalArg != "" {
		if raw, ok := args[spec.positionalArg]; ok {
			if value := strings.TrimSpace(fmt.Sprint(raw)); value != "" {
				flags = append(flags, value)
			}
		}
	}
	for key, flag := range spec.stringFlags {
		if raw, ok := args[key]; ok {
			if value := strings.TrimSpace(fmt.Sprint(raw)); value != "" {
				flags = append(flags, flag, value)
			}
		}
	}
	for key, flag := range spec.boolFlags {
		if raw, ok := args[key]; ok {
			if b, isBool := raw.(bool); isBool {
				flags = append(flags, fmt.Sprintf("%s=%t", flag, b))
			}
		}
	}
	return flags, nil
}

// --- documentation tools ---

func registerDocTools(s *mcp.Server, docs *runtz.Docs) {
	s.AddTool(&mcp.Tool{
		Name:        "runtz_docs_list",
		Description: "List the available runtz documentation pages (offline snapshot) with their titles and summaries.",
		InputSchema: objectSchema(nil),
		Handler: func(ctx context.Context, _ json.RawMessage) (string, error) {
			var b strings.Builder
			b.WriteString("runtz documentation pages:\n\n")
			for _, doc := range docs.List() {
				fmt.Fprintf(&b, "- %s (slug: %s)\n", doc.Title, doc.Slug)
				if doc.Summary != "" {
					fmt.Fprintf(&b, "  %s\n", doc.Summary)
				}
			}
			b.WriteString("\nUse runtz_docs_read with a slug to read a full page.")
			return b.String(), nil
		},
	})

	s.AddTool(&mcp.Tool{
		Name:        "runtz_docs_read",
		Description: "Read a full runtz documentation page by its slug (use runtz_docs_list to discover slugs).",
		InputSchema: objectSchema(map[string]any{
			"slug": strProp("Documentation page slug, e.g. cli or scan-container.", ""),
		}, "slug"),
		Handler: func(ctx context.Context, arguments json.RawMessage) (string, error) {
			var in struct {
				Slug string `json:"slug"`
			}
			if err := json.Unmarshal(arguments, &in); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}
			doc, ok := docs.Get(in.Slug)
			if !ok {
				return "", fmt.Errorf("no doc with slug %q (call runtz_docs_list to see slugs)", in.Slug)
			}
			return fmt.Sprintf("# %s\n\n%s", doc.Title, doc.Body), nil
		},
	})

	s.AddTool(&mcp.Tool{
		Name:        "runtz_docs_search",
		Description: "Search the runtz documentation for a query and return matching pages with snippets.",
		InputSchema: objectSchema(map[string]any{
			"query": strProp("Search terms, e.g. 'container scan token'.", ""),
		}, "query"),
		Handler: func(ctx context.Context, arguments json.RawMessage) (string, error) {
			var in struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal(arguments, &in); err != nil {
				return "", fmt.Errorf("invalid arguments: %w", err)
			}
			hits := docs.Search(in.Query)
			if len(hits) == 0 {
				return fmt.Sprintf("No documentation pages matched %q.", in.Query), nil
			}
			var b strings.Builder
			fmt.Fprintf(&b, "%d page(s) matched %q:\n\n", len(hits), in.Query)
			for _, hit := range hits {
				fmt.Fprintf(&b, "- %s (slug: %s)\n  %s\n", hit.Doc.Title, hit.Doc.Slug, hit.Snippet)
			}
			return b.String(), nil
		},
	})
}

func registerDocResources(s *mcp.Server, docs *runtz.Docs) {
	for _, doc := range docs.List() {
		doc := doc
		s.AddResource(&mcp.Resource{
			URI:         runtz.ResourceURI(doc.Slug),
			Name:        doc.Title,
			Description: doc.Summary,
			MIMEType:    "text/markdown",
			Read: func(ctx context.Context) (string, error) {
				return fmt.Sprintf("# %s\n\n%s", doc.Title, doc.Body), nil
			},
		})
	}
}

// --- schema helpers ---

func objectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{"type": "object"}
	if len(properties) > 0 {
		schema["properties"] = properties
	} else {
		schema["properties"] = map[string]any{}
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func strProp(description, def string) map[string]any {
	p := map[string]any{"type": "string", "description": description}
	if def != "" {
		p["default"] = def
	}
	return p
}

func boolProp(description string) map[string]any {
	return map[string]any{"type": "boolean", "description": description}
}
