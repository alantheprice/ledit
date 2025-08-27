package core

import (
	"context"
	"fmt"
	"time"

	"github.com/alantheprice/ledit/pkg/llm"
	"github.com/alantheprice/ledit/pkg/prompts"
	"github.com/alantheprice/ledit/pkg/types"
)

// DefaultExecutionEngine implements the ExecutionEngine interface
type DefaultExecutionEngine struct {
	optimizerEnabled bool
}

func NewExecutionEngine() *DefaultExecutionEngine {
	return &DefaultExecutionEngine{
		optimizerEnabled: true,
	}
}

// NewOptimizedEngine creates an execution engine with optimization enabled
func NewOptimizedEngine() ExecutionEngine {
	return NewOptimizedExecutionEngine()
}

func (e *DefaultExecutionEngine) Execute(ctx context.Context, request *ExecutionRequest) (*ExecutionResult, error) {
	startTime := time.Now()

	// Build the prompt
	promptCtx := &PromptContext{
		TaskType:     request.TaskType,
		UserInput:    fmt.Sprintf("%v", request.Input), // Convert input to string
		Filename:     "",
		FileContent:  "",
		Instructions: fmt.Sprintf("%v", request.Input),
		Metadata:     make(map[string]interface{}),
		Config:       request.Config,
	}

	prompt, err := request.PromptBuilder.BuildPrompt(promptCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Build the context if context builder is provided
	var llmContext *LLMContext
	if request.ContextBuilder != nil {
		contextInput := &ContextInput{
			UserIntent:    fmt.Sprintf("%v", request.Input),
			WorkspacePath: ".", // Current directory
			TargetFiles:   []string{},
			Config:        request.Config,
		}

		llmContext, err = request.ContextBuilder.BuildContext(contextInput)
		if err != nil {
			return nil, fmt.Errorf("failed to build context: %w", err)
		}

		// Integrate context into prompt
		if llmContext.WorkspaceInfo != "" {
			// Prepend workspace context to user prompt
			prompt.UserPrompt = llmContext.WorkspaceInfo + "\n\n" + prompt.UserPrompt
		}
	}

	// Select the best model
	model := e.selectBestModel(prompt.ModelHints, request.Config)

	// Execute the LLM request
	rawOutput, err := e.executeLLMRequest(ctx, prompt, model, request)
	if err != nil {
		return &ExecutionResult{
			Success:       false,
			Duration:      time.Since(startTime),
			ModelUsed:     model,
			OptimizedUsed: false,
			Errors:        []error{err},
		}, err
	}

	// Process the output
	processedOutput, err := request.OutputProcessor.ProcessOutput(rawOutput)
	if err != nil {
		return &ExecutionResult{
			Success:       false,
			Duration:      time.Since(startTime),
			ModelUsed:     model,
			OptimizedUsed: false,
			TokenUsage:    rawOutput.TokenUsage,
			Errors:        []error{err},
		}, err
	}

	// Validate the output if requested
	if request.Options.ValidateOutput {
		if err := request.OutputProcessor.ValidateOutput(processedOutput); err != nil {
			processedOutput.Errors = append(processedOutput.Errors, err.Error())
			processedOutput.Success = false
		}
	}

	return &ExecutionResult{
		Success:       processedOutput.Success,
		Output:        processedOutput,
		TokenUsage:    rawOutput.TokenUsage,
		Duration:      time.Since(startTime),
		ModelUsed:     model,
		OptimizedUsed: false, // Would be true if optimization was used
		Errors:        []error{},
	}, nil
}

func (e *DefaultExecutionEngine) ExecuteWithOptimization(ctx context.Context, request *ExecutionRequest) (*ExecutionResult, error) {
	// Use the optimized execution engine
	optimizedEngine := NewOptimizedExecutionEngine()
	return optimizedEngine.ExecuteWithOptimization(ctx, request)
}

func (e *DefaultExecutionEngine) selectBestModel(preferences ModelPreferences, config interface{}) string {
	// Try preferred models first
	for _, model := range preferences.Preferred {
		if e.isModelAvailable(model) {
			return model
		}
	}

	// Fall back to fallback models
	for _, model := range preferences.Fallbacks {
		if e.isModelAvailable(model) {
			return model
		}
	}

	// Use default model from config
	// This is a simplified implementation
	return "openai:gpt-4o" // Default fallback
}

func (e *DefaultExecutionEngine) isModelAvailable(model string) bool {
	// This would check if the model is available/configured
	// For now, assume all models are available
	return true
}

func (e *DefaultExecutionEngine) executeLLMRequest(ctx context.Context, prompt *Prompt, model string, request *ExecutionRequest) (*RawOutput, error) {
	// Set up timeout
	timeout := request.Options.Timeout
	if timeout == 0 {
		timeout = 2 * time.Minute // Default timeout
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute the LLM request using existing LLM infrastructure
	response, tokenUsage, err := e.callLLM(timeoutCtx, prompt, model, request)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	return &RawOutput{
		Content:    response,
		Model:      model,
		TokenUsage: tokenUsage,
		Metadata:   make(map[string]interface{}),
		Timestamp:  time.Now(),
	}, nil
}

func (e *DefaultExecutionEngine) callLLM(ctx context.Context, prompt *Prompt, model string, request *ExecutionRequest) (string, *types.TokenUsage, error) {
	// Convert our prompt messages to the format expected by the LLM API
	var messages []prompts.Message
	for _, msg := range prompt.Messages {
		messages = append(messages, prompts.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Use the existing LLM infrastructure
	response, tokenUsage, err := llm.GetLLMResponse(
		model,
		messages,
		"", // filename - not applicable for core execution
		request.Config, // Already expected to be *config.Config
		request.Options.Timeout,
	)

	if err != nil {
		return "", nil, fmt.Errorf("LLM API call failed: %w", err)
	}

	return response, tokenUsage, nil
}

// Factory functions for creating common configurations

// CreateCodeGenerationRequest creates a request for code generation
func CreateCodeGenerationRequest(instructions string, filename string, config interface{}) *ExecutionRequest {
	return &ExecutionRequest{
		TaskType:        TaskTypeCodeGeneration,
		PromptBuilder:   NewCodeGenerationPromptBuilder(),
		ContextBuilder:  NewWorkspaceContextBuilder(),
		OutputProcessor: NewCodeGenerationOutputProcessor(),
		Input:           instructions,
		Options: ExecutionOptions{
			UseOptimization: true,
			Timeout:         3 * time.Minute,
			RetryCount:      1,
			ValidateOutput:  true,
			Debug:           false,
		},
	}
}

// CreateAgentAnalysisRequest creates a request for agent analysis
func CreateAgentAnalysisRequest(userIntent string, config interface{}) *ExecutionRequest {
	return &ExecutionRequest{
		TaskType:        TaskTypeAgentAnalysis,
		PromptBuilder:   NewAgentAnalysisPromptBuilder(),
		ContextBuilder:  NewWorkspaceContextBuilder(),
		OutputProcessor: NewAgentAnalysisOutputProcessor(),
		Input:           userIntent,
		Options: ExecutionOptions{
			UseOptimization: true,
			Timeout:         2 * time.Minute,
			RetryCount:      2,
			ValidateOutput:  true,
			Debug:           false,
		},
	}
}

// CreateAgentExecutionRequest creates a request for agent execution
func CreateAgentExecutionRequest(taskDescription string, config interface{}) *ExecutionRequest {
	return &ExecutionRequest{
		TaskType:        TaskTypeAgentExecution,
		PromptBuilder:   NewAgentExecutionPromptBuilder(),
		ContextBuilder:  NewFileOnlyContextBuilder(), // Focus on specific files for execution
		OutputProcessor: NewSimpleExecutionOutputProcessor(), // Use simple processor for execution results
		Input:           taskDescription,
		Options: ExecutionOptions{
			UseOptimization: true,
			Timeout:         2 * time.Minute,
			RetryCount:      1,
			ValidateOutput:  false, // Execution may have diverse outputs
			Debug:           false,
		},
	}
}