package release

import (
	"fmt"
	"os"
	"os/exec"
)

// RunGoreleaser executes goreleaser release --clean in the given directory.
func RunGoreleaser(dir, configPath string) error {
	if _, err := exec.LookPath("goreleaser"); err != nil {
		return fmt.Errorf("goreleaser not found in PATH")
	}

	cmd := exec.Command("goreleaser", "release", "--clean", "--config", configPath)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
