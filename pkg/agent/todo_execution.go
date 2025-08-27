package agent

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/alantheprice/ledit/pkg/llm"
	"github.com/alantheprice/ledit/pkg/prompts"
	ui "github.com/alantheprice/ledit/pkg/ui"
	"github.com/alantheprice/ledit/pkg/workspace"
)

// executeTodoWithSmartRetry executes a todo with context-aware retry logic
func executeTodoWithSmartRetry(ctx *SimplifiedAgentContext, todo *TodoItem) error {
	const maxRetries = 2

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := executeTodo(ctx, todo)
		if err == nil {
			return nil
		}

		if shouldSwitchToShellCommand(err, attempt, todo) {
			ctx.Logger.LogProcessStep("🔄 Switching to shell command approach for filesystem task")
			return executeShellCommandTodo(ctx, todo)
		}

		if attempt < maxRetries {
			ctx.Logger.LogProcessStep(fmt.Sprintf("🔄 Retry %d/%d: %v", attempt+1, maxRetries, err))
			time.Sleep(time.Second * 2)
		}
	}

	return fmt.Errorf("failed after %d retries", maxRetries+1)
}

// shouldSwitchToShellCommand determines if we should switch execution strategy
func shouldSwitchToShellCommand(err error, attempt int, todo *TodoItem) bool {
	return strings.Contains(err.Error(), "code review requires revisions") &&
		attempt == 0 &&
		containsFilesystemKeywords(todo.Content)
}

// executeTodo executes a todo using the optimized editing service
func executeTodo(ctx *SimplifiedAgentContext, todo *TodoItem) error {
	ctx.Logger.LogProcessStep(fmt.Sprintf("🔧 Executing: %s", todo.Content))

	executionType := analyzeTodoExecutionType(todo.Content, todo.Description)
	
	switch executionType {
	case ExecutionTypeAnalysis:
		if err := executeAnalysisTodo(ctx, todo); err != nil {
			return err
		}
		refineTodosWithAnalysis(ctx, todo)
		return nil
	case ExecutionTypeDirectEdit:
		return executeDirectEditTodo(ctx, todo)
	case ExecutionTypeShellCommand:
		return executeShellCommandTodo(ctx, todo)
	case ExecutionTypeCodeCommand:
		return executeOptimizedCodeEditingTodo(ctx, todo)
	case ExecutionTypeContinuation:
		return executeContinuationTodo(ctx, todo)
	default:
		return executeOptimizedCodeEditingTodo(ctx, todo)
	}
}

// analyzeTodoExecutionType determines the best way to execute a todo
func analyzeTodoExecutionType(content, description string) ExecutionType {
	contentLower := strings.ToLower(content)
	descriptionLower := strings.ToLower(description)

	if isContinuationTodo(contentLower, descriptionLower) {
		return ExecutionTypeContinuation
	}
	if isDirectEditTodo(contentLower, descriptionLower) {
		return ExecutionTypeDirectEdit
	}
	if isShellCommandTodo(contentLower, descriptionLower) {
		return ExecutionTypeShellCommand
	}
	if isAnalysisTodo(contentLower) {
		return ExecutionTypeAnalysis
	}

	return ExecutionTypeCodeCommand
}

// isContinuationTodo checks if this is a continuation workflow todo
func isContinuationTodo(contentLower, descriptionLower string) bool {
	continuationKeywords := []string{
		"continue with next phase", "continue with", "next phase of", 
		"continue to", "proceed with next",
	}
	return containsAnyKeyword(contentLower, descriptionLower, continuationKeywords)
}

// isDirectEditTodo checks if this is a simple direct edit
func isDirectEditTodo(contentLower, descriptionLower string) bool {
	// First check simple direct edit keywords
	directEditKeywords := []string{
		"update readme", "update documentation", "add comment", "fix typo", 
		"update description", "add example", "update text",
	}
	if containsAnyKeyword(contentLower, descriptionLower, directEditKeywords) {
		return true
	}

	// Check file creation patterns
	fileCreationPatterns := []string{
		"generate.*\\.md", "create.*\\.md", "write.*\\.md",
		"generate.*\\.txt", "create.*\\.txt", "write.*\\.txt",
		"generate.*\\.json", "create.*\\.json", "write.*\\.json",
		"generate.*\\.yaml", "create.*\\.yaml", "write.*\\.yaml",
		"generate.*\\.yml", "create.*\\.yml", "write.*\\.yml",
		"generate.*documentation", "create.*documentation", "write.*documentation",
		"generate.*api.*doc", "create.*api.*doc", "write.*api.*doc",
	}
	if matchesAnyPattern(contentLower, descriptionLower, fileCreationPatterns) {
		return true
	}

	// Check for documentation updates
	return strings.Contains(contentLower, "update") && 
		(strings.Contains(contentLower, "readme") || 
		 strings.Contains(contentLower, "documentation") || 
		 strings.Contains(contentLower, "docs"))
}

// isShellCommandTodo checks if this needs shell command execution
func isShellCommandTodo(contentLower, descriptionLower string) bool {
	shellKeywords := []string{
		"create directory", "mkdir", "create folder", "setup project", "initialize",
		"install", "setup monorepo", "create backend", "create frontend", "run", "execute command",
		"create the", "directory for", "backend directory", "frontend directory",
		"directory in", "directory called", "directory named", " directory ", "new directory",
	}
	return containsAnyKeyword(contentLower, descriptionLower, shellKeywords)
}

// isAnalysisTodo checks if this is analysis-only
func isAnalysisTodo(contentLower string) bool {
	analysisKeywords := []string{
		"analyze", "examine", "explore", "read", "review", "understand", "study", 
		"investigate", "check", "verify", "validate", "list", "show", "display", 
		"find", "search", "discover", "identify",
	}
	for _, keyword := range analysisKeywords {
		if strings.Contains(contentLower, keyword) {
			return true
		}
	}
	return false
}

// containsAnyKeyword checks if content or description contains any of the keywords
func containsAnyKeyword(content, description string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(content, keyword) || strings.Contains(description, keyword) {
			return true
		}
	}
	return false
}

// matchesAnyPattern checks if content or description matches any regex pattern
func matchesAnyPattern(content, description string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			return true
		}
		if matched, _ := regexp.MatchString(pattern, description); matched {
			return true
		}
	}
	return false
}

// executeAnalysisTodo handles analysis-only todos with direct LLM exploration
func executeAnalysisTodo(ctx *SimplifiedAgentContext, todo *TodoItem) error {
	ctx.Logger.LogProcessStep("🔍 Performing analysis (no code changes)")

	workspaceContext := getWorkspaceContext(ctx)
	prompt := buildAnalysisPrompt(ctx, todo, workspaceContext)

	messages := []prompts.Message{
		{Role: "system", Content: llm.GetSystemMessageForInformational()},
		{Role: "user", Content: prompt},
	}

	response, tokenUsage, err := executeAgentWorkflowWithTools(ctx, messages, "analysis")
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	trackAnalysisResults(ctx, todo, response, tokenUsage)
	ctx.Logger.LogProcessStep("📊 Analysis completed and stored")
	ui.Out().Print(fmt.Sprintf("\n📋 Analysis Result for Todo: %s\n%s\n", todo.Content, response))

	return nil
}

// getWorkspaceContext gets workspace context with fallback
func getWorkspaceContext(ctx *SimplifiedAgentContext) string {
	workspaceContext := workspace.GetProgressiveWorkspaceContext(ctx.UserIntent, ctx.Config)
	if workspaceContext == "" {
		workspaceContext = "Workspace context not available"
	}
	return workspaceContext
}

// buildAnalysisPrompt builds the analysis prompt
func buildAnalysisPrompt(ctx *SimplifiedAgentContext, todo *TodoItem, workspaceContext string) string {
	return fmt.Sprintf(`Task: %s

Use available tools to complete this analysis task effectively.`, todo.Description)
}

// trackAnalysisResults stores analysis results and findings
func trackAnalysisResults(ctx *SimplifiedAgentContext, todo *TodoItem, response string, tokenUsage *llm.TokenUsage) {
	model := ctx.Config.OrchestrationModel
	if model == "" {
		model = ctx.Config.EditingModel
	}
	if tokenUsage != nil {
		trackTokenUsage(ctx, tokenUsage, model)
	}

	ctx.AnalysisResults[todo.ID] = response

	if ctx.ContextManager != nil && ctx.PersistentCtx != nil {
		findings := extractFindingsFromAnalysis(response, todo)
		for _, finding := range findings {
			err := ctx.ContextManager.AddFinding(ctx.PersistentCtx, finding)
			if err != nil {
				ctx.Logger.LogError(fmt.Errorf("failed to store finding in context: %w", err))
			} else {
				ctx.Logger.LogProcessStep(fmt.Sprintf("💡 Finding stored: %s", finding.Title))
			}
		}
	}
}

// containsFilesystemKeywords checks if content contains filesystem operation keywords
func containsFilesystemKeywords(content string) bool {
	contentLower := strings.ToLower(content)
	keywords := []string{
		"create directory", "mkdir", "create folder", "directory", 
		"filesystem", "file system", "setup project",
	}
	for _, keyword := range keywords {
		if strings.Contains(contentLower, keyword) {
			return true
		}
	}
	return false
}