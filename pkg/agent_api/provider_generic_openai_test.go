package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenericOpenAIProvider(t *testing.T) {
	config := &ProviderConfig{
		Type:           "test-openai",
		DisplayName:    "Test OpenAI Provider",
		BaseURL:        "https://api.test.com/v1/chat/completions",
		APIKeyEnvVar:   "TEST_API_KEY",
		APIKeyRequired: false, // No API key required for test
		Features: ProviderFeatures{
			Tools:     true,
			Streaming: true,
		},
		DefaultModel:   "test-model",
		DefaultTimeout: 2 * time.Minute,
	}

	provider, err := NewGenericOpenAIProvider(config)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "test-model", provider.GetModel())
	assert.Equal(t, "Test OpenAI Provider", provider.GetName())
	assert.Equal(t, "test-openai", string(provider.GetType()))
	assert.True(t, provider.SupportsTools())
	assert.True(t, provider.SupportsStreaming())
}

func TestGenericOpenAIProvider_SendChatRequest(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Return a mock OpenAI response
		response := `{
			"id": "test-response",
			"object": "chat.completion",
			"created": 1234567890,
			"model": "test-model",
			"choices": [{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "Hello! How can I help you today?"
				},
				"finish_reason": "stop"
			}],
			"usage": {
				"prompt_tokens": 10,
				"completion_tokens": 20,
				"total_tokens": 30
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := &ProviderConfig{
		Type:           "test-openai",
		DisplayName:    "Test OpenAI Provider",
		BaseURL:        server.URL, // Use test server URL
		APIKeyRequired: false,
		Features: ProviderFeatures{
			Tools:     true,
			Streaming: false,
		},
		DefaultModel: "test-model",
	}

	provider, err := NewGenericOpenAIProvider(config)
	require.NoError(t, err)

	// Create a test request
	req := &ProviderChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		Options: &RequestOptions{
			MaxTokens: intPtr(100),
		},
	}

	// Send the request
	ctx := context.Background()
	response, err := provider.SendChatRequest(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, response)

	// Verify response
	assert.Equal(t, "test-response", response.ID)
	assert.Equal(t, "test-model", response.Model)
	assert.Len(t, response.Choices, 1)
	assert.Equal(t, "assistant", response.Choices[0].Message.Role)
	assert.Equal(t, "Hello! How can I help you today?", response.Choices[0].Message.Content)
	assert.Equal(t, 10, response.Usage.PromptTokens)
	assert.Equal(t, 20, response.Usage.CompletionTokens)
	assert.Equal(t, 30, response.Usage.TotalTokens)
}

func TestGenericOpenAIProvider_SendChatRequestStream(t *testing.T) {
	// Create a mock server that returns streaming data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Return streaming response
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Write streaming chunks
		chunks := []string{
			`data: {"id":"test-stream","object":"chat.completion.chunk","created":1234567890,"model":"test-model","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
			`data: {"id":"test-stream","object":"chat.completion.chunk","created":1234567890,"model":"test-model","choices":[{"index":0,"delta":{"content":" there!"},"finish_reason":null}]}`,
			`data: {"id":"test-stream","object":"chat.completion.chunk","created":1234567890,"model":"test-model","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":10,"total_tokens":15}}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	defer server.Close()

	config := &ProviderConfig{
		Type:           "test-openai-stream",
		DisplayName:    "Test OpenAI Streaming Provider",
		BaseURL:        server.URL,
		APIKeyRequired: false,
		Features: ProviderFeatures{
			Tools:     true,
			Streaming: true,
		},
		DefaultModel: "test-model",
	}

	provider, err := NewGenericOpenAIProvider(config)
	require.NoError(t, err)

	// Track streamed content
	var streamedContent strings.Builder
	var callbackCount int

	callback := func(content string) {
		callbackCount++
		streamedContent.WriteString(content)
	}

	// Create a test request
	req := &ProviderChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		Options: &RequestOptions{
			Stream: true,
		},
	}

	// Send the streaming request
	ctx := context.Background()
	response, err := provider.SendChatRequestStream(ctx, req, callback)
	require.NoError(t, err)
	require.NotNil(t, response)

	// Verify streamed content
	assert.Equal(t, "Hello there!", streamedContent.String())
	assert.Greater(t, callbackCount, 0)

	// Verify final response
	assert.Len(t, response.Choices, 1)
	assert.Equal(t, "assistant", response.Choices[0].Message.Role)
	assert.Equal(t, "Hello there!", response.Choices[0].Message.Content)
	assert.Equal(t, 5, response.Usage.PromptTokens)
	assert.Equal(t, 10, response.Usage.CompletionTokens)
	assert.Equal(t, 15, response.Usage.TotalTokens)
}

func TestGenericOpenAIProvider_ErrorHandling(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{
			"error": {
				"message": "Invalid request",
				"type": "invalid_request_error",
				"code": "bad_request"
			}
		}`))
	}))
	defer server.Close()

	config := &ProviderConfig{
		Type:           "test-error",
		DisplayName:    "Test Error Provider",
		BaseURL:        server.URL,
		APIKeyRequired: false,
		DefaultModel:   "test-model",
	}

	provider, err := NewGenericOpenAIProvider(config)
	require.NoError(t, err)

	req := &ProviderChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	_, err = provider.SendChatRequest(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid request")
	assert.Contains(t, err.Error(), "400")
}

func TestGenericOpenAIProvider_GetAvailableModels(t *testing.T) {
	// Create a mock server for the models endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/models") {
			response := `{
				"data": [
					{"id": "model-1", "object": "model", "created": 1234567890},
					{"id": "model-2", "object": "model", "created": 1234567891}
				]
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &ProviderConfig{
		Type:           "test-models",
		DisplayName:    "Test Models Provider",
		BaseURL:        server.URL + "/chat/completions", // This will be modified to /models
		APIKeyRequired: false,
		Features: ProviderFeatures{
			Tools: true,
		},
		DefaultModel: "model-1",
	}

	provider, err := NewGenericOpenAIProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	models, err := provider.GetAvailableModels(ctx)
	require.NoError(t, err)
	require.Len(t, models, 2)

	assert.Equal(t, "model-1", models[0].ID)
	assert.Equal(t, "model-1", models[0].Name)
	assert.True(t, models[0].IsDefault)
	assert.Contains(t, models[0].Features, "tools")

	assert.Equal(t, "model-2", models[1].ID)
	assert.Equal(t, "model-2", models[1].Name)
	assert.False(t, models[1].IsDefault)
}

func TestGenericOpenAIProviderFactory(t *testing.T) {
	config := &ProviderConfig{
		Type:           "factory-test",
		DisplayName:    "Factory Test Provider",
		BaseURL:        "https://api.factorytest.com/v1/chat/completions",
		APIKeyRequired: false,
		DefaultModel:   "factory-model",
	}

	provider, err := GenericOpenAIProviderFactory(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Verify it implements the Provider interface
	assert.Equal(t, "Factory Test Provider", provider.GetName())
	assert.Equal(t, "factory-test", string(provider.GetType()))
	assert.Equal(t, "factory-model", provider.GetModel())
}
