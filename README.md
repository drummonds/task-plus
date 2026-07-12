# task-plus

Go CLI tool that standardizes common development workflows across repositories.

<!-- auto:version -->Latest: v0.1.82<!-- /auto:version -->

Try https://h3-task-plus.statichost.page/ for documentation.

## Install

```bash
go install codeberg.org/hum3/task-plus/cmd/task-plus@latest
```

A shorter alias `tp` is also available:

```bash
go install codeberg.org/hum3/task-plus/cmd/tp@latest
```

Both binaries are identical — `tp` is just shorter to type.

## Commands

| Command | Description |
|---------|-------------|
| [`check`](https://h3-task-plus.statichost.page/check.html) | Validate task-plus.yml and Taskfile.yml configuration |
| [`release`](https://h3-task-plus.statichost.page/release-workflow.html) | Interactive release workflow |
| `release:version-update` | Scaffold a Taskfile task to update version strings |
| `repos` | Manage git remotes for release |
| [`pages`](https://h3-task-plus.statichost.page/pages.html) | Serve, deploy, configure, and migrate documentation |
| `md2html` | Convert markdown files to Bulma-styled HTML |
| `md_update` | Update auto-marker sections in a markdown file (toc, pages, links) |
| `readme` | Update auto-marker sections in README.md |
| [`wt`](https://h3-task-plus.statichost.page/worktrees.html) | Manage git worktrees for isolated Claude tasks |
| [`claude`](https://h3-task-plus.statichost.page/worktrees.html#the-claude-command) | Run claude with --dangerously-skip-permissions (requires worktree + sandbox) |
| `self` | Manage task-plus itself |

### Global Flags

- `--init` — create a default `task-plus.yml` config file (statichost.eu pre-configured)
- `-a` — list available commands
- `--version` — print version

### `tp check`

Validates project configuration and prints a report. Checks:

- `task-plus.yml` — parses YAML, validates type/forge/changelog_format/deploy targets, flags unknown fields
- `Taskfile.yml` — checks standard tasks (fmt, vet, test, check), detects name conflicts and inversions
- Cross-repo — validates `-docs` sibling relationship, checks for stale `docs/` or misplaced config
- In a `-docs` repo, warns about `.md` files missing the `DOC-` prefix

```bash
tp check
tp check --dir /path/to/project
```

### `tp release`

Interactive release workflow. Replaces duplicated `task release` Taskfile patterns.

```bash
tp release
tp release --dry-run
tp release --yes --dir /path/to/project
```

Flags:
- `--dry-run` — show what would happen without making changes
- `--yes` — auto-confirm all prompts
- `--dir <path>` — project directory (default: `.`)

**Taskfile guard:** If `Taskfile.yml` contains a `release:` task, tp refuses to run (to avoid conflict).

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
10. Cleanup old releases (GitHub, Codeberg, GitLab)
11. Local install
12. Deploy documentation (if configured)
13. Run `post:release` Taskfile task (if present)

### `tp release:version-update`

Scaffolds a sample Taskfile task for updating version strings during release. The release workflow calls this task (if it exists) with `VERSION=vX.Y.Z` after the version is confirmed.

```bash
tp release:version-update --init
```

### `tp repos`

Manage which git remotes are pushed to during release.

```bash
tp repos              # show configured and available remotes
tp repos info         # same as above
tp repos add <name>   # add a git remote to the release push list
tp repos remove <name> # remove a remote from the release push list
```

### `tp pages`

Serve, deploy, and manage documentation. When run from a main project repo with a `-docs` sibling, automatically delegates to the docs repo.

```bash
tp pages                    # build and serve docs/ over HTTP
tp pages --port 3000        # custom port
tp pages deploy             # deploy to configured targets
tp pages deploy --dry-run   # show what would happen
tp pages config             # show configured deploy targets
tp pages migrate            # create a -docs sibling repo from docs/
tp pages migrate clean      # remove docs/ and pages config from main repo after migration
```

Flags (serve mode):
- `--port <n>` — HTTP port (default: `8080`)
- `--dir <path>` — project directory (default: `.`)

### `tp md2html`

Converts markdown files to Bulma-styled HTML pages with breadcrumb navigation. Supports auto-marker comments:

| Marker | Output |
|--------|--------|
| &lt;!-- auto:toc --&gt; | Table of contents from h2+ headings |
| &lt;!-- auto:pages --&gt; | Bulma sidebar nav of all HTML pages, grouped by subdirectory |
| &lt;!-- auto:links --&gt; | Project links (git remotes, statichost) |

Breadcrumbs auto-detect the docs root by walking up from `--dst` to find `index.html`/`index.md`, so subdirectory pages link correctly to the root.

By default, files whose `.html` output is at least as new as the source `.md` are skipped. Pass `--rebuild` to force a full rebuild (e.g. after a template change).

Typical docs repo Taskfile pattern — convert subdirectories first, `index.md` last so `auto:pages` sees all HTML files:

```yaml
docs:build:
  cmds:
    - task-plus md2html --src docs/research --dst docs/research
    - task-plus md2html --src docs --dst docs
    - task-plus md2html --file ../myapp/README.md --dst docs
    - task-plus md2html --file docs/index.md --dst docs  # index last (auto:pages)
```

### `tp md_update`

Updates auto-marker sections in a markdown file without converting to HTML. Same markers as `md2html` (auto:toc, auto:pages, auto:links) but writes back to the source `.md` file.

```bash
tp md_update docs/index.md              # scan file's directory for pages
tp md_update --dst docs docs/index.md   # explicit pages directory
```

### `tp wt`

Manage git worktrees for running Claude tasks in isolation. Each worktree gets its own branch (`task/<name>`), sandbox settings, and VS Code configuration. Task names are auto-prefixed with `WT` (e.g. `demo` → `WTdemo`, worktree dir `project-WTdemo`).

```bash
tp wt start my-feature             # create worktree, add it to the project .code-workspace
tp wt start my-feature /c          # shorthand: create, work, then clean up (delegates to wt clean)
tp wt agent my-feature --spec="implement login"  # register agent + run claude
tp wt review my-feature            # diff task branch against main
tp wt clean my-feature             # merge, remove from workspace + recent list, clean up
tp wt clean my-feature --yes       # same, skipping the confirmation (for agents/scripts)
tp wt list                         # list active worktrees
tp wt dashboard                    # agent dashboard (web UI; --term for terminal)
tp wt --init                       # print Taskfile snippets for wt: tasks
```

Task names "doc" and "docs" are reserved (they clash with the `-docs` repo convention).

**Editor integration.** `wt` keeps a `<project>.code-workspace` file beside the repo, listing the main checkout plus every active worktree. Open that workspace once in your editor — `wt start` then makes new worktrees appear inside that window and `wt clean` makes them disappear, with no extra windows. This works without relying on the editor's CLI control socket (often missing outside the integrated terminal, and absent entirely on some VSCodium setups). When a live control socket *is* reachable, `wt` additionally pushes `--add`/`--remove` straight into the running window, so plain folder windows update too.

### `tp claude`

Runs `claude --dangerously-skip-permissions` with safety guards. Requires:
1. Running inside a git worktree (not the main repo)
2. `.claude/settings.json` with sandbox enabled

```bash
tp claude
tp claude "implement the search feature"
```

### `tp self`

Manage the task-plus installation.

```bash
tp self update    # update to latest version via go install
```

## Config

Optional `task-plus.yml` in project root (generate with `task-plus --init`):

```yaml
type: library           # or "binary" (auto-detected from .goreleaser.yaml)
check: [task check]     # commands to run first
changelog_format: keepachangelog  # or "simple"
wasm: []                # optional WASM build commands
goreleaser_config: .goreleaser.yaml
forge: github           # forge override for the release remote (github, gitlab, forgejo)
release_remote: github  # remote whose forge drives goreleaser/proxy decisions (default: first in remotes)
remotes:                # git remotes pushed to during release (default: [origin])
  - origin              # plain string for recognised hosts
  - name: forgejo-local # map form for hosts that need overrides
    forge: forgejo      # github, gitlab, forgejo, or none (dumb mirror, no release API)
    token_env: HOMELAB_FORGEJO_TOKEN  # env var holding this host's API token
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

### Forge arrangements

`tp release` supports Codeberg + GitHub mirror, GitHub only, Codeberg only, and
self-hosted Forgejo setups. Run the interactive interview to identify your
arrangement and write it to `task-plus.yml`:

```bash
tp check --setup
```

Release cleanup deletes old releases on every configured remote whose forge is
reachable (gh/glab CLI or Forgejo API token). GitHub/GitLab commands are pinned
to the right repo via `-R owner/repo` derived from each remote's URL.

**Forgejo/Codeberg API tokens:** if a remote sets `token_env`, only that
variable is used. Otherwise the default chain is `CODEBERG_APIKEY`, then
`FORGEJO_TOKEN`, then `GITEA_TOKEN`. Goreleaser publishing to a Forgejo host
(`release: gitea:` block) expects `GITEA_TOKEN`; `tp release` passes the forge
token automatically when it's unset. Self-hosted GitLab additionally needs
`glab` host configuration — not managed by tp.

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

## Links

<!-- auto:links -->
| | |
|---|---|
| Documentation | https://h3-task-plus.statichost.page/ |
| Source (Codeberg) | https://codeberg.org/hum3/task-plus |
| Mirror (GitHub) | https://github.com/drummonds/task-plus |
<!-- /auto:links -->
