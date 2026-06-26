package vscode

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"
)

// Run invokes the editor CLI with the given args, ensuring the child can reach
// a running window. The CLI only talks to a live instance when
// VSCODE_IPC_HOOK_CLI points at its control socket; that variable is often
// missing from the environment tp inherits (e.g. when launched outside the
// integrated terminal). Run recovers it from a discoverable socket so that
// --add/--remove attach to the existing window instead of spawning a new one.
//
// When no live socket can be found the CLI falls back to opening a new window;
// callers should treat this as best-effort and rely on the workspace file for
// reliable folder management.
func (e Editor) Run(args ...string) error {
	cmd := exec.Command(e.Cmd, args...)
	cmd.Env = os.Environ()
	if sock := IPCSocket(); sock != "" && os.Getenv("VSCODE_IPC_HOOK_CLI") == "" {
		cmd.Env = append(cmd.Env, "VSCODE_IPC_HOOK_CLI="+sock)
	}
	return cmd.Run()
}

// IPCSocket returns the path to a live VS Code CLI control socket, or "" if none
// is reachable. It prefers an already-set VSCODE_IPC_HOOK_CLI and otherwise
// scans the usual runtime directories for the newest vscode-ipc-*.sock that
// still accepts a connection (stale sockets from exited instances are skipped).
func IPCSocket() string {
	if s := os.Getenv("VSCODE_IPC_HOOK_CLI"); s != "" && socketAlive(s) {
		return s
	}

	var dirs []string
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		dirs = append(dirs, d)
	}
	if d := os.Getenv("TMPDIR"); d != "" {
		dirs = append(dirs, d)
	}
	dirs = append(dirs, "/tmp")

	type cand struct {
		path string
		mod  time.Time
	}
	var cands []cand
	seen := map[string]bool{}
	for _, dir := range dirs {
		if seen[dir] {
			continue
		}
		seen[dir] = true
		matches, _ := filepath.Glob(filepath.Join(dir, "vscode-ipc-*.sock"))
		for _, m := range matches {
			info, err := os.Stat(m)
			if err != nil {
				continue
			}
			cands = append(cands, cand{path: m, mod: info.ModTime()})
		}
	}
	// Newest first — most likely the focused/active instance.
	sort.Slice(cands, func(i, j int) bool { return cands[i].mod.After(cands[j].mod) })
	for _, c := range cands {
		if socketAlive(c.path) {
			return c.path
		}
	}
	return ""
}

// socketAlive reports whether a Unix socket accepts a connection, i.e. an editor
// is still listening on it.
func socketAlive(path string) bool {
	conn, err := net.DialTimeout("unix", path, 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
