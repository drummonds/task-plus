package ports

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// Clean kills processes listening on ports in the range [start, end] inclusive.
func Clean(start, end int) error {
	if _, err := exec.LookPath("lsof"); err != nil {
		return fmt.Errorf("lsof not found: install lsof to use clean:ports")
	}

	killed := 0
	for port := start; port <= end; port++ {
		for _, pid := range findPIDs(port) {
			fmt.Printf("Killing PID %d on port %d\n", pid, port)
			if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
				fmt.Printf("  Warning: kill %d: %v\n", pid, err)
			} else {
				killed++
			}
		}
	}
	fmt.Printf("Ports %d-%d cleared (%d processes killed)\n", start, end, killed)
	return nil
}

func findPIDs(port int) []int {
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port)).Output()
	if err != nil {
		return nil
	}
	var pids []int
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if pid, err := strconv.Atoi(strings.TrimSpace(line)); err == nil {
			pids = append(pids, pid)
		}
	}
	return pids
}
