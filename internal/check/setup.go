package check

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"codeberg.org/hum3/task-plus/internal/config"
	"codeberg.org/hum3/task-plus/internal/forge"
	"codeberg.org/hum3/task-plus/internal/git"
	"codeberg.org/hum3/task-plus/internal/prompt"
	"codeberg.org/hum3/task-plus/internal/release"
	"gopkg.in/yaml.v3"
)

// remoteInfo is the interview's working state for one git remote.
type remoteInfo struct {
	name          string
	url           string
	host          string
	forgeType     forge.Type
	overrideForge string // written to config as forge: when non-empty
	tokenEnv      string // written to config as token_env: when non-empty
	selected      bool
}

// Setup interactively identifies the repo's remote arrangement and writes
// remotes/release_remote to task-plus.yml.
func Setup(dir string) error {
	cfg, err := config.Load(dir)
	if err != nil {
		return err
	}

	names, err := git.Remotes(dir)
	if err != nil {
		return fmt.Errorf("listing git remotes: %w", err)
	}
	if len(names) == 0 {
		return fmt.Errorf("no git remotes found — add one first, e.g.: git remote add origin <url>")
	}

	prompt.Printf("Git remotes:\n")
	var infos []remoteInfo
	for _, name := range names {
		url, err := git.RemoteURL(dir, name)
		if err != nil {
			continue
		}
		spec := cfg.RemoteSpec(name)
		host, _, _ := forge.ExtractOwnerRepo(url)
		ft := forge.DetectFromURL(url)
		if spec.Forge != "" {
			ft = forge.Type(spec.Forge)
		}
		prompt.Printf("  %-16s %s (%s)\n", name, url, ft)
		infos = append(infos, remoteInfo{
			name:          name,
			url:           url,
			host:          host,
			forgeType:     ft,
			overrideForge: spec.Forge,
			tokenEnv:      spec.TokenEnv,
		})
	}

	// Configured remotes missing from git: the interview output is authoritative.
	for _, name := range cfg.RemoteNames() {
		if !git.HasRemote(dir, name) {
			prompt.Printf("  Note: remote %q is in task-plus.yml but not in git — it will be dropped (or run 'git remote add %s <url>' first and re-run)\n", name, name)
		}
	}
	prompt.Printf("\n")

	// 1. Which remotes to push releases to
	for i := range infos {
		in := &infos[i]
		hint := ""
		if cfg.RemotesExplicit && !slices.Contains(cfg.RemoteNames(), in.name) {
			hint = " — not currently in task-plus.yml"
		}
		in.selected = prompt.Confirm(fmt.Sprintf("Push releases to %s (%s)%s?", in.name, in.url, hint))
	}
	var sel []remoteInfo
	for _, in := range infos {
		if in.selected {
			sel = append(sel, in)
		}
	}
	if len(sel) == 0 {
		return fmt.Errorf("at least one remote must be selected")
	}

	// 2. Forge type for unrecognised hosts
	forgeChoices := []string{"github", "gitlab", "forgejo", "none (no release API)"}
	forgeValues := []string{"github", "gitlab", "forgejo", "none"}
	for i := range sel {
		in := &sel[i]
		if in.forgeType != forge.Unknown {
			continue
		}
		idx := prompt.Select(
			fmt.Sprintf("Forge type for %s (%s)", in.name, in.host),
			forgeChoices, 2) // self-hosted instances are most often Forgejo/Gitea
		in.overrideForge = forgeValues[idx]
		in.forgeType = forge.Type(in.overrideForge)
	}

	// 3. API token env var for self-hosted Forgejo (codeberg.org uses the
	// CODEBERG_APIKEY/FORGEJO_TOKEN/GITEA_TOKEN default chain)
	for i := range sel {
		in := &sel[i]
		if in.forgeType != forge.Forgejo || strings.EqualFold(in.host, "codeberg.org") {
			continue
		}
		def := in.tokenEnv
		if def == "" {
			def = "FORGEJO_TOKEN"
		}
		in.tokenEnv = prompt.AskString(fmt.Sprintf("API token env var for %s", in.host), def)
		if os.Getenv(in.tokenEnv) == "" {
			prompt.Printf("  Warning: %s is not set in the environment\n", in.tokenEnv)
		}
	}

	// 4. Release remote (drives goreleaser/proxy decisions)
	releaseRemote := sel[0].name
	if len(sel) > 1 {
		selNames := make([]string, len(sel))
		for i, in := range sel {
			selNames[i] = in.name
		}
		def := slices.Index(selNames, cfg.GetReleaseRemote())
		if def < 0 {
			def = max(slices.Index(selNames, "origin"), 0)
		}
		releaseRemote = selNames[prompt.Select("Release remote", selNames, def)]
	}

	prompt.Printf("\nArrangement: %s\n", classifyArrangement(sel))

	// 5. Goreleaser cross-check (warn only — tp never edits .goreleaser.yaml)
	if cfg.IsBinary() {
		warnGoreleaserMismatch(dir, cfg, sel, releaseRemote)
	}

	// 6. Write
	remotes := make([]config.Remote, len(sel))
	for i, in := range sel {
		remotes[i] = config.Remote{Name: in.name, Forge: in.overrideForge, TokenEnv: in.tokenEnv}
	}
	preview, err := yaml.Marshal(struct {
		Remotes       []config.Remote `yaml:"remotes"`
		ReleaseRemote string          `yaml:"release_remote"`
	}{remotes, releaseRemote})
	if err != nil {
		return err
	}
	prompt.Printf("\ntask-plus.yml will be updated with:\n\n%s\n", preview)
	if !prompt.Confirm("Write task-plus.yml?") {
		prompt.Printf("Aborted — nothing written.\n")
		return nil
	}
	if err := config.SaveRemotes(dir, remotes, releaseRemote); err != nil {
		return err
	}
	prompt.Printf("Wrote task-plus.yml\n")
	return nil
}

// classifyArrangement names the remote/forge arrangement for the user.
func classifyArrangement(remotes []remoteInfo) string {
	isCodeberg := func(r remoteInfo) bool {
		return r.forgeType == forge.Forgejo && strings.EqualFold(r.host, "codeberg.org")
	}
	if len(remotes) == 1 {
		r := remotes[0]
		switch {
		case r.forgeType == forge.GitHub:
			return "GitHub only"
		case isCodeberg(r):
			return "Codeberg only"
		case r.forgeType == forge.Forgejo:
			return "self-hosted Forgejo (" + r.host + ")"
		case r.forgeType == forge.GitLab:
			return "GitLab only"
		}
		return "custom"
	}
	if len(remotes) == 2 {
		var codeberg, github bool
		for _, r := range remotes {
			if isCodeberg(r) {
				codeberg = true
			}
			if r.forgeType == forge.GitHub {
				github = true
			}
		}
		if codeberg && github {
			return "Codeberg + GitHub mirror"
		}
	}
	return fmt.Sprintf("custom (%d remotes)", len(remotes))
}

// warnGoreleaserMismatch prints guidance when the goreleaser release target
// doesn't match any selected remote's forge.
func warnGoreleaserMismatch(dir string, cfg *config.Config, sel []remoteInfo, releaseRemote string) {
	target, err := release.ReleaseTarget(dir, cfg.GoreleaserConfig)
	if err != nil {
		return
	}
	targetType := release.TargetForgeType(target)
	for _, in := range sel {
		if string(in.forgeType) == targetType {
			return
		}
	}
	prompt.Printf("\nWarning: goreleaser publishes releases to %s but no selected remote is %s.\n", targetType, targetType)
	for _, in := range sel {
		if in.forgeType != forge.Forgejo || in.name != releaseRemote {
			continue
		}
		_, owner, repo := forge.ExtractOwnerRepo(in.url)
		prompt.Printf("Suggested release block for %s (Forgejo uses the Gitea API):\n\n", cfg.GoreleaserConfig)
		prompt.Printf("release:\n  gitea:\n    owner: %s\n    name: %s\n\ngitea_urls:\n  api: https://%s/api/v1\n\n", owner, repo, in.host)
		prompt.Printf("Goreleaser reads the token from GITEA_TOKEN (tp release passes your forge token automatically).\n")
	}
}
