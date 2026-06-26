package vscode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func readFolders(t *testing.T, wsPath string) []WorkspaceFolder {
	t.Helper()
	ws, existed, err := loadWorkspace(wsPath)
	if err != nil {
		t.Fatalf("loadWorkspace: %v", err)
	}
	if !existed {
		t.Fatalf("workspace %s does not exist", wsPath)
	}
	return ws.Folders
}

func TestAddAndRemoveWorktreeFolder(t *testing.T) {
	base := t.TempDir()
	repoRoot := filepath.Join(base, "proj")
	wt := filepath.Join(base, "proj-WTfoo")
	wsPath := WorkspacePath(repoRoot, "proj")

	if want := filepath.Join(base, "proj.code-workspace"); wsPath != want {
		t.Fatalf("WorkspacePath = %q, want %q", wsPath, want)
	}

	// First add creates the file with root + worktree as relative siblings.
	created, err := AddWorktreeFolder(wsPath, repoRoot, wt)
	if err != nil {
		t.Fatalf("AddWorktreeFolder: %v", err)
	}
	if !created {
		t.Fatalf("expected created=true on first add")
	}
	folders := readFolders(t, wsPath)
	if len(folders) != 2 || folders[0].Path != "proj" || folders[1].Path != "proj-WTfoo" {
		t.Fatalf("folders = %+v, want [proj proj-WTfoo]", folders)
	}

	// Re-adding the same folder is idempotent and reports not-created.
	created, err = AddWorktreeFolder(wsPath, repoRoot, wt)
	if err != nil {
		t.Fatalf("AddWorktreeFolder (2nd): %v", err)
	}
	if created {
		t.Fatalf("expected created=false when file already exists")
	}
	if got := readFolders(t, wsPath); len(got) != 2 {
		t.Fatalf("idempotency broken: %+v", got)
	}

	// Removing the worktree leaves only the root.
	if err := RemoveWorktreeFolder(wsPath, wt); err != nil {
		t.Fatalf("RemoveWorktreeFolder: %v", err)
	}
	folders = readFolders(t, wsPath)
	if len(folders) != 1 || folders[0].Path != "proj" {
		t.Fatalf("after remove folders = %+v, want [proj]", folders)
	}
}

func TestWorkspacePreservesUnknownKeys(t *testing.T) {
	base := t.TempDir()
	repoRoot := filepath.Join(base, "proj")
	wt := filepath.Join(base, "proj-WTfoo")
	wsPath := WorkspacePath(repoRoot, "proj")

	seed := `{
	"folders": [{"path": "proj"}],
	"settings": {"editor.tabSize": 2},
	"extensions": {"recommendations": ["golang.go"]}
}`
	if err := os.WriteFile(wsPath, []byte(seed), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := AddWorktreeFolder(wsPath, repoRoot, wt); err != nil {
		t.Fatalf("AddWorktreeFolder: %v", err)
	}

	data, err := os.ReadFile(wsPath)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if _, ok := got["settings"]; !ok {
		t.Errorf("settings key was dropped")
	}
	if _, ok := got["extensions"]; !ok {
		t.Errorf("extensions key was dropped")
	}
	folders := readFolders(t, wsPath)
	if len(folders) != 2 {
		t.Errorf("folders = %+v, want 2 entries", folders)
	}
}

func TestRemoveFromMissingWorkspaceIsNoError(t *testing.T) {
	wsPath := filepath.Join(t.TempDir(), "absent.code-workspace")
	if err := RemoveWorktreeFolder(wsPath, "/whatever"); err != nil {
		t.Fatalf("expected nil error for missing workspace, got %v", err)
	}
}
