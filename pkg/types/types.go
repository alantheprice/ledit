package types

import (
	"time"
)

// FileInfo represents analyzed information for a single file.
type FileInfo struct {
	Path                    string    `json:"path"`
	Hash                    string    `json:"hash"`
	Summary                 string    `json:"summary"`
	Exports                 string    `json:"exports"`
	References              []string  `json:"references"`
	TokenCount              int       `json:"token_count"`
	LastAnalyzed            time.Time `json:"last_analyzed"`
	SecurityConcerns        []string  `json:"security_concerns"`
	IgnoredSecurityConcerns []string  `json:"ignored_security_concerns"`
}

// WorkspaceFile represents the entire workspace state, including file information and additional context.
type WorkspaceFile struct {
	Files map[string]FileInfo `json:"files"`
	// Additional context can be added here if needed, e.g., GitInfo, FileSystemInfo
	GitInfo      GitWorkspaceInfo `json:"git_info"`
	FileSystem   FileSystemInfo   `json:"file_system"`
	IgnoredFiles []string         `json:"ignored_files"` // List of files/patterns ignored by .leditignore
}

// GitWorkspaceInfo holds Git repository context information.
type GitWorkspaceInfo struct {
	CurrentBranch       string   `json:"current_branch"`
	LastCommit          string   `json:"last_commit"`
	Uncommitted         bool     `json:"uncommitted"`
	Remotes             []string `json:"remotes"`
	IsGitRepo           bool     `json:"is_git_repo"`
	GitRootPath         string   `json:"git_root_path"`
	IsGitRoot           bool     `json:"is_git_root"`
	CurrentDirInGitRoot string   `json:"current_dir_in_git_root"`
}

// FileSystemInfo holds information about the file system structure.
type FileSystemInfo struct {
	Structure           string `json:"structure"` // A string representation of the directory tree
	BaseFolderStructure string `json:"base_folder_structure"`
}

// OrchestrationRequirement defines a single step in the orchestration plan.
type OrchestrationRequirement struct {
	Filepath                 string `json:"filepath"`
	Instruction              string `json:"instruction"`
	Status                   string `json:"status"` // "pending", "completed", "failed"
	ValidationFailureContext string `json:"validation_failure_context,omitempty"`
	LastLLMResponse          string `json:"last_llm_response,omitempty"`
}

// OrchestrationPlan holds the entire list of requirements for a feature.
type OrchestrationPlan struct {
	Requirements []OrchestrationRequirement `json:"requirements"`
}
