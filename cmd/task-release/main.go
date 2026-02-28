package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/drummonds/task-release/internal/config"
	"github.com/drummonds/task-release/internal/prompt"
	"github.com/drummonds/task-release/internal/workflow"
)

var (
	// Set by goreleaser
	appVersion = "dev"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "show what would happen without making changes")
	yes := flag.Bool("yes", false, "auto-confirm all prompts")
	dir := flag.String("dir", ".", "project directory")
	showVersion := flag.Bool("version", false, "print version")
	flag.Parse()

	if *showVersion {
		fmt.Println("task-release", appVersion)
		return
	}

	if *yes {
		prompt.AutoConfirm = true
	}

	absDir, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load(absDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("task-release %s\n", appVersion)
	fmt.Printf("Project: %s (%s)\n", absDir, cfg.Type)

	if err := workflow.Run(cfg, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}
}
