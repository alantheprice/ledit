package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/alantheprice/ledit/pkg/llm"
	"github.com/alantheprice/ledit/pkg/prompts"
	ui "github.com/alantheprice/ledit/pkg/ui"
	"github.com/alantheprice/ledit/pkg/utils"
)

// executeContinuationTodo handles workflow continuation by generating the next batch of todos
func executeContinuationTodo(ctx *SimplifiedAgentContext, todo *TodoItem) error {
	ctx.Logger.LogProcessStep("🔄 Processing continuation todo - generating next phase")

	if err := requestContinuationApproval(ctx, todo); err != nil {
		return err
	}

	return generateContinuationTodos(ctx, todo)
}

// requestContinuationApproval asks user for approval if not in skip-prompt mode
func requestContinuationApproval(ctx *SimplifiedAgentContext, todo *TodoItem) error {
	if ctx.SkipPrompt {
		return nil
	}

	ctx.Logger.LogProcessStep("⏳ Requesting user approval for workflow continuation...")
	
	displayContinuationPrompt(ctx, todo)
	
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(strings.TrimSpace(response)) != "y" {
		ctx.Logger.LogProcessStep("❌ User chose not to continue workflow")
		return fmt.Errorf("workflow continuation cancelled by user")
	}

	ctx.Logger.LogProcessStep("✅ User approved workflow continuation")
	ui.Out().Printf("\n🚀 Continuing with next phase...\n\n")
	return nil
}

// displayContinuationPrompt shows the continuation prompt to the user
func displayContinuationPrompt(ctx *SimplifiedAgentContext, todo *TodoItem) {
	ui.Out().Printf("\n🔄 Workflow Continuation Required\n")
	ui.Out().Printf("The current phase is complete. Ready to continue with the next set of tasks?\n\n")
	ui.Out().Printf("Original request: %s\n\n", ctx.UserIntent)

	completedCount := countCompletedTodos(ctx.Todos, todo.Content)
	ui.Out().Printf("✅ Completed %d tasks in this phase\n\n", completedCount)
	ui.Out().Printf("Continue with next phase? (y/N): ")
}

// countCompletedTodos counts completed todos excluding the current one
func countCompletedTodos(todos []TodoItem, currentContent string) int {
	count := 0
	for _, todo := range todos {
		if todo.Status == "completed" && todo.Content != currentContent {
			count++
		}
	}
	return count
}

// generateContinuationTodos generates new todos for the next phase
func generateContinuationTodos(ctx *SimplifiedAgentContext, todo *TodoItem) error {
	completedTasks := extractCompletedTasks(ctx.Todos, todo.Content)
	continuationPrompt := buildContinuationPrompt(ctx, todo, completedTasks)

	response, tokenUsage, err := llm.GetLLMResponseWithTools(
		ctx.Config.OrchestrationModel,
		[]prompts.Message{{Role: "user", Content: continuationPrompt}},
		"You are an expert project manager continuing a complex workflow. Generate the next logical set of todos.",
		ctx.Config,
		60*time.Second,
	)

	if err != nil {
		return fmt.Errorf("failed to generate continuation todos: %w", err)
	}

	trackTokenUsage(ctx, tokenUsage, ctx.Config.OrchestrationModel)

	newTodos, err := parseTodosFromResponseForContinuation(response)
	if err != nil {
		return fmt.Errorf("failed to parse continuation todos: %w", err)
	}

	ctx.Todos = append(ctx.Todos, newTodos...)
	ctx.Logger.LogProcessStep(fmt.Sprintf("✅ Generated %d continuation todos for next phase", len(newTodos)))
	return nil
}

// extractCompletedTasks extracts list of completed task descriptions
func extractCompletedTasks(todos []TodoItem, excludeContent string) []string {
	var completedTasks []string
	for _, todo := range todos {
		if todo.Content != excludeContent {
			completedTasks = append(completedTasks, todo.Content)
		}
	}
	return completedTasks
}

// buildContinuationPrompt builds the prompt for continuation todos
func buildContinuationPrompt(ctx *SimplifiedAgentContext, todo *TodoItem, completedTasks []string) string {
	return fmt.Sprintf(`CONTINUATION WORKFLOW

Original Request: %s

COMPLETED IN PREVIOUS PHASE:
%s

CONTINUATION TODO: %s
Description: %s

Based on the original request and what has been completed, generate the NEXT 10 todos to continue this workflow. Focus on the logical next steps that build upon the completed work.

Consider:
- What components/features are still needed?
- What files/directories need to be created next?  
- What configuration or setup steps are missing?
- What testing or validation needs to happen?

Generate todos that continue naturally from where the previous phase left off.`,
		ctx.UserIntent,
		strings.Join(completedTasks, "\n- "),
		todo.Content,
		todo.Description)
}

// executeParallelTodos executes a set of independent todos concurrently
func executeParallelTodos(ctx *SimplifiedAgentContext, todos []TodoItem) error {
	if !canExecuteInParallel(todos) {
		return fmt.Errorf("todos cannot be executed in parallel")
	}

	workerCount := getOptimalWorkerCount(len(todos), ctx.Config.EditingModel)
	ctx.Logger.LogProcessStep(fmt.Sprintf("🔧 Executing %d todos in parallel using %d workers", len(todos), workerCount))

	return executeWithWorkerPool(ctx, todos, workerCount)
}

// executeWithWorkerPool executes todos using a worker pool pattern
func executeWithWorkerPool(ctx *SimplifiedAgentContext, todos []TodoItem, workerCount int) error {
	todosChan := make(chan TodoItem, len(todos))
	resultsChan := make(chan ParallelTodoResult, len(todos))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			executeParallelWorker(ctx, workerID, todosChan, resultsChan)
		}(i)
	}

	// Send todos to workers
	for _, todo := range todos {
		todosChan <- todo
	}
	close(todosChan)

	// Wait for completion
	wg.Wait()
	close(resultsChan)

	// Collect results
	return collectParallelResults(resultsChan, len(todos))
}

// executeParallelWorker is a worker function for parallel todo execution
func executeParallelWorker(ctx *SimplifiedAgentContext, workerID int, todosChan <-chan TodoItem, resultsChan chan<- ParallelTodoResult) {
	for todo := range todosChan {
		ctx.Logger.LogProcessStep(fmt.Sprintf("⚙️ Worker %d executing: %s", workerID, todo.Content))
		
		err := executeTodo(ctx, &todo)
		
		result := ParallelTodoResult{
			TodoID:   todo.ID,
			Content:  todo.Content,
			Success:  err == nil,
			Error:    err,
			WorkerID: workerID,
		}
		
		resultsChan <- result
	}
}

// collectParallelResults collects and processes results from parallel execution
func collectParallelResults(resultsChan <-chan ParallelTodoResult, expectedCount int) error {
	successCount := 0
	var errors []string

	for result := range resultsChan {
		if result.Success {
			successCount++
		} else if result.Error != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.Content, result.Error))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("parallel execution had %d failures: %s", len(errors), strings.Join(errors, "; "))
	}

	return nil
}

// ParallelTodoResult represents the result of a parallel todo execution
type ParallelTodoResult struct {
	TodoID   string
	Content  string
	Success  bool
	Error    error
	WorkerID int
}

// canExecuteInParallel checks if todos can be safely executed in parallel
func canExecuteInParallel(todos []TodoItem) bool {
	// Simple heuristic: documentation todos can be executed in parallel
	for _, todo := range todos {
		if !isDocumentationTodo(todo) {
			return false
		}
	}
	return len(todos) > 1
}

// isDocumentationTodo checks if a todo is documentation-related
func isDocumentationTodo(todo TodoItem) bool {
	content := strings.ToLower(todo.Content + " " + todo.Description)
	docKeywords := []string{
		"documentation", "readme", "docs", "comment", "docstring", 
		"api doc", "user guide", "tutorial", "example", "changelog",
	}
	
	for _, keyword := range docKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// Note: parseTodosFromResponse is defined in todo_creation.go but with different signature
// This version is for continuation workflows
func parseTodosFromResponseForContinuation(response string) ([]TodoItem, error) {
	clean, err := utils.ExtractJSON(response)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON from response: %w", err)
	}

	var todoStructs []struct {
		Content     string `json:"content"`
		Description string `json:"description"`
		Priority    int    `json:"priority"`
		FilePath    string `json:"file_path"`
	}

	if err := json.Unmarshal([]byte(clean), &todoStructs); err != nil {
		return nil, fmt.Errorf("failed to parse todos JSON: %w", err)
	}

	var todos []TodoItem
	for _, ts := range todoStructs {
		todos = append(todos, TodoItem{
			ID:          generateTodoID(),
			Content:     ts.Content,
			Description: ts.Description,
			Status:      "pending",
			Priority:    ts.Priority,
			FilePath:    strings.TrimSpace(ts.FilePath),
		})
	}

	return todos, nil
}

// getOptimalWorkerCount determines optimal number of workers for parallel execution
func getOptimalWorkerCount(todoCount int, modelName string) int {
	// Conservative approach: limit based on todo count and model capabilities
	maxWorkers := 3 // Conservative limit to avoid overwhelming the LLM API
	
	if todoCount <= 2 {
		return todoCount
	}
	
	// For more todos, use up to maxWorkers
	if todoCount >= maxWorkers {
		return maxWorkers
	}
	
	return todoCount
}

// Helper functions for specialized todo creation

// createDocumentationTodos creates todos specifically for documentation tasks
func createDocumentationTodos(ctx *SimplifiedAgentContext) error {
	ctx.Logger.LogProcessStep("📚 Creating documentation-focused todos")
	
	prompt := buildDocumentationPrompt(ctx)
	return createSpecializedTodos(ctx, prompt, "documentation")
}

// createCreationTodos creates todos for file/component creation
func createCreationTodos(ctx *SimplifiedAgentContext) error {
	ctx.Logger.LogProcessStep("🔨 Creating file/component creation todos")
	
	prompt := buildCreationPrompt(ctx)
	return createSpecializedTodos(ctx, prompt, "creation")
}

// createAnalysisTodos creates todos for analysis and exploration
func createAnalysisTodos(ctx *SimplifiedAgentContext) error {
	ctx.Logger.LogProcessStep("🔍 Creating analysis-focused todos")
	
	prompt := buildAnalysisPromptForSpecialized(ctx)
	return createSpecializedTodos(ctx, prompt, "analysis")
}

// createSpecializedTodos is a helper for creating specialized todo types
func createSpecializedTodos(ctx *SimplifiedAgentContext, prompt, todoType string) error {
	messages := []prompts.Message{
		{Role: "system", Content: fmt.Sprintf("You are an expert at creating %s todos. Return JSON array only.", todoType)},
		{Role: "user", Content: prompt},
	}

	response, tokenUsage, err := llm.GetLLMResponse(
		ctx.Config.OrchestrationModel, 
		messages, 
		"", 
		ctx.Config, 
		GetSmartTimeout(ctx.Config, ctx.Config.OrchestrationModel, "analysis"),
	)
	if err != nil {
		return fmt.Errorf("failed to create %s todos: %w", todoType, err)
	}

	trackTokenUsage(ctx, tokenUsage, ctx.Config.OrchestrationModel)

	newTodos, err := parseTodosFromResponseForContinuation(response)
	if err != nil {
		return fmt.Errorf("failed to parse %s todos: %w", todoType, err)
	}

	ctx.Todos = append(ctx.Todos, newTodos...)
	ctx.Logger.LogProcessStep(fmt.Sprintf("✅ Created %d %s todos", len(newTodos), todoType))
	return nil
}

// buildDocumentationPrompt creates prompt for documentation todos
func buildDocumentationPrompt(ctx *SimplifiedAgentContext) string {
	return fmt.Sprintf(`Create documentation todos for: %s

Focus on:
- README files
- API documentation  
- Code comments
- User guides
- Examples

Return JSON array of todos.`, ctx.UserIntent)
}

// buildCreationPrompt creates prompt for creation todos  
func buildCreationPrompt(ctx *SimplifiedAgentContext) string {
	return fmt.Sprintf(`Create file/component creation todos for: %s

Focus on:
- New files to create
- Directory structure
- Configuration files
- Component scaffolding  
- Initial implementations

Return JSON array of todos.`, ctx.UserIntent)
}

// buildAnalysisPromptForSpecialized creates prompt for analysis todos (specialized version)
func buildAnalysisPromptForSpecialized(ctx *SimplifiedAgentContext) string {
	return fmt.Sprintf(`Create analysis and exploration todos for: %s

Focus on:
- Code review and analysis
- Architecture exploration
- Dependency analysis
- Security review
- Performance analysis

Return JSON array of todos.`, ctx.UserIntent)
}