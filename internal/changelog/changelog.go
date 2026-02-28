package changelog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const changelogFile = "CHANGELOG.md"

// FormatEntry returns a changelog heading for the given version and format.
func FormatEntry(version, format, comment string) string {
	date := time.Now().Format("2006-01-02")
	switch format {
	case "simple":
		return fmt.Sprintf("## %s %s", version, date)
	default: // keepachangelog
		return fmt.Sprintf("## [%s] - %s", version, date)
	}
}

// Update inserts a new version entry into CHANGELOG.md.
// It places the entry after the [Unreleased] heading (if present) or at the top of the file.
func Update(dir, version, format, comment string) error {
	path := filepath.Join(dir, changelogFile)
	data, err := os.ReadFile(path)
	if err != nil {
		// Create a new changelog if it doesn't exist
		return createNew(path, version, format, comment)
	}

	content := string(data)
	entry := FormatEntry(version, format, comment)
	body := ""
	if comment != "" {
		body = "\n\n### Changed\n\n- " + comment
	}

	// Try to insert after [Unreleased]
	unreleased := "## [Unreleased]"
	if idx := strings.Index(content, unreleased); idx >= 0 {
		insertAt := idx + len(unreleased)
		// Skip any whitespace after [Unreleased]
		for insertAt < len(content) && (content[insertAt] == '\n' || content[insertAt] == '\r') {
			insertAt++
		}
		newContent := content[:idx] + unreleased + "\n\n" + entry + body + "\n\n" + content[insertAt:]
		return os.WriteFile(path, []byte(newContent), 0644)
	}

	// No [Unreleased] — find the first "## " heading and insert before it
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") {
			before := strings.Join(lines[:i], "\n")
			after := strings.Join(lines[i:], "\n")
			newContent := before + entry + body + "\n\n" + after
			return os.WriteFile(path, []byte(newContent), 0644)
		}
	}

	// No headings at all — append
	newContent := content + "\n" + entry + body + "\n"
	return os.WriteFile(path, []byte(newContent), 0644)
}

func createNew(path, version, format, comment string) error {
	entry := FormatEntry(version, format, comment)
	body := ""
	if comment != "" {
		body = "\n\n### Changed\n\n- " + comment
	}
	content := "# Changelog\n\n## [Unreleased]\n\n" + entry + body + "\n"
	return os.WriteFile(path, []byte(content), 0644)
}
