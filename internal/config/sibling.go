package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveDocsRepo finds the -docs sibling repo for a project directory.
// Checks (in order): explicit docs_repo config, then ../<basename>-docs convention.
// Returns the absolute path to the docs repo, or empty string if not found.
func (c *Config) ResolveDocsRepo() string {
	if c.DocsRepo != "" {
		abs := c.DocsRepo
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(c.Dir, abs)
		}
		if isDocsRepo(abs) {
			return abs
		}
		return ""
	}
	// Convention: ../<project>-docs
	candidate := docsRepoPath(c.Dir)
	if isDocsRepo(candidate) {
		return candidate
	}
	return ""
}

// ResolveParentRepo finds the parent project repo from a -docs repo.
// Checks (in order): explicit parent_repo config, then ../<basename-without-docs> convention.
// Returns the absolute path to the parent repo, or empty string if not found.
func (c *Config) ResolveParentRepo() string {
	if c.ParentRepo != "" {
		abs := c.ParentRepo
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(c.Dir, abs)
		}
		if isProjectRepo(abs) {
			return abs
		}
		return ""
	}
	// Convention: strip -docs suffix
	base := filepath.Base(c.Dir)
	if !strings.HasSuffix(base, "-docs") {
		return ""
	}
	parent := filepath.Join(filepath.Dir(c.Dir), strings.TrimSuffix(base, "-docs"))
	if isProjectRepo(parent) {
		return parent
	}
	return ""
}

// docsRepoPath returns the conventional -docs sibling path for a project directory.
func docsRepoPath(projectDir string) string {
	return filepath.Join(filepath.Dir(projectDir), filepath.Base(projectDir)+"-docs")
}

// isDocsRepo checks if dir exists and contains a task-plus.yml.
func isDocsRepo(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, configFile))
	return err == nil
}

// isProjectRepo checks if dir exists and looks like a Go project (has go.mod or task-plus.yml).
func isProjectRepo(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	for _, f := range []string{"go.mod", configFile} {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			return true
		}
	}
	return false
}

// DocsRepoPath returns the conventional -docs sibling path for a project directory.
// This is used by migrate to create the repo even when it doesn't exist yet.
func DocsRepoPath(projectDir string) string {
	return docsRepoPath(projectDir)
}

// HasDocsDir returns true if the project has a docs/ directory.
func HasDocsDir(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "docs"))
	return err == nil && info.IsDir()
}

// LoadDocsRepo loads the config from the resolved -docs sibling.
// Returns nil if no sibling is found.
func (c *Config) LoadDocsRepo() (*Config, error) {
	path := c.ResolveDocsRepo()
	if path == "" {
		return nil, fmt.Errorf("no -docs sibling found for %s", c.Dir)
	}
	return Load(path)
}
