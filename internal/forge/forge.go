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
	Unknown Type = "unknown"
)

// Forge holds the detected forge type for a repository.
type Forge struct {
	Type Type
	URL  string // git remote URL used for API calls
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

// HasCLI returns true if the appropriate CLI tool is available in PATH.
func (f Forge) HasCLI() bool {
	switch f.Type {
	case GitHub:
		_, err := exec.LookPath("gh")
		return err == nil
	case GitLab:
		_, err := exec.LookPath("glab")
		return err == nil
	case Forgejo:
		return os.Getenv("CODEBERG_APIKEY") != ""
	default:
		return false
	}
}

// ListReleases returns release tag names from the forge.
func (f Forge) ListReleases(dir string) ([]string, error) {
	switch f.Type {
	case GitHub:
		return listReleasesGitHub(dir)
	case GitLab:
		return listReleasesGitLab(dir)
	case Forgejo:
		return listReleasesForgejo(f.URL)
	default:
		return nil, fmt.Errorf("unknown forge type %q", f.Type)
	}
}

// DeleteRelease deletes a release by tag on the forge.
func (f Forge) DeleteRelease(dir, tag string) error {
	switch f.Type {
	case GitHub:
		return deleteReleaseGitHub(dir, tag)
	case GitLab:
		return deleteReleaseGitLab(dir, tag)
	case Forgejo:
		return deleteReleaseForgejo(f.URL, tag)
	default:
		return fmt.Errorf("unknown forge type %q", f.Type)
	}
}

func listReleasesGitHub(dir string) ([]string, error) {
	cmd := exec.Command("gh", "release", "list", "--limit", "100", "--json", "tagName", "-q", ".[].tagName")
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

func deleteReleaseGitHub(dir, tag string) error {
	cmd := exec.Command("gh", "release", "delete", tag, "--yes", "--cleanup-tag")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh release delete %s: %w\n%s", tag, err, out)
	}
	return nil
}

func listReleasesGitLab(dir string) ([]string, error) {
	cmd := exec.Command("glab", "release", "list", "--per-page", "100")
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

func deleteReleaseGitLab(dir, tag string) error {
	cmd := exec.Command("glab", "release", "delete", tag, "-y", "--with-tag")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("glab release delete %s: %w\n%s", tag, err, out)
	}
	return nil
}
