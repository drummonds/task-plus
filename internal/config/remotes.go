package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// AddRemote adds a remote name to the remotes list in task-plus.yml.
// Creates the file if it doesn't exist.
func AddRemote(dir, name string) error {
	cfg, err := Load(dir)
	if err != nil {
		return err
	}
	if slices.Contains(cfg.Remotes, name) {
		return fmt.Errorf("remote %q already configured", name)
	}

	path := filepath.Join(dir, configFile)
	data, err := os.ReadFile(path)
	if err != nil {
		// No config file — create one with just remotes
		content := fmt.Sprintf("remotes:\n  - %s\n", name)
		return os.WriteFile(path, []byte(content), 0644)
	}

	text := string(data)
	if idx := strings.Index(text, "\nremotes:"); idx >= 0 {
		// Find end of remotes block (last "  - " line after "remotes:")
		lines := strings.Split(text, "\n")
		var result []string
		inRemotes := false
		inserted := false
		for _, line := range lines {
			result = append(result, line)
			if strings.TrimSpace(line) == "remotes:" {
				inRemotes = true
				continue
			}
			if inRemotes {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "- ") {
					// Still in remotes list — check if next line exits the block
					continue
				}
				// End of remotes block — insert before this line
				if !inserted {
					// Insert before the current (non-list) line
					result = append(result[:len(result)-1], fmt.Sprintf("  - %s", name), line)
					inserted = true
				}
				inRemotes = false
			}
		}
		if !inserted {
			// Remotes block was at end of file
			result = append(result, fmt.Sprintf("  - %s", name))
		}
		return os.WriteFile(path, []byte(strings.Join(result, "\n")), 0644)
	}

	// No remotes section — need to write current defaults + new remote
	// Rebuild remotes block with existing defaults + new name
	remotes := append(cfg.Remotes, name)
	var block strings.Builder
	block.WriteString("\nremotes:\n")
	for _, r := range remotes {
		fmt.Fprintf(&block, "  - %s\n", r)
	}
	text += block.String()
	return os.WriteFile(path, []byte(text), 0644)
}

// RemoveRemote removes a remote name from the remotes list in task-plus.yml.
func RemoveRemote(dir, name string) error {
	cfg, err := Load(dir)
	if err != nil {
		return err
	}
	if !slices.Contains(cfg.Remotes, name) {
		return fmt.Errorf("remote %q not configured", name)
	}
	if len(cfg.Remotes) == 1 {
		return fmt.Errorf("cannot remove last remote %q", name)
	}

	path := filepath.Join(dir, configFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	var result []string
	inRemotes := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "remotes:" {
			inRemotes = true
			result = append(result, line)
			continue
		}
		if inRemotes {
			trimmed := strings.TrimSpace(line)
			if entry, ok := strings.CutPrefix(trimmed, "- "); ok {
				if strings.TrimSpace(entry) == name {
					continue // skip this line
				}
				result = append(result, line)
				continue
			}
			inRemotes = false
		}
		result = append(result, line)
	}
	return os.WriteFile(path, []byte(strings.Join(result, "\n")), 0644)
}
