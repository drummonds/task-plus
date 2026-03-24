package check

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	findings := checkConfig(dir)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].level != levelWarn {
		t.Errorf("expected WARN, got %v", findings[0].level)
	}
}

func TestCheckConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte(":\n  :\n[bad"), 0644)
	findings := checkConfig(dir)
	if len(findings) == 0 {
		t.Fatal("expected findings")
	}
	if findings[0].level != levelError {
		t.Errorf("expected ERROR, got %v", findings[0].level)
	}
}

func TestCheckConfig_UnknownField(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("bogus_field: true\n"), 0644)
	findings := checkConfig(dir)
	hasUnknown := false
	for _, f := range findings {
		if f.level == levelWarn && f.message == `Unknown field "bogus_field"` {
			hasUnknown = true
		}
	}
	if !hasUnknown {
		t.Errorf("expected unknown field warning, got %v", findings)
	}
}

func TestCheckConfig_InvalidType(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("type: invalid\n"), 0644)
	findings := checkConfig(dir)
	hasError := false
	for _, f := range findings {
		if f.level == levelError && f.message == `Invalid type "invalid" (expected: binary, library, docs)` {
			hasError = true
		}
	}
	if !hasError {
		t.Errorf("expected invalid type error, got %v", findings)
	}
}

func TestCheckConfig_ChangemeWarning(t *testing.T) {
	dir := t.TempDir()
	content := "pages_deploy:\n  - type: statichost\n    site: CHANGEME\n"
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte(content), 0644)
	findings := checkConfig(dir)
	hasWarn := false
	for _, f := range findings {
		if f.level == levelWarn && f.message == "pages_deploy[0]: site is still 'CHANGEME'" {
			hasWarn = true
		}
	}
	if !hasWarn {
		t.Errorf("expected CHANGEME warning, got %v", findings)
	}
}

func TestCheckConfig_LanguagesDetected(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("remotes: [origin]\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname = \"test\"\n"), 0644)
	findings := checkConfig(dir)
	hasLangs := false
	for _, f := range findings {
		if f.level == levelOK && f.message == "Languages: go, python" {
			hasLangs = true
		}
	}
	if !hasLangs {
		t.Errorf("expected languages OK, got %v", findings)
	}
}

func TestCheckConfig_GoWithoutGoMod(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("languages: [go]\n"), 0644)
	findings := checkConfig(dir)
	hasError := false
	for _, f := range findings {
		if f.level == levelError && f.message == "Language 'go' but no go.mod found" {
			hasError = true
		}
	}
	if !hasError {
		t.Errorf("expected go/go.mod error, got %v", findings)
	}
}

func TestCheckConfig_PythonWithoutPyproject(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("languages: [python]\n"), 0644)
	findings := checkConfig(dir)
	hasError := false
	for _, f := range findings {
		if f.level == levelError && f.message == "Language 'python' but no pyproject.toml found" {
			hasError = true
		}
	}
	if !hasError {
		t.Errorf("expected python/pyproject.toml error, got %v", findings)
	}
}

func TestCheckConfig_UnknownLanguage(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("languages: [rust]\n"), 0644)
	findings := checkConfig(dir)
	hasWarn := false
	for _, f := range findings {
		if f.level == levelWarn && f.message == `Unknown language "rust" (expected: go, python)` {
			hasWarn = true
		}
	}
	if !hasWarn {
		t.Errorf("expected unknown language warning, got %v", findings)
	}
}

func TestCheckTaskfile_Missing(t *testing.T) {
	dir := t.TempDir()
	findings := checkTaskfile(dir)
	if len(findings) != 1 || findings[0].level != levelWarn {
		t.Errorf("expected 1 WARN, got %v", findings)
	}
}

func TestCheckTaskfile_StandardTasks(t *testing.T) {
	dir := t.TempDir()
	// Non-Go project: only test + check are standard
	content := "version: '3'\ntasks:\n  test:\n    cmds: [echo]\n  check:\n    cmds: [echo]\n"
	_ = os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(content), 0644)
	findings := checkTaskfile(dir)
	hasOK := false
	for _, f := range findings {
		if f.level == levelOK && f.message == "Standard tasks: test, check" {
			hasOK = true
		}
	}
	if !hasOK {
		t.Errorf("expected standard tasks OK, got %v", findings)
	}
}

func TestCheckTaskfile_StandardTasksGo(t *testing.T) {
	dir := t.TempDir()
	// Go project: fmt, vet, test, check are all standard
	content := "version: '3'\ntasks:\n  fmt:\n    cmds: [echo]\n  vet:\n    cmds: [echo]\n  test:\n    cmds: [echo]\n  check:\n    cmds: [echo]\n"
	_ = os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(content), 0644)
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("languages: [go]\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0644)
	findings := checkTaskfile(dir)
	hasOK := false
	for _, f := range findings {
		if f.level == levelOK && f.message == "Standard tasks: fmt, vet, test, check" {
			hasOK = true
		}
	}
	if !hasOK {
		t.Errorf("expected standard tasks OK, got %v", findings)
	}
}

func TestCheckTaskfile_MissingStandardTasks(t *testing.T) {
	dir := t.TempDir()
	// Non-Go project: only test + check are standard; neither present
	content := "version: '3'\ntasks:\n  fmt:\n    cmds: [echo]\n"
	_ = os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(content), 0644)
	findings := checkTaskfile(dir)
	hasWarn := false
	for _, f := range findings {
		if f.level == levelWarn && f.message == "Missing standard tasks: test, check" {
			hasWarn = true
		}
	}
	if !hasWarn {
		t.Errorf("expected missing tasks warning, got %v", findings)
	}
}

func TestCheckTaskfile_Conflict(t *testing.T) {
	dir := t.TempDir()
	content := "version: '3'\ntasks:\n  release:\n    cmds: [echo]\n"
	_ = os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(content), 0644)
	findings := checkTaskfile(dir)
	hasError := false
	for _, f := range findings {
		if f.level == levelError && f.message == `Task "release" conflicts with 'tp release' command — remove it` {
			hasError = true
		}
	}
	if !hasError {
		t.Errorf("expected conflict error, got %v", findings)
	}
}

func TestCheckTaskfile_Inversion(t *testing.T) {
	dir := t.TempDir()
	content := "version: '3'\ntasks:\n  build:docs:\n    cmds: [echo]\n  tidy:deps:\n    cmds: [echo]\n"
	_ = os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(content), 0644)
	findings := checkTaskfile(dir)

	inversions := 0
	for _, f := range findings {
		if f.level == levelWarn {
			switch f.message {
			case "Has 'build:docs' — rename to 'docs:build' (subject:action convention)":
				inversions++
			case "Has 'tidy:deps' — rename to 'deps:tidy' (subject:action convention)":
				inversions++
			}
		}
	}
	if inversions != 2 {
		t.Errorf("expected 2 inversion warnings, got %d from %v", inversions, findings)
	}
}

func TestCheckTaskfile_NoInversionWhenPreferredExists(t *testing.T) {
	dir := t.TempDir()
	// Both preferred and inverted exist — no warning (user may have aliases)
	content := "version: '3'\ntasks:\n  docs:build:\n    cmds: [echo]\n  build:docs:\n    cmds: [echo]\n"
	_ = os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(content), 0644)
	findings := checkTaskfile(dir)
	for _, f := range findings {
		if f.level == levelWarn && f.message == "Has 'build:docs' — rename to 'docs:build' (subject:action convention)" {
			t.Errorf("should not warn when preferred form also exists")
		}
	}
}

func TestCheckStatichost_NoTargets(t *testing.T) {
	dir := t.TempDir()
	findings := checkStatichost(dir)
	if len(findings) != 1 || findings[0].level != levelOK {
		t.Errorf("expected 1 OK, got %v", findings)
	}
}

func TestCheckStatichost_Reachable(t *testing.T) {
	dir := t.TempDir()
	yaml := "pages_deploy:\n  - type: statichost\n    site: h3-task-plus\n"
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte(yaml), 0644)

	// Use a transport that returns 200 for any request
	orig := statichostHTTPClient
	statichostHTTPClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
		}),
	}
	defer func() { statichostHTTPClient = orig }()

	findings := checkStatichost(dir)
	hasOK := false
	for _, f := range findings {
		if f.level == levelOK && f.message == "h3-task-plus reachable (200)" {
			hasOK = true
		}
	}
	if !hasOK {
		t.Errorf("expected reachable OK, got %v", findings)
	}
}

func TestCheckStatichost_Unreachable(t *testing.T) {
	dir := t.TempDir()
	yaml := "pages_deploy:\n  - type: statichost\n    site: h3-nonexistent\n"
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte(yaml), 0644)

	orig := statichostHTTPClient
	statichostHTTPClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 404, Body: http.NoBody}, nil
		}),
	}
	defer func() { statichostHTTPClient = orig }()

	findings := checkStatichost(dir)
	hasWarn := false
	for _, f := range findings {
		if f.level == levelWarn && f.message == "h3-nonexistent returned HTTP 404" {
			hasWarn = true
		}
	}
	if !hasWarn {
		t.Errorf("expected unreachable warning, got %v", findings)
	}
}

func TestCheckStatichost_RCSite(t *testing.T) {
	dir := t.TempDir()
	yaml := "pages_deploy:\n  - type: statichost\n    site: h3-mysite\n    rc_site: h3-mysite-rc\n"
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte(yaml), 0644)

	orig := statichostHTTPClient
	statichostHTTPClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
		}),
	}
	defer func() { statichostHTTPClient = orig }()

	findings := checkStatichost(dir)
	// Should check both main and RC sites
	count := 0
	for _, f := range findings {
		if f.level == levelOK {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 OK findings (main + rc), got %d from %v", count, findings)
	}
}

func TestRemoteURLToModulePath(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"ssh://git@codeberg.org/hum3/task-plus.git", "codeberg.org/hum3/task-plus"},
		{"git@github.com:drummonds/task-plus.git", "github.com/drummonds/task-plus"},
		{"https://github.com/drummonds/task-plus.git", "github.com/drummonds/task-plus"},
		{"https://codeberg.org/hum3/task-plus", "codeberg.org/hum3/task-plus"},
		{"ssh://git@codeberg.org/hum3/go-luca.git", "codeberg.org/hum3/go-luca"},
		{"git@codeberg.org:hum3/go-luca.git", "codeberg.org/hum3/go-luca"},
	}
	for _, tt := range tests {
		got := remoteURLToModulePath(tt.url)
		if got != tt.want {
			t.Errorf("remoteURLToModulePath(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestReadGoModulePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "go.mod")

	// Valid go.mod
	_ = os.WriteFile(path, []byte("module codeberg.org/hum3/task-plus\n\ngo 1.25.3\n"), 0644)
	got, err := readGoModulePath(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "codeberg.org/hum3/task-plus" {
		t.Errorf("got %q, want %q", got, "codeberg.org/hum3/task-plus")
	}

	// No module directive
	_ = os.WriteFile(path, []byte("go 1.25.3\n"), 0644)
	_, err = readGoModulePath(path)
	if err == nil {
		t.Error("expected error for missing module directive")
	}
}

func TestCheckGoModule_NotGoProject(t *testing.T) {
	dir := t.TempDir()
	findings := checkGoModule(dir)
	if len(findings) != 1 || findings[0].level != levelOK {
		t.Errorf("expected 1 OK for non-Go project, got %v", findings)
	}
}

func TestCheckTaskfile_LintGolangciLint(t *testing.T) {
	dir := t.TempDir()
	taskfile := "version: '3'\ntasks:\n  lint:\n    cmds:\n      - golangci-lint run ./...\n  fmt:\n    cmds: [echo]\n  vet:\n    cmds: [echo]\n  test:\n    cmds: [echo]\n  check:\n    cmds: [echo]\n"
	_ = os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0644)
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("languages: [go]\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.25.3\n"), 0644)
	findings := checkTaskfile(dir)
	found := false
	for _, f := range findings {
		if f.level == levelOK && f.message == "lint task uses golangci-lint" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected golangci-lint OK, got %v", findings)
	}
}

func TestCheckTaskfile_LintStaticcheck_Warn(t *testing.T) {
	dir := t.TempDir()
	taskfile := "version: '3'\ntasks:\n  lint:\n    cmds:\n      - staticcheck ./...\n  fmt:\n    cmds: [echo]\n  vet:\n    cmds: [echo]\n  test:\n    cmds: [echo]\n  check:\n    cmds: [echo]\n"
	_ = os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0644)
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("languages: [go]\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.25.3\n"), 0644)
	findings := checkTaskfile(dir)
	found := false
	for _, f := range findings {
		if f.level == levelWarn && f.message == "lint uses staticcheck — migrate to golangci-lint (wraps staticcheck + more; https://golangci-lint.run/)" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected staticcheck warning, got %v", findings)
	}
}

func TestCheckTaskfile_LintStaticcheck_Override(t *testing.T) {
	dir := t.TempDir()
	taskfile := "version: '3'\ntasks:\n  lint:\n    cmds:\n      - staticcheck ./...\n  fmt:\n    cmds: [echo]\n  vet:\n    cmds: [echo]\n  test:\n    cmds: [echo]\n  check:\n    cmds: [echo]\n"
	_ = os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0644)
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("languages: [go]\nlinter: staticcheck\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.25.3\n"), 0644)
	findings := checkTaskfile(dir)
	found := false
	for _, f := range findings {
		if f.level == levelOK && f.message == "lint task uses staticcheck (override)" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected staticcheck override OK, got %v", findings)
	}
}

func TestCheckTaskfile_GoFix_Present(t *testing.T) {
	dir := t.TempDir()
	taskfile := "version: '3'\ntasks:\n  fmt:\n    cmds:\n      - go fmt ./...\n      - go fix ./...\n  vet:\n    cmds: [echo]\n  test:\n    cmds: [echo]\n  check:\n    cmds: [echo]\n"
	_ = os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0644)
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("languages: [go]\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.26.0\n"), 0644)
	findings := checkTaskfile(dir)
	found := false
	for _, f := range findings {
		if f.level == levelOK && f.message == "fmt task includes go fix" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected go fix OK, got %v", findings)
	}
}

func TestCheckTaskfile_GoFix_Missing(t *testing.T) {
	dir := t.TempDir()
	taskfile := "version: '3'\ntasks:\n  fmt:\n    cmds:\n      - go fmt ./...\n  vet:\n    cmds: [echo]\n  test:\n    cmds: [echo]\n  check:\n    cmds: [echo]\n"
	_ = os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0644)
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("languages: [go]\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.26.0\n"), 0644)
	findings := checkTaskfile(dir)
	found := false
	for _, f := range findings {
		if f.level == levelWarn && f.message == "Go 1.26+ project — add 'go fix ./...' to fmt task" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected go fix warning, got %v", findings)
	}
}

func TestCheckTaskfile_GoFix_NotNeeded_OlderGo(t *testing.T) {
	dir := t.TempDir()
	taskfile := "version: '3'\ntasks:\n  fmt:\n    cmds:\n      - go fmt ./...\n  vet:\n    cmds: [echo]\n  test:\n    cmds: [echo]\n  check:\n    cmds: [echo]\n"
	_ = os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0644)
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("languages: [go]\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.25.3\n"), 0644)
	findings := checkTaskfile(dir)
	for _, f := range findings {
		if f.message == "Go 1.26+ project — add 'go fix ./...' to fmt task" {
			t.Error("should not warn about go fix for Go < 1.26")
		}
		if f.message == "fmt task includes go fix" {
			t.Error("should not report go fix OK for Go < 1.26")
		}
	}
}

func TestReadGoVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "go.mod")

	_ = os.WriteFile(path, []byte("module example.com/test\n\ngo 1.25.3\n"), 0644)
	major, minor, ok := readGoVersion(dir)
	if !ok || major != 1 || minor != 25 {
		t.Errorf("got %d.%d (ok=%v), want 1.25", major, minor, ok)
	}

	_ = os.WriteFile(path, []byte("module example.com/test\n\ngo 1.26.0\n"), 0644)
	major, minor, ok = readGoVersion(dir)
	if !ok || major != 1 || minor != 26 {
		t.Errorf("got %d.%d (ok=%v), want 1.26", major, minor, ok)
	}

	// No go.mod
	noDir := t.TempDir()
	_, _, ok = readGoVersion(noDir)
	if ok {
		t.Error("expected ok=false for missing go.mod")
	}
}

func TestTaskContains(t *testing.T) {
	data := []byte("version: '3'\ntasks:\n  lint:\n    cmds:\n      - golangci-lint run ./...\n  fmt:\n    cmds:\n      - go fmt ./...\n")

	if !taskContains(data, "lint", "golangci-lint") {
		t.Error("expected lint to contain golangci-lint")
	}
	if taskContains(data, "lint", "staticcheck") {
		t.Error("lint should not contain staticcheck")
	}
	if !taskContains(data, "fmt", "go fmt") {
		t.Error("expected fmt to contain go fmt")
	}
	if taskContains(data, "fmt", "golangci-lint") {
		t.Error("fmt should not contain golangci-lint")
	}
	if taskContains(data, "nonexistent", "anything") {
		t.Error("nonexistent task should not match")
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
}

func gitCommitAndTag(t *testing.T, dir, tag string) {
	t.Helper()
	// Need at least one commit for the tag
	f := filepath.Join(dir, tag+".txt")
	_ = os.WriteFile(f, []byte(tag), 0644)
	cmds := [][]string{
		{"git", "add", "-A"},
		{"git", "commit", "-m", "release " + tag},
		{"git", "tag", "-a", tag, "-m", tag},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
}

func TestCheckVersionSection_AllMatch(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	gitCommitAndTag(t, dir, "v0.1.5")

	// Changelog matches
	_ = os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog\n\n## [0.1.5] - 2026-03-23\n"), 0644)

	s := checkVersionSection(dir)
	if !strings.Contains(s.name, "v0.1.5") {
		t.Errorf("section name %q should contain version", s.name)
	}
	for _, f := range s.findings {
		if f.level == levelError {
			t.Errorf("unexpected error: %s", f.message)
		}
	}
}

func TestCheckVersionSection_ChangelogMismatch(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	gitCommitAndTag(t, dir, "v0.1.5")

	// Changelog has different version
	_ = os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog\n\n## [0.1.4] - 2026-03-20\n"), 0644)

	s := checkVersionSection(dir)
	hasError := false
	for _, f := range s.findings {
		if f.level == levelError && strings.Contains(f.message, "changelog") {
			hasError = true
		}
	}
	if !hasError {
		t.Errorf("expected changelog mismatch error, got %v", s.findings)
	}
}

func TestCheckVersionSection_NoTags(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	// Need at least one commit
	_ = os.WriteFile(filepath.Join(dir, "dummy.txt"), []byte("x"), 0644)
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = dir
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = dir
	_ = cmd.Run()

	s := checkVersionSection(dir)
	if s.name != "Version" {
		t.Errorf("section name %q, want %q", s.name, "Version")
	}
}

func TestCheckVersionSection_PyprojectMatch(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	gitCommitAndTag(t, dir, "v0.2.0")

	_ = os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog\n\n## [0.2.0] - 2026-03-23\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname = \"mypkg\"\nversion = \"0.2.0\"\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("languages: [python]\n"), 0644)

	s := checkVersionSection(dir)
	if !strings.Contains(s.name, "v0.2.0") {
		t.Errorf("section name %q should contain version", s.name)
	}
	for _, f := range s.findings {
		if f.level == levelError {
			t.Errorf("unexpected error: %s", f.message)
		}
	}
}

func TestCheckVersionSection_PyprojectMismatch(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	gitCommitAndTag(t, dir, "v0.2.0")

	_ = os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog\n\n## [0.2.0] - 2026-03-23\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname = \"mypkg\"\nversion = \"0.1.0\"\n"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("languages: [python]\n"), 0644)

	s := checkVersionSection(dir)
	hasError := false
	for _, f := range s.findings {
		if f.level == levelError && strings.Contains(f.message, "pyproject.toml") {
			hasError = true
		}
	}
	if !hasError {
		t.Errorf("expected pyproject mismatch error, got %v", s.findings)
	}
}

func TestCheckConfig_LinterOverride(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("linter: staticcheck\n"), 0644)
	findings := checkConfig(dir)
	found := false
	for _, f := range findings {
		if f.level == levelOK && f.message == "Linter: staticcheck" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected linter OK, got %v", findings)
	}
}

func TestCheckConfig_LinterInvalid(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "task-plus.yml"), []byte("linter: bogus\n"), 0644)
	findings := checkConfig(dir)
	found := false
	for _, f := range findings {
		if f.level == levelError && f.message == `Invalid linter "bogus" (expected: staticcheck, golangci-lint)` {
			found = true
		}
	}
	if !found {
		t.Errorf("expected linter error, got %v", findings)
	}
}

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
