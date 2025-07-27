package workspace

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/alantheprice/ledit/pkg/types" // Updated import
	"github.com/alantheprice/ledit/pkg/utils"
)

// GetWorkspaceInfo generates a formatted string containing workspace information.
func GetWorkspaceInfo(workspace *types.WorkspaceFile, fullContextFiles, summaryContextFiles []string) string {
	var sb strings.Builder

	sb.WriteString("--- Workspace File System Structure ---\n")
	sb.WriteString(workspace.FileSystem.BaseFolderStructure) // Use BaseFolderStructure
	sb.WriteString("\n")

	if workspace.GitInfo.CurrentBranch != "" {
		sb.WriteString("--- Git Repository Info ---\n")
		sb.WriteString(fmt.Sprintf("Current Branch: %s\n", workspace.GitInfo.CurrentBranch))
		sb.WriteString(fmt.Sprintf("Last Commit: %s\n", workspace.GitInfo.LastCommit))
		sb.WriteString(fmt.Sprintf("Uncommitted Changes: %t\n", workspace.GitInfo.Uncommitted))
		if len(workspace.GitInfo.Remotes) > 0 {
			sb.WriteString("Remotes:\n")
			for _, remote := range workspace.GitInfo.Remotes {
				sb.WriteString(fmt.Sprintf("  - %s\n", remote))
			}
		}
		sb.WriteString("\n")
	}

	if len(workspace.IgnoredFiles) > 0 {
		sb.WriteString("--- Ignored Files (by .leditignore) ---\n")
		for _, ignored := range workspace.IgnoredFiles {
			sb.WriteString(fmt.Sprintf("- %s\n", ignored))
		}
		sb.WriteString("\n")
	}

	if len(fullContextFiles) > 0 {
		sb.WriteString("### Full Context Files:\n")
		for _, file := range fullContextFiles {
			fileInfo, exists := workspace.Files[file]
			if exists && fileInfo.Summary != "File is too large to analyze." && fileInfo.Summary != "Skipped due to confirmed security concerns." {
				content, err := utils.ReadFile(file)
				if err != nil {
					sb.WriteString(fmt.Sprintf("```error # %s\nCould not read file: %v\n```\n", file, err))
					continue
				}
				sb.WriteString(fmt.Sprintf("```%s # %s\n%s\n```END\n", getLanguageFromFilename(file), file, content))
			} else if exists && fileInfo.Summary == "Skipped due to confirmed security concerns." {
				sb.WriteString(fmt.Sprintf("```text # %s\nSummary: %s\nSecurity Concerns: %s\n```END\n", file, fileInfo.Summary, strings.Join(fileInfo.SecurityConcerns, ", ")))
			} else if exists && fileInfo.Summary == "File is too large to analyze." {
				sb.WriteString(fmt.Sprintf("```text # %s\nSummary: %s\n```END\n", file, fileInfo.Summary))
			} else {
				sb.WriteString(fmt.Sprintf("```text # %s\nSummary: File not found in workspace or could not be analyzed.\n```END\n", file))
			}
		}
		sb.WriteString("\n")
	}

	if len(summaryContextFiles) > 0 {
		sb.WriteString("### Summary Context Files:\n")
		for _, file := range summaryContextFiles {
			fileInfo, exists := workspace.Files[file]
			if exists {
				sb.WriteString(fmt.Sprintf("%s\nSummary: %s\nExports: %s\nReferences: %s\n", file, fileInfo.Summary, fileInfo.Exports, strings.Join(fileInfo.References, ", ")))
				if len(fileInfo.SecurityConcerns) > 0 {
					sb.WriteString(fmt.Sprintf("Security Concerns: %s\n", strings.Join(fileInfo.SecurityConcerns, ", ")))
				}
				sb.WriteString("\n")
			} else {
				sb.WriteString(fmt.Sprintf("%s\nSummary: File not found in workspace or could not be analyzed.\n\n", file))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("--- End of full content from workspace ---\n")
	return sb.String()
}

// getLanguageFromFilename infers the programming language from a file's extension.
func getLanguageFromFilename(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".hpp":
		return "cpp"
	case ".sh", ".bash":
		return "bash"
	case ".md":
		return "markdown"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".xml":
		return "xml"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".sql":
		return "sql"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".kt":
		return "kotlin"
	case ".rs":
		return "rust"
	case ".dart":
		return "dart"
	case ".pl", ".pm":
		return "perl"
	case ".lua":
		return "lua"
	case ".vim":
		return "vimscript"
	case ".toml":
		return "toml"
	default:
		return "text" // Default to plain text
	}
}
