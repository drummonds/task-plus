package md2html

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratePagesTable(t *testing.T) {
	dir := t.TempDir()

	// Create some HTML files with <title> tags.
	for _, f := range []struct {
		name, title string
	}{
		{"alpha.html", "Alpha Page"},
		{"beta.html", "Beta Page"},
	} {
		content := "<html><head><title>" + f.title + "</title></head><body></body></html>"
		os.WriteFile(filepath.Join(dir, f.name), []byte(content), 0644)
	}
	// index.html should be excluded.
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0644)

	table := GeneratePagesTable(dir)
	if !strings.Contains(table, "Alpha Page") {
		t.Errorf("expected Alpha Page in table, got:\n%s", table)
	}
	if !strings.Contains(table, "beta.html") {
		t.Errorf("expected beta.html in table, got:\n%s", table)
	}
	if strings.Contains(table, "index.html") {
		t.Errorf("index.html should be excluded from pages table")
	}
}

func TestGeneratePagesTableEmpty(t *testing.T) {
	dir := t.TempDir()
	table := GeneratePagesTable(dir)
	if table != "\n" {
		t.Errorf("expected single newline for empty dir, got: %q", table)
	}
}

func TestMarkerReplacementEndToEnd(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "out")
	os.MkdirAll(dst, 0755)

	// Create an existing HTML page so auto:pages has something to list.
	os.WriteFile(filepath.Join(dst, "guide.html"),
		[]byte("<html><head><title>Guide</title></head><body></body></html>"), 0644)

	// Create a markdown file with markers.
	md := "# My Docs\n\n<!-- auto:pages -->\n<!-- /auto:pages -->\n\nSome text.\n"
	os.WriteFile(filepath.Join(dir, "index.md"), []byte(md), 0644)

	cfg := Config{
		Src:  dir,
		Dst:  dst,
		File: filepath.Join(dir, "index.md"),
	}
	if err := Run(cfg); err != nil {
		t.Fatalf("Run: %v", err)
	}

	out, err := os.ReadFile(filepath.Join(dst, "index.html"))
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	html := string(out)

	if !strings.Contains(html, "Guide") {
		t.Errorf("expected Guide in output, got:\n%s", html)
	}
	if !strings.Contains(html, "guide.html") {
		t.Errorf("expected guide.html link in output, got:\n%s", html)
	}
	if !strings.Contains(html, "Some text.") {
		t.Errorf("expected hand-written content preserved, got:\n%s", html)
	}
}
