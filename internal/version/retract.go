package version

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// ParseRetracted reads go.mod in dir and returns all retracted versions.
func ParseRetracted(dir string) ([]Version, error) {
	f, err := os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		return nil, nil // no go.mod → no retractions
	}
	defer f.Close()

	var retracted []Version
	inBlock := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "retract (") {
			inBlock = true
			continue
		}
		if inBlock && line == ")" {
			inBlock = false
			continue
		}

		// Single-line: retract v1.2.3 // comment
		if strings.HasPrefix(line, "retract ") && !inBlock {
			ver := extractVersion(strings.TrimPrefix(line, "retract "))
			if v, err := Parse(ver); err == nil {
				retracted = append(retracted, v)
			}
			continue
		}

		// Inside block: v1.2.3 // comment
		if inBlock {
			ver := extractVersion(line)
			if v, err := Parse(ver); err == nil {
				retracted = append(retracted, v)
			}
		}
	}
	return retracted, scanner.Err()
}

// extractVersion pulls the version string from a retract line,
// stripping comments and whitespace.
func extractVersion(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "//"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

// IsRetracted checks if v is in the retracted set.
func IsRetracted(v Version, retracted []Version) bool {
	return slices.Contains(retracted, v)
}
