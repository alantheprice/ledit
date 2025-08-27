package workspace

import (
	"io/fs"
	"path/filepath"

	"github.com/alantheprice/ledit/pkg/config"
	"github.com/alantheprice/ledit/pkg/workspaceinfo"
)

// validateAndUpdateWorkspace validates and updates workspace information
func validateAndUpdateWorkspace(rootDir string, cfg *config.Config) (workspaceinfo.WorkspaceFile, error) {
	// Create new workspace if file doesn't exist
	ws := workspaceinfo.WorkspaceFile{
		Files:     make(map[string]workspaceinfo.WorkspaceFileInfo),
		Languages: []string{},
	}

	// Detect languages if not already done
	if len(ws.Languages) == 0 {
		ws.Languages = detectLanguages(rootDir)
	}

	// Detect project insights
	ws.ProjectInsights = detectProjectInsightsHeuristics(rootDir, ws)

	// Detect build command if not set
	if ws.BuildCommand == "" {
		ws.BuildCommand = detectBuildCommand(rootDir)
	}

	// Detect workspace context
	detectWorkspaceContext(&ws, rootDir, nil)

	// Return the workspace (saving is handled elsewhere)
	return ws, nil
}

// detectBuildCommand detects the build command for the project
func detectBuildCommand(rootDir string) string {
	// Check for common build files and return appropriate commands
	if exists(filepath.Join(rootDir, "package.json")) {
		return "npm run build"
	}
	if exists(filepath.Join(rootDir, "go.mod")) {
		return "go build"
	}
	if exists(filepath.Join(rootDir, "Cargo.toml")) {
		return "cargo build"
	}
	if exists(filepath.Join(rootDir, "Makefile")) {
		return "make"
	}
	return ""
}

// detectWorkspaceContext detects and updates workspace context information
func detectWorkspaceContext(workspace *workspaceinfo.WorkspaceFile, rootDir string, logger interface{}) {
	// Basic context detection - can be expanded as needed
	if workspace.ProjectInsights.Monorepo == "" {
		if hasMultiplePackageFiles(rootDir) {
			workspace.ProjectInsights.Monorepo = "yes"
		} else {
			workspace.ProjectInsights.Monorepo = "no"
		}
	}
}

// hasMultiplePackageFiles checks if there are multiple package management files
func hasMultiplePackageFiles(rootDir string) bool {
	packageFiles := []string{"package.json", "go.mod", "Cargo.toml", "pom.xml"}
	count := 0
	
	for _, file := range packageFiles {
		if exists(filepath.Join(rootDir, file)) {
			count++
		}
	}
	
	return count > 1
}

// detectLanguages detects programming languages in the project
func detectLanguages(rootDir string) []string {
	langMap := make(map[string]bool)
	
	// Simple language detection based on file extensions
	extToLang := map[string]string{
		".go":   "go",
		".js":   "javascript", 
		".ts":   "typescript",
		".py":   "python",
		".java": "java",
		".rb":   "ruby",
		".rs":   "rust",
		".cpp":  "cpp",
		".c":    "c",
		".php":  "php",
		".cs":   "csharp",
	}

	// Walk directory and detect languages
	filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		
		ext := filepath.Ext(path)
		if lang, found := extToLang[ext]; found {
			langMap[lang] = true
		}
		
		return nil
	})
	
	// Convert map to slice
	var languages []string
	for lang := range langMap {
		languages = append(languages, lang)
	}
	
	return languages
}