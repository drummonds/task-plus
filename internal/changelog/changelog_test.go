package changelog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLatestVersion_Keepachangelog(t *testing.T) {
	dir := t.TempDir()
	content := "# Changelog\n\n## [Unreleased]\n\n## [0.1.69] - 2026-03-23\n\n## [0.1.68] - 2026-03-20\n"
	_ = os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(content), 0644)
	got := LatestVersion(dir)
	if got != "0.1.69" {
		t.Errorf("got %q, want %q", got, "0.1.69")
	}
}

func TestLatestVersion_Simple(t *testing.T) {
	dir := t.TempDir()
	content := "# Changelog\n\n## 0.2.0 2026-03-23\n\n## 0.1.0 2026-01-01\n"
	_ = os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(content), 0644)
	got := LatestVersion(dir)
	if got != "0.2.0" {
		t.Errorf("got %q, want %q", got, "0.2.0")
	}
}

func TestLatestVersion_NoFile(t *testing.T) {
	dir := t.TempDir()
	got := LatestVersion(dir)
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestLatestVersion_NoVersionEntries(t *testing.T) {
	dir := t.TempDir()
	content := "# Changelog\n\n## [Unreleased]\n\nNothing here yet.\n"
	_ = os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(content), 0644)
	got := LatestVersion(dir)
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestLatestVersion_Prerelease(t *testing.T) {
	dir := t.TempDir()
	content := "# Changelog\n\n## [1.0.0-beta.1] - 2026-03-23\n"
	_ = os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(content), 0644)
	got := LatestVersion(dir)
	if got != "1.0.0-beta.1" {
		t.Errorf("got %q, want %q", got, "1.0.0-beta.1")
	}
}
