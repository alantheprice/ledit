package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alantheprice/ledit/pkg/config"
	"github.com/alantheprice/ledit/pkg/git"
	"github.com/alantheprice/ledit/pkg/types" // Added import for common types
)

// GatherAdditionalWorkspaceContext collects git information and file system structure.
func GatherAdditionalWorkspaceContext(cfg *config.Config) (*types.GitWorkspaceInfo, *types.FileSystemInfo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	gitInfo := &types.GitWorkspaceInfo{}
	gitRoot, err := git.GetGitRootDir()
	if err == nil {
		gitInfo.IsGitRepo = true
		gitInfo.GitRootPath = gitRoot
		gitInfo.IsGitRoot = (gitRoot == cwd)
		relPath, relErr := filepath.Rel(gitRoot, cwd)
		if relErr == nil {
			gitInfo.CurrentDirInGitRoot = relPath
		} else {
			gitInfo.CurrentDirInGitRoot = cwd // Fallback to absolute if relative fails
		}
	} else {
		gitInfo.IsGitRepo = false
	}

	fileSystemInfo := &types.FileSystemInfo{}
	var treeBuilder strings.Builder
	treeBuilder.WriteString(".\n") // Start with current directory

	// Walk the directory tree, limiting depth to 2 for brevity
	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(cwd, path)
		if err != nil {
			return err
		}

		// Skip .git and .ledit directories
		if info.IsDir() && (info.Name() == ".git" || info.Name() == ".ledit") && relPath != "." {
			return filepath.SkipDir
		}

		depth := strings.Count(relPath, string(os.PathSeparator))
		if relPath == "." {
			depth = 0
		}

		if depth <= 2 { // Limit to 2 levels deep
			if relPath != "." {
				indent := strings.Repeat("  ", depth)
				if info.IsDir() {
					treeBuilder.WriteString(fmt.Sprintf("%s├── %s/\n", indent, info.Name()))
				} else {
					treeBuilder.WriteString(fmt.Sprintf("%s├── %s\n", indent, info.Name()))
				}
			}
		} else if depth == 3 && info.IsDir() { // Add a "..." for deeper directories
			indent := strings.Repeat("  ", depth)
			treeBuilder.WriteString(fmt.Sprintf("%s└── ...\n", indent))
			return filepath.SkipDir // Skip further traversal into this directory
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Warning: Error walking file system for context: %v\n", err)
		fileSystemInfo.BaseFolderStructure = "Could not generate file system structure."
	} else {
		fileSystemInfo.BaseFolderStructure = treeBuilder.String()
	}

	return gitInfo, fileSystemInfo, nil
}

// UpdateWorkspaceFile updates the Git and FileSystem context within the provided WorkspaceFile.
func UpdateWorkspaceFile(ws *types.WorkspaceFile, cfg *config.Config) (*types.WorkspaceFile, error) {
	// Gather and update additional workspace context
	gitInfo, fileSystemInfo, err := GatherAdditionalWorkspaceContext(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to gather additional workspace context: %w", err)
	}
	ws.GitInfo = *gitInfo
	ws.FileSystem = *fileSystemInfo

	// File summaries are now handled by the calling layer (e.g., llm.GetLLMCodeResponse)
	// and are expected to be updated in the ws.Files map before saving.

	return ws, nil
}
