package cleanup

import "testing"

func TestPlanDeletions(t *testing.T) {
	tags := []string{
		"v0.3.2", "v0.3.1", "v0.3.0",
		"v0.2.3", "v0.2.2", "v0.2.1", "v0.2.0",
		"v0.1.5", "v0.1.4", "v0.1.3", "v0.1.2", "v0.1.1", "v0.1.0",
	}

	toDelete := PlanDeletions(tags, 2, 2)

	// Keep minors: 0.3, 0.2 (2 most recent)
	// Keep patches: 2 per minor
	// So keep: v0.3.2, v0.3.1, v0.2.3, v0.2.2
	// Delete: v0.3.0, v0.2.1, v0.2.0, all of v0.1.x

	deleteMap := make(map[string]string) // tag -> reason
	for _, d := range toDelete {
		deleteMap[d.Tag] = d.Reason
	}

	shouldKeep := []string{"v0.3.2", "v0.3.1", "v0.2.3", "v0.2.2"}
	for _, k := range shouldKeep {
		if _, ok := deleteMap[k]; ok {
			t.Errorf("should keep %s but it's in delete list", k)
		}
	}

	shouldDelete := []string{"v0.3.0", "v0.2.1", "v0.2.0", "v0.1.5", "v0.1.0"}
	for _, d := range shouldDelete {
		if _, ok := deleteMap[d]; !ok {
			t.Errorf("should delete %s but it's not in delete list", d)
		}
	}

	// Check reasons
	if r := deleteMap["v0.3.0"]; r != "old patch version" {
		t.Errorf("v0.3.0 reason = %q, want %q", r, "old patch version")
	}
	if r := deleteMap["v0.1.5"]; r != "old minor version (keeping 0.3.x–0.2.x)" {
		t.Errorf("v0.1.5 reason = %q, want %q", r, "old minor version (keeping 0.3.x–0.2.x)")
	}
}

func TestPlanDeletionsEmpty(t *testing.T) {
	toDelete := PlanDeletions(nil, 2, 5)
	if len(toDelete) != 0 {
		t.Errorf("expected no deletions, got %v", toDelete)
	}
}
