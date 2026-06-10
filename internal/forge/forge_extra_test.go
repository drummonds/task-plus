package forge

import (
	"slices"
	"testing"
)

func TestAPITokenPrecedence(t *testing.T) {
	clear := func(t *testing.T) {
		t.Setenv("CODEBERG_APIKEY", "")
		t.Setenv("FORGEJO_TOKEN", "")
		t.Setenv("GITEA_TOKEN", "")
		t.Setenv("CUSTOM_TOKEN", "")
	}

	t.Run("token_env wins and is exclusive", func(t *testing.T) {
		clear(t)
		t.Setenv("CODEBERG_APIKEY", "cb")
		t.Setenv("CUSTOM_TOKEN", "custom")
		f := Forge{Type: Forgejo, TokenEnv: "CUSTOM_TOKEN"}
		if got := f.APIToken(); got != "custom" {
			t.Errorf("APIToken = %q, want custom", got)
		}
		// token_env set but empty → no fallback to the chain
		f.TokenEnv = "UNSET_TOKEN_VAR"
		if got := f.APIToken(); got != "" {
			t.Errorf("APIToken = %q, want empty (no fallback when TokenEnv set)", got)
		}
	})

	t.Run("default chain order", func(t *testing.T) {
		clear(t)
		t.Setenv("GITEA_TOKEN", "gitea")
		f := Forge{Type: Forgejo}
		if got := f.APIToken(); got != "gitea" {
			t.Errorf("APIToken = %q, want gitea", got)
		}
		t.Setenv("FORGEJO_TOKEN", "forgejo")
		if got := f.APIToken(); got != "forgejo" {
			t.Errorf("APIToken = %q, want forgejo", got)
		}
		t.Setenv("CODEBERG_APIKEY", "cb")
		if got := f.APIToken(); got != "cb" {
			t.Errorf("APIToken = %q, want cb", got)
		}
	})
}

func TestRepoFlag(t *testing.T) {
	tests := []struct {
		url  string
		want []string
	}{
		{"https://github.com/user/repo.git", []string{"-R", "user/repo"}},
		{"git@github.com:user/repo.git", []string{"-R", "user/repo"}},
		{"https://gitlab.com/user/repo.git", []string{"-R", "user/repo"}},
		{"https://github.example.com/user/repo.git", []string{"-R", "github.example.com/user/repo"}},
		{"not-a-url", nil},
	}
	for _, tt := range tests {
		got := Forge{URL: tt.url}.repoFlag()
		if !slices.Equal(got, tt.want) {
			t.Errorf("repoFlag(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestSupportsReleases(t *testing.T) {
	for _, tt := range []struct {
		typ  Type
		want bool
	}{
		{GitHub, true}, {GitLab, true}, {Forgejo, true},
		{None, false}, {Unknown, false},
	} {
		if got := (Forge{Type: tt.typ}).SupportsReleases(); got != tt.want {
			t.Errorf("SupportsReleases(%s) = %v, want %v", tt.typ, got, tt.want)
		}
	}
}

func TestIsPublicHost(t *testing.T) {
	for host, want := range map[string]bool{
		"github.com":   true,
		"GitHub.com":   true,
		"codeberg.org": true,
		"gitlab.com":   true,
		"gitea.com":    true,
		"git.home.lan": false,
		"example.com":  false,
	} {
		if got := IsPublicHost(host); got != want {
			t.Errorf("IsPublicHost(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestHasCLINone(t *testing.T) {
	if (Forge{Type: None}).HasCLI() {
		t.Error("HasCLI(None) = true, want false")
	}
}
