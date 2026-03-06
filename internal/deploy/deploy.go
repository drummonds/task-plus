package deploy

import "fmt"

// Target describes a documentation deployment target configured in task-plus.yml.
type Target struct {
	Type string `yaml:"type"`
	Site string `yaml:"site"` // statichost site name
}

// Deployer deploys documentation to a hosting provider.
type Deployer interface {
	Name() string
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
	default:
		return nil, fmt.Errorf("unknown deploy type: %q (supported: github, statichost)", t.Type)
	}
}
