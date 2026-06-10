package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func hasFinding(findings []finding, lvl level, substr string) bool {
	for _, f := range findings {
		if f.level == lvl && strings.Contains(f.message, substr) {
			return true
		}
	}
	return false
}

func TestCheckConfig_RemoteEntryUnknownField(t *testing.T) {
	dir := t.TempDir()
	yml := `remotes:
  - origin
  - name: local
    forge: forgejo
    bogus: true
`
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte(yml), 0644)
	findings := checkConfig(dir)
	if !hasFinding(findings, levelWarn, `remotes[1]: unknown field "bogus"`) {
		t.Errorf("missing unknown-field warning, findings: %v", findings)
	}
}

func TestCheckConfig_RemoteEntryMissingName(t *testing.T) {
	dir := t.TempDir()
	yml := "remotes:\n  - forge: forgejo\n"
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte(yml), 0644)
	findings := checkConfig(dir)
	if !hasFinding(findings, levelError, "requires 'name'") {
		t.Errorf("missing missing-name error, findings: %v", findings)
	}
}

func TestCheckConfig_RemoteInvalidForge(t *testing.T) {
	dir := t.TempDir()
	yml := "remotes:\n  - name: local\n    forge: bitbucket\n"
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte(yml), 0644)
	findings := checkConfig(dir)
	if !hasFinding(findings, levelError, `invalid forge "bitbucket"`) {
		t.Errorf("missing invalid-forge error, findings: %v", findings)
	}
}

func TestCheckConfig_RemoteForgeNoneValid(t *testing.T) {
	dir := t.TempDir()
	yml := "remotes:\n  - name: mirror\n    forge: none\n"
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte(yml), 0644)
	findings := checkConfig(dir)
	if !hasFinding(findings, levelOK, "Remote mirror: forge none") {
		t.Errorf("forge none not accepted, findings: %v", findings)
	}
}

func TestCheckConfig_TokenEnvUnset(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DEFINITELY_UNSET_TOKEN_VAR", "")
	yml := "remotes:\n  - name: local\n    forge: forgejo\n    token_env: DEFINITELY_UNSET_TOKEN_VAR\n"
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte(yml), 0644)
	findings := checkConfig(dir)
	if !hasFinding(findings, levelWarn, "DEFINITELY_UNSET_TOKEN_VAR is not set") {
		t.Errorf("missing token_env warning, findings: %v", findings)
	}
}

func TestCheckGoreleaser_NotBinary(t *testing.T) {
	dir := t.TempDir()
	if findings := checkGoreleaser(dir); len(findings) != 0 {
		t.Errorf("expected no findings for non-binary project, got %v", findings)
	}
}

func TestCheckGoreleaser_TargetMismatch(t *testing.T) {
	dir := t.TempDir()
	// Binary project releasing to github, but no remotes in git at all
	_ = os.WriteFile(filepath.Join(dir, ".goreleaser.yaml"),
		[]byte("release:\n  github:\n    owner: me\n    name: repo\n"), 0644)
	findings := checkGoreleaser(dir)
	if !hasFinding(findings, levelWarn, "no configured remote is github") {
		t.Errorf("missing target-mismatch warning, findings: %v", findings)
	}
}

func TestCheckGoreleaser_GiteaTokenChain(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITEA_TOKEN", "")
	t.Setenv("CODEBERG_APIKEY", "")
	t.Setenv("FORGEJO_TOKEN", "")
	_ = os.WriteFile(filepath.Join(dir, ".goreleaser.yaml"),
		[]byte("release:\n  gitea:\n    owner: me\n    name: repo\n"), 0644)
	findings := checkGoreleaser(dir)
	if !hasFinding(findings, levelWarn, "No Forgejo API token set") {
		t.Errorf("missing forgejo token warning, findings: %v", findings)
	}

	t.Setenv("FORGEJO_TOKEN", "tok")
	findings = checkGoreleaser(dir)
	if !hasFinding(findings, levelOK, "passes the forge API token") {
		t.Errorf("missing token-bridge OK finding, findings: %v", findings)
	}
}
