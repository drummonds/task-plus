package deploy

import (
	"fmt"
	"os"
	"os/exec"
)

// Rsync deploys documentation to a self-hosted static docs server by
// rsyncing the built docs to /srv/sites/<site>/ over SSH. The server is
// expected to have a directory per site (created by its Caddy setup).
type Rsync struct {
	Site string
	Host string // ssh target, e.g. deploy@docs.example.com
}

func (r *Rsync) Name() string { return fmt.Sprintf("%s (%s)", r.Host, r.Site) }

// Validate checks the site directory exists on the server, which proves SSH
// works and the site has been added to the Caddyfile (setup.sh creates the
// directory from the Caddyfile's hostname list).
func (r *Rsync) Validate() error {
	cmd := exec.Command("ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10",
		r.Host, "test -d /srv/sites/"+r.Site)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("site %q not ready on %s (add it to the server's Caddyfile and re-apply): %v %s",
			r.Site, r.Host, err, out)
	}
	return nil
}

func (r *Rsync) Deploy(projectDir, docsDir string, dryRun bool) error {
	if _, err := os.Stat(docsDir); os.IsNotExist(err) {
		return fmt.Errorf("docs directory not found: %s", docsDir)
	}
	if dryRun {
		fmt.Printf("  (dry-run) Would rsync %s to %s:/srv/sites/%s/\n", docsDir, r.Host, r.Site)
		return nil
	}
	if err := r.Validate(); err != nil {
		return err
	}
	fmt.Printf("  Rsyncing to %s:/srv/sites/%s/ ...\n", r.Host, r.Site)
	cmd := exec.Command("rsync", "-az", "--delete",
		"-e", "ssh -o BatchMode=yes",
		docsDir+"/", r.Host+":/srv/sites/"+r.Site+"/")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rsync deploy failed: %w", err)
	}
	return nil
}
