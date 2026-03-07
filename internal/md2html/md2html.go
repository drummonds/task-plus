// Package md2html converts markdown files to Bulma-styled HTML pages.
package md2html

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

//go:embed template.html index.html
var templateFS embed.FS

// Config holds the parameters for a conversion run.
type Config struct {
	Src      string // source markdown directory
	Dst      string // destination HTML directory
	Label    string // breadcrumb label for this doc set
	Project  string // project name for breadcrumb root
	File     string // single file to convert (overrides Src directory scan)
	Index    bool   // generate index.html listing all pages
	Subtitle string // subtitle for the index page
}

// pageInfo holds metadata about a converted page for the index.
type pageInfo struct {
	Title    string
	Filename string
}

type breadcrumb struct {
	Label string
	URL   string
}

type pageData struct {
	Title       string
	Project     string
	Content     template.HTML
	Breadcrumbs []breadcrumb
	HasMermaid  bool
}

// Run converts all .md files in Src to .html files in Dst.
func Run(cfg Config) error {
	if cfg.Project == "" {
		cfg.Project = detectProject()
	}
	if cfg.Subtitle == "" {
		cfg.Subtitle = "Documentation"
	}

	tmplBytes, err := templateFS.ReadFile("template.html")
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}
	tmpl, err := template.New("page").Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)

	if err := os.MkdirAll(cfg.Dst, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", cfg.Dst, err)
	}

	// Single file mode
	if cfg.File != "" {
		cfg.Src = filepath.Dir(cfg.File)
		name := filepath.Base(cfg.File)
		if _, err := convertFile(md, tmpl, cfg, name); err != nil {
			return fmt.Errorf("convert %s: %w", name, err)
		}
		return nil
	}

	entries, err := os.ReadDir(cfg.Src)
	if err != nil {
		return fmt.Errorf("read %s: %w", cfg.Src, err)
	}

	var pages []pageInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		// Skip _index.md from normal conversion; it's used as index intro content.
		if entry.Name() == "_index.md" {
			continue
		}
		info, err := convertFile(md, tmpl, cfg, entry.Name())
		if err != nil {
			return fmt.Errorf("convert %s: %w", entry.Name(), err)
		}
		pages = append(pages, info)
	}

	if cfg.Index {
		if err := generateIndex(md, cfg, pages); err != nil {
			return fmt.Errorf("generate index: %w", err)
		}
	}

	return nil
}

func convertFile(md goldmark.Markdown, tmpl *template.Template, cfg Config, name string) (pageInfo, error) {
	content, err := os.ReadFile(filepath.Join(cfg.Src, name))
	if err != nil {
		return pageInfo{}, err
	}

	title := extractTitle(content, name)

	var buf bytes.Buffer
	if err := md.Convert(content, &buf); err != nil {
		return pageInfo{}, fmt.Errorf("goldmark: %w", err)
	}

	rendered := replaceMermaidBlocks(buf.String())
	hasMermaid := mermaidBlockRe.MatchString(buf.String())

	outName := strings.TrimSuffix(name, ".md") + ".html"

	// The output file always lands in cfg.Dst alongside index.html,
	// so the breadcrumb root is always a sibling link.
	rootURL := "index.html"

	data := pageData{
		Title:   title,
		Project: cfg.Project,
		Content: template.HTML(rendered),
		Breadcrumbs: []breadcrumb{
			{Label: cfg.Project, URL: rootURL},
			{Label: cfg.Label, URL: ""},
			{Label: title, URL: ""},
		},
		HasMermaid: hasMermaid,
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return pageInfo{}, fmt.Errorf("template: %w", err)
	}

	outPath := filepath.Join(cfg.Dst, outName)
	if err := os.WriteFile(outPath, out.Bytes(), 0o644); err != nil {
		return pageInfo{}, err
	}
	fmt.Printf("%s -> %s\n", filepath.Join(cfg.Src, name), outPath)
	return pageInfo{Title: title, Filename: outName}, nil
}

// indexData holds the template data for the index page.
type indexData struct {
	Project  string
	Subtitle string
	Intro    template.HTML
	Pages    []pageInfo
}

// generateIndex creates an index.html listing all converted pages.
func generateIndex(md goldmark.Markdown, cfg Config, pages []pageInfo) error {
	idxBytes, err := templateFS.ReadFile("index.html")
	if err != nil {
		return fmt.Errorf("read index template: %w", err)
	}
	idxTmpl, err := template.New("index").Parse(string(idxBytes))
	if err != nil {
		return fmt.Errorf("parse index template: %w", err)
	}

	// Render optional _index.md intro content.
	var intro template.HTML
	introPath := filepath.Join(cfg.Src, "_index.md")
	if introContent, err := os.ReadFile(introPath); err == nil {
		var buf bytes.Buffer
		if err := md.Convert(introContent, &buf); err == nil {
			intro = template.HTML(buf.String())
		}
	}

	data := indexData{
		Project:  cfg.Project,
		Subtitle: cfg.Subtitle,
		Intro:    intro,
		Pages:    pages,
	}

	var out bytes.Buffer
	if err := idxTmpl.Execute(&out, data); err != nil {
		return fmt.Errorf("template: %w", err)
	}

	outPath := filepath.Join(cfg.Dst, "index.html")
	if err := os.WriteFile(outPath, out.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Printf("index -> %s\n", outPath)
	return nil
}

func extractTitle(content []byte, fallback string) string {
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return strings.TrimSuffix(fallback, ".md")
}

// mermaidBlockRe matches <pre><code class="language-mermaid">...</code></pre> blocks
// produced by goldmark from ```mermaid fenced code.
var mermaidBlockRe = regexp.MustCompile(`(?s)<pre><code class="language-mermaid">(.*?)</code></pre>`)

// htmlEntityDecoder restores HTML entities back to plain text for mermaid.js.
var htmlEntityDecoder = strings.NewReplacer(
	"&gt;", ">", "&lt;", "<", "&amp;", "&", "&quot;", `"`,
)

// replaceMermaidBlocks converts goldmark's mermaid code blocks into
// <pre class="mermaid"> elements for client-side rendering by mermaid.js.
func replaceMermaidBlocks(htmlStr string) string {
	return mermaidBlockRe.ReplaceAllStringFunc(htmlStr, func(match string) string {
		subs := mermaidBlockRe.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		src := htmlEntityDecoder.Replace(subs[1])
		return `<pre class="mermaid">` + src + `</pre>`
	})
}

// detectProject parses go.mod in CWD to extract the last path element of the module name.
func detectProject() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "project"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "module ") {
			mod := strings.TrimPrefix(line, "module ")
			mod = strings.TrimSpace(mod)
			parts := strings.Split(mod, "/")
			return parts[len(parts)-1]
		}
	}
	return "project"
}
