package runtz

import "testing"

func TestLoadDocs(t *testing.T) {
	docs, err := LoadDocs()
	if err != nil {
		t.Fatalf("LoadDocs: %v", err)
	}
	if len(docs.List()) == 0 {
		t.Fatal("expected embedded docs, got none")
	}

	cli, ok := docs.Get("cli")
	if !ok {
		t.Fatal("expected a 'cli' doc")
	}
	if cli.Title == "" {
		t.Error("expected the cli doc to have a title")
	}
	if cli.Body == "" {
		t.Error("expected the cli doc to have a body")
	}
}

func TestParseDocFrontmatter(t *testing.T) {
	doc := parseDoc("cli", "---\ntitle: CLI\ndescription: Use the runtz CLI.\n---\n\n# CLI\n\nBody text.")
	if doc.Title != "CLI" {
		t.Errorf("title = %q, want CLI", doc.Title)
	}
	if doc.Summary != "Use the runtz CLI." {
		t.Errorf("summary = %q", doc.Summary)
	}
	if got := doc.Body; got == "" || got[0] == '-' {
		t.Errorf("body should not include frontmatter: %q", got)
	}
}

func TestSearch(t *testing.T) {
	docs, err := LoadDocs()
	if err != nil {
		t.Fatalf("LoadDocs: %v", err)
	}
	hits := docs.Search("container image")
	if len(hits) == 0 {
		t.Fatal("expected at least one hit for 'container image'")
	}
}
