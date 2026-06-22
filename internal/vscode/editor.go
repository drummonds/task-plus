package vscode

import (
	"os"
	"os/exec"
	"strings"
)

// Editor identifies a VS Code-family editor and how to drive it.
type Editor struct {
	Cmd       string // CLI command on PATH, e.g. "code" or "codium"
	ConfigDir string // config dir name under $XDG_CONFIG_HOME, e.g. "Code" or "VSCodium"
}

// knownEditors lists supported editors in PATH-fallback preference order.
var knownEditors = []Editor{
	{Cmd: "code", ConfigDir: "Code"},
	{Cmd: "codium", ConfigDir: "VSCodium"},
	{Cmd: "code-insiders", ConfigDir: "Code - Insiders"},
}

// DetectEditor determines which VS Code-family editor to drive.
//
// It prefers the editor hosting the current integrated terminal so that
// `--add`/`--remove` connect to that running instance over its inherited IPC
// socket (rather than spawning fresh windows from a mismatched fork). When the
// host editor can't be determined from the environment, it falls back to the
// first known CLI found on PATH. The bool is false if none is available.
func DetectEditor() (Editor, bool) {
	if ed, ok := editorFromEnv(); ok {
		if _, err := exec.LookPath(ed.Cmd); err == nil {
			return ed, true
		}
	}
	for _, ed := range knownEditors {
		if _, err := exec.LookPath(ed.Cmd); err == nil {
			return ed, true
		}
	}
	return Editor{}, false
}

// editorFromEnv identifies the host editor from VS Code integrated-terminal
// environment variables. This is the only reliable signal when several forks
// (e.g. both `code` and `codium`) are installed side by side.
func editorFromEnv() (Editor, bool) {
	if os.Getenv("TERM_PROGRAM") != "vscode" {
		return Editor{}, false
	}
	// VSCODE_GIT_ASKPASS_NODE points at the editor's Electron binary, e.g.
	//   /usr/share/codium/codium  or  /usr/share/code/code
	// VSCODE_GIT_ASKPASS_MAIN is a fallback with the same install path embedded.
	hint := strings.ToLower(os.Getenv("VSCODE_GIT_ASKPASS_NODE"))
	if hint == "" {
		hint = strings.ToLower(os.Getenv("VSCODE_GIT_ASKPASS_MAIN"))
	}
	switch {
	case strings.Contains(hint, "codium"):
		return Editor{Cmd: "codium", ConfigDir: "VSCodium"}, true
	case strings.Contains(hint, "insiders"):
		return Editor{Cmd: "code-insiders", ConfigDir: "Code - Insiders"}, true
	case strings.Contains(hint, "code"):
		return Editor{Cmd: "code", ConfigDir: "Code"}, true
	}
	return Editor{}, false
}
