package release

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RunGoreleaser executes goreleaser release --clean in the given directory.
// extraEnv entries ("KEY=value") are appended to the inherited environment.
func RunGoreleaser(dir, configPath string, extraEnv ...string) error {
	if _, err := exec.LookPath("goreleaser"); err != nil {
		return fmt.Errorf("goreleaser not found in PATH")
	}

	cmd := exec.Command("goreleaser", "release", "--clean", "--config", configPath)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ReleaseTarget returns which forge the goreleaser config publishes releases
// to: "github", "gitea", "gitlab", or "" when no release block is configured
// (goreleaser then defaults to github).
func ReleaseTarget(dir, configPath string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, configPath))
	if err != nil {
		return "", err
	}
	var cfg struct {
		Release struct {
			GitHub map[string]any `yaml:"github"`
			Gitea  map[string]any `yaml:"gitea"`
			GitLab map[string]any `yaml:"gitlab"`
		} `yaml:"release"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parsing %s: %w", configPath, err)
	}
	switch {
	case cfg.Release.Gitea != nil:
		return "gitea", nil
	case cfg.Release.GitLab != nil:
		return "gitlab", nil
	case cfg.Release.GitHub != nil:
		return "github", nil
	}
	return "", nil
}

// TargetForgeType maps a goreleaser release target to the matching forge type
// name used in task-plus config ("forgejo" for gitea, etc.). An empty target
// means goreleaser's default, github.
func TargetForgeType(target string) string {
	switch target {
	case "gitea":
		return "forgejo"
	case "gitlab":
		return "gitlab"
	default:
		return "github"
	}
}
