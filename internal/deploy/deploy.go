package deploy

import (
	"fmt"
	"strings"
)

// Target describes a documentation deployment target configured in task-plus.yml.
type Target struct {
	Type   string `yaml:"type"`
	Site   string `yaml:"site"`    // site name (statichost site or /srv/sites/<site> for rsync)
	RCSite string `yaml:"rc_site"` // optional RC site for pre-release verification
	Dir    string `yaml:"dir"`     // output directory to deploy (default "docs")
	Host   string `yaml:"host"`    // rsync only: ssh target; usually supplied by task-plus.local.yaml's rsync_host
}

// SiteURL returns the public URL the deployed site is served at.
// For rsync targets with no host resolved yet it returns "".
func (t Target) SiteURL() string {
	if t.Type == "rsync" {
		if t.Host == "" {
			return ""
		}
		domain := t.Host
		if i := strings.Index(domain, "@"); i >= 0 {
			domain = domain[i+1:]
		}
		return "https://" + t.Site + "." + domain + "/"
	}
	return "https://" + t.Site + ".statichost.page/"
}

// DocsDir returns the configured output directory, defaulting to "docs".
func (t Target) DocsDir() string {
	if t.Dir != "" {
		return t.Dir
	}
	return "docs"
}

// HasRCSite returns true if this target has an RC site configured.
func (t Target) HasRCSite() bool {
	return t.RCSite != ""
}

// Deployer deploys documentation to a hosting provider.
type Deployer interface {
	Name() string
	// Validate checks that the deploy target is reachable and ready.
	// Call this before irreversible release steps to catch problems early.
	Validate() error
	Deploy(projectDir, docsDir string, dryRun bool) error
}

// New creates a Deployer for the given target configuration.
func New(t Target) (Deployer, error) {
	switch t.Type {
	case "github":
		return &GitHub{}, nil
	case "statichost":
		if t.Site == "" {
			return nil, fmt.Errorf("statichost deploy requires 'site' field")
		}
		return &Statichost{Site: t.Site}, nil
	case "rsync":
		if t.Site == "" {
			return nil, fmt.Errorf("rsync deploy requires 'site' field")
		}
		if t.Host == "" {
			return nil, fmt.Errorf("rsync deploy requires a host (set 'host:' on the target or 'rsync_host:' in task-plus.local.yaml)")
		}
		return &Rsync{Site: t.Site, Host: t.Host}, nil
	default:
		return nil, fmt.Errorf("unknown deploy type: %q (supported: github, statichost, rsync)", t.Type)
	}
}
