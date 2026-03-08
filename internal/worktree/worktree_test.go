package worktree

import "testing"

func TestParseTaskArgs_ReservedNames(t *testing.T) {
	for _, name := range []string{"doc", "docs", "Doc", "Docs", "DOCS"} {
		_, _, err := parseTaskArgs([]string{"--task=" + name})
		if err == nil {
			t.Errorf("expected error for reserved task name %q", name)
		}
	}
}

func TestParseTaskArgs_AllowedNames(t *testing.T) {
	for _, name := range []string{"add-docs-page", "documentation", "doctor"} {
		task, _, err := parseTaskArgs([]string{"--task=" + name})
		if err != nil {
			t.Errorf("unexpected error for task name %q: %v", name, err)
		}
		if task != name {
			t.Errorf("expected task %q, got %q", name, task)
		}
	}
}
