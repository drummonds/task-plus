package vscode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WorkspaceFolder is one entry in a .code-workspace "folders" array. Paths are
// stored relative to the workspace file's directory when possible.
type WorkspaceFolder struct {
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}

// workspaceFile is a minimal model of a .code-workspace document. Unknown
// top-level keys (settings, extensions, launch, …) are preserved verbatim via
// Extra so editing folders never discards user configuration.
type workspaceFile struct {
	Folders []WorkspaceFolder          `json:"folders"`
	Extra   map[string]json.RawMessage `json:"-"`
}

// WorkspacePath returns the conventional location of the project workspace file:
// a sibling of the repo root named "<project>.code-workspace". Keeping it beside
// the repo (not inside it) means worktrees and root are simple siblings and the
// file never needs to be git-ignored.
func WorkspacePath(repoRoot, projName string) string {
	return filepath.Join(filepath.Dir(repoRoot), projName+".code-workspace")
}

// AddWorktreeFolder ensures the workspace file exists with the repo root as its
// first folder and adds folderPath as a folder. It is idempotent. Returns true
// if the file was newly created (caller may then want to open it).
func AddWorktreeFolder(wsPath, repoRoot, folderPath string) (created bool, err error) {
	ws, existed, err := loadWorkspace(wsPath)
	if err != nil {
		return false, err
	}

	rootRel := relTo(wsPath, repoRoot)
	if !hasFolder(ws.Folders, rootRel, repoRoot, wsPath) {
		ws.Folders = append([]WorkspaceFolder{{Path: rootRel}}, ws.Folders...)
	}

	rel := relTo(wsPath, folderPath)
	if !hasFolder(ws.Folders, rel, folderPath, wsPath) {
		ws.Folders = append(ws.Folders, WorkspaceFolder{Path: rel})
	}

	return !existed, saveWorkspace(wsPath, ws)
}

// RemoveWorktreeFolder removes folderPath from the workspace file's folders.
// Missing files or absent entries are not errors.
func RemoveWorktreeFolder(wsPath, folderPath string) error {
	ws, existed, err := loadWorkspace(wsPath)
	if err != nil || !existed {
		return err
	}

	rel := relTo(wsPath, folderPath)
	filtered := ws.Folders[:0]
	for _, f := range ws.Folders {
		if sameFolder(f.Path, rel, folderPath, wsPath) {
			continue
		}
		filtered = append(filtered, f)
	}
	ws.Folders = filtered
	return saveWorkspace(wsPath, ws)
}

func loadWorkspace(wsPath string) (workspaceFile, bool, error) {
	data, err := os.ReadFile(wsPath)
	if os.IsNotExist(err) {
		return workspaceFile{}, false, nil
	}
	if err != nil {
		return workspaceFile{}, false, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return workspaceFile{}, true, fmt.Errorf("parse %s: %w", filepath.Base(wsPath), err)
	}

	var ws workspaceFile
	if f, ok := raw["folders"]; ok {
		if err := json.Unmarshal(f, &ws.Folders); err != nil {
			return workspaceFile{}, true, fmt.Errorf("parse folders in %s: %w", filepath.Base(wsPath), err)
		}
		delete(raw, "folders")
	}
	ws.Extra = raw
	return ws, true, nil
}

func saveWorkspace(wsPath string, ws workspaceFile) error {
	out := map[string]json.RawMessage{}
	for k, v := range ws.Extra {
		out[k] = v
	}
	folders, err := json.Marshal(ws.Folders)
	if err != nil {
		return err
	}
	out["folders"] = folders

	data, err := json.MarshalIndent(out, "", "\t")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(wsPath, data, 0644)
}

// relTo returns target expressed relative to the workspace file's directory,
// falling back to the absolute path when a relative one can't be computed.
func relTo(wsPath, target string) string {
	rel, err := filepath.Rel(filepath.Dir(wsPath), target)
	if err != nil {
		return target
	}
	return rel
}

func hasFolder(folders []WorkspaceFolder, rel, abs, wsPath string) bool {
	for _, f := range folders {
		if sameFolder(f.Path, rel, abs, wsPath) {
			return true
		}
	}
	return false
}

// sameFolder reports whether a stored folder path refers to the same directory
// as the target, comparing both relative and absolute forms.
func sameFolder(stored, rel, abs, wsPath string) bool {
	if stored == rel {
		return true
	}
	storedAbs := stored
	if !filepath.IsAbs(storedAbs) {
		storedAbs = filepath.Join(filepath.Dir(wsPath), stored)
	}
	return filepath.Clean(storedAbs) == filepath.Clean(abs)
}
