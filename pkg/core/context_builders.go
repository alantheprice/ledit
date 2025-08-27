package core

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alantheprice/ledit/pkg/workspace"
)

// WorkspaceContextBuilder builds workspace context for LLM requests
type WorkspaceContextBuilder struct {
	maxTokens      int
	includeHidden  bool
	priorityTypes  []string
}

func NewWorkspaceContextBuilder() *WorkspaceContextBuilder {
	return &WorkspaceContextBuilder{
		maxTokens:     10000, // Default token budget for workspace context
		includeHidden: false,
		priorityTypes: []string{".go", ".py", ".js", ".ts", ".java", ".cpp", ".c", ".rs"},
	}
}

func (wb *WorkspaceContextBuilder) GetContextID() string {
	return "workspace_context_v1"
}

func (wb *WorkspaceContextBuilder) BuildContext(input *ContextInput) (*LLMContext, error) {
	// Get workspace information
	workspaceInfo := wb.getWorkspaceOverview(input.WorkspacePath)
	
	// Get relevant files based on user intent and target files
	relevantFiles, err := wb.getRelevantFiles(input)
	if err != nil {
		return nil, fmt.Errorf("failed to get relevant files: %w", err)
	}

	// Build project structure summary
	projectStructure := wb.buildProjectStructure(input.WorkspacePath)

	// Calculate total token count
	tokenCount := wb.calculateTokenCount(workspaceInfo, relevantFiles, projectStructure)

	return &LLMContext{
		WorkspaceInfo:    workspaceInfo,
		RelevantFiles:    relevantFiles,
		Dependencies:     wb.extractDependencies(relevantFiles),
		ProjectStructure: projectStructure,
		TokenCount:       tokenCount,
	}, nil
}

func (wb *WorkspaceContextBuilder) GetTokenBudget() TokenBudget {
	return TokenBudget{
		Total:      wb.maxTokens,
		System:     1000,
		Context:    wb.maxTokens - 3000, // Reserve space for user prompt and system
		UserPrompt: 1500,
		Reserved:   500,
	}
}

func (wb *WorkspaceContextBuilder) ShouldInclude(content string, metadata map[string]interface{}) bool {
	// Check file extension priority
	if ext, ok := metadata["extension"].(string); ok {
		for _, priority := range wb.priorityTypes {
			if ext == priority {
				return true
			}
		}
	}

	// Check relevance score
	if relevance, ok := metadata["relevance"].(float64); ok {
		return relevance > 0.3 // Include files with relevance > 30%
	}

	// Check file size (exclude very large files)
	if size, ok := metadata["size"].(int); ok {
		return size < 50000 // Exclude files larger than 50KB
	}

	return false
}

func (wb *WorkspaceContextBuilder) getWorkspaceOverview(workspacePath string) string {
	// Get workspace context using existing workspace package
	return workspace.GetWorkspaceContext("", nil)
}

func (wb *WorkspaceContextBuilder) getRelevantFiles(input *ContextInput) ([]FileContext, error) {
	var files []FileContext
	
	// If target files are specified, prioritize them
	if len(input.TargetFiles) > 0 {
		for _, filePath := range input.TargetFiles {
			fileCtx, err := wb.buildFileContext(filePath, 1.0)
			if err != nil {
				continue // Skip files that can't be read
			}
			files = append(files, fileCtx)
		}
	} else {
		// Auto-discover relevant files based on user intent
		discoveredFiles := wb.discoverRelevantFiles(input.UserIntent, input.WorkspacePath)
		for _, filePath := range discoveredFiles {
			relevance := wb.calculateRelevance(filePath, input.UserIntent)
			if relevance > 0.2 { // Only include files with 20%+ relevance
				fileCtx, err := wb.buildFileContext(filePath, relevance)
				if err != nil {
					continue
				}
				files = append(files, fileCtx)
			}
		}
	}

	// Sort by relevance (highest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Relevance > files[j].Relevance
	})

	// Limit by token budget
	return wb.limitFilesByTokens(files), nil
}

func (wb *WorkspaceContextBuilder) buildFileContext(filePath string, relevance float64) (FileContext, error) {
	// This would read the file content and create context
	// For now, return a placeholder
	return FileContext{
		Path:       filePath,
		Content:    "",        // Would be populated by reading the file
		Language:   wb.detectLanguage(filePath),
		Summary:    "",        // Would be generated or cached
		TokenCount: 0,         // Would be calculated
		Relevance:  relevance,
	}, nil
}

func (wb *WorkspaceContextBuilder) detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".java":
		return "java"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".c":
		return "c"
	case ".rs":
		return "rust"
	default:
		return "text"
	}
}

func (wb *WorkspaceContextBuilder) discoverRelevantFiles(userIntent, workspacePath string) []string {
	// This would use smart file discovery based on user intent
	// For now, return a basic implementation
	return []string{} // Would be implemented with actual file discovery
}

func (wb *WorkspaceContextBuilder) calculateRelevance(filePath, userIntent string) float64 {
	// This would calculate file relevance based on content and intent
	// For now, return a basic relevance score
	return 0.5 // Would be implemented with actual relevance calculation
}

func (wb *WorkspaceContextBuilder) limitFilesByTokens(files []FileContext) []FileContext {
	budget := wb.GetTokenBudget()
	availableTokens := budget.Context
	
	var result []FileContext
	usedTokens := 0
	
	for _, file := range files {
		if usedTokens+file.TokenCount <= availableTokens {
			result = append(result, file)
			usedTokens += file.TokenCount
		} else {
			break
		}
	}
	
	return result
}

func (wb *WorkspaceContextBuilder) buildProjectStructure(workspacePath string) string {
	// This would build a project structure summary
	return "Project structure would be generated here"
}

func (wb *WorkspaceContextBuilder) extractDependencies(files []FileContext) []string {
	var deps []string
	for _, file := range files {
		// Extract dependencies from file content
		// This would be implemented with actual dependency extraction
		deps = append(deps, fmt.Sprintf("dependencies for %s", file.Path))
	}
	return deps
}

func (wb *WorkspaceContextBuilder) calculateTokenCount(workspaceInfo string, files []FileContext, projectStructure string) int {
	count := float64(len(strings.Fields(workspaceInfo))) * 1.3 // Rough token estimation
	count += float64(len(strings.Fields(projectStructure))) * 1.3
	
	for _, file := range files {
		count += float64(file.TokenCount)
	}
	
	return int(count)
}

// FileOnlyContextBuilder builds context for single-file operations
type FileOnlyContextBuilder struct {
	maxTokens int
}

func NewFileOnlyContextBuilder() *FileOnlyContextBuilder {
	return &FileOnlyContextBuilder{
		maxTokens: 4000,
	}
}

func (fb *FileOnlyContextBuilder) GetContextID() string {
	return "file_only_context_v1"
}

func (fb *FileOnlyContextBuilder) BuildContext(input *ContextInput) (*LLMContext, error) {
	var relevantFiles []FileContext
	
	// Focus only on target files
	for _, filePath := range input.TargetFiles {
		fileCtx, err := fb.buildFileContext(filePath)
		if err != nil {
			continue
		}
		relevantFiles = append(relevantFiles, fileCtx)
	}
	
	return &LLMContext{
		WorkspaceInfo:    "", // No workspace context for file-only operations
		RelevantFiles:    relevantFiles,
		Dependencies:     []string{},
		ProjectStructure: "",
		TokenCount:       fb.calculateTokenCount(relevantFiles),
	}, nil
}

func (fb *FileOnlyContextBuilder) GetTokenBudget() TokenBudget {
	return TokenBudget{
		Total:      fb.maxTokens,
		System:     500,
		Context:    fb.maxTokens - 1500,
		UserPrompt: 800,
		Reserved:   200,
	}
}

func (fb *FileOnlyContextBuilder) ShouldInclude(content string, metadata map[string]interface{}) bool {
	// For file-only context, include everything up to token limit
	return true
}

func (fb *FileOnlyContextBuilder) buildFileContext(filePath string) (FileContext, error) {
	// This would read and process the specific file
	return FileContext{
		Path:       filePath,
		Content:    "", // Would be populated
		Language:   fb.detectLanguage(filePath),
		Summary:    "",
		TokenCount: 0,  // Would be calculated
		Relevance:  1.0, // Always high relevance for target files
	}, nil
}

func (fb *FileOnlyContextBuilder) detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	default:
		return "text"
	}
}

func (fb *FileOnlyContextBuilder) calculateTokenCount(files []FileContext) int {
	count := 0
	for _, file := range files {
		count += file.TokenCount
	}
	return count
}