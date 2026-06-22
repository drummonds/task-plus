package vscode

import "testing"

func TestEditorFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		termProg string
		node     string
		main     string
		wantCmd  string
		wantDir  string
		wantOK   bool
	}{
		{
			name:     "codium via askpass node",
			termProg: "vscode",
			node:     "/usr/share/codium/codium",
			wantCmd:  "codium",
			wantDir:  "VSCodium",
			wantOK:   true,
		},
		{
			name:     "vscode via askpass node",
			termProg: "vscode",
			node:     "/usr/share/code/code",
			wantCmd:  "code",
			wantDir:  "Code",
			wantOK:   true,
		},
		{
			name:     "insiders via askpass node",
			termProg: "vscode",
			node:     "/usr/share/code-insiders/code-insiders",
			wantCmd:  "code-insiders",
			wantDir:  "Code - Insiders",
			wantOK:   true,
		},
		{
			name:     "falls back to askpass main",
			termProg: "vscode",
			main:     "/usr/share/codium/resources/app/extensions/git/dist/askpass-main.js",
			wantCmd:  "codium",
			wantDir:  "VSCodium",
			wantOK:   true,
		},
		{
			name:     "not a vscode terminal",
			termProg: "iTerm.app",
			node:     "/usr/share/codium/codium",
			wantOK:   false,
		},
		{
			name:     "vscode terminal but no hints",
			termProg: "vscode",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TERM_PROGRAM", tt.termProg)
			t.Setenv("VSCODE_GIT_ASKPASS_NODE", tt.node)
			t.Setenv("VSCODE_GIT_ASKPASS_MAIN", tt.main)

			ed, ok := editorFromEnv()
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if ed.Cmd != tt.wantCmd || ed.ConfigDir != tt.wantDir {
				t.Errorf("got {%q, %q}, want {%q, %q}", ed.Cmd, ed.ConfigDir, tt.wantCmd, tt.wantDir)
			}
		})
	}
}
