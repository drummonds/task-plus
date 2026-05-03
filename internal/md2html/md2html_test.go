package md2html

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDocsRootURL(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	research := filepath.Join(docs, "research")
	_ = os.MkdirAll(research, 0755)

	_ = os.WriteFile(filepath.Join(docs, "index.md"), []byte("# Index"), 0644)

	if got := docsRootURL(docs); got != "index.html" {
		t.Errorf("docsRootURL(docs) = %q, want %q", got, "index.html")
	}

	if got := docsRootURL(research); got != "../index.html" {
		t.Errorf("docsRootURL(research) = %q, want %q", got, "../index.html")
	}
}

func TestBreadcrumbs(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	nested := filepath.Join(docs, "example")
	_ = os.MkdirAll(nested, 0755)

	// Create index.md at docs root so docsRootURL can find it.
	_ = os.WriteFile(filepath.Join(docs, "index.md"), []byte("# Home\n"), 0644)

	// Root index: single active "Home" crumb.
	t.Run("root_index", func(t *testing.T) {
		cfg := Config{Src: docs, Dst: docs, Project: "test"}
		_ = os.WriteFile(filepath.Join(docs, "index.md"), []byte("# Home\n"), 0644)
		if err := Run(cfg); err != nil {
			t.Fatal(err)
		}
		out, _ := os.ReadFile(filepath.Join(docs, "index.html"))
		html := string(out)
		if !strings.Contains(html, `<li class="is-active"><a href="#" aria-current="page">Home</a></li>`) {
			t.Error("root index should have active Home crumb")
		}
		// Should NOT have a linked Home crumb.
		if strings.Contains(html, `<a href="index.html">Home</a>`) {
			t.Error("root index should not link Home to itself")
		}
	})

	// Root subpage: Home link + active page title.
	t.Run("root_subpage", func(t *testing.T) {
		_ = os.WriteFile(filepath.Join(docs, "about.md"), []byte("# About Us\n"), 0644)
		cfg := Config{Src: docs, Dst: docs, Project: "test", File: filepath.Join(docs, "about.md")}
		if err := Run(cfg); err != nil {
			t.Fatal(err)
		}
		out, _ := os.ReadFile(filepath.Join(docs, "about.html"))
		html := string(out)
		if !strings.Contains(html, `<a href="index.html">Home</a>`) {
			t.Error("subpage should have clickable Home link")
		}
		if !strings.Contains(html, `<li class="is-active"><a href="#" aria-current="page">About Us</a></li>`) {
			t.Error("subpage should have active page title")
		}
	})

	// Nested page: Home link with ../ prefix.
	t.Run("nested_page", func(t *testing.T) {
		_ = os.WriteFile(filepath.Join(nested, "index.md"), []byte("# Example\n"), 0644)
		cfg := Config{Src: nested, Dst: nested, Project: "test"}
		if err := Run(cfg); err != nil {
			t.Fatal(err)
		}
		out, _ := os.ReadFile(filepath.Join(nested, "index.html"))
		html := string(out)
		if !strings.Contains(html, `<a href="../index.html">Home</a>`) {
			t.Error("nested page should link Home to ../index.html")
		}
		if !strings.Contains(html, `<li class="is-active"><a href="#" aria-current="page">Example</a></li>`) {
			t.Error("nested page should have active page title")
		}
	})

	// NoBreadcrumbs: no breadcrumb nav at all.
	t.Run("no_breadcrumbs", func(t *testing.T) {
		_ = os.WriteFile(filepath.Join(docs, "wasm.md"), []byte("# WASM Demo\n"), 0644)
		cfg := Config{Src: docs, Dst: docs, Project: "test", NoBreadcrumbs: true, File: filepath.Join(docs, "wasm.md")}
		if err := Run(cfg); err != nil {
			t.Fatal(err)
		}
		out, _ := os.ReadFile(filepath.Join(docs, "wasm.html"))
		html := string(out)
		if strings.Contains(html, "breadcrumb") {
			t.Error("NoBreadcrumbs should suppress all breadcrumb markup")
		}
	})
}

func TestMarkerReplacementEndToEnd(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "out")
	_ = os.MkdirAll(dst, 0755)

	// Create an existing HTML page so auto:pages has something to list.
	_ = os.WriteFile(filepath.Join(dst, "guide.html"),
		[]byte("<html><head><title>Guide</title></head><body></body></html>"), 0644)

	// Create a markdown file with markers.
	md := "# My Docs\n\n<!-- auto:pages -->\n<!-- /auto:pages -->\n\nSome text.\n"
	_ = os.WriteFile(filepath.Join(dir, "index.md"), []byte(md), 0644)

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

func TestIncrementalRebuild(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "page.md")
	outPath := filepath.Join(dir, "page.html")

	if err := os.WriteFile(srcPath, []byte("# Page\n\noriginal\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{Src: dir, Dst: dir, Project: "test", NoBreadcrumbs: true}
	if err := Run(cfg); err != nil {
		t.Fatal(err)
	}
	firstOut, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("first build did not produce output: %v", err)
	}

	// Force output mtime ahead of source so a rebuild would only happen if Force is set.
	future := time.Now().Add(2 * time.Hour)
	if err := os.Chtimes(outPath, future, future); err != nil {
		t.Fatal(err)
	}
	// Mutate source content but keep mtime older than output.
	if err := os.WriteFile(srcPath, []byte("# Page\n\nchanged\n"), 0644); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(srcPath, past, past); err != nil {
		t.Fatal(err)
	}

	// Default: skip — output should still match the first build.
	if err := Run(cfg); err != nil {
		t.Fatal(err)
	}
	skipped, _ := os.ReadFile(outPath)
	if string(skipped) != string(firstOut) {
		t.Error("expected output to be skipped when output mtime > source mtime")
	}

	// Force: rebuild even when output is newer.
	cfg.Force = true
	if err := Run(cfg); err != nil {
		t.Fatal(err)
	}
	rebuilt, _ := os.ReadFile(outPath)
	if !strings.Contains(string(rebuilt), "changed") {
		t.Error("Force=true should rebuild and pick up new source content")
	}

	// Source newer than output: rebuild without Force.
	cfg.Force = false
	if err := os.WriteFile(srcPath, []byte("# Page\n\nfresh\n"), 0644); err != nil {
		t.Fatal(err)
	}
	stale := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(outPath, stale, stale); err != nil {
		t.Fatal(err)
	}
	if err := Run(cfg); err != nil {
		t.Fatal(err)
	}
	fresh, _ := os.ReadFile(outPath)
	if !strings.Contains(string(fresh), "fresh") {
		t.Error("rebuild expected when source mtime > output mtime")
	}

	// Output missing: rebuild without Force.
	if err := os.Remove(outPath); err != nil {
		t.Fatal(err)
	}
	if err := Run(cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("missing output should be rebuilt: %v", err)
	}
}
