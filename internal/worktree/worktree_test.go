package worktree

import (
	"os/exec"
	"strings"
	"testing"
)

func TestParseTaskArgs_ReservedNames(t *testing.T) {
	for _, name := range []string{"doc", "docs", "Doc", "Docs", "DOCS"} {
		_, _, err := parseTaskArgs([]string{name})
		if err == nil {
			t.Errorf("expected error for reserved task name %q", name)
		}
	}
}

func TestParseTaskArgs_AllowedNames(t *testing.T) {
	for _, tc := range []struct {
		input, want string
	}{
		{"add-docs-page", "WTadd-docs-page"},
		{"documentation", "WTdocumentation"},
		{"doctor", "WTdoctor"},
	} {
		task, _, err := parseTaskArgs([]string{tc.input})
		if err != nil {
			t.Errorf("unexpected error for task name %q: %v", tc.input, err)
		}
		if task != tc.want {
			t.Errorf("input %q: expected task %q, got %q", tc.input, tc.want, task)
		}
	}
}

func TestParseTaskArgs_Positional(t *testing.T) {
	task, _, err := parseTaskArgs([]string{"banner"})
	if err != nil {
		t.Fatal(err)
	}
	if task != "WTbanner" {
		t.Errorf("expected WTbanner, got %s", task)
	}
}

func TestParseTaskArgs_FlagStillWorks(t *testing.T) {
	task, _, err := parseTaskArgs([]string{"--task=banner"})
	if err != nil {
		t.Fatal(err)
	}
	if task != "WTbanner" {
		t.Errorf("expected WTbanner, got %s", task)
	}
}

func TestParseTaskArgs_AlreadyPrefixed(t *testing.T) {
	task, _, err := parseTaskArgs([]string{"WTbanner"})
	if err != nil {
		t.Fatal(err)
	}
	if task != "WTbanner" {
		t.Errorf("expected WTbanner, got %s", task)
	}
}

func TestRejectIfInsideWorktree_MainRepo(t *testing.T) {
	// This test runs from the main repo, so it should NOT reject.
	// Skip if git-dir != git-common-dir (i.e. test itself is in a worktree).
	gd, err := exec.Command("git", "rev-parse", "--git-dir").Output()
	if err != nil {
		t.Skip("not in a git repo")
	}
	cd, err := exec.Command("git", "rev-parse", "--git-common-dir").Output()
	if err != nil {
		t.Skip("git too old for --git-common-dir")
	}
	if strings.TrimSpace(string(gd)) != strings.TrimSpace(string(cd)) {
		// We're actually in a worktree — verify it rejects.
		err := rejectIfInsideWorktree()
		if err == nil {
			t.Fatal("expected error when running inside a worktree")
		}
		return
	}
	if err := rejectIfInsideWorktree(); err != nil {
		t.Errorf("unexpected error in main repo: %v", err)
	}
}
