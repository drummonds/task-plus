package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, configFile), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readConfig(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, configFile))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSaveRemotesPreservesOtherKeys(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `# my project config
remotes:
  - origin
  - github

pages_build:
  - task docs:build

pages_deploy:
  - type: statichost
    site: mysite
`)

	remotes := []Remote{
		{Name: "origin"},
		{Name: "forgejo-local", Forge: "forgejo", TokenEnv: "HOMELAB_FORGEJO_TOKEN"},
	}
	if err := SaveRemotes(dir, remotes, "origin"); err != nil {
		t.Fatal(err)
	}

	out := readConfig(t, dir)
	for _, want := range []string{
		"# my project config",
		"task docs:build",
		"site: mysite",
		"name: forgejo-local",
		"forge: forgejo",
		"token_env: HOMELAB_FORGEJO_TOKEN",
		"release_remote: origin",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "github") {
		t.Errorf("output still mentions dropped remote github:\n%s", out)
	}

	// Round-trip: loads back with the same remotes
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.RemoteNames(); len(got) != 2 || got[1] != "forgejo-local" {
		t.Errorf("RemoteNames after save = %v", got)
	}
	if spec := cfg.RemoteSpec("forgejo-local"); spec.TokenEnv != "HOMELAB_FORGEJO_TOKEN" {
		t.Errorf("RemoteSpec after save = %+v", spec)
	}
}

func TestSaveRemotesPlainEntriesStayScalars(t *testing.T) {
	dir := t.TempDir()
	if err := SaveRemotes(dir, []Remote{{Name: "origin"}, {Name: "github"}}, ""); err != nil {
		t.Fatal(err)
	}
	out := readConfig(t, dir)
	if strings.Contains(out, "name:") {
		t.Errorf("plain remotes serialized as maps:\n%s", out)
	}
	if !strings.Contains(out, "- origin") || !strings.Contains(out, "- github") {
		t.Errorf("expected scalar list entries:\n%s", out)
	}
	if strings.Contains(out, "release_remote") {
		t.Errorf("release_remote written despite empty arg:\n%s", out)
	}
}

func TestSaveRemotesCreatesFile(t *testing.T) {
	dir := t.TempDir()
	if err := SaveRemotes(dir, []Remote{{Name: "origin"}}, "origin"); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GetReleaseRemote() != "origin" {
		t.Errorf("GetReleaseRemote = %q", cfg.GetReleaseRemote())
	}
}

func TestSaveRemotesEmptyList(t *testing.T) {
	dir := t.TempDir()
	if err := SaveRemotes(dir, nil, ""); err == nil {
		t.Error("SaveRemotes(nil) succeeded, want error")
	}
}

func TestAddRemoteWithStructuredEntry(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `remotes:
  - origin
  - name: forgejo-local
    forge: forgejo
    token_env: HOMELAB_FORGEJO_TOKEN
pages_build:
  - task docs:build
`)

	if err := AddRemote(dir, "github"); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.RemoteNames()
	if len(got) != 3 || got[2] != "github" {
		t.Errorf("RemoteNames = %v, want [origin forgejo-local github]", got)
	}
	// Structured entry survives intact (the old line editor corrupted this case)
	if spec := cfg.RemoteSpec("forgejo-local"); spec.Forge != "forgejo" || spec.TokenEnv != "HOMELAB_FORGEJO_TOKEN" {
		t.Errorf("structured entry corrupted: %+v", spec)
	}
	if !strings.Contains(readConfig(t, dir), "task docs:build") {
		t.Error("unrelated key lost")
	}
}

func TestAddRemoteDuplicate(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "remotes:\n  - origin\n")
	if err := AddRemote(dir, "origin"); err == nil {
		t.Error("AddRemote(origin) succeeded, want duplicate error")
	}
}

func TestAddRemoteNoConfigFile(t *testing.T) {
	dir := t.TempDir()
	if err := AddRemote(dir, "github"); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.RemoteNames(); len(got) != 1 || got[0] != "github" {
		t.Errorf("RemoteNames = %v, want [github]", got)
	}
}

func TestRemoveRemoteStructuredEntry(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `remotes:
  - origin
  - name: forgejo-local
    forge: forgejo
`)
	if err := RemoveRemote(dir, "forgejo-local"); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.RemoteNames(); len(got) != 1 || got[0] != "origin" {
		t.Errorf("RemoteNames = %v, want [origin]", got)
	}
}

func TestRemoveRemoteLast(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "remotes:\n  - origin\n")
	if err := RemoveRemote(dir, "origin"); err == nil {
		t.Error("RemoveRemote of last remote succeeded, want error")
	}
}

func TestRemoveRemoteNotConfigured(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "remotes:\n  - origin\n  - github\n")
	if err := RemoveRemote(dir, "gitlab"); err == nil {
		t.Error("RemoveRemote(gitlab) succeeded, want error")
	}
}
