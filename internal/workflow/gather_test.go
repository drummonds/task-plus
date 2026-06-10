package workflow

import (
	"strings"
	"testing"

	"codeberg.org/hum3/task-plus/internal/config"
	"codeberg.org/hum3/task-plus/internal/forge"
)

func TestCheckArrangement(t *testing.T) {
	rf := func(name string, typ forge.Type) RemoteForge {
		return RemoteForge{Name: name, Forge: forge.Forge{Type: typ, URL: "https://example/" + name}}
	}

	tests := []struct {
		name       string
		explicit   bool
		gitRemotes []string
		forges     []RemoteForge
		wantErr    string // substring, "" = no error
	}{
		{
			name:       "single github remote zero-config",
			gitRemotes: []string{"origin"},
			forges:     []RemoteForge{rf("origin", forge.GitHub)},
		},
		{
			name:       "single codeberg remote zero-config",
			gitRemotes: []string{"origin"},
			forges:     []RemoteForge{rf("origin", forge.Forgejo)},
		},
		{
			name:       "multiple git remotes without explicit config",
			gitRemotes: []string{"origin", "github"},
			forges:     []RemoteForge{rf("origin", forge.Forgejo)},
			wantErr:    "tp check --setup",
		},
		{
			name:       "multiple git remotes with explicit config",
			explicit:   true,
			gitRemotes: []string{"origin", "github"},
			forges:     []RemoteForge{rf("origin", forge.Forgejo), rf("github", forge.GitHub)},
		},
		{
			name:       "unknown forge host",
			explicit:   true,
			gitRemotes: []string{"origin"},
			forges:     []RemoteForge{rf("origin", forge.Unknown)},
			wantErr:    "unrecognised forge host",
		},
		{
			name:       "forge none is allowed",
			explicit:   true,
			gitRemotes: []string{"origin", "mirror"},
			forges:     []RemoteForge{rf("origin", forge.Forgejo), rf("mirror", forge.None)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{RemotesExplicit: tt.explicit}
			err := checkArrangement(cfg, tt.gitRemotes, tt.forges)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("checkArrangement() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("checkArrangement() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}
