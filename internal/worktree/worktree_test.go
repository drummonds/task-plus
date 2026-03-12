package worktree

import "testing"

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
