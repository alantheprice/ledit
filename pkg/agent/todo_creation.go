package agent

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/alantheprice/ledit/pkg/llm"
	"github.com/alantheprice/ledit/pkg/prompts"
	"github.com/alantheprice/ledit/pkg/utils"
	"github.com/alantheprice/ledit/pkg/workspace"
)

// createTodos generates a list of todos based on user intent
func createTodos(ctx *SimplifiedAgentContext) error {
	prompt := buildTodoCreationPrompt(ctx)
	todos, err := generateTodosFromLLM(ctx, prompt)
	if err != nil {
		return err
	}

	convertAndSortTodos(ctx, todos)
	return nil
}

// buildTodoCreationPrompt creates the prompt for todo generation
func buildTodoCreationPrompt(ctx *SimplifiedAgentContext) string {
	promptCache := GetPromptCache()
	userPromptTemplate := promptCache.GetCachedPromptWithFallback(
		"agent_todo_creation_user_optimized.txt",
		getDefaultTodoTemplate(),
	)

	workspaceContext := workspace.GetProgressiveWorkspaceContext(ctx.UserIntent, ctx.Config)
	rolloverContext := buildRolloverContext(ctx)

	prompt := strings.ReplaceAll(userPromptTemplate, "{USER_REQUEST}", ctx.UserIntent)
	prompt = strings.ReplaceAll(prompt, "{WORKSPACE_CONTEXT}", workspaceContext)
	prompt = strings.ReplaceAll(prompt, "{ROLLOVER_CONTEXT}", rolloverContext)

	return prompt
}

// buildRolloverContext builds context from previous analysis
func buildRolloverContext(ctx *SimplifiedAgentContext) string {
	if ctx.ContextManager == nil || ctx.PersistentCtx == nil {
		return ""
	}

	var rolloverContext strings.Builder
	rolloverCtxData := ctx.ContextManager.GetRolloverContext(ctx.PersistentCtx)

	addFindings(&rolloverContext, rolloverCtxData)
	addKnowledge(&rolloverContext, rolloverCtxData)
	addCodePatterns(&rolloverContext, rolloverCtxData)

	return rolloverContext.String()
}

// addFindings adds recent findings to rollover context
func addFindings(builder *strings.Builder, data map[string]interface{}) {
	if recentFindings, ok := data["recent_findings"]; ok {
		if findings, ok := recentFindings.([]AnalysisFinding); ok && len(findings) > 0 {
			builder.WriteString("\n\nRECENT FINDINGS:\n")
			for _, finding := range findings {
				builder.WriteString(fmt.Sprintf("- %s: %s\n", finding.Type, finding.Title))
			}
		}
	}
}

// addKnowledge adds key knowledge to rollover context
func addKnowledge(builder *strings.Builder, data map[string]interface{}) {
	if keyKnowledge, ok := data["key_knowledge"]; ok {
		if knowledge, ok := keyKnowledge.([]KnowledgeItem); ok && len(knowledge) > 0 {
			builder.WriteString("\n\nKNOWLEDGE:\n")
			for _, item := range knowledge {
				builder.WriteString(fmt.Sprintf("- %s: %s\n", item.Category, item.Title))
			}
		}
	}
}

// addCodePatterns adds code patterns to rollover context
func addCodePatterns(builder *strings.Builder, data map[string]interface{}) {
	if codePatterns, ok := data["code_patterns"]; ok {
		if patterns, ok := codePatterns.([]CodePattern); ok && len(patterns) > 0 {
			builder.WriteString("\n\nCODE PATTERNS:\n")
			for _, pattern := range patterns {
				builder.WriteString(fmt.Sprintf("- %s: %s\n", pattern.Type, pattern.Name))
			}
		}
	}
}

// generateTodosFromLLM gets todos from LLM with fallback strategy
func generateTodosFromLLM(ctx *SimplifiedAgentContext, prompt string) ([]todoStruct, error) {
	promptCache := GetPromptCache()
	systemPrompt := promptCache.GetCachedPromptWithFallback(
		"agent_todo_creation_system_optimized.txt",
		"Create specific, actionable development todos. Ground todos in workspace context using tools (read_file, grep_search, run_shell_command) to validate assumptions. Include analysis todo if uncertain about file locations or details. Always return valid JSON.",
	)

	messages := []prompts.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	}

	// Try primary approach
	response, err := tryPrimaryTodoGeneration(ctx, messages)
	if err != nil {
		// Fallback approach
		response, err = tryFallbackTodoGeneration(ctx)
		if err != nil {
			return nil, fmt.Errorf("both primary and fallback attempts failed: %w", err)
		}
	}

	return parseTodosFromResponse(ctx, response)
}

// tryPrimaryTodoGeneration attempts todo generation with primary model
func tryPrimaryTodoGeneration(ctx *SimplifiedAgentContext, messages []prompts.Message) (string, error) {
	smartTimeout := GetSmartTimeout(ctx.Config, ctx.Config.OrchestrationModel, "analysis")
	response, tokenUsage, err := llm.GetLLMResponse(ctx.Config.OrchestrationModel, messages, "", ctx.Config, smartTimeout)
	
	if err != nil {
		return "", err
	}

	trackTokenUsage(ctx, tokenUsage, ctx.Config.OrchestrationModel)
	return response, nil
}

// tryFallbackTodoGeneration attempts simplified todo generation
func tryFallbackTodoGeneration(ctx *SimplifiedAgentContext) (string, error) {
	ctx.Logger.LogProcessStep("⚠️ Primary model failed, trying fallback approach")

	fallbackMessages := []prompts.Message{
		{Role: "system", Content: "You create development todos. Keep it simple and return JSON only."},
		{Role: "user", Content: fmt.Sprintf("Create 1-2 simple todos for: %s\nReturn JSON array only.", ctx.UserIntent)},
	}

	smartTimeout := GetSmartTimeout(ctx.Config, ctx.Config.OrchestrationModel, "analysis")
	fallbackTimeout := time.Duration(float64(smartTimeout) * 1.5)
	response, tokenUsage, err := llm.GetLLMResponse(ctx.Config.OrchestrationModel, fallbackMessages, "", ctx.Config, fallbackTimeout)

	if err != nil {
		ctx.Logger.LogProcessStep(fmt.Sprintf("⚠️ Fallback attempt failed: %v", err))
		return "", err
	}

	ctx.Logger.LogProcessStep("✅ Fallback approach succeeded")
	trackTokenUsage(ctx, tokenUsage, ctx.Config.OrchestrationModel)
	return response, nil
}

// todoStruct represents the JSON structure for todos from LLM
type todoStruct struct {
	Content     string `json:"content"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	FilePath    string `json:"file_path"`
}

// parseTodosFromResponse extracts todos from LLM response
func parseTodosFromResponse(ctx *SimplifiedAgentContext, response string) ([]todoStruct, error) {
	clean, err := utils.ExtractJSON(response)
	if err != nil {
		clean, err = extractJSONFromEnd(response)
		if err != nil {
			ctx.Logger.LogProcessStep(fmt.Sprintf("⚠️ JSON extraction failed. LLM Response: %s", response))
			return nil, fmt.Errorf("failed to extract JSON from response: %w", err)
		}
		ctx.Logger.LogProcessStep("✅ Successfully extracted JSON from end of response")
	}

	var todos []todoStruct
	if err := json.Unmarshal([]byte(clean), &todos); err != nil {
		ctx.Logger.LogProcessStep(fmt.Sprintf("⚠️ JSON parsing failed, creating fallback todo: %v", err))
		todos = createFallbackTodos(ctx.UserIntent)
		ctx.Logger.LogProcessStep("✅ Created fallback todo for analysis")
	}

	return todos, nil
}

// extractJSONFromEnd attempts to extract JSON from end of response
func extractJSONFromEnd(response string) (string, error) {
	if lastBracket := strings.LastIndex(response, "["); lastBracket != -1 {
		potentialJSON := response[lastBracket:]
		if json.Valid([]byte(potentialJSON)) {
			return potentialJSON, nil
		}
	}
	return "", fmt.Errorf("no valid JSON found at end of response")
}

// createFallbackTodos creates simple fallback todos when parsing fails
func createFallbackTodos(userIntent string) []todoStruct {
	return []todoStruct{
		{
			Content:     "Analyze user request: " + userIntent,
			Description: "Analyze and understand what the user is asking for: " + userIntent,
			Priority:    1,
			FilePath:    "",
		},
	}
}

// convertAndSortTodos converts todo structs to TodoItems and sorts them
func convertAndSortTodos(ctx *SimplifiedAgentContext, todos []todoStruct) {
	for _, todo := range todos {
		ctx.Todos = append(ctx.Todos, TodoItem{
			ID:          generateTodoID(),
			Content:     todo.Content,
			Description: todo.Description,
			Status:      "pending",
			Priority:    todo.Priority,
			FilePath:    strings.TrimSpace(todo.FilePath),
		})
	}

	// Sort by priority
	sort.Slice(ctx.Todos, func(i, j int) bool {
		return ctx.Todos[i].Priority < ctx.Todos[j].Priority
	})
}

// generateTodoID creates a unique ID for a todo
func generateTodoID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// getDefaultTodoTemplate returns the default template for todo creation
func getDefaultTodoTemplate() string {
	return `Expert developer: break down request into actionable todos using workspace context.

Request: "{USER_REQUEST}"

## Workspace
{WORKSPACE_CONTEXT}

{ROLLOVER_CONTEXT}

GUIDELINES:
- Max 10 todos; use continuation todo #10 for complex multi-phase work
- Monorepo: focus on ONE component at a time
- Use tools to validate: read_file, grep_search, shell commands
- Include analysis todo if locations/details uncertain
- Ground in actual files, avoid speculation

JSON format:
[{"content":"Brief task","description":"Details","priority":1,"file_path":"optional/path.ext"}]

Return ONLY JSON array.`
}