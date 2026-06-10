package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	// A single shared scanner so sequential prompts don't lose buffered input.
	scanner *bufio.Scanner = bufio.NewScanner(os.Stdin)
	writer  io.Writer      = os.Stdout
)

// SetIO overrides stdin/stdout for testing.
func SetIO(r io.Reader, w io.Writer) {
	scanner = bufio.NewScanner(r)
	writer = w
}

// Confirm asks a yes/no question. Default is yes.
func Confirm(msg string) bool {
	_, _ = fmt.Fprintf(writer, "%s [Y/n] ", msg)
	if !scanner.Scan() {
		return false
	}
	ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return ans == "" || ans == "y" || ans == "yes"
}

// AskString prompts for a string with a default value.
func AskString(msg, def string) string {
	if def != "" {
		_, _ = fmt.Fprintf(writer, "%s [%s]: ", msg, def)
	} else {
		_, _ = fmt.Fprintf(writer, "%s: ", msg)
	}
	if !scanner.Scan() {
		return def
	}
	ans := strings.TrimSpace(scanner.Text())
	if ans == "" {
		return def
	}
	return ans
}

// Printf writes interview/info output to the same stream as the prompts,
// so interactive flows stay capturable in tests via SetIO.
func Printf(format string, args ...any) {
	_, _ = fmt.Fprintf(writer, format, args...)
}

// Select prints numbered options and returns the chosen index.
// Empty or invalid input returns def.
func Select(msg string, options []string, def int) int {
	if def < 0 || def >= len(options) {
		def = 0
	}
	_, _ = fmt.Fprintf(writer, "%s:\n", msg)
	for i, opt := range options {
		marker := " "
		if i == def {
			marker = "*"
		}
		_, _ = fmt.Fprintf(writer, "  %s %d) %s\n", marker, i+1, opt)
	}
	_, _ = fmt.Fprintf(writer, "Choice [%d]: ", def+1)
	if !scanner.Scan() {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(strings.TrimSpace(scanner.Text()), "%d", &n); err != nil || n < 1 || n > len(options) {
		return def
	}
	return n - 1
}

// AutoConfirm controls whether prompts auto-accept defaults.
var AutoConfirm bool

// ConfirmOrAuto returns true immediately if AutoConfirm is set.
func ConfirmOrAuto(msg string) bool {
	if AutoConfirm {
		_, _ = fmt.Fprintf(writer, "%s [Y/n] y (auto)\n", msg)
		return true
	}
	return Confirm(msg)
}

// AskStringOrAuto returns the default immediately if AutoConfirm is set.
func AskStringOrAuto(msg, def string) string {
	if AutoConfirm {
		_, _ = fmt.Fprintf(writer, "%s [%s]: %s (auto)\n", msg, def, def)
		return def
	}
	return AskString(msg, def)
}
