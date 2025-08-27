package llm

import (
	"github.com/alantheprice/ledit/pkg/config"
)

// ContextHandler is a function type that defines how context requests are handled.
// It takes a slice of ContextRequest and returns a string response and an error.
type ContextHandler func([]ContextRequest, *config.Config) (string, error)

// Global context handler for tool execution
var globalContextHandler ContextHandler

// SetGlobalContextHandler sets the global context handler for tool execution
func SetGlobalContextHandler(handler ContextHandler) {
	globalContextHandler = handler
}

// ContextRequest represents a request for additional context from the LLM.
type ContextRequest struct {
	Type  string `json:"type"`
	Query string `json:"query"`
}

// ContextResponse represents the LLM's response containing context requests.
type ContextResponse struct {
	ContextRequests []ContextRequest `json:"context_requests"`
}