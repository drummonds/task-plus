package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"git.bytestone.uk/hum3/task-plus/internal/cleanup"
	"git.bytestone.uk/hum3/task-plus/internal/config"
	"git.bytestone.uk/hum3/task-plus/internal/forge"
	"git.bytestone.uk/hum3/task-plus/internal/git"
	"git.bytestone.uk/hum3/task-plus/internal/version"
)

// Gather performs read-only state probing to populate the plan.
func Gather(ctx *Context) error {
	p := &ctx.Plan

	// Validate at least one git remote exists
	remotes, err := git.Remotes(ctx.Config.Dir)
	if err != nil {
		return fmt.Errorf("listing git remotes: %w", err)
	}
	if len(remotes) == 0 {
		return fmt.Errorf("no git remotes configured — add a remote before releasing")
	}

	// Validate configured push-target remotes exist
	for _, name := range ctx.Config.RemoteNames() {
		if !git.HasRemote(ctx.Config.Dir, name) {
			return fmt.Errorf("configured remote %q not found in git — run 'tp check --setup' to fix the remotes config", name)
		}
	}

	// If there's a docs repo, check it has a remote origin
	if docsDir := ctx.Config.ResolveDocsRepo(); docsDir != "" {
		docsRemotes, err := git.Remotes(docsDir)
		if err != nil {
			fmt.Printf("  WARN: cannot list -docs remotes: %v\n", err)
		} else if len(docsRemotes) == 0 {
			return fmt.Errorf("docs repo %s has no git remotes — add a remote before releasing", docsDir)
		}
	}

	// Validate language marker files
	if ctx.Config.HasGo() && !ctx.Config.HasGoMod() {
		return fmt.Errorf("language 'go' configured but no go.mod found")
	}
	if ctx.Config.HasPython() && !ctx.Config.HasPyproject() {
		return fmt.Errorf("language 'python' configured but no pyproject.toml found")
	}

	// Git status
	out, err := git.Status(ctx.Config.Dir)
	if err != nil {
		return err
	}
	p.StatusOutput = out
	p.GitDirty = out != ""

	// Tags + retracted versions → suggested version
	tags, err := git.Tags(ctx.Config.Dir)
	if err != nil {
		return err
	}

	retracted, err := version.ParseRetracted(ctx.Config.Dir)
	if err != nil {
		return fmt.Errorf("parsing retracted versions: %w", err)
	}
	p.Retracted = retracted

	latest, found := version.LatestFromTags(tags, retracted)
	p.LatestTag = latest
	p.FoundTag = found
	if found {
		p.SuggestedVersion = latest.BumpPastRetracted(retracted)
	} else {
		p.SuggestedVersion = version.Version{Major: 0, Minor: 1, Patch: 0}
	}

	// RC mode: suggest next rcN for the suggested version
	if ctx.RC {
		p.SuggestedVersion = p.SuggestedVersion.BumpRC(tags)
		// Skip fork detection in RC mode
	} else if ctx.Promote {
		// Promote mode: find latest RC tag and derive final version
		rc, rcFound := version.LatestRCFromTags(tags, p.SuggestedVersion.Base())
		if !rcFound && found {
			// Try with the suggested version's base
			rc, rcFound = version.LatestRCFromTags(tags, p.SuggestedVersion.Base())
		}
		if !rcFound {
			return fmt.Errorf("no RC tags found to promote — run 'tp release --rc' first")
		}
		p.LatestRC = rc
		p.SuggestedVersion = rc.Base()
	} else {
		// Fork detection: compare go.mod module path vs primary remote
		if ctx.Config.Fork != nil {
			p.IsFork = *ctx.Config.Fork
		} else {
			modPath, err := version.ModulePath(ctx.Config.Dir)
			if err == nil {
				remotePath, err := version.GitRemoteModulePath(ctx.Config.Dir, ctx.Config.PrimaryRemote())
				if err == nil && remotePath != "" && remotePath != modPath {
					p.IsFork = true
				}
			}
		}
		if p.IsFork {
			branch, err := git.CurrentBranch(ctx.Config.Dir)
			if err != nil {
				return fmt.Errorf("getting current branch: %w", err)
			}
			p.ForkBranch = branch
			// Suggest pre-release version based on latest tag + branch name
			base := p.SuggestedVersion.Base()
			if found {
				base = latest
			}
			p.SuggestedVersion = base.BumpPrerelease(branch, tags)
		}
	}

	// Taskfile release:version-update task?
	p.HasVersionUpdate = config.HasTaskfileTask(ctx.Config.Dir, "release:version-update")

	// Taskfile release:install task?
	p.HasReleaseInstall = config.HasTaskfileTask(ctx.Config.Dir, "release:install")

	// Goreleaser config exists?
	configPath := filepath.Join(ctx.Config.Dir, ctx.Config.GoreleaserConfig)
	if _, err := os.Stat(configPath); err == nil {
		p.HasGoreleaserCfg = true
	}

	// Detect forge per configured remote; check CLI/API availability for cleanup.
	for _, r := range ctx.Config.Remotes {
		override := r.Forge
		if override == "" && r.Name == ctx.Config.GetReleaseRemote() {
			// Legacy global forge override applies to the release remote.
			override = ctx.Config.Forge
		}
		f, err := forge.Detect(ctx.Config.Dir, r.Name, override)
		if err != nil {
			return fmt.Errorf("detecting forge for %s: %w", r.Name, err)
		}
		f.TokenEnv = r.TokenEnv
		p.RemoteForges = append(p.RemoteForges, RemoteForge{
			Name:      r.Name,
			Forge:     f,
			HasAccess: f.SupportsReleases() && f.HasCLI(),
		})
	}

	if err := checkArrangement(ctx.Config, remotes, p.RemoteForges); err != nil {
		return err
	}

	// Plan release cleanup on every remote with API access (errors swallowed:
	// cleanup is optional and the push must not be blocked by a flaky API).
	for i := range p.RemoteForges {
		rf := &p.RemoteForges[i]
		if !rf.HasAccess {
			continue
		}
		releases, err := rf.Forge.ListReleases(ctx.Config.Dir)
		if err == nil {
			rf.Deletions = cleanup.PlanDeletions(releases, ctx.Config.Cleanup.KeepPatches, ctx.Config.Cleanup.KeepMinors)
		}
	}

	// The release remote's forge drives goreleaser/proxy decisions.
	for _, rf := range p.RemoteForges {
		if rf.Name == ctx.Config.GetReleaseRemote() {
			p.Forge = rf.Forge
			p.HasForgeCLI = rf.HasAccess
			break
		}
	}

	return nil
}

// checkArrangement validates that the remote/forge arrangement is configured
// well enough to release, directing the user to 'tp check --setup' otherwise.
func checkArrangement(cfg *config.Config, gitRemotes []string, forges []RemoteForge) error {
	if len(gitRemotes) > 1 && !cfg.RemotesExplicit {
		return fmt.Errorf("multiple git remotes (%s) but no remotes configured in task-plus.yml — run 'tp check --setup'",
			strings.Join(gitRemotes, ", "))
	}
	for _, rf := range forges {
		if rf.Forge.Type == forge.Unknown {
			return fmt.Errorf("remote %q (%s): unrecognised forge host — run 'tp check --setup' (or set 'forge: none' for this remote in task-plus.yml)",
				rf.Name, rf.Forge.URL)
		}
	}
	return nil
}
