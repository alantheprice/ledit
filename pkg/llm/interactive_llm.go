package llm

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alantheprice/ledit/pkg/config"
	"github.com/alantheprice/ledit/pkg/prompts"
	ui "github.com/alantheprice/ledit/pkg/ui"
	"github.com/alantheprice/ledit/pkg/utils"
)

// CallLLMWithInteractiveContext handles interactive LLM calls with tool support
func CallLLMWithInteractiveContext(
	modelName string,
	initialMessages []prompts.Message,
	filename string,
	cfg *config.Config,
	timeout time.Duration,
	contextHandler ContextHandler,
) (string, error) {
	logger := utils.GetLogger(cfg.SkipPrompt)
	
	// Initialize session
	session := &InteractiveSession{
		ModelName:      modelName,
		Config:         cfg,
		Logger:         logger,
		Timeout:        timeout,
		ContextHandler: contextHandler,
	}

	// Setup initial context
	if err := session.setupInitialContext(initialMessages); err != nil {
		return "", fmt.Errorf("failed to setup initial context: %w", err)
	}

	// Execute interactive loop
	return session.executeInteractiveLoop()
}

// InteractiveSession manages an interactive LLM session with tool calling
type InteractiveSession struct {
	ModelName       string
	Config          *config.Config
	Logger          *utils.Logger
	Timeout         time.Duration
	ContextHandler  ContextHandler
	CurrentMessages []prompts.Message
	MaxIterations   int
}

// setupInitialContext prepares the initial message context
func (s *InteractiveSession) setupInitialContext(initialMessages []prompts.Message) error {
	s.Logger.Logf("DEBUG: Setting up interactive context with model: %s", s.ModelName)
	s.Logger.Logf("DEBUG: Initial messages count: %d", len(initialMessages))

	// Detect mentioned files and enhance messages
	userPrompt := s.extractUserPrompt(initialMessages)
	detector := NewFileDetector()
	mentionedFiles := detector.DetectMentionedFiles(userPrompt)

	// Enhance messages with tools and file warnings
	s.CurrentMessages = s.enhanceMessagesWithTools(initialMessages, mentionedFiles)
	s.logSystemPromptHash()

	s.MaxIterations = 5 // Reasonable default for interactive sessions
	return nil
}

// extractUserPrompt extracts user content from messages
func (s *InteractiveSession) extractUserPrompt(messages []prompts.Message) string {
	var userPrompt string
	for _, msg := range messages {
		if msg.Role == "user" {
			userPrompt += fmt.Sprintf("%v ", msg.Content)
		}
	}
	return userPrompt
}

// enhanceMessagesWithTools adds tool information to messages
func (s *InteractiveSession) enhanceMessagesWithTools(messages []prompts.Message, mentionedFiles []string) []prompts.Message {
	toolInfo := FormatToolsForPrompt()
	fileWarning := ""
	if len(mentionedFiles) > 0 {
		fileWarning = GenerateFileReadPrompt(mentionedFiles)
	}

	var enhanced []prompts.Message
	for i, msg := range messages {
		if i == 0 && msg.Role == "system" {
			// Enhance existing system message
			originalContent := fmt.Sprintf("%v", msg.Content)
			enhancedContent := fmt.Sprintf("%s\n\n%s%s", originalContent, toolInfo, fileWarning)
			enhanced = append(enhanced, prompts.Message{
				Role:    msg.Role,
				Content: enhancedContent,
			})
		} else {
			enhanced = append(enhanced, msg)
		}
	}

	// Add system message if none exists
	if len(enhanced) == 0 || enhanced[0].Role != "system" {
		systemMessage := prompts.Message{
			Role:    "system",
			Content: toolInfo + fileWarning,
		}
		enhanced = append([]prompts.Message{systemMessage}, enhanced...)
	}

	return enhanced
}

// logSystemPromptHash logs system prompt hash for drift detection
func (s *InteractiveSession) logSystemPromptHash() {
	if len(s.CurrentMessages) > 0 && s.CurrentMessages[0].Role == "system" {
		contentStr := fmt.Sprintf("%v", s.CurrentMessages[0].Content)
		h := sha1.Sum([]byte(contentStr))
		ui.Out().Printf("[tools] system_prompt_hash: %x\n", h)
		
		if rl := utils.GetRunLogger(); rl != nil {
			msgDump, _ := json.Marshal(s.CurrentMessages)
			rl.LogEvent("interactive_start", map[string]any{
				"model":    s.ModelName,
				"messages": string(msgDump),
			})
		}
	}
}

// executeInteractiveLoop runs the main interactive loop with tool calling
func (s *InteractiveSession) executeInteractiveLoop() (string, error) {
	for iteration := 0; iteration < s.MaxIterations; iteration++ {
		s.Logger.Logf("DEBUG: Starting iteration %d/%d", iteration+1, s.MaxIterations)

		// Get LLM response
		response, tokenUsage, err := s.getLLMResponse()
		if err != nil {
			return "", fmt.Errorf("LLM request failed on iteration %d: %w", iteration+1, err)
		}

		s.logTokenUsage(tokenUsage)

		// Check for tool calls
		if !containsToolCall(response) {
			s.Logger.Log("DEBUG: No tool calls detected, returning response")
			return response, nil
		}

		// Execute tool calls and continue
		if err := s.handleToolCalls(response); err != nil {
			return "", fmt.Errorf("tool execution failed on iteration %d: %w", iteration+1, err)
		}
	}

	return "", fmt.Errorf("exceeded maximum iterations (%d) in interactive loop", s.MaxIterations)
}

// getLLMResponse gets response from LLM
func (s *InteractiveSession) getLLMResponse() (string, *TokenUsage, error) {
	return GetLLMResponse(s.ModelName, s.CurrentMessages, "", s.Config, s.Timeout)
}

// logTokenUsage logs token usage information
func (s *InteractiveSession) logTokenUsage(tokenUsage *TokenUsage) {
	if tokenUsage != nil && tokenUsage.TotalTokens > 0 {
		s.Logger.Logf("DEBUG: Token usage - Total: %d, Prompt: %d, Completion: %d",
			tokenUsage.TotalTokens, tokenUsage.PromptTokens, tokenUsage.CompletionTokens)
	}
}

// handleToolCalls processes tool calls in the response
func (s *InteractiveSession) handleToolCalls(response string) error {
	s.Logger.Log("DEBUG: Tool calls detected, parsing...")

	// Parse tool calls
	toolCalls, err := parseToolCalls(response)
	if err != nil {
		return fmt.Errorf("failed to parse tool calls: %w", err)
	}

	s.Logger.Logf("DEBUG: Parsed %d tool calls", len(toolCalls))

	// Add assistant response to conversation
	s.CurrentMessages = append(s.CurrentMessages, prompts.Message{
		Role:    "assistant",
		Content: response,
	})

	// Execute tool calls
	for _, toolCall := range toolCalls {
		if err := s.executeToolCall(toolCall); err != nil {
			s.Logger.Logf("WARNING: Tool call execution failed: %v", err)
			// Continue with other tool calls rather than failing entirely
		}
	}

	return nil
}

// executeToolCall executes a single tool call
func (s *InteractiveSession) executeToolCall(toolCall ToolCall) error {
	s.Logger.Logf("DEBUG: Executing tool call: %s", toolCall.Function.Name)

	result, err := ExecuteBasicToolCallWithContext(context.Background(), toolCall, s.Config)
	if err != nil {
		result = fmt.Sprintf("Error executing %s: %v", toolCall.Function.Name, err)
	}

	// Sanitize output for logging
	sanitizedResult := sanitizeOutput(result)
	s.Logger.Logf("DEBUG: Tool call result length: %d", len(sanitizedResult))

	// Add tool result to conversation
	s.CurrentMessages = append(s.CurrentMessages, prompts.Message{
		Role:    "tool",
		Content: result,
	})

	return nil
}

// extractContextRequests extracts context requests from LLM response (legacy support)
func extractContextRequests(response string) ([]ContextRequest, error) {
	// Look for context request patterns in the response
	var requests []ContextRequest
	
	// Simple pattern matching for now - could be enhanced
	if strings.Contains(strings.ToLower(response), "need more context") ||
	   strings.Contains(strings.ToLower(response), "additional information") {
		requests = append(requests, ContextRequest{
			Type:  "general",
			Query: "Additional context needed",
		})
	}

	return requests, nil
}