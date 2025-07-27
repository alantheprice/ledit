package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetGitRootDir returns the absolute path to the root of the Git repository
// containing the current working directory.
func GetGitRootDir() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git root directory: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
