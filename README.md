# task-plus

Go CLI tool that standardizes common development workflows across repositories. Subcommand architecture — start with `release`, more commands to come.

## Install

```bash
go install github.com/drummonds/task-plus/cmd/task-plus@latest
```

A shorter alias `tp` is also available:

```bash
go install github.com/drummonds/task-plus/cmd/tp@latest
```

Both binaries are identical — `tp` is just shorter to type.

## Commands

### `task-plus release`

Interactive release workflow. Replaces duplicated `task release` Taskfile patterns.

```bash
task-plus release
task-plus release --dry-run
task-plus release --yes --dir /path/to/project
```

Flags:
- `--dry-run` — show what would happen without making changes
- `--yes` — auto-confirm all prompts
- `--dir <path>` — project directory (default: `.`)

**Taskfile guard:** If the project's `Taskfile.yml` contains a `release:` task, `task-plus release` refuses to run (to avoid conflict). Remove the Taskfile release task and use `task-plus release` directly.

#### Release Workflow

1. Run checks (e.g. `task check`)
2. Show git status
3. Git add / commit (if dirty)
4. Detect version (latest tag + patch bump)
5. Update CHANGELOG.md
6. Git tag
7. WASM build (if configured)
8. Git push (branch + tags)
9. Goreleaser (if binary project)
10. Cleanup old GitHub releases
11. Local install
12. Deploy documentation (if configured)

### `task-plus pages`

Serve the `docs/` directory over HTTP for local preview.

```bash
task-plus pages
task-plus pages --port 3000 --dir /path/to/project
```

Flags:
- `--port <n>` — HTTP port (default: `8080`)
- `--dir <path>` — project directory (default: `.`)

### Global Flags

- `--init` — create a default `task-plus.yml` config file (statichost.eu pre-configured)
- `-a` — list available commands
- `--version` — print version

## Config

Optional `task-plus.yml` in project root (generate with `task-plus --init`):

```yaml
type: library           # or "binary" (auto-detected from .goreleaser.yaml)
check: [task check]     # commands to run first
changelog_format: keepachangelog  # or "simple"
wasm: []                # optional WASM build commands
goreleaser_config: .goreleaser.yaml
install: true           # auto-run "go install" (skip prompt; omit to be asked)
cleanup:
  keep_patches: 2       # per minor version
  keep_minors: 5
pages_build: [task docs:build]  # commands to build docs before serving/deploying
pages_deploy:                   # deploy docs during release (multiple targets supported)
  - type: github                # push docs/ to gh-pages branch
  - type: statichost
    site: myproject             # site name on statichost.eu
```

All fields optional — sensible defaults are auto-detected.

### Documentation Deployment

Configure `pages_deploy` in `task-plus.yml` to deploy documentation as part of the release workflow. Multiple targets can be active simultaneously.

**Supported providers:**

| Type | Description | Requirements |
|------|-------------|--------------|
| `github` | Pushes `docs/` to `gh-pages` branch via `git subtree push` | Git remote configured |
| `statichost` | Uploads `docs/` to [statichost.eu](https://www.statichost.eu/) | `site` field required; uses `shcli` (auto-downloaded if missing) |

If `pages_build` commands are configured, they run before deployment.

Example `task-plus.yml` for deploying to both GitHub Pages and statichost.eu:

```yaml
pages_build: [task docs:build]
pages_deploy:
  - type: github
  - type: statichost
    site: my-docs
```
