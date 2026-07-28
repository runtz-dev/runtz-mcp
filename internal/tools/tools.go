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

// Register adds every runtz documentation tool and resource to the server.
func Register(s *mcp.Server, docs *runtz.Docs) {
	registerDocTools(s, docs)
	registerDocResources(s, docs)
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
