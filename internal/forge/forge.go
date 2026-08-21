package forge

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Type identifies a git forge provider.
type Type string

const (
	GitHub  Type = "github"
	GitLab  Type = "gitlab"
	Forgejo Type = "forgejo"
	None    Type = "none" // explicit "no release API" (e.g. a dumb mirror)
	Unknown Type = "unknown"
)

// Forge holds the detected forge type for a repository.
type Forge struct {
	Type     Type
	URL      string // git remote URL used for API calls
	TokenEnv string // env var holding the API token (Forgejo); empty = default chain
}

// Detect determines the forge from a config override or the git remote URL.
// The remote parameter specifies which git remote to inspect (e.g. "origin").
func Detect(dir, remote, override string) (Forge, error) {
	cmd := exec.Command("git", "remote", "get-url", remote)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	var url string
	if err == nil {
		url = strings.TrimSpace(string(out))
	}
	if override != "" {
		return Forge{Type: Type(override), URL: url}, nil
	}
	if url == "" {
		return Forge{Type: Unknown}, nil
	}
	return Forge{Type: detectFromURL(url), URL: url}, nil
}

// DetectFromURL returns the forge type for a git remote URL. Exported for CLI use.
func DetectFromURL(url string) Type {
	return detectFromURL(url)
}

// extractHost returns the hostname from an SSH or HTTPS git URL.
func extractHost(url string) string {
	// SSH: git@host:path
	if strings.HasPrefix(url, "git@") {
		rest := url[4:]
		if host, _, ok := strings.Cut(rest, ":"); ok {
			return host
		}
		return rest
	}
	// HTTPS: https://host/path
	if _, rest, ok := strings.Cut(url, "://"); ok {
		if i := strings.IndexAny(rest, ":/"); i >= 0 {
			return rest[:i]
		}
		return rest
	}
	return url
}

// ExtractOwnerRepo parses a git remote URL and returns host, owner, and repo.
// Supports SSH (git@host:owner/repo.git) and HTTPS (https://host/owner/repo.git).
func ExtractOwnerRepo(url string) (host, owner, repo string) {
	host = extractHost(url)
	var path string
	// SSH: git@host:owner/repo
	if strings.HasPrefix(url, "git@") && !strings.Contains(url, "://") {
		if _, rest, ok := strings.Cut(url[4:], ":"); ok {
			path = rest
		}
	} else if strings.HasPrefix(url, "ssh://") {
		// ssh://git@host/owner/repo
		if _, rest, ok := strings.Cut(url, "://"); ok {
			if hostPart, after, ok := strings.Cut(rest, "/"); ok {
				if _, h, ok := strings.Cut(hostPart, "@"); ok {
					host = h
				}
				path = after
			}
		}
	} else if _, rest, ok := strings.Cut(url, "://"); ok {
		// HTTPS: https://host/owner/repo
		if _, after, ok := strings.Cut(rest, "/"); ok {
			path = after
		}
	}
	path = strings.TrimSuffix(path, ".git")
	if o, r, ok := strings.Cut(path, "/"); ok {
		owner = o
		repo = r
	}
	return
}

// detectFromURL maps a git remote URL to a forge type.
func detectFromURL(url string) Type {
	host := strings.ToLower(extractHost(url))
	switch {
	case host == "github.com":
		return GitHub
	case host == "gitlab.com" || strings.Contains(host, "gitlab"):
		return GitLab
	case host == "codeberg.org" || strings.Contains(host, "gitea") || strings.Contains(host, "forgejo"):
		return Forgejo
	default:
		return Unknown
	}
}

// APIToken returns the Forgejo/Gitea API token for this forge. If TokenEnv is
// set, only that variable is consulted; otherwise the default chain is
// CODEBERG_APIKEY, then FORGEJO_TOKEN, then GITEA_TOKEN.
func (f Forge) APIToken() string {
	if f.TokenEnv != "" {
		return os.Getenv(f.TokenEnv)
	}
	for _, env := range []string{"CODEBERG_APIKEY", "FORGEJO_TOKEN", "GITEA_TOKEN"} {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	return ""
}

// tokenHint names the env var(s) consulted by APIToken, for error messages.
func (f Forge) tokenHint() string {
	if f.TokenEnv != "" {
		return f.TokenEnv
	}
	return "CODEBERG_APIKEY/FORGEJO_TOKEN/GITEA_TOKEN"
}

// SupportsReleases returns true if the forge has a release API we can use.
func (f Forge) SupportsReleases() bool {
	switch f.Type {
	case GitHub, GitLab, Forgejo:
		return true
	default:
		return false
	}
}

// IsPublicHost returns true for hosts reachable by public infrastructure
// such as proxy.golang.org (as opposed to a local/self-hosted forge).
func IsPublicHost(host string) bool {
	switch strings.ToLower(host) {
	case "github.com", "gitlab.com", "codeberg.org", "gitea.com":
		return true
	default:
		return false
	}
}

// HasCLI returns true if the appropriate CLI tool is available in PATH
// (or, for Forgejo, an API token is set).
func (f Forge) HasCLI() bool {
	switch f.Type {
	case GitHub:
		_, err := exec.LookPath("gh")
		return err == nil
	case GitLab:
		_, err := exec.LookPath("glab")
		return err == nil
	case Forgejo:
		return f.APIToken() != ""
	default:
		return false
	}
}

// repoFlag returns ["-R", "owner/repo"] (or host/owner/repo for non-github.com
// hosts) so gh/glab target the right repo when several remotes exist.
func (f Forge) repoFlag() []string {
	host, owner, repo := ExtractOwnerRepo(f.URL)
	if owner == "" || repo == "" {
		return nil
	}
	spec := owner + "/" + repo
	if !strings.EqualFold(host, "github.com") && !strings.EqualFold(host, "gitlab.com") {
		spec = host + "/" + spec
	}
	return []string{"-R", spec}
}

// Archive marks the repository archived (read-only) on the forge.
func (f Forge) Archive(dir string) error {
	switch f.Type {
	case GitHub:
		_, owner, repo := ExtractOwnerRepo(f.URL)
		if owner == "" || repo == "" {
			return fmt.Errorf("cannot parse owner/repo from %q", f.URL)
		}
		cmd := exec.Command("gh", "repo", "archive", owner+"/"+repo, "--yes")
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("gh repo archive: %w\n%s", err, out)
		}
		return nil
	case Forgejo:
		token := f.APIToken()
		if token == "" {
			return fmt.Errorf("no API token set (%s)", f.tokenHint())
		}
		return archiveRepoForgejo(f.URL, token, true)
	default:
		return fmt.Errorf("forge type %q does not support archiving", f.Type)
	}
}

// ListReleases returns release tag names from the forge.
func (f Forge) ListReleases(dir string) ([]string, error) {
	switch f.Type {
	case GitHub:
		return listReleasesGitHub(dir, f.repoFlag())
	case GitLab:
		return listReleasesGitLab(dir, f.repoFlag())
	case Forgejo:
		return listReleasesForgejo(f.URL, f.APIToken())
	default:
		return nil, fmt.Errorf("forge type %q has no release API", f.Type)
	}
}

// DeleteRelease deletes a release by tag on the forge.
func (f Forge) DeleteRelease(dir, tag string) error {
	switch f.Type {
	case GitHub:
		return deleteReleaseGitHub(dir, tag, f.repoFlag())
	case GitLab:
		return deleteReleaseGitLab(dir, tag, f.repoFlag())
	case Forgejo:
		token := f.APIToken()
		if token == "" {
			return fmt.Errorf("no API token set (%s)", f.tokenHint())
		}
		return deleteReleaseForgejo(f.URL, tag, token)
	default:
		return fmt.Errorf("forge type %q has no release API", f.Type)
	}
}

func listReleasesGitHub(dir string, repoFlag []string) ([]string, error) {
	args := append([]string{"release", "list", "--limit", "100", "--json", "tagName", "-q", ".[].tagName"}, repoFlag...)
	cmd := exec.Command("gh", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gh release list: %w\n%s", err, out)
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil, nil
	}
	return strings.Split(s, "\n"), nil
}

func deleteReleaseGitHub(dir, tag string, repoFlag []string) error {
	args := append([]string{"release", "delete", tag, "--yes", "--cleanup-tag"}, repoFlag...)
	cmd := exec.Command("gh", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh release delete %s: %w\n%s", tag, err, out)
	}
	return nil
}

func listReleasesGitLab(dir string, repoFlag []string) ([]string, error) {
	args := append([]string{"release", "list", "--per-page", "100"}, repoFlag...)
	cmd := exec.Command("glab", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("glab release list: %w\n%s", err, out)
	}
	return parseGLabReleaseList(string(out)), nil
}

// parseGLabReleaseList extracts version tags from glab release list output.
// Each line's first whitespace-delimited field is checked for a leading "v".
func parseGLabReleaseList(output string) []string {
	var tags []string
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 && strings.HasPrefix(fields[0], "v") {
			tags = append(tags, fields[0])
		}
	}
	return tags
}

func deleteReleaseGitLab(dir, tag string, repoFlag []string) error {
	args := append([]string{"release", "delete", tag, "-y", "--with-tag"}, repoFlag...)
	cmd := exec.Command("glab", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("glab release delete %s: %w\n%s", tag, err, out)
	}
	return nil
}
