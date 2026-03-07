package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun_NoExistingDocs(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "myproject")
	os.MkdirAll(projectDir, 0755)
	os.WriteFile(filepath.Join(projectDir, "task-plus.yml"), []byte("type: library\n"), 0644)

	if err := Run(projectDir); err != nil {
		t.Fatal(err)
	}

	docsRepo := filepath.Join(parent, "myproject-docs")
	// Verify created files
	for _, f := range []string{"task-plus.yml", "Taskfile.yml", "README.md", ".gitignore"} {
		if _, err := os.Stat(filepath.Join(docsRepo, f)); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}
	// Verify docs/index.html template created
	if _, err := os.Stat(filepath.Join(docsRepo, "docs", "index.html")); err != nil {
		t.Error("missing docs/index.html")
	}
}

func TestRun_WithExistingDocs(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "myproject")
	os.MkdirAll(filepath.Join(projectDir, "docs"), 0755)
	os.WriteFile(filepath.Join(projectDir, "docs", "test.html"), []byte("<html>test</html>"), 0644)
	os.WriteFile(filepath.Join(projectDir, "task-plus.yml"), []byte("pages_deploy:\n  - type: statichost\n    site: mysite\n"), 0644)

	if err := Run(projectDir); err != nil {
		t.Fatal(err)
	}

	docsRepo := filepath.Join(parent, "myproject-docs")
	// Verify docs were copied
	if _, err := os.Stat(filepath.Join(docsRepo, "docs", "test.html")); err != nil {
		t.Error("docs/test.html not copied")
	}
	// Verify config has deploy target
	data, err := os.ReadFile(filepath.Join(docsRepo, "task-plus.yml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !contains(content, "statichost") {
		t.Error("docs task-plus.yml should contain statichost deploy target")
	}
	if !contains(content, "docs") {
		t.Error("docs task-plus.yml should have type: docs")
	}
}

func TestRun_AlreadyDocs(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("type: docs\nparent_repo: ../foo\n"), 0644)

	err := Run(dir)
	if err == nil {
		t.Fatal("expected error for docs project")
	}
}

func TestRun_AlreadyExists(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "myproject")
	docsDir := filepath.Join(parent, "myproject-docs")
	os.MkdirAll(projectDir, 0755)
	os.MkdirAll(docsDir, 0755)
	os.WriteFile(filepath.Join(projectDir, "task-plus.yml"), []byte("type: library\n"), 0644)

	err := Run(projectDir)
	if err == nil {
		t.Fatal("expected error when -docs already exists")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
