package version

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		input   string
		want    Version
		wantErr bool
	}{
		{"v1.2.3", Version{1, 2, 3}, false},
		{"v0.1.0", Version{0, 1, 0}, false},
		{"v10.20.30", Version{10, 20, 30}, false},
		{"1.2.3", Version{}, true},
		{"v1.2", Version{}, true},
		{"v1.2.3-beta", Version{}, true},
		{"", Version{}, true},
	}
	for _, tt := range tests {
		got, err := Parse(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestBumpPatch(t *testing.T) {
	v := Version{0, 1, 5}
	got := v.BumpPatch()
	want := Version{0, 1, 6}
	if got != want {
		t.Errorf("BumpPatch() = %v, want %v", got, want)
	}
}

func TestLatestFromTags(t *testing.T) {
	tags := []string{"v0.1.0", "v0.2.0", "v0.1.5", "not-a-tag", "v0.3.0"}
	got, found := LatestFromTags(tags)
	if !found {
		t.Fatal("expected to find a version")
	}
	want := Version{0, 3, 0}
	if got != want {
		t.Errorf("LatestFromTags() = %v, want %v", got, want)
	}
}

func TestLatestFromTagsEmpty(t *testing.T) {
	_, found := LatestFromTags(nil)
	if found {
		t.Error("expected not found for nil tags")
	}
}

func TestString(t *testing.T) {
	v := Version{1, 2, 3}
	if got := v.String(); got != "v1.2.3" {
		t.Errorf("String() = %q, want %q", got, "v1.2.3")
	}
	if got := v.TagString(); got != "1.2.3" {
		t.Errorf("TagString() = %q, want %q", got, "1.2.3")
	}
}
