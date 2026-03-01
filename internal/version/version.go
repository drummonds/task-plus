package version

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var semverRe = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

type Version struct {
	Major, Minor, Patch int
}

func (v Version) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// TagString returns the version without 'v' prefix (for changelogs).
func (v Version) TagString() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Parse parses a "vX.Y.Z" string.
func Parse(s string) (Version, error) {
	m := semverRe.FindStringSubmatch(s)
	if m == nil {
		return Version{}, fmt.Errorf("invalid version: %q", s)
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

// BumpPatch returns a new version with patch incremented.
func (v Version) BumpPatch() Version {
	return Version{v.Major, v.Minor, v.Patch + 1}
}

// Less returns true if v < other.
func (v Version) Less(other Version) bool {
	if v.Major != other.Major {
		return v.Major < other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor < other.Minor
	}
	return v.Patch < other.Patch
}

// LatestFromTags finds the latest non-retracted semver tag from a list of tag strings.
func LatestFromTags(tags []string, retracted ...[]Version) (Version, bool) {
	var exclude []Version
	if len(retracted) > 0 {
		exclude = retracted[0]
	}
	var versions []Version
	for _, t := range tags {
		t = strings.TrimSpace(t)
		v, err := Parse(t)
		if err == nil && !IsRetracted(v, exclude) {
			versions = append(versions, v)
		}
	}
	if len(versions) == 0 {
		return Version{}, false
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Less(versions[j])
	})
	return versions[len(versions)-1], true
}

// BumpPastRetracted bumps patch, skipping any retracted versions.
func (v Version) BumpPastRetracted(retracted []Version) Version {
	next := v.BumpPatch()
	for IsRetracted(next, retracted) {
		next = next.BumpPatch()
	}
	return next
}
