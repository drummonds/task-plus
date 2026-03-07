package migrate

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/drummonds/task-plus/internal/config"
	"github.com/drummonds/task-plus/internal/deploy"
	"gopkg.in/yaml.v3"
)

// Run creates a -docs sibling repo for the project at dir.
// If docs/ exists, copies its contents across. Otherwise scaffolds a fresh docs repo.
func Run(dir string) error {
	cfg, err := config.Load(dir)
	if err != nil {
		return err
	}

	if cfg.IsDocs() {
		return fmt.Errorf("already a docs project — nothing to migrate")
	}

	docsRepoDir := config.DocsRepoPath(dir)
	if _, err := os.Stat(docsRepoDir); err == nil {
		return fmt.Errorf("%s already exists", docsRepoDir)
	}

	projectName := filepath.Base(dir)
	docsRepoName := projectName + "-docs"
	fmt.Printf("Creating %s\n", docsRepoDir)

	if err := os.MkdirAll(filepath.Join(docsRepoDir, "docs"), 0755); err != nil {
		return err
	}

	// Copy existing docs/ if present
	srcDocs := filepath.Join(dir, "docs")
	hasDocs := config.HasDocsDir(dir)
	if hasDocs {
		fmt.Println("  Copying docs/ contents...")
		if err := copyDir(srcDocs, filepath.Join(docsRepoDir, "docs")); err != nil {
			return fmt.Errorf("copying docs: %w", err)
		}
	} else {
		fmt.Println("  Creating template docs/index.html...")
		if err := writeTemplateIndex(filepath.Join(docsRepoDir, "docs", "index.html"), projectName); err != nil {
			return err
		}
	}

	// Create task-plus.yml in docs repo
	docsCfg := docsConfig(cfg)
	if err := writeDocsConfig(docsRepoDir, docsCfg); err != nil {
		return err
	}

	// Create Taskfile.yml
	if err := writeDocsTaskfile(docsRepoDir); err != nil {
		return err
	}

	// Create README.md
	if err := writeDocsReadme(docsRepoDir, projectName, docsRepoName); err != nil {
		return err
	}

	// Create .gitignore
	if err := os.WriteFile(filepath.Join(docsRepoDir, ".gitignore"), []byte("dist/\n"), 0644); err != nil {
		return err
	}

	// Git init
	cmd := exec.Command("git", "init")
	cmd.Dir = docsRepoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	fmt.Printf("\nCreated %s\n", docsRepoDir)
	fmt.Println("Next steps:")
	fmt.Println("  1. Review the generated files")
	fmt.Println("  2. Run 'tp pages' from either repo to test")
	fmt.Println("  3. Run 'tp pages migrate clean' to remove docs/ from the main repo")
	return nil
}

// Clean removes docs/ and pages config from the main project after migration.
func Clean(dir string) error {
	cfg, err := config.Load(dir)
	if err != nil {
		return err
	}

	if cfg.IsDocs() {
		return fmt.Errorf("this is a docs project — run clean from the main project")
	}

	docsRepoDir := cfg.ResolveDocsRepo()
	if docsRepoDir == "" {
		return fmt.Errorf("no -docs sibling found — run 'tp pages migrate' first")
	}

	// Verify -docs repo has been committed
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = docsRepoDir
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("checking -docs repo status: %w", err)
	}
	if len(strings.TrimSpace(string(out))) > 0 {
		return fmt.Errorf("%s has uncommitted changes — commit first before cleaning", docsRepoDir)
	}

	// Remove docs/ from main repo
	docsDir := filepath.Join(dir, "docs")
	if config.HasDocsDir(dir) {
		fmt.Printf("  Removing %s\n", docsDir)
		if err := os.RemoveAll(docsDir); err != nil {
			return fmt.Errorf("removing docs/: %w", err)
		}
	}

	// Remove pages_build and pages_deploy from main task-plus.yml, add docs_repo if needed
	if err := cleanMainConfig(dir, docsRepoDir); err != nil {
		return fmt.Errorf("updating config: %w", err)
	}

	fmt.Println("  Done. Main repo cleaned.")
	fmt.Println("  Commit the changes when ready.")
	return nil
}

// docsConfig builds the task-plus.yml content for the -docs repo.
func docsConfig(mainCfg *config.Config) docsYAML {
	d := docsYAML{
		Type:       "docs",
		ParentRepo: "../" + filepath.Base(mainCfg.Dir),
	}
	if len(mainCfg.PagesBuild) > 0 {
		d.PagesBuild = mainCfg.PagesBuild
	}
	if len(mainCfg.PagesDeploy) > 0 {
		d.PagesDeploy = mainCfg.PagesDeploy
	}
	return d
}

type docsYAML struct {
	Type        string          `yaml:"type"`
	ParentRepo  string          `yaml:"parent_repo"`
	PagesBuild  []string        `yaml:"pages_build,omitempty"`
	PagesDeploy []deploy.Target `yaml:"pages_deploy,omitempty"`
}

func writeDocsConfig(dir string, cfg docsYAML) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "task-plus.yml"), data, 0644)
}

func writeDocsTaskfile(dir string) error {
	content := `version: "3"

tasks:
  docs:build:
    desc: Build documentation (markdown to HTML)
    cmds:
      - task-plus md2html

  clean:
    desc: Remove generated files
    cmds:
      - rm -f docs/*.html
`
	return os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(content), 0644)
}

func writeDocsReadme(dir, projectName, repoName string) error {
	content := fmt.Sprintf(`# %s

Documentation for [%s](https://github.com/drummonds/%s).

Built and deployed using [task-plus](https://github.com/drummonds/task-plus).

## Local preview

    tp pages

## Deploy

    tp pages deploy
`, repoName, projectName, projectName)
	return os.WriteFile(filepath.Join(dir, "README.md"), []byte(content), 0644)
}

func writeTemplateIndex(path, projectName string) error {
	content := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>%s Documentation</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bulma@0.9.4/css/bulma.min.css">
</head>
<body>
  <section class="section">
    <div class="container">
      <h1 class="title">%s</h1>
      <p>Documentation coming soon.</p>
    </div>
  </section>
</body>
</html>
`, projectName, projectName)
	return os.WriteFile(path, []byte(content), 0644)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

// cleanMainConfig removes pages_build and pages_deploy from task-plus.yml
// and ensures docs_repo is set if not using the default convention.
func cleanMainConfig(dir, docsRepoDir string) error {
	path := filepath.Join(dir, "task-plus.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil // no config to clean
	}

	lines := strings.Split(string(data), "\n")
	var result []string
	skip := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip pages_build and pages_deploy blocks
		if trimmed == "pages_build:" || strings.HasPrefix(trimmed, "pages_build:") {
			skip = true
			continue
		}
		if trimmed == "pages_deploy:" || strings.HasPrefix(trimmed, "pages_deploy:") {
			skip = true
			continue
		}
		if skip {
			// Still in a list/block if indented
			if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
				continue
			}
			skip = false
		}
		result = append(result, line)
	}

	// Check if docs_repo needs to be explicit (non-conventional name)
	conventionalPath := config.DocsRepoPath(dir)
	if docsRepoDir != conventionalPath {
		rel, err := filepath.Rel(dir, docsRepoDir)
		if err != nil {
			rel = docsRepoDir
		}
		result = append(result, fmt.Sprintf("docs_repo: %s", rel))
	}

	return os.WriteFile(path, []byte(strings.Join(result, "\n")), 0644)
}
