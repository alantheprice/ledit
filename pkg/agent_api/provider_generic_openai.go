package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// GenericOpenAIProvider implements a flexible provider for any OpenAI-compatible API
type GenericOpenAIProvider struct {
	*BaseProvider
	config       *ProviderConfig
	httpClient   *http.Client
	streamClient *http.Client
	apiKey       string
}

// NewGenericOpenAIProvider creates a new generic OpenAI-compatible provider
func NewGenericOpenAIProvider(config *ProviderConfig) (*GenericOpenAIProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("provider config cannot be nil")
	}

	var apiKey string
	if config.APIKeyRequired {
		apiKey = os.Getenv(config.APIKeyEnvVar)
		if apiKey == "" {
			return nil, fmt.Errorf("API key required but %s environment variable not set", config.APIKeyEnvVar)
		}
	}

	timeout := config.DefaultTimeout
	if timeout == 0 {
		timeout = 2 * time.Minute
	}

	// Create base provider
	baseProvider := NewBaseProvider(config.DisplayName, config.Type, config.BaseURL, apiKey)

	// Set feature flags from configuration
	baseProvider.supportsVision = config.Features.Vision
	baseProvider.supportsTools = config.Features.Tools
	baseProvider.supportsStreaming = config.Features.Streaming
	baseProvider.supportsReasoning = config.Features.Reasoning

	// Set default model
	if config.DefaultModel != "" {
		baseProvider.model = config.DefaultModel
	}

	provider := &GenericOpenAIProvider{
		BaseProvider: baseProvider,
		config:       config,
		apiKey:       apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		streamClient: &http.Client{
			Timeout: 15 * time.Minute, // Longer timeout for streaming
		},
	}

	return provider, nil
}

// SendChatRequest sends a chat request to the provider
func (p *GenericOpenAIProvider) SendChatRequest(ctx context.Context, req *ProviderChatRequest) (*ChatResponse, error) {
	// Build OpenAI-compatible request
	openaiReq := p.buildOpenAIRequest(req)

	// Serialize request
	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if p.debug {
		fmt.Printf("Sending request to %s:\n%s\n", p.config.BaseURL, string(reqBody))
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	p.setHeaders(httpReq)

	// Send request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if p.debug {
		fmt.Printf("Response from %s: %s\n", p.config.DisplayName, string(respBody))
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return nil, p.handleErrorResponse(resp.StatusCode, respBody)
	}

	// Parse response
	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Calculate cost if configuration is available
	if chatResp.Usage.PromptTokens > 0 || chatResp.Usage.CompletionTokens > 0 {
		registry := GetProviderRegistry()
		cost := registry.CalculateCost(p.config.Type, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens, p.model)
		chatResp.Usage.EstimatedCost = cost
	}

	return &chatResp, nil
}

// SendChatRequestStream sends a streaming chat request
func (p *GenericOpenAIProvider) SendChatRequestStream(ctx context.Context, req *ProviderChatRequest, callback StreamCallback) (*ChatResponse, error) {
	if !p.config.Features.Streaming {
		return nil, fmt.Errorf("provider %s does not support streaming", p.config.DisplayName)
	}

	// Build streaming request
	openaiReq := p.buildOpenAIRequest(req)
	openaiReq.Stream = true

	// Serialize request
	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if p.debug {
		fmt.Printf("Sending streaming request to %s\n", p.config.BaseURL)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	p.setHeaders(httpReq)

	// Send request
	resp, err := p.streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, p.handleErrorResponse(resp.StatusCode, respBody)
	}

	// Process streaming response
	return p.processStreamingResponse(resp.Body, callback)
}

// CheckConnection verifies the provider is accessible
func (p *GenericOpenAIProvider) CheckConnection(ctx context.Context) error {
	// Try to make a simple request or check the models endpoint
	// For OpenAI-compatible APIs, we'll try a minimal chat request
	testReq := &ProviderChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Hi"},
		},
		Options: &RequestOptions{
			MaxTokens: intPtr(1),
		},
	}

	_, err := p.SendChatRequest(ctx, testReq)
	return err
}

// GetAvailableModels returns the list of available models
func (p *GenericOpenAIProvider) GetAvailableModels(ctx context.Context) ([]ModelDetails, error) {
	// Try to fetch models from the provider's models endpoint
	modelsURL := strings.Replace(p.config.BaseURL, "/chat/completions", "/models", 1)

	req, err := http.NewRequestWithContext(ctx, "GET", modelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create models request: %w", err)
	}

	// Set headers
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	for key, value := range p.config.ExtraHeaders {
		req.Header.Set(key, value)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		// If models endpoint is not available, return a default model
		return []ModelDetails{
			{
				ID:            p.config.DefaultModel,
				Name:          p.config.DefaultModel,
				ContextLength: 8192, // Default context length
				IsDefault:     true,
			},
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Models endpoint not available, return default
		return []ModelDetails{
			{
				ID:            p.config.DefaultModel,
				Name:          p.config.DefaultModel,
				ContextLength: 8192,
				IsDefault:     true,
			},
		}, nil
	}

	// Parse models response
	var modelsResp struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
		} `json:"data"`
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read models response: %w", err)
	}

	if err := json.Unmarshal(respBody, &modelsResp); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	// Convert to ModelDetails
	models := make([]ModelDetails, 0, len(modelsResp.Data))
	for _, model := range modelsResp.Data {
		details := ModelDetails{
			ID:            model.ID,
			Name:          model.ID,
			ContextLength: 8192, // Default, could be improved with model-specific logic
			IsDefault:     model.ID == p.config.DefaultModel,
		}

		// Add features based on provider configuration
		if p.config.Features.Vision {
			details.Features = append(details.Features, "vision")
		}
		if p.config.Features.Tools {
			details.Features = append(details.Features, "tools")
		}
		if p.config.Features.Reasoning {
			details.Features = append(details.Features, "reasoning")
		}

		models = append(models, details)
	}

	return models, nil
}

// GetModelContextLimit returns the context limit for the current model
func (p *GenericOpenAIProvider) GetModelContextLimit() (int, error) {
	// Default context limit - could be improved with model-specific logic
	return 8192, nil
}

// buildOpenAIRequest converts a ProviderChatRequest to an OpenAI-compatible request
func (p *GenericOpenAIProvider) buildOpenAIRequest(req *ProviderChatRequest) *ChatRequest {
	openaiReq := &ChatRequest{
		Model:    p.model,
		Messages: req.Messages,
		Tools:    req.Tools,
	}

	if req.Options != nil {
		if req.Options.MaxTokens != nil {
			openaiReq.MaxTokens = *req.Options.MaxTokens
		}
		if req.Options.ReasoningEffort != "" {
			openaiReq.Reasoning = req.Options.ReasoningEffort
		}
		openaiReq.Stream = req.Options.Stream

		// Set tool choice if tools are provided
		if len(req.Tools) > 0 {
			openaiReq.ToolChoice = "auto"
		}
	}

	return openaiReq
}

// setHeaders sets the appropriate headers for the request
func (p *GenericOpenAIProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	// Set extra headers from configuration
	for key, value := range p.config.ExtraHeaders {
		req.Header.Set(key, value)
	}
}

// handleErrorResponse handles error responses from the provider
func (p *GenericOpenAIProvider) handleErrorResponse(statusCode int, body []byte) error {
	var errorResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
		return fmt.Errorf("%s error (%d): %s", p.config.DisplayName, statusCode, errorResp.Error.Message)
	}

	return fmt.Errorf("%s error (%d): %s", p.config.DisplayName, statusCode, string(body))
}

// processStreamingResponse processes a streaming response
func (p *GenericOpenAIProvider) processStreamingResponse(reader io.Reader, callback StreamCallback) (*ChatResponse, error) {
	scanner := bufio.NewScanner(reader)

	var finalResponse *ChatResponse
	var contentBuilder strings.Builder
	var reasoningBuilder strings.Builder

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || line == "data: [DONE]" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		// Extract JSON data
		jsonData := strings.TrimPrefix(line, "data: ")

		// Use the existing StreamingChatResponse type from streaming.go
		var streamChunk StreamingChatResponse
		if err := json.Unmarshal([]byte(jsonData), &streamChunk); err != nil {
			if p.debug {
				fmt.Printf("Failed to parse streaming chunk: %s\n", jsonData)
			}
			continue
		}

		// Process the chunk
		if len(streamChunk.Choices) > 0 {
			choice := streamChunk.Choices[0]

			// Accumulate content from delta
			if choice.Delta.Content != "" {
				contentBuilder.WriteString(choice.Delta.Content)

				// Call the callback with the chunk
				if callback != nil {
					callback(choice.Delta.Content)
				}
			}

			// Accumulate reasoning content
			if choice.Delta.ReasoningContent != "" {
				reasoningBuilder.WriteString(choice.Delta.ReasoningContent)
			}

			// Initialize the final response structure on first chunk
			if finalResponse == nil {
				finalResponse = &ChatResponse{
					ID:      streamChunk.ID,
					Object:  "chat.completion",
					Created: streamChunk.Created,
					Model:   streamChunk.Model,
					Choices: []Choice{
						{
							Index: 0,
							Message: struct {
								Role             string      `json:"role"`
								Content          string      `json:"content"`
								ReasoningContent string      `json:"reasoning_content,omitempty"`
								Images           []ImageData `json:"images,omitempty"`
								ToolCalls        []ToolCall  `json:"tool_calls,omitempty"`
							}{
								Role: "assistant",
							},
						},
					},
				}
			}

			// Update usage information if provided
			if streamChunk.Usage != nil {
				finalResponse.Usage.PromptTokens = streamChunk.Usage.PromptTokens
				finalResponse.Usage.CompletionTokens = streamChunk.Usage.CompletionTokens
				finalResponse.Usage.TotalTokens = streamChunk.Usage.TotalTokens
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading streaming response: %w", err)
	}

	// Finalize the response
	if finalResponse != nil {
		finalResponse.Choices[0].Message.Content = contentBuilder.String()
		finalResponse.Choices[0].Message.ReasoningContent = reasoningBuilder.String()

		// Calculate cost if we have token information
		if finalResponse.Usage.PromptTokens > 0 || finalResponse.Usage.CompletionTokens > 0 {
			registry := GetProviderRegistry()
			cost := registry.CalculateCost(p.config.Type, finalResponse.Usage.PromptTokens, finalResponse.Usage.CompletionTokens, p.model)
			finalResponse.Usage.EstimatedCost = cost
		}
	} else {
		// Create a minimal response if no data was received
		finalResponse = &ChatResponse{
			Choices: []Choice{
				{
					Message: struct {
						Role             string      `json:"role"`
						Content          string      `json:"content"`
						ReasoningContent string      `json:"reasoning_content,omitempty"`
						Images           []ImageData `json:"images,omitempty"`
						ToolCalls        []ToolCall  `json:"tool_calls,omitempty"`
					}{
						Role:    "assistant",
						Content: "",
					},
				},
			},
		}
	}

	// Final callback to indicate completion
	if callback != nil {
		callback("")
	}

	return finalResponse, nil
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

// ProviderFactoryFunc creates a generic OpenAI provider factory function
func GenericOpenAIProviderFactory(config *ProviderConfig) (Provider, error) {
	return NewGenericOpenAIProvider(config)
}
