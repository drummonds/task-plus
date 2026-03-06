package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	// No goreleaser → library
	if cfg.Type != "library" {
		t.Errorf("Type = %q, want library", cfg.Type)
	}

	// No Taskfile → fallback checks
	if len(cfg.Check) != 3 {
		t.Errorf("Check = %v, want 3 fallback commands", cfg.Check)
	}

	if cfg.ChangelogFormat != "keepachangelog" {
		t.Errorf("ChangelogFormat = %q, want keepachangelog", cfg.ChangelogFormat)
	}

	if cfg.Cleanup.KeepPatches != 2 {
		t.Errorf("KeepPatches = %d, want 2", cfg.Cleanup.KeepPatches)
	}
}

func TestDetectBinary(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".goreleaser.yaml"), []byte("{}"), 0644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Type != "binary" {
		t.Errorf("Type = %q, want binary", cfg.Type)
	}
}

func TestDetectTaskfile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte("{}"), 0644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Check) != 1 || cfg.Check[0] != "task check" {
		t.Errorf("Check = %v, want [task check]", cfg.Check)
	}
}

func TestDetectSimpleChangelog(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog\n\n## 0.1.0 2026-01-01\n"), 0644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ChangelogFormat != "simple" {
		t.Errorf("ChangelogFormat = %q, want simple", cfg.ChangelogFormat)
	}
}

func TestInstallNil(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Install != nil {
		t.Error("Install should be nil by default")
	}
	if cfg.ShouldInstall() {
		t.Error("ShouldInstall should return false when nil")
	}
}

func TestInstallFromYAML(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("install: true\n"), 0644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Install == nil {
		t.Fatal("Install should not be nil")
	}
	if !cfg.ShouldInstall() {
		t.Error("ShouldInstall should return true")
	}
}

func TestPagesBuildDetectDocsBuild(t *testing.T) {
	dir := t.TempDir()
	taskfile := `version: "3"
tasks:
  docs:build:
    cmds:
      - echo build docs
  build:
    cmds:
      - go build ./...
`
	os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.PagesBuild) != 1 || cfg.PagesBuild[0] != "task docs:build" {
		t.Errorf("PagesBuild = %v, want [task docs:build]", cfg.PagesBuild)
	}
}

func TestPagesBuildDetectBuildDocsLegacy(t *testing.T) {
	dir := t.TempDir()
	taskfile := `version: "3"
tasks:
  build:docs:
    cmds:
      - echo build docs
  build:
    cmds:
      - go build ./...
`
	os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.PagesBuild) != 1 || cfg.PagesBuild[0] != "task build:docs" {
		t.Errorf("PagesBuild = %v, want [task build:docs]", cfg.PagesBuild)
	}
}

func TestPagesBuildIgnoresGenericBuild(t *testing.T) {
	dir := t.TempDir()
	taskfile := `version: "3"
tasks:
  build:
    cmds:
      - go build ./...
`
	os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.PagesBuild) != 0 {
		t.Errorf("PagesBuild = %v, want empty (generic build: should not match)", cfg.PagesBuild)
	}
}

func TestPagesBuildNone(t *testing.T) {
	dir := t.TempDir()
	taskfile := `version: "3"
tasks:
  test:
    cmds:
      - go test ./...
`
	os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.PagesBuild) != 0 {
		t.Errorf("PagesBuild = %v, want empty", cfg.PagesBuild)
	}
}

func TestPagesBuildFromYAML(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("pages_build: [\"make docs\"]\n"), 0644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.PagesBuild) != 1 || cfg.PagesBuild[0] != "make docs" {
		t.Errorf("PagesBuild = %v, want [make docs]", cfg.PagesBuild)
	}
}

func TestPagesDeployFromYAML(t *testing.T) {
	dir := t.TempDir()
	yaml := `pages_deploy:
  - type: github
  - type: statichost
    site: myproject
`
	os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte(yaml), 0644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.PagesDeploy) != 2 {
		t.Fatalf("PagesDeploy = %v, want 2 targets", cfg.PagesDeploy)
	}
	if cfg.PagesDeploy[0].Type != "github" {
		t.Errorf("PagesDeploy[0].Type = %q, want github", cfg.PagesDeploy[0].Type)
	}
	if cfg.PagesDeploy[1].Type != "statichost" {
		t.Errorf("PagesDeploy[1].Type = %q, want statichost", cfg.PagesDeploy[1].Type)
	}
	if cfg.PagesDeploy[1].Site != "myproject" {
		t.Errorf("PagesDeploy[1].Site = %q, want myproject", cfg.PagesDeploy[1].Site)
	}
	if !cfg.HasPagesDeploy() {
		t.Error("HasPagesDeploy should return true")
	}
}

func TestPagesDeployEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HasPagesDeploy() {
		t.Error("HasPagesDeploy should return false when no targets configured")
	}
}

func TestLoadYAML(t *testing.T) {
	dir := t.TempDir()
	yaml := `type: binary
check: [make test]
changelog_format: simple
cleanup:
  keep_patches: 3
  keep_minors: 10
`
	os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte(yaml), 0644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Type != "binary" {
		t.Errorf("Type = %q, want binary", cfg.Type)
	}
	if len(cfg.Check) != 1 || cfg.Check[0] != "make test" {
		t.Errorf("Check = %v, want [make test]", cfg.Check)
	}
	if cfg.ChangelogFormat != "simple" {
		t.Errorf("ChangelogFormat = %q, want simple", cfg.ChangelogFormat)
	}
	if cfg.Cleanup.KeepPatches != 3 {
		t.Errorf("KeepPatches = %d, want 3", cfg.Cleanup.KeepPatches)
	}
}
