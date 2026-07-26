package runtz

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// docsFS is an offline snapshot of the runtz documentation, embedded at build
// time. Rebuild the binary to refresh it. Keeping the docs in-binary means the
// server never needs network access to answer documentation questions.
//
//go:embed docs/*.md
var docsFS embed.FS

// Doc is a single documentation page.
type Doc struct {
	Slug    string
	Title   string
	Summary string
	Body    string
}

// Docs is the loaded documentation set, indexed by slug.
type Docs struct {
	ordered []Doc
	bySlug  map[string]Doc
}

// LoadDocs parses the embedded snapshot. It is cheap and safe to call once at
// startup.
func LoadDocs() (*Docs, error) {
	entries, err := fs.ReadDir(docsFS, "docs")
	if err != nil {
		return nil, err
	}

	d := &Docs{bySlug: map[string]Doc{}}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		raw, err := docsFS.ReadFile("docs/" + entry.Name())
		if err != nil {
			return nil, err
		}
		slug := strings.TrimSuffix(entry.Name(), ".md")
		doc := parseDoc(slug, string(raw))
		d.ordered = append(d.ordered, doc)
		d.bySlug[slug] = doc
	}

	sort.Slice(d.ordered, func(i, j int) bool { return d.ordered[i].Slug < d.ordered[j].Slug })
	return d, nil
}

// List returns all docs in slug order.
func (d *Docs) List() []Doc { return d.ordered }

// Get returns a doc by slug.
func (d *Docs) Get(slug string) (Doc, bool) {
	doc, ok := d.bySlug[strings.TrimSpace(slug)]
	return doc, ok
}

// Search returns docs whose title or body contains every whitespace-separated
// term in query (case-insensitive), each with a short matching snippet.
func (d *Docs) Search(query string) []SearchHit {
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return nil
	}

	var hits []SearchHit
	for _, doc := range d.ordered {
		haystack := strings.ToLower(doc.Title + "\n" + doc.Body)
		matchesAll := true
		for _, term := range terms {
			if !strings.Contains(haystack, term) {
				matchesAll = false
				break
			}
		}
		if matchesAll {
			hits = append(hits, SearchHit{Doc: doc, Snippet: snippet(doc.Body, terms[0])})
		}
	}
	return hits
}

// SearchHit pairs a matching doc with a snippet around the first term.
type SearchHit struct {
	Doc     Doc
	Snippet string
}

// parseDoc extracts the title/summary from optional YAML-ish frontmatter and the
// remaining body. It avoids a YAML dependency by reading the two keys it needs.
func parseDoc(slug, raw string) Doc {
	doc := Doc{Slug: slug, Body: raw}

	if strings.HasPrefix(raw, "---") {
		if end := strings.Index(raw[3:], "---"); end >= 0 {
			frontmatter := raw[3 : 3+end]
			doc.Body = strings.TrimLeft(raw[3+end+3:], "\n")
			for _, line := range strings.Split(frontmatter, "\n") {
				key, value, ok := strings.Cut(line, ":")
				if !ok {
					continue
				}
				value = strings.TrimSpace(strings.Trim(strings.TrimSpace(value), `"'`))
				switch strings.TrimSpace(key) {
				case "title":
					doc.Title = value
				case "description":
					doc.Summary = value
				}
			}
		}
	}

	if doc.Title == "" {
		doc.Title = deriveTitle(slug, doc.Body)
	}
	return doc
}

func deriveTitle(slug, body string) string {
	for _, line := range strings.Split(body, "\n") {
		if h := strings.TrimSpace(line); strings.HasPrefix(h, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(h, "# "))
		}
	}
	return strings.ReplaceAll(slug, "-", " ")
}

func snippet(body, term string) string {
	lower := strings.ToLower(body)
	idx := strings.Index(lower, term)
	if idx < 0 {
		return firstLine(body)
	}
	start := idx - 80
	if start < 0 {
		start = 0
	}
	end := idx + 160
	if end > len(body) {
		end = len(body)
	}
	out := strings.TrimSpace(strings.ReplaceAll(body[start:end], "\n", " "))
	if start > 0 {
		out = "…" + out
	}
	if end < len(body) {
		out += "…"
	}
	return out
}

func firstLine(body string) string {
	for _, line := range strings.Split(body, "\n") {
		if l := strings.TrimSpace(line); l != "" && !strings.HasPrefix(l, "#") {
			return l
		}
	}
	return ""
}

// ResourceURI returns the canonical MCP resource URI for a doc slug.
func ResourceURI(slug string) string { return fmt.Sprintf("runtz-docs://%s", slug) }
