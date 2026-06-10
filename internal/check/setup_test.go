package check

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"codeberg.org/hum3/task-plus/internal/config"
	"codeberg.org/hum3/task-plus/internal/forge"
	"codeberg.org/hum3/task-plus/internal/prompt"
)

func setupGitRepo(t *testing.T, remotes map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{{"git", "init"}}
	for name, url := range remotes {
		cmds = append(cmds, []string{"git", "remote", "add", name, url})
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	return dir
}

func runSetup(t *testing.T, dir, input string) string {
	t.Helper()
	var out bytes.Buffer
	prompt.SetIO(strings.NewReader(input), &out)
	t.Cleanup(func() { prompt.SetIO(os.Stdin, os.Stdout) })
	if err := Setup(dir); err != nil {
		t.Fatalf("Setup: %v\n%s", err, out.String())
	}
	return out.String()
}

func TestSetupGitHubOnly(t *testing.T) {
	dir := setupGitRepo(t, map[string]string{"origin": "https://github.com/user/repo.git"})

	// y: push to origin; y: write config
	out := runSetup(t, dir, "y\ny\n")
	if !strings.Contains(out, "GitHub only") {
		t.Errorf("arrangement not classified as GitHub only:\n%s", out)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.RemoteNames(); len(got) != 1 || got[0] != "origin" {
		t.Errorf("RemoteNames = %v, want [origin]", got)
	}
	if cfg.GetReleaseRemote() != "origin" {
		t.Errorf("GetReleaseRemote = %q", cfg.GetReleaseRemote())
	}
	if spec := cfg.RemoteSpec("origin"); spec.Forge != "" {
		t.Errorf("no forge override expected for github.com, got %q", spec.Forge)
	}
}

func TestSetupLocalForgejo(t *testing.T) {
	dir := setupGitRepo(t, map[string]string{"origin": "https://git.home.lan/user/repo.git"})

	// y: push to origin; 3: forge type forgejo; HOMELAB_TOKEN: token env; y: write
	out := runSetup(t, dir, "y\n3\nHOMELAB_TOKEN\ny\n")
	if !strings.Contains(out, "self-hosted Forgejo (git.home.lan)") {
		t.Errorf("arrangement not classified as self-hosted Forgejo:\n%s", out)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	spec := cfg.RemoteSpec("origin")
	if spec.Forge != "forgejo" || spec.TokenEnv != "HOMELAB_TOKEN" {
		t.Errorf("RemoteSpec = %+v, want forge=forgejo token_env=HOMELAB_TOKEN", spec)
	}
}

func TestSetupCodebergPlusGitHub(t *testing.T) {
	dir := setupGitRepo(t, map[string]string{
		"origin": "https://codeberg.org/user/repo.git",
		"github": "https://github.com/user/repo.git",
	})

	// y, y: push to both; 1 or default: release remote (origin is the default); y: write
	out := runSetup(t, dir, "y\ny\n\ny\n")
	if !strings.Contains(out, "Codeberg + GitHub mirror") {
		t.Errorf("arrangement not classified as Codeberg + GitHub mirror:\n%s", out)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.RemoteNames(); len(got) != 2 {
		t.Errorf("RemoteNames = %v, want 2 remotes", got)
	}
	if cfg.GetReleaseRemote() != "origin" {
		t.Errorf("GetReleaseRemote = %q, want origin", cfg.GetReleaseRemote())
	}
	// codeberg.org needs no token_env (default chain)
	if spec := cfg.RemoteSpec("origin"); spec.TokenEnv != "" {
		t.Errorf("unexpected token_env for codeberg.org: %q", spec.TokenEnv)
	}
}

func TestSetupDeselectMirror(t *testing.T) {
	dir := setupGitRepo(t, map[string]string{
		"origin": "https://codeberg.org/user/repo.git",
		"github": "https://github.com/user/repo.git",
	})

	// Keep only origin: answer differs per remote iteration order, so pick by
	// answering yes to origin and no to github. Remote order from git is
	// alphabetical (github, origin), so: n (github), y (origin), y (write).
	runSetup(t, dir, "n\ny\ny\n")

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.RemoteNames(); len(got) != 1 || got[0] != "origin" {
		t.Errorf("RemoteNames = %v, want [origin]", got)
	}
}

func TestSetupNoRemotes(t *testing.T) {
	dir := setupGitRepo(t, nil)
	prompt.SetIO(strings.NewReader(""), &bytes.Buffer{})
	t.Cleanup(func() { prompt.SetIO(os.Stdin, os.Stdout) })
	if err := Setup(dir); err == nil {
		t.Error("Setup succeeded with no remotes, want error")
	}
}

func TestClassifyArrangement(t *testing.T) {
	ri := func(host string, ft forge.Type) remoteInfo {
		return remoteInfo{host: host, forgeType: ft}
	}
	tests := []struct {
		name    string
		remotes []remoteInfo
		want    string
	}{
		{"github only", []remoteInfo{ri("github.com", forge.GitHub)}, "GitHub only"},
		{"codeberg only", []remoteInfo{ri("codeberg.org", forge.Forgejo)}, "Codeberg only"},
		{"local forgejo", []remoteInfo{ri("git.home.lan", forge.Forgejo)}, "self-hosted Forgejo (git.home.lan)"},
		{"gitlab only", []remoteInfo{ri("gitlab.com", forge.GitLab)}, "GitLab only"},
		{"codeberg + github", []remoteInfo{ri("codeberg.org", forge.Forgejo), ri("github.com", forge.GitHub)}, "Codeberg + GitHub mirror"},
		{"three remotes", []remoteInfo{ri("a", forge.GitHub), ri("b", forge.GitHub), ri("c", forge.GitHub)}, "custom (3 remotes)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyArrangement(tt.remotes); got != tt.want {
				t.Errorf("classifyArrangement = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetupDropsConfigRemoteMissingFromGit(t *testing.T) {
	dir := setupGitRepo(t, map[string]string{"origin": "https://codeberg.org/user/repo.git"})
	// Config mentions a github remote that doesn't exist in git
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"),
		[]byte("remotes:\n  - origin\n  - github\n"), 0644)

	out := runSetup(t, dir, "y\ny\n")
	if !strings.Contains(out, `"github" is in task-plus.yml but not in git`) {
		t.Errorf("missing dropped-remote notice:\n%s", out)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.RemoteNames(); len(got) != 1 || got[0] != "origin" {
		t.Errorf("RemoteNames = %v, want [origin]", got)
	}
}
