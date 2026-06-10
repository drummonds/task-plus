package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// AddRemote adds a remote name to the remotes list in task-plus.yml.
// Creates the file if it doesn't exist.
func AddRemote(dir, name string) error {
	cfg, err := Load(dir)
	if err != nil {
		return err
	}
	if slices.Contains(cfg.RemoteNames(), name) {
		return fmt.Errorf("remote %q already configured", name)
	}

	remotes := append(cfg.Remotes, Remote{Name: name})
	if _, err := os.Stat(filepath.Join(dir, configFile)); err != nil {
		// No config file — create one listing just the new remote
		remotes = []Remote{{Name: name}}
	}
	return SaveRemotes(dir, remotes, "")
}

// RemoveRemote removes a remote name from the remotes list in task-plus.yml.
func RemoveRemote(dir, name string) error {
	cfg, err := Load(dir)
	if err != nil {
		return err
	}
	if !slices.Contains(cfg.RemoteNames(), name) {
		return fmt.Errorf("remote %q not configured", name)
	}
	if len(cfg.Remotes) == 1 {
		return fmt.Errorf("cannot remove last remote %q", name)
	}

	remotes := slices.DeleteFunc(slices.Clone(cfg.Remotes), func(r Remote) bool {
		return r.Name == name
	})
	return SaveRemotes(dir, remotes, "")
}
