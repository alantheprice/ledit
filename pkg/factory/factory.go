package factory

import (
	"context"
	"fmt"

	api "github.com/alantheprice/ledit/pkg/agent_api"
	"github.com/alantheprice/ledit/pkg/agent_providers"
)

// DeepInfraClientWrapper wraps DeepInfraProvider to implement the full ClientInterface
type DeepInfraClientWrapper struct {
	provider *providers.DeepInfraProvider
}

// Delegate all methods to the provider
func (w *DeepInfraClientWrapper) SendChatRequest(messages []api.Message, tools []api.Tool, reasoning string) (*api.ChatResponse, error) {
	return w.provider.SendChatRequest(messages, tools, reasoning)
}

func (w *DeepInfraClientWrapper) SendChatRequestStream(messages []api.Message, tools []api.Tool, reasoning string, callback api.StreamCallback) (*api.ChatResponse, error) {
	return w.provider.SendChatRequestStream(messages, tools, reasoning, callback)
}

func (w *DeepInfraClientWrapper) CheckConnection() error {
	return w.provider.CheckConnection()
}

func (w *DeepInfraClientWrapper) SetDebug(debug bool) {
	w.provider.SetDebug(debug)
}

func (w *DeepInfraClientWrapper) SetModel(model string) error {
	return w.provider.SetModel(model)
}

func (w *DeepInfraClientWrapper) GetModel() string {
	return w.provider.GetModel()
}

func (w *DeepInfraClientWrapper) GetProvider() string {
	return w.provider.GetProvider()
}

func (w *DeepInfraClientWrapper) GetModelContextLimit() (int, error) {
	return w.provider.GetModelContextLimit()
}

func (w *DeepInfraClientWrapper) SupportsVision() bool {
	return w.provider.SupportsVision()
}

func (w *DeepInfraClientWrapper) GetVisionModel() string {
	return w.provider.GetVisionModel()
}

func (w *DeepInfraClientWrapper) SendVisionRequest(messages []api.Message, tools []api.Tool, reasoning string) (*api.ChatResponse, error) {
	return w.provider.SendVisionRequest(messages, tools, reasoning)
}

func (w *DeepInfraClientWrapper) ListModels() ([]api.ModelInfo, error) {
	return w.provider.ListModels()
}

// TPS methods that the provider doesn't implement
func (w *DeepInfraClientWrapper) GetLastTPS() float64 {
	return 0.0 // Provider doesn't track TPS
}

func (w *DeepInfraClientWrapper) GetAverageTPS() float64 {
	return 0.0 // Provider doesn't track TPS
}

func (w *DeepInfraClientWrapper) GetTPSStats() map[string]float64 {
	return map[string]float64{} // Provider doesn't track TPS
}

func (w *DeepInfraClientWrapper) ResetTPSStats() {
	// No-op - provider doesn't track TPS
}

// OpenRouterClientWrapper wraps OpenRouterProvider to implement the full ClientInterface
type OpenRouterClientWrapper struct {
	provider *providers.OpenRouterProvider
}

// Delegate all methods to the provider
func (w *OpenRouterClientWrapper) SendChatRequest(messages []api.Message, tools []api.Tool, reasoning string) (*api.ChatResponse, error) {
	return w.provider.SendChatRequest(messages, tools, reasoning)
}

func (w *OpenRouterClientWrapper) SendChatRequestStream(messages []api.Message, tools []api.Tool, reasoning string, callback api.StreamCallback) (*api.ChatResponse, error) {
	return w.provider.SendChatRequestStream(messages, tools, reasoning, callback)
}

func (w *OpenRouterClientWrapper) CheckConnection() error {
	return w.provider.CheckConnection()
}

func (w *OpenRouterClientWrapper) SetDebug(debug bool) {
	w.provider.SetDebug(debug)
}

func (w *OpenRouterClientWrapper) SetModel(model string) error {
	return w.provider.SetModel(model)
}

func (w *OpenRouterClientWrapper) GetModel() string {
	return w.provider.GetModel()
}

func (w *OpenRouterClientWrapper) GetProvider() string {
	return w.provider.GetProvider()
}

func (w *OpenRouterClientWrapper) GetModelContextLimit() (int, error) {
	return w.provider.GetModelContextLimit()
}

func (w *OpenRouterClientWrapper) SupportsVision() bool {
	return w.provider.SupportsVision()
}

func (w *OpenRouterClientWrapper) GetVisionModel() string {
	return w.provider.GetVisionModel()
}

func (w *OpenRouterClientWrapper) SendVisionRequest(messages []api.Message, tools []api.Tool, reasoning string) (*api.ChatResponse, error) {
	return w.provider.SendVisionRequest(messages, tools, reasoning)
}

func (w *OpenRouterClientWrapper) ListModels() ([]api.ModelInfo, error) {
	return w.provider.ListModels()
}

// TPS methods that the provider now implements
func (w *OpenRouterClientWrapper) GetLastTPS() float64 {
	return w.provider.GetLastTPS()
}

func (w *OpenRouterClientWrapper) GetAverageTPS() float64 {
	return w.provider.GetAverageTPS()
}

func (w *OpenRouterClientWrapper) GetTPSStats() map[string]float64 {
	return w.provider.GetTPSStats()
}

func (w *OpenRouterClientWrapper) ResetTPSStats() {
	w.provider.ResetTPSStats()
}

// CreateProviderClient is a factory function that creates providers
func CreateProviderClient(clientType api.ClientType, model string) (api.ClientInterface, error) {
	// Try the new registry-based approach first
	registry := api.GetProviderRegistry()
	if registry.IsProviderAvailable(clientType) {
		provider, err := registry.GetProvider(clientType)
		if err == nil {
			// Set the model if specified
			if model != "" {
				if err := provider.SetModel(model); err != nil {
					return nil, fmt.Errorf("failed to set model %s for provider %s: %w", model, clientType, err)
				}
			}
			// Wrap the new Provider in a ClientInterface adapter
			return NewClientInterfaceFromProvider(provider), nil
		}
		// If registry fails, fall back to legacy implementations
	}

	// Legacy factory implementation for backward compatibility
	switch clientType {
	case api.OpenAIClientType:
		return api.NewOpenAIClientWrapper(model)
	case api.DeepInfraClientType:
		// Use the real DeepInfra provider wrapped to implement ClientInterface
		provider, err := providers.NewDeepInfraProviderWithModel(model)
		if err != nil {
			return nil, err
		}
		return &DeepInfraClientWrapper{provider: provider}, nil
	case api.OllamaClientType, api.OllamaLocalClientType:
		return api.NewOllamaLocalClient(model)
	case api.OllamaTurboClientType:
		return api.NewOllamaTurboClient(model)
	case api.OpenRouterClientType:
		// Use the real OpenRouter provider wrapped to implement ClientInterface
		provider, err := providers.NewOpenRouterProviderWithModel(model)
		if err != nil {
			return nil, err
		}
		return &OpenRouterClientWrapper{provider: provider}, nil
	default:
		return nil, fmt.Errorf("unknown client type: %s", clientType)
	}
}

// CreateProviderFromRegistry creates a provider using the new registry system
func CreateProviderFromRegistry(clientType api.ClientType, model string) (api.Provider, error) {
	registry := api.GetProviderRegistry()

	provider, err := registry.GetProvider(clientType)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", clientType, err)
	}

	// Set the model if specified
	if model != "" {
		if err := provider.SetModel(model); err != nil {
			return nil, fmt.Errorf("failed to set model %s for provider %s: %w", model, clientType, err)
		}
	}

	return provider, nil
}

// ProviderClientWrapper wraps the new Provider interface to implement ClientInterface
type ProviderClientWrapper struct {
	provider api.Provider
}

// NewClientInterfaceFromProvider creates a ClientInterface adapter for a Provider
func NewClientInterfaceFromProvider(provider api.Provider) api.ClientInterface {
	return &ProviderClientWrapper{provider: provider}
}

// Implement ClientInterface methods by delegating to the Provider

func (w *ProviderClientWrapper) SendChatRequest(messages []api.Message, tools []api.Tool, reasoning string) (*api.ChatResponse, error) {
	req := &api.ProviderChatRequest{
		Messages: messages,
		Tools:    tools,
		Options: &api.RequestOptions{
			ReasoningEffort: reasoning,
		},
	}
	return w.provider.SendChatRequest(context.Background(), req)
}

func (w *ProviderClientWrapper) SendChatRequestStream(messages []api.Message, tools []api.Tool, reasoning string, callback api.StreamCallback) (*api.ChatResponse, error) {
	req := &api.ProviderChatRequest{
		Messages: messages,
		Tools:    tools,
		Options: &api.RequestOptions{
			ReasoningEffort: reasoning,
			Stream:          true,
		},
	}
	return w.provider.SendChatRequestStream(context.Background(), req, callback)
}

func (w *ProviderClientWrapper) CheckConnection() error {
	return w.provider.CheckConnection(context.Background())
}

func (w *ProviderClientWrapper) SetDebug(debug bool) {
	w.provider.SetDebug(debug)
}

func (w *ProviderClientWrapper) SetModel(model string) error {
	return w.provider.SetModel(model)
}

func (w *ProviderClientWrapper) GetModel() string {
	return w.provider.GetModel()
}

func (w *ProviderClientWrapper) GetProvider() string {
	return w.provider.GetName()
}

func (w *ProviderClientWrapper) GetModelContextLimit() (int, error) {
	return w.provider.GetModelContextLimit()
}

func (w *ProviderClientWrapper) SupportsVision() bool {
	return w.provider.SupportsVision()
}

func (w *ProviderClientWrapper) GetVisionModel() string {
	// For now, return the current model - could be enhanced to return vision-specific model
	return w.provider.GetModel()
}

func (w *ProviderClientWrapper) SendVisionRequest(messages []api.Message, tools []api.Tool, reasoning string) (*api.ChatResponse, error) {
	// Use the regular chat request - the provider should handle vision content in messages
	return w.SendChatRequest(messages, tools, reasoning)
}

func (w *ProviderClientWrapper) ListModels() ([]api.ModelInfo, error) {
	// Get models from the new interface and convert to old format
	modelDetails, err := w.provider.GetAvailableModels(context.Background())
	if err != nil {
		return nil, err
	}

	models := make([]api.ModelInfo, len(modelDetails))
	for i, detail := range modelDetails {
		models[i] = api.ModelInfo{
			ID:            detail.ID,
			Name:          detail.Name,
			ContextLength: detail.ContextLength,
		}
	}

	return models, nil
}

// TPS tracking methods - not implemented by the new Provider interface
func (w *ProviderClientWrapper) GetLastTPS() float64 {
	return 0.0
}

func (w *ProviderClientWrapper) GetAverageTPS() float64 {
	return 0.0
}

func (w *ProviderClientWrapper) GetTPSStats() map[string]float64 {
	return map[string]float64{}
}

func (w *ProviderClientWrapper) ResetTPSStats() {
	// No-op
}
