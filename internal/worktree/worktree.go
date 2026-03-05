// Package worktree manages git worktrees for running Claude tasks in isolation.
package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const settingsJSON = `{
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true,
    "filesystem": {
      "denyRead": ["~/.ssh", "~/.aws"]
    }
  }
}
`

// Sandbox stub files that Claude Code creates as a known bug.
var sandboxStubs = []string{
	".bashrc",
	".gitconfig",
	"HEAD",
}

// Run dispatches wt sub-subcommands.
func Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: task-plus wt <start|review|merge|clean|list|--init>")
	}

	switch args[0] {
	case "start":
		return runStart(args[1:])
	case "review":
		return runReview(args[1:])
	case "merge":
		return runMerge(args[1:])
	case "clean":
		return runClean(args[1:])
	case "list":
		return runList(args[1:])
	case "--init":
		printInit()
		return nil
	default:
		return fmt.Errorf("unknown wt command: %s\nUsage: task-plus wt <start|review|merge|clean|list|--init>", args[0])
	}
}

func runStart(args []string) error {
	task, spec, dir, err := parseStartArgs(args)
	if err != nil {
		return err
	}

	projName, err := projectName(dir)
	if err != nil {
		return err
	}

	wtPath := worktreePath(dir, projName, task)
	branch := "task/" + task

	// Create worktree
	fmt.Printf("Creating worktree at %s on branch %s\n", wtPath, branch)
	if err := git(dir, "worktree", "add", wtPath, "-b", branch); err != nil {
		return fmt.Errorf("git worktree add: %w", err)
	}

	// Write .claude/settings.json
	claudeDir := filepath.Join(wtPath, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("mkdir .claude: %w", err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
		return fmt.Errorf("write settings.json: %w", err)
	}

	// Add ignores to worktree's git exclude (avoids modifying tracked .gitignore)
	if err := addToGitExclude(wtPath, append([]string{".claude/settings.json"}, sandboxStubs...)); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not update git exclude: %v\n", err)
	}

	// Also ensure .claude/settings.json is in the main repo's .gitignore
	addToGitignore(dir, []string{".claude/settings.json"})

	// Run claude
	if spec != "" {
		fmt.Printf("Running claude in %s\n", wtPath)
		c := exec.Command("claude", spec)
		c.Dir = wtPath
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin
		if err := c.Run(); err != nil {
			return fmt.Errorf("claude: %w", err)
		}
	}

	return nil
}

func runReview(args []string) error {
	task, dir, err := parseTaskArgs(args)
	if err != nil {
		return err
	}
	branch := "task/" + task

	cmd := exec.Command("git", "-C", dir, "diff", "main..."+branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runMerge(args []string) error {
	task, dir, err := parseTaskArgs(args)
	if err != nil {
		return err
	}

	projName, err := projectName(dir)
	if err != nil {
		return err
	}

	wtPath := worktreePath(dir, projName, task)
	branch := "task/" + task

	fmt.Printf("Merging %s into current branch\n", branch)
	if err := git(dir, "merge", branch); err != nil {
		return fmt.Errorf("merge: %w", err)
	}

	fmt.Printf("Removing worktree at %s\n", wtPath)
	if err := git(dir, "worktree", "remove", wtPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: worktree remove: %v\n", err)
	}

	if err := git(dir, "branch", "-d", branch); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: branch delete: %v\n", err)
	}

	return nil
}

func runClean(args []string) error {
	task, dir, err := parseTaskArgs(args)
	if err != nil {
		return err
	}

	projName, err := projectName(dir)
	if err != nil {
		return err
	}

	wtPath := worktreePath(dir, projName, task)
	branch := "task/" + task

	fmt.Printf("Removing worktree at %s (force)\n", wtPath)
	if err := git(dir, "worktree", "remove", wtPath, "--force"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: worktree remove: %v\n", err)
	}

	if err := git(dir, "branch", "-D", branch); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: branch delete: %v\n", err)
	}

	return nil
}

func runList(args []string) error {
	dir := "."
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--dir" {
			dir = args[i+1]
		}
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	return git(absDir, "worktree", "list")
}

// parseStartArgs extracts --task, --spec, --dir from args.
func parseStartArgs(args []string) (task, spec, dir string, err error) {
	dir = "."
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--task" && i+1 < len(args):
			task = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--task="):
			task = args[i][len("--task="):]
		case args[i] == "--spec" && i+1 < len(args):
			spec = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--spec="):
			spec = args[i][len("--spec="):]
		case args[i] == "--dir" && i+1 < len(args):
			dir = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--dir="):
			dir = args[i][len("--dir="):]
		}
	}
	if task == "" {
		return "", "", "", fmt.Errorf("--task is required\nUsage: task-plus wt start --task=<name> --spec=\"<prompt>\"")
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", "", "", err
	}
	return task, spec, absDir, nil
}

// parseTaskArgs extracts --task and --dir from args.
func parseTaskArgs(args []string) (task, dir string, err error) {
	dir = "."
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--task" && i+1 < len(args):
			task = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--task="):
			task = args[i][len("--task="):]
		case args[i] == "--dir" && i+1 < len(args):
			dir = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--dir="):
			dir = args[i][len("--dir="):]
		}
	}
	if task == "" {
		return "", "", fmt.Errorf("--task is required")
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}
	return task, absDir, nil
}

func projectName(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return filepath.Base(strings.TrimSpace(string(out))), nil
}

func worktreePath(dir, projName, task string) string {
	// Place worktree alongside the repo root
	topOut, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return filepath.Join(filepath.Dir(dir), projName+"-"+task)
	}
	top := strings.TrimSpace(string(topOut))
	return filepath.Join(filepath.Dir(top), projName+"-"+task)
}

func git(dir string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// addToGitExclude writes entries to the worktree's per-checkout exclude file.
func addToGitExclude(wtPath string, entries []string) error {
	out, err := exec.Command("git", "-C", wtPath, "rev-parse", "--git-dir").Output()
	if err != nil {
		return err
	}
	gitDir := strings.TrimSpace(string(out))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(wtPath, gitDir)
	}

	excludePath := filepath.Join(gitDir, "info", "exclude")
	if err := os.MkdirAll(filepath.Dir(excludePath), 0755); err != nil {
		return err
	}

	existing, _ := os.ReadFile(excludePath)
	content := string(existing)

	var toAdd []string
	for _, entry := range entries {
		if !strings.Contains(content, entry) {
			toAdd = append(toAdd, entry)
		}
	}

	if len(toAdd) > 0 {
		f, err := os.OpenFile(excludePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		if len(content) > 0 && content[len(content)-1] != '\n' {
			f.WriteString("\n")
		}
		f.WriteString("# task-plus worktree sandbox\n")
		for _, entry := range toAdd {
			f.WriteString(entry + "\n")
		}
	}
	return nil
}

// addToGitignore appends entries to .gitignore if not already present.
func addToGitignore(dir string, entries []string) {
	gitignorePath := filepath.Join(dir, ".gitignore")
	existing, _ := os.ReadFile(gitignorePath)
	content := string(existing)

	var toAdd []string
	for _, entry := range entries {
		if !strings.Contains(content, entry) {
			toAdd = append(toAdd, entry)
		}
	}

	if len(toAdd) > 0 {
		f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		defer f.Close()

		if len(content) > 0 && content[len(content)-1] != '\n' {
			f.WriteString("\n")
		}
		for _, entry := range toAdd {
			f.WriteString(entry + "\n")
		}
	}
}

func printInit() {
	fmt.Print(`# Add these to your Taskfile.yml.
# Requires task-plus to be installed.
#
# Usage:
#   task wt:start TASK=my-feature SPEC="implement the login page"
#   task wt:review TASK=my-feature
#   task wt:merge TASK=my-feature
#   task wt:clean TASK=my-feature
#   task wt:list

  wt:start:
    desc: Create a worktree and run Claude in it
    requires:
      vars: [TASK, SPEC]
    cmds:
      - task-plus wt start --task={{.TASK}} --spec="{{.SPEC}}"

  wt:review:
    desc: Review changes in a worktree task
    requires:
      vars: [TASK]
    cmds:
      - task-plus wt review --task={{.TASK}}

  wt:merge:
    desc: Merge task branch and remove worktree
    requires:
      vars: [TASK]
    cmds:
      - task-plus wt merge --task={{.TASK}}

  wt:clean:
    desc: Remove worktree and delete branch without merging
    requires:
      vars: [TASK]
    cmds:
      - task-plus wt clean --task={{.TASK}}

  wt:list:
    desc: List active worktrees
    cmds:
      - task-plus wt list
`)
}
