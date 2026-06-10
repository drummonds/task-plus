package release

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseTarget(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"github explicit", "release:\n  github:\n    owner: me\n    name: repo\n", "github"},
		{"gitea", "release:\n  gitea:\n    owner: me\n    name: repo\n", "gitea"},
		{"gitlab", "release:\n  gitlab:\n    owner: me\n    name: repo\n", "gitlab"},
		{"no release block", "project_name: foo\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, ".goreleaser.yaml"), []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			got, err := ReleaseTarget(dir, ".goreleaser.yaml")
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("ReleaseTarget = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReleaseTargetMissingFile(t *testing.T) {
	if _, err := ReleaseTarget(t.TempDir(), ".goreleaser.yaml"); err == nil {
		t.Error("ReleaseTarget succeeded for missing file, want error")
	}
}

func TestTargetForgeType(t *testing.T) {
	for target, want := range map[string]string{
		"gitea":  "forgejo",
		"gitlab": "gitlab",
		"github": "github",
		"":       "github",
	} {
		if got := TargetForgeType(target); got != want {
			t.Errorf("TargetForgeType(%q) = %q, want %q", target, got, want)
		}
	}
}
