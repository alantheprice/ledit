package core

import (
	"context"
	"fmt"

	"github.com/alantheprice/ledit/pkg/config"
)

// CoreService provides high-level interfaces for code generation and agent operations
type CoreService struct {
	engine ExecutionEngine
	config *config.Config
}

// NewCoreService creates a new core service instance
func NewCoreService(cfg *config.Config) *CoreService {
	return &CoreService{
		engine: NewOptimizedEngine(),
		config: cfg,
	}
}

// GenerateCode generates code based on instructions using the modular architecture
func (s *CoreService) GenerateCode(ctx context.Context, instructions string, filename string) (*CodeGenerationResult, error) {
	// Create execution request
	request := CreateCodeGenerationRequest(instructions, filename, s.config)
	request.Config = s.config

	// Execute with optimization
	result, err := s.engine.ExecuteWithOptimization(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("code generation failed: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("code generation was not successful: %v", result.Errors)
	}

	// Convert to legacy format for compatibility
	codeResult := &CodeGenerationResult{
		Files:      s.convertFileResults(result.Output.Files),
		Success:    result.Success,
		TokenUsage: result.TokenUsage,
		Model:      result.ModelUsed,
		Duration:   result.Duration,
		Optimized:  result.OptimizedUsed,
	}

	return codeResult, nil
}

// AnalyzeAgentRequest analyzes a user request and creates todos using the modular architecture
func (s *CoreService) AnalyzeAgentRequest(ctx context.Context, userIntent string) (*AgentAnalysisResult, error) {
	// Create execution request
	request := CreateAgentAnalysisRequest(userIntent, s.config)
	request.Config = s.config

	// Execute with optimization
	result, err := s.engine.ExecuteWithOptimization(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("agent analysis failed: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("agent analysis was not successful: %v", result.Errors)
	}

	// Extract todos from the result
	todos := s.extractTodos(result.Output)

	analysisResult := &AgentAnalysisResult{
		Todos:      todos,
		Success:    result.Success,
		TokenUsage: result.TokenUsage,
		Model:      result.ModelUsed,
		Duration:   result.Duration,
		Optimized:  result.OptimizedUsed,
	}

	return analysisResult, nil
}

// ExecuteAgentTask executes a specific agent task using the modular architecture
func (s *CoreService) ExecuteAgentTask(ctx context.Context, taskDescription string) (*AgentExecutionResult, error) {
	// Create execution request
	request := CreateAgentExecutionRequest(taskDescription, s.config)
	request.Config = s.config

	// Execute with optimization
	result, err := s.engine.ExecuteWithOptimization(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	executionResult := &AgentExecutionResult{
		Actions:    result.Output.Actions,
		Files:      s.convertFileResults(result.Output.Files),
		Success:    result.Success,
		TokenUsage: result.TokenUsage,
		Model:      result.ModelUsed,
		Duration:   result.Duration,
		Optimized:  result.OptimizedUsed,
		Errors:     s.convertErrors(result.Errors),
	}

	return executionResult, nil
}

// GetOptimizedPrompt gets an optimized prompt for a specific task type and model
func (s *CoreService) GetOptimizedPrompt(taskType TaskType, model string) (*Prompt, error) {
	var builder PromptBuilder

	switch taskType {
	case TaskTypeCodeGeneration:
		builder = NewCodeGenerationPromptBuilder()
	case TaskTypeAgentAnalysis:
		builder = NewAgentAnalysisPromptBuilder()
	case TaskTypeAgentExecution:
		builder = NewAgentExecutionPromptBuilder()
	default:
		return nil, fmt.Errorf("unsupported task type: %s", taskType)
	}

	// Build prompt with minimal context for optimization purposes
	promptCtx := &PromptContext{
		TaskType:     taskType,
		UserInput:    "sample input",
		Instructions: "sample instructions",
		Metadata:     make(map[string]interface{}),
		Config:       s.config,
	}

	return builder.BuildPrompt(promptCtx)
}

// GetAvailableTaskTypes returns all available task types
func (s *CoreService) GetAvailableTaskTypes() []TaskType {
	return []TaskType{
		TaskTypeCodeGeneration,
		TaskTypeCodeReview,
		TaskTypeAgentAnalysis,
		TaskTypeAgentExecution,
		TaskTypeQuestion,
		TaskTypeRefactoring,
		TaskTypeTesting,
	}
}

// GetModelPreferences returns model preferences for a task type
func (s *CoreService) GetModelPreferences(taskType TaskType) (ModelPreferences, error) {
	var builder PromptBuilder

	switch taskType {
	case TaskTypeCodeGeneration:
		builder = NewCodeGenerationPromptBuilder()
	case TaskTypeAgentAnalysis:
		builder = NewAgentAnalysisPromptBuilder()
	case TaskTypeAgentExecution:
		builder = NewAgentExecutionPromptBuilder()
	default:
		return ModelPreferences{}, fmt.Errorf("unsupported task type: %s", taskType)
	}

	return builder.GetModelPreferences(), nil
}

// Helper methods for converting between formats

func (s *CoreService) convertFileResults(files []FileResult) []string {
	var result []string
	for _, file := range files {
		result = append(result, file.Path)
	}
	return result
}

func (s *CoreService) extractTodos(output *ProcessedOutput) []TodoItem {
	var todos []TodoItem
	
	if dataMap, ok := output.Data.(map[string]interface{}); ok {
		if todosData, ok := dataMap["todos"]; ok {
		if todoStructs, ok := todosData.([]TodoStruct); ok {
			for i, ts := range todoStructs {
				todo := TodoItem{
					ID:          fmt.Sprintf("todo_%d", i+1),
					Content:     ts.Content,
					Description: ts.Description,
					Priority:    ts.Priority,
					FilePath:    ts.FilePath,
					Status:      "pending",
				}
				todos = append(todos, todo)
			}
		}
		}
	}
	
	return todos
}

func (s *CoreService) convertErrors(errors []error) []string {
	var result []string
	for _, err := range errors {
		result = append(result, err.Error())
	}
	return result
}

// Result types for compatibility with existing code

type CodeGenerationResult struct {
	Files      []string
	Success    bool
	TokenUsage interface{}
	Model      string
	Duration   interface{}
	Optimized  bool
}

type AgentAnalysisResult struct {
	Todos      []TodoItem
	Success    bool
	TokenUsage interface{}
	Model      string
	Duration   interface{}
	Optimized  bool
}

type AgentExecutionResult struct {
	Actions    []Action
	Files      []string
	Success    bool
	TokenUsage interface{}
	Model      string
	Duration   interface{}
	Optimized  bool
	Errors     []string
}

type TodoItem struct {
	ID          string
	Content     string
	Description string
	Priority    int
	FilePath    string
	Status      string
}

// Integration helper functions

// IntegrateWithExistingCodeGeneration shows how to integrate with existing code generation
func (s *CoreService) IntegrateWithExistingCodeGeneration(instructions string, filename string) (string, error) {
	ctx := context.Background()
	
	result, err := s.GenerateCode(ctx, instructions, filename)
	if err != nil {
		return "", err
	}

	// For compatibility, return the first generated file content or a diff
	// This would need to be adapted based on the existing interface
	if len(result.Files) > 0 {
		return result.Files[0], nil // Return first file path
	}

	return "", fmt.Errorf("no files generated")
}

// IntegrateWithExistingAgent shows how to integrate with existing agent workflow
func (s *CoreService) IntegrateWithExistingAgent(userIntent string) ([]TodoItem, error) {
	ctx := context.Background()
	
	result, err := s.AnalyzeAgentRequest(ctx, userIntent)
	if err != nil {
		return nil, err
	}

	return result.Todos, nil
}