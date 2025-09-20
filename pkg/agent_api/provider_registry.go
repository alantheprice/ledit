package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// ProviderFeatures defines capabilities supported by a provider
type ProviderFeatures struct {
	Vision    bool `yaml:"vision" json:"vision"`
	Tools     bool `yaml:"tools" json:"tools"`
	Streaming bool `yaml:"streaming" json:"streaming"`
	Reasoning bool `yaml:"reasoning" json:"reasoning"`
	Audio     bool `yaml:"audio" json:"audio"`
}

// CostTier represents a pricing tier for models
type CostTier struct {
	ModelPattern string  `yaml:"model_pattern" json:"model_pattern"`
	InputPer1M   float64 `yaml:"input_per_1m" json:"input_per_1m"`
	OutputPer1M  float64 `yaml:"output_per_1m" json:"output_per_1m"`
}

// CostConfig defines pricing configuration for a provider
type CostConfig struct {
	Type  string     `yaml:"type" json:"type"` // "tiered", "flat", "custom"
	Tiers []CostTier `yaml:"tiers,omitempty" json:"tiers,omitempty"`
}

// ProviderConfig defines the configuration for a provider
type ProviderConfig struct {
	Type           ClientType `yaml:"type" json:"type"`
	DisplayName    string     `yaml:"display_name" json:"display_name"`
	BaseURL        string     `yaml:"base_url" json:"base_url"`
	APIKeyEnvVar   string     `yaml:"api_key_env" json:"api_key_env"`
	APIKeyRequired bool       `yaml:"api_key_required" json:"api_key_required"`

	// Feature flags
	Features ProviderFeatures `yaml:"features" json:"features"`

	// Default settings
	DefaultModel   string        `yaml:"default_model" json:"default_model"`
	DefaultTimeout time.Duration `yaml:"default_timeout" json:"default_timeout"`

	// Cost configuration
	CostConfig *CostConfig `yaml:"cost_config,omitempty" json:"cost_config,omitempty"`

	// Provider-specific configurations
	ExtraHeaders map[string]string      `yaml:"extra_headers,omitempty" json:"extra_headers,omitempty"`
	ExtraParams  map[string]interface{} `yaml:"extra_params,omitempty" json:"extra_params,omitempty"`
}

// TokenCounterFunc defines a custom token counting function
type TokenCounterFunc func(text string, model string) (int, error)

// CostCalculatorFunc defines a custom cost calculation function
type CostCalculatorFunc func(promptTokens, completionTokens int, model string) float64

// RequestBuilderFunc defines a custom request builder function
type RequestBuilderFunc func(ctx context.Context, req *ProviderChatRequest) (interface{}, error)

// ResponseParserFunc defines a custom response parser function
type ResponseParserFunc func(data []byte) (*ChatResponse, error)

// ProviderFactory function type for creating providers
type ProviderFactory func(config *ProviderConfig) (Provider, error)

// ProviderRegistry manages provider configurations and instances
type ProviderRegistry struct {
	providers map[ClientType]*ProviderConfig
	factories map[ClientType]ProviderFactory
	instances map[ClientType]Provider
	mu        sync.RWMutex

	// Plugin functions
	tokenCounters   map[ClientType]TokenCounterFunc
	costCalculators map[ClientType]CostCalculatorFunc
	requestBuilders map[ClientType]RequestBuilderFunc
	responseParsers map[ClientType]ResponseParserFunc
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers:       make(map[ClientType]*ProviderConfig),
		factories:       make(map[ClientType]ProviderFactory),
		instances:       make(map[ClientType]Provider),
		tokenCounters:   make(map[ClientType]TokenCounterFunc),
		costCalculators: make(map[ClientType]CostCalculatorFunc),
		requestBuilders: make(map[ClientType]RequestBuilderFunc),
		responseParsers: make(map[ClientType]ResponseParserFunc),
	}
}

// Global registry instance
var globalRegistry *ProviderRegistry
var registryOnce sync.Once

// GetProviderRegistry returns the global provider registry
func GetProviderRegistry() *ProviderRegistry {
	registryOnce.Do(func() {
		globalRegistry = NewProviderRegistry()
	})
	return globalRegistry
}

// RegisterProvider registers a provider configuration
func (r *ProviderRegistry) RegisterProvider(config *ProviderConfig) error {
	if config == nil {
		return fmt.Errorf("provider config cannot be nil")
	}

	// Validate the configuration
	if err := r.validateConfig(config); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[config.Type] = config
	return nil
}

// RegisterProviderFactory registers a factory function for a provider type
func (r *ProviderRegistry) RegisterProviderFactory(providerType ClientType, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.factories[providerType] = factory
}

// GetProviderConfig returns the configuration for a provider type
func (r *ProviderRegistry) GetProviderConfig(providerType ClientType) (*ProviderConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config, exists := r.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("provider type %s not registered", providerType)
	}

	return config, nil
}

// GetProvider returns a provider instance, creating it if necessary
func (r *ProviderRegistry) GetProvider(providerType ClientType) (Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if we already have an instance
	if instance, exists := r.instances[providerType]; exists {
		return instance, nil
	}

	// Get the configuration
	config, exists := r.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("provider type %s not registered", providerType)
	}

	// Get the factory function
	factory, exists := r.factories[providerType]
	if !exists {
		// Fall back to generic OpenAI provider for OpenAI-compatible APIs
		factory = r.factories["generic-openai"]
		if factory == nil {
			return nil, fmt.Errorf("no factory registered for provider type %s", providerType)
		}
	}

	// Create the provider instance
	instance, err := factory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", providerType, err)
	}

	// Cache the instance
	r.instances[providerType] = instance

	return instance, nil
}

// IsProviderAvailable checks if a provider is available (has valid configuration and API key)
func (r *ProviderRegistry) IsProviderAvailable(providerType ClientType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config, exists := r.providers[providerType]
	if !exists {
		return false
	}

	// Check if API key is required and available
	if config.APIKeyRequired {
		apiKey := os.Getenv(config.APIKeyEnvVar)
		return apiKey != ""
	}

	return true
}

// ListProviders returns all registered provider types
func (r *ProviderRegistry) ListProviders() []ClientType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]ClientType, 0, len(r.providers))
	for providerType := range r.providers {
		providers = append(providers, providerType)
	}

	return providers
}

// ListAvailableProviders returns all available provider types
func (r *ProviderRegistry) ListAvailableProviders() []ClientType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]ClientType, 0, len(r.providers))
	for providerType := range r.providers {
		if r.IsProviderAvailable(providerType) {
			providers = append(providers, providerType)
		}
	}

	return providers
}

// LoadFromDirectory loads provider configurations from a directory
func (r *ProviderRegistry) LoadFromDirectory(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, not an error
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read provider directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		configPath := filepath.Join(dir, entry.Name())
		if err := r.LoadFromFile(configPath); err != nil {
			return fmt.Errorf("failed to load provider config from %s: %w", configPath, err)
		}
	}

	return nil
}

// LoadFromFile loads a provider configuration from a YAML file
func (r *ProviderRegistry) LoadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", filename, err)
	}

	var config ProviderConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse YAML config %s: %w", filename, err)
	}

	// Validate the configuration
	if err := r.validateConfig(&config); err != nil {
		return fmt.Errorf("invalid configuration in %s: %w", filename, err)
	}

	return r.RegisterProvider(&config)
}

// validateConfig validates a provider configuration
func (r *ProviderRegistry) validateConfig(config *ProviderConfig) error {
	if config.Type == "" {
		return fmt.Errorf("provider type is required")
	}

	if config.DisplayName == "" {
		return fmt.Errorf("display name is required")
	}

	if config.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	if config.APIKeyRequired && config.APIKeyEnvVar == "" {
		return fmt.Errorf("API key environment variable is required when API key is required")
	}

	return nil
}

// CalculateCost calculates the cost for a request using the provider's cost configuration
func (r *ProviderRegistry) CalculateCost(providerType ClientType, promptTokens, completionTokens int, model string) float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check for custom cost calculator
	if calculator, exists := r.costCalculators[providerType]; exists {
		return calculator(promptTokens, completionTokens, model)
	}

	// Use provider configuration
	config, exists := r.providers[providerType]
	if !exists || config.CostConfig == nil {
		// Default fallback cost
		inputCost := float64(promptTokens) * 0.001 / 1000
		outputCost := float64(completionTokens) * 0.002 / 1000
		return inputCost + outputCost
	}

	// Find matching tier
	for _, tier := range config.CostConfig.Tiers {
		// Simple pattern matching - in practice, you'd want regex or glob
		if tier.ModelPattern == model || tier.ModelPattern == "*" {
			inputCost := float64(promptTokens) * tier.InputPer1M / 1000000
			outputCost := float64(completionTokens) * tier.OutputPer1M / 1000000
			return inputCost + outputCost
		}
	}

	// Fallback to default
	inputCost := float64(promptTokens) * 0.001 / 1000
	outputCost := float64(completionTokens) * 0.002 / 1000
	return inputCost + outputCost
}

// SetTokenCounter sets a custom token counter for a provider
func (r *ProviderRegistry) SetTokenCounter(providerType ClientType, counter TokenCounterFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tokenCounters[providerType] = counter
}

// SetCostCalculator sets a custom cost calculator for a provider
func (r *ProviderRegistry) SetCostCalculator(providerType ClientType, calculator CostCalculatorFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.costCalculators[providerType] = calculator
}

// SetRequestBuilder sets a custom request builder for a provider
func (r *ProviderRegistry) SetRequestBuilder(providerType ClientType, builder RequestBuilderFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.requestBuilders[providerType] = builder
}

// SetResponseParser sets a custom response parser for a provider
func (r *ProviderRegistry) SetResponseParser(providerType ClientType, parser ResponseParserFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.responseParsers[providerType] = parser
}

// LoadDefaultConfigurations loads built-in default configurations
func (r *ProviderRegistry) LoadDefaultConfigurations() error {
	// OpenAI
	openaiConfig := &ProviderConfig{
		Type:           OpenAIClientType,
		DisplayName:    "OpenAI",
		BaseURL:        "https://api.openai.com/v1/chat/completions",
		APIKeyEnvVar:   "OPENAI_API_KEY",
		APIKeyRequired: true,
		Features: ProviderFeatures{
			Vision:    true,
			Tools:     true,
			Streaming: true,
			Reasoning: true,
			Audio:     false,
		},
		DefaultModel:   "gpt-4o-mini",
		DefaultTimeout: 2 * time.Minute,
		CostConfig: &CostConfig{
			Type: "tiered",
			Tiers: []CostTier{
				{ModelPattern: "gpt-4o-mini", InputPer1M: 0.15, OutputPer1M: 0.60},
				{ModelPattern: "gpt-4o", InputPer1M: 2.50, OutputPer1M: 10.00},
				{ModelPattern: "o1-preview", InputPer1M: 15.00, OutputPer1M: 60.00},
				{ModelPattern: "o1-mini", InputPer1M: 3.00, OutputPer1M: 12.00},
			},
		},
	}

	// DeepInfra
	deepinfraConfig := &ProviderConfig{
		Type:           DeepInfraClientType,
		DisplayName:    "DeepInfra",
		BaseURL:        "https://api.deepinfra.com/v1/openai/chat/completions",
		APIKeyEnvVar:   "DEEPINFRA_API_KEY",
		APIKeyRequired: true,
		Features: ProviderFeatures{
			Vision:    true,
			Tools:     true,
			Streaming: true,
			Reasoning: false,
			Audio:     false,
		},
		DefaultModel:   "meta-llama/Llama-3.3-70B-Instruct",
		DefaultTimeout: 2 * time.Minute,
		CostConfig: &CostConfig{
			Type: "tiered",
			Tiers: []CostTier{
				{ModelPattern: "*", InputPer1M: 0.27, OutputPer1M: 0.27},
			},
		},
	}

	// OpenRouter
	openrouterConfig := &ProviderConfig{
		Type:           OpenRouterClientType,
		DisplayName:    "OpenRouter",
		BaseURL:        "https://openrouter.ai/api/v1/chat/completions",
		APIKeyEnvVar:   "OPENROUTER_API_KEY",
		APIKeyRequired: true,
		Features: ProviderFeatures{
			Vision:    true,
			Tools:     true,
			Streaming: true,
			Reasoning: false,
			Audio:     false,
		},
		DefaultModel:   "anthropic/claude-3.5-sonnet",
		DefaultTimeout: 2 * time.Minute,
		CostConfig: &CostConfig{
			Type: "tiered",
			Tiers: []CostTier{
				{ModelPattern: "*", InputPer1M: 1.00, OutputPer1M: 3.00}, // Approximate
			},
		},
	}

	// Ollama Local
	ollamaConfig := &ProviderConfig{
		Type:           OllamaLocalClientType,
		DisplayName:    "Ollama (Local)",
		BaseURL:        "http://localhost:11434/api/chat",
		APIKeyEnvVar:   "",
		APIKeyRequired: false,
		Features: ProviderFeatures{
			Vision:    false,
			Tools:     true,
			Streaming: true,
			Reasoning: false,
			Audio:     false,
		},
		DefaultModel:   "llama3.2",
		DefaultTimeout: 5 * time.Minute,
		CostConfig: &CostConfig{
			Type: "flat",
			Tiers: []CostTier{
				{ModelPattern: "*", InputPer1M: 0.0, OutputPer1M: 0.0},
			},
		},
	}

	// Ollama Turbo
	ollamaTurboConfig := &ProviderConfig{
		Type:           OllamaTurboClientType,
		DisplayName:    "Ollama Turbo",
		BaseURL:        "https://ollama.com/v1/chat/completions",
		APIKeyEnvVar:   "OLLAMA_API_KEY",
		APIKeyRequired: true,
		Features: ProviderFeatures{
			Vision:    false,
			Tools:     true,
			Streaming: true,
			Reasoning: false,
			Audio:     false,
		},
		DefaultModel:   "llama3.2",
		DefaultTimeout: 2 * time.Minute,
		CostConfig: &CostConfig{
			Type: "tiered",
			Tiers: []CostTier{
				{ModelPattern: "*", InputPer1M: 0.1, OutputPer1M: 0.1},
			},
		},
	}

	// Register all configurations
	configs := []*ProviderConfig{
		openaiConfig,
		deepinfraConfig,
		openrouterConfig,
		ollamaConfig,
		ollamaTurboConfig,
	}

	for _, config := range configs {
		if err := r.RegisterProvider(config); err != nil {
			return fmt.Errorf("failed to register default config for %s: %w", config.Type, err)
		}
	}

	return nil
}

// Initialize sets up the global registry with default configurations
func Initialize() error {
	registry := GetProviderRegistry()

	// Register the generic OpenAI factory as the default factory
	registry.RegisterProviderFactory("generic-openai", GenericOpenAIProviderFactory)

	// Load default configurations
	if err := registry.LoadDefaultConfigurations(); err != nil {
		return fmt.Errorf("failed to load default configurations: %w", err)
	}

	// Load user configurations from multiple directories
	configDirs := []string{
		filepath.Join(os.Getenv("HOME"), ".ledit", "providers"),
		filepath.Join(".", ".ledit", "providers"),
	}

	for _, dir := range configDirs {
		if err := registry.LoadFromDirectory(dir); err != nil {
			return fmt.Errorf("failed to load configurations from %s: %w", dir, err)
		}
	}

	return nil
}
