# task-release

Go CLI tool that standardizes the release workflow across repositories. Replaces duplicated `task release` Taskfile patterns with an interactive, configurable tool.

## Install

```bash
go install github.com/drummonds/task-release/cmd/task-release@latest
```

## Usage

Run in any Git repo:

```bash
task-release
```

### Flags

- `--dry-run` — show what would happen without making changes
- `--yes` — auto-confirm all prompts
- `--dir <path>` — project directory (default: `.`)
- `--version` — print version

## Workflow

1. Load config (`task-release.yml`, auto-detects defaults)
2. Run checks (e.g. `task check`)
3. Show git status
4. Git add (if dirty)
5. Git commit (if dirty)
6. Detect version (latest tag + patch bump)
7. Update CHANGELOG.md
8. Git tag
9. WASM build (if configured)
10. Git push (branch + tag)
11. Goreleaser (if binary project)
12. Cleanup old GitHub releases

## Config

Optional `task-release.yml` in project root:

```yaml
type: library           # or "binary" (auto-detected from .goreleaser.yaml)
check: [task check]     # commands to run first
changelog_format: keepachangelog  # or "simple"
wasm: []                # optional WASM build commands
goreleaser_config: .goreleaser.yaml
install: true              # auto-run "go install" (skip prompt; omit to be asked)
cleanup:
  keep_patches: 2       # per minor version
  keep_minors: 5
```

All fields optional — sensible defaults are auto-detected.
