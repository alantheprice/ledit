package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alantheprice/ledit/pkg/types" // Updated import
)

const (
	leditDir      = ".ledit"
	workspaceFile = "workspace.json"
)

// GetWorkspaceFilePath returns the full path to the workspace.json file.
func GetWorkspaceFilePath(dir string) string {
	return filepath.Join(dir, leditDir, workspaceFile)
}

// LoadWorkspace loads the WorkspaceFile from the specified directory.
func LoadWorkspace(dir string) (*types.WorkspaceFile, error) {
	filePath := GetWorkspaceFilePath(dir)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return an empty workspace with initialized maps if file doesn't exist
			return &types.WorkspaceFile{
				Files:      make(map[string]types.FileInfo),
				GitInfo:    types.GitWorkspaceInfo{},
				FileSystem: types.FileSystemInfo{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read workspace file %s: %w", filePath, err)
	}

	var ws types.WorkspaceFile // Updated type
	if err := json.Unmarshal(data, &ws); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workspace file %s: %w", filePath, err)
	}

	// Ensure maps are initialized even if unmarshaling from an empty file or old format
	if ws.Files == nil {
		ws.Files = make(map[string]types.FileInfo) // Updated type
	}

	return &ws, nil
}

// SaveWorkspace saves the WorkspaceFile to the specified directory.
func SaveWorkspace(dir string, ws *types.WorkspaceFile) error { // Updated type
	workspaceDir := filepath.Join(dir, leditDir)
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return fmt.Errorf("failed to create .ledit directory: %w", err)
	}

	filePath := GetWorkspaceFilePath(dir)
	data, err := json.MarshalIndent(ws, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal workspace data: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write workspace file %s: %w", filePath, err)
	}
	return nil
}
