package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetGitRootDir returns the absolute path to the root directory of the current Git repository.
func GetGitRootDir() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	var out []byte
	var err error
	if out, err = cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("could not find git root: %v", string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// GetFileGitPath returns the path of the given filename relative to the Git repository root.
func GetFileGitPath(filename string) (string, error) {
	gitRoot, err := GetGitRootDir()
	if err != nil {
		return filename, fmt.Errorf("failed to get git root directory: %w", err)
	}
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return filename, fmt.Errorf("failed to get absolute path for %s: %w", filename, err)
	}
	relPath, err := filepath.Rel(gitRoot, absPath)
	if err != nil {
		return filename, fmt.Errorf("failed to get relative path for %s: %w", filename, err)
	}
	return relPath, nil
}

// AddAndCommitFile stages the specified file and commits it with the given message.
func AddAndCommitFile(newFilename, message string) error {
	if err := exec.Command("git", "add", newFilename).Run(); err != nil {
		return fmt.Errorf("error adding changes to git: %v", err)
	}
	if err := exec.Command("git", "commit", "-m", message).Run(); err != nil {
		return fmt.Errorf("error committing changes to git: %v", err)
	}
	fmt.Printf("Changes committed to git for %s\n", newFilename)
	return nil
}

// GetGitStatus returns the current branch, number of uncommitted changes, and number of staged changes.
func GetGitStatus() (currentBranch string, uncommittedChanges int, stagedChanges int, err error) {
	// Get current branch
	cmdBranch := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchOut, err := cmdBranch.CombinedOutput()
	if err != nil {
		// If not in a git repo, or no commits yet, this might fail.
		// Return empty string and 0 counts, but still indicate an error if it's not just "not a git repo".
		if strings.Contains(strings.ToLower(string(branchOut)), "not a git repository") {
			return "", 0, 0, nil // Not an error if it's just not a git repo
		}
		return "", 0, 0, fmt.Errorf("failed to get git branch: %v", string(branchOut))
	}
	currentBranch = strings.TrimSpace(string(branchOut))

	// Get status --porcelain to count changes
	cmdStatus := exec.Command("git", "status", "--porcelain", "-u", "--no-ahead-behind")
	statusOut, err := cmdStatus.CombinedOutput()
	if err != nil {
		return currentBranch, 0, 0, fmt.Errorf("failed to get git status: %v", string(statusOut))
	}

	lines := strings.Split(strings.TrimSpace(string(statusOut)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// X is the status of the index (staged), Y is the status of the working tree (uncommitted)
		// XY PATH
		// M  file.txt (staged modified, uncommitted modified)
		// M  file.txt (staged modified, uncommitted unchanged)
		//  M file.txt (staged unchanged, uncommitted modified)
		// A  file.txt (staged added)
		// D  file.txt (staged deleted)
		//  A file.txt (untracked added) - this is not staged or uncommitted in the sense of tracked files
		// ?? file.txt (untracked)

		// Staged changes (X column)
		if len(line) >= 1 && line[0] != ' ' && line[0] != '?' { // ' ' means not staged, '?' means untracked
			stagedChanges++
		}
		// Uncommitted changes (Y column)
		if len(line) >= 2 && line[1] != ' ' && line[1] != '?' { // ' ' means not uncommitted, '?' means untracked
			uncommittedChanges++
		}
	}

	return currentBranch, uncommittedChanges, stagedChanges, nil
}
