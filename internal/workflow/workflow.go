package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"codeberg.org/hum3/task-plus/internal/cleanup"
	"codeberg.org/hum3/task-plus/internal/config"
	"codeberg.org/hum3/task-plus/internal/forge"
	"codeberg.org/hum3/task-plus/internal/version"
)

// Plan holds all gathered state and user decisions for a release.
type Plan struct {
	// Gathered state
	GitDirty          bool
	StatusOutput      string
	SuggestedVersion  version.Version
	LatestTag         version.Version
	FoundTag          bool
	Retracted         []version.Version
	ReleasesToDelete  []cleanup.Deletion
	HasGoreleaserCfg  bool
	HasVersionUpdate  bool
	HasReleaseInstall bool
	Forge             forge.Forge
	HasForgeCLI       bool
	IsFork            bool
	ForkBranch        string
	LatestRC          version.Version // latest RC tag for promote mode

	// User decisions
	DoGitAdd      bool
	CommitMsg     string
	Version       version.Version
	Comment       string
	DoPush        bool
	DoGoreleaser  bool
	DoPublishPyPI bool
	DoCleanup     bool
	DoInstall     bool
	DoDeploy      bool
}

// Context carries config and flags through the workflow.
type Context struct {
	Config  *config.Config
	DryRun  bool
	RC      bool   // --rc: create release candidate tag
	Promote bool   // --promote: promote latest RC to final release
	Comment string // pre-set release comment (from --comment flag)
	Plan    Plan
}

// RunOption configures the release workflow.
type RunOption func(*Context)

// WithRC enables release candidate mode.
func WithRC() RunOption { return func(ctx *Context) { ctx.RC = true } }

// WithPromote enables promote mode (RC → final).
func WithPromote() RunOption { return func(ctx *Context) { ctx.Promote = true } }

// WithComment sets a pre-set release comment.
func WithComment(c string) RunOption {
	return func(ctx *Context) {
		if c != "" {
			ctx.Comment = c
		}
	}
}

// Run executes the full release workflow: Check → Gather → Ask → Execute.
func Run(cfg *config.Config, dryRun bool, opts ...RunOption) error {
	ctx := &Context{Config: cfg, DryRun: dryRun}
	for _, opt := range opts {
		opt(ctx)
	}

	if ctx.RC && ctx.Promote {
		return fmt.Errorf("--rc and --promote are mutually exclusive")
	}

	// Binary projects run goreleaser which requires a clean git state.
	if cfg.IsBinary() && !ctx.RC {
		if err := checkDistClean(cfg.Dir); err != nil {
			return err
		}
	}

	// 1. Precheck — fast checks before user interaction
	if len(cfg.Precheck) > 0 {
		fmt.Println("\n=== Precheck ===")
		if err := runCmds(ctx, cfg.Precheck); err != nil {
			return fmt.Errorf("precheck: %w", err)
		}
	}

	// 2. Gather — read-only state probing
	fmt.Println("\n=== Gather state ===")
	if err := Gather(ctx); err != nil {
		return fmt.Errorf("Gather: %w", err)
	}

	// 3. Ask — all user prompts
	fmt.Println("\n=== Questions ===")
	if err := Ask(ctx); err != nil {
		return fmt.Errorf("questions: %w", err)
	}

	// 4. Check — full checks (including tests) after questions
	// RC mode skips full checks — the promote step will run them.
	if !ctx.RC {
		fmt.Println("\n=== Run checks ===")
		if err := runCmds(ctx, cfg.Check); err != nil {
			return fmt.Errorf("Run checks: %w", err)
		}
	}

	// 4b. Validate deploy targets before any irreversible steps
	if ctx.Plan.DoDeploy {
		fmt.Println("\n=== Validate deploy targets ===")
		if err := validateDeploy(ctx); err != nil {
			return fmt.Errorf("deploy validation: %w", err)
		}
	}

	// 5. Execute — all mutations
	fmt.Println("\n=== Execute ===")
	if err := Execute(ctx); err != nil {
		return fmt.Errorf("Execute: %w", err)
	}

	if !ctx.RC {
		fmt.Printf("\nRelease %s complete!\n", ctx.Plan.Version)
	}
	return nil
}

// runCmds runs a list of shell commands in the project directory.
func runCmds(ctx *Context, cmds []string) error {
	for _, cmd := range cmds {
		fmt.Printf("  $ %s\n", cmd)
		if ctx.DryRun {
			continue
		}
		parts := strings.Fields(cmd)
		c := exec.Command(parts[0], parts[1:]...)
		c.Dir = ctx.Config.Dir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("%s failed: %w", cmd, err)
		}
	}
	return nil
}

// checkDistClean removes dist/ if it exists and would make git dirty for goreleaser.
func checkDistClean(dir string) error {
	distDir := filepath.Join(dir, "dist")
	if _, err := os.Stat(distDir); err == nil {
		fmt.Println("  Removing stale dist/ to ensure clean git state for goreleaser...")
		if err := os.RemoveAll(distDir); err != nil {
			return fmt.Errorf("failed to remove dist/: %w", err)
		}
	}
	return nil
}
