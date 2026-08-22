// Package secrets fetches API tokens from a Bitwarden vault using a
// session key cached by `tp unlock` (24h TTL, 0600 file). It replaces the
// external bw-unlock-24h / with-secrets shell scripts: tokens are read via
// the bw CLI and set only in the process environment — never printed or
// written to disk.
package secrets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const sessionTTL = 24 * time.Hour

// SessionFile returns the path of the cached Bitwarden session key.
func SessionFile() string {
	dir := os.Getenv("XDG_CACHE_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".cache")
	}
	return filepath.Join(dir, "bw-session", "session")
}

// ItemsFile returns the path of the env-var → vault-item mapping.
func ItemsFile() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "bw-session", "items.conf")
}

// Unlock prompts for the master password (via the bw CLI, which reads it
// interactively — tp never sees it) and caches the session key for 24h.
func Unlock() error {
	cmd := exec.Command("bw", "unlock", "--raw")
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bw unlock: %w", err)
	}
	key := strings.TrimSpace(out.String())
	if key == "" {
		return fmt.Errorf("bw unlock returned no session key")
	}
	path := SessionFile()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(key), 0600); err != nil {
		return err
	}
	fmt.Printf("Bitwarden session cached (%s, mode 0600). Valid for 24h.\n", path)
	return nil
}

// Session returns the cached session key, enforcing the 24h TTL. On expiry
// the key is deleted and the vault locked. The returned error message tells
// the user to run `tp unlock`.
func Session() (string, error) {
	path := SessionFile()
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("no cached Bitwarden session — run: tp unlock")
	}
	if time.Since(info.ModTime()) > sessionTTL {
		_ = os.Remove(path)
		_ = exec.Command("bw", "lock").Run()
		return "", fmt.Errorf("Bitwarden session expired (>24h) — run: tp unlock")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// SessionState reports the cache state for diagnostics without touching the
// vault: "ok" with remaining validity, or a problem description.
func SessionState() string {
	info, err := os.Stat(SessionFile())
	if err != nil {
		return "no session — run: tp unlock"
	}
	age := time.Since(info.ModTime())
	if age > sessionTTL {
		return "expired — run: tp unlock"
	}
	return fmt.Sprintf("ok, %s left", (sessionTTL - age).Round(time.Minute))
}

// item is one entry of items.conf: env var → vault item (password field, or
// a named custom field).
type item struct {
	Var   string
	ID    string
	Field string
}

// parseItems parses items.conf data: ENV_VAR<TAB>item-id[<TAB>field] per
// line, # comments and blank lines ignored.
func parseItems(data []byte) []item {
	var items []item
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			continue
		}
		it := item{Var: parts[0], ID: parts[1]}
		if len(parts) > 2 {
			it.Field = parts[2]
		}
		items = append(items, it)
	}
	return items
}

// Load fetches every token mapped in items.conf that is not already set in
// the environment, and sets it with os.Setenv so child processes inherit it.
func Load() error {
	session, err := Session()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(ItemsFile())
	if err != nil {
		return fmt.Errorf("missing %s (ENV_VAR<TAB>item[<TAB>field] per line)", ItemsFile())
	}
	for _, it := range parseItems(data) {
		if os.Getenv(it.Var) != "" {
			continue
		}
		val, err := fetch(session, it)
		if err != nil {
			return err
		}
		if err := os.Setenv(it.Var, val); err != nil {
			return err
		}
	}
	return nil
}

// fetch reads one secret from the vault via the bw CLI.
func fetch(session string, it item) (string, error) {
	var args []string
	if it.Field == "" {
		args = []string{"get", "password", it.ID}
	} else {
		args = []string{"get", "item", it.ID}
	}
	cmd := exec.Command("bw", args...)
	cmd.Env = append(os.Environ(), "BW_SESSION="+session)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("bw get %s (%s): %v: %s", it.ID, it.Var, err, strings.TrimSpace(errb.String()))
	}
	if it.Field == "" {
		return strings.TrimSpace(out.String()), nil
	}
	var parsed struct {
		Fields []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		return "", fmt.Errorf("bw get item %s: %w", it.ID, err)
	}
	for _, f := range parsed.Fields {
		if f.Name == it.Field {
			return f.Value, nil
		}
	}
	return "", fmt.Errorf("field %q of item %s (%s) not found", it.Field, it.ID, it.Var)
}
