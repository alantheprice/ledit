package api

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderRegistry_RegisterProvider(t *testing.T) {
	registry := NewProviderRegistry()

	config := &ProviderConfig{
		Type:           "test-provider",
		DisplayName:    "Test Provider",
		BaseURL:        "https://api.test.com/v1/chat/completions",
		APIKeyEnvVar:   "TEST_API_KEY",
		APIKeyRequired: true,
		Features: ProviderFeatures{
			Tools:     true,
			Streaming: true,
		},
		DefaultModel:   "test-model",
		DefaultTimeout: 2 * time.Minute,
	}

	err := registry.RegisterProvider(config)
	require.NoError(t, err)

	// Verify the provider was registered
	retrievedConfig, err := registry.GetProviderConfig("test-provider")
	require.NoError(t, err)
	assert.Equal(t, config.DisplayName, retrievedConfig.DisplayName)
	assert.Equal(t, config.BaseURL, retrievedConfig.BaseURL)
}

func TestProviderRegistry_IsProviderAvailable(t *testing.T) {
	registry := NewProviderRegistry()

	// Register a test provider that requires an API key
	config := &ProviderConfig{
		Type:           "test-with-key",
		DisplayName:    "Test With Key",
		BaseURL:        "https://api.test.com/v1/chat/completions",
		APIKeyEnvVar:   "TEST_WITH_KEY_API_KEY",
		APIKeyRequired: true,
		Features:       ProviderFeatures{Tools: true},
		DefaultModel:   "test-model",
	}
	err := registry.RegisterProvider(config)
	require.NoError(t, err)

	// Should not be available without API key
	assert.False(t, registry.IsProviderAvailable("test-with-key"))

	// Set API key and check again
	os.Setenv("TEST_WITH_KEY_API_KEY", "test-key")
	defer os.Unsetenv("TEST_WITH_KEY_API_KEY")
	assert.True(t, registry.IsProviderAvailable("test-with-key"))

	// Register a provider that doesn't require an API key
	configNoKey := &ProviderConfig{
		Type:           "test-no-key",
		DisplayName:    "Test No Key",
		BaseURL:        "http://localhost:8080/v1/chat/completions",
		APIKeyRequired: false,
		Features:       ProviderFeatures{Tools: true},
		DefaultModel:   "test-model",
	}
	err = registry.RegisterProvider(configNoKey)
	require.NoError(t, err)

	// Should be available without API key
	assert.True(t, registry.IsProviderAvailable("test-no-key"))
}

func TestProviderRegistry_LoadFromFile(t *testing.T) {
	registry := NewProviderRegistry()

	// Create a temporary YAML file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-provider.yaml")

	yamlContent := `
type: yaml-test-provider
display_name: "YAML Test Provider"
base_url: "https://api.yamltest.com/v1/chat/completions"
api_key_env: "YAML_TEST_API_KEY"
api_key_required: true
features:
  tools: true
  streaming: true
  vision: false
default_model: "yaml-test-model"
default_timeout: "3m"
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Load the configuration
	err = registry.LoadFromFile(configFile)
	require.NoError(t, err)

	// Verify the configuration was loaded
	config, err := registry.GetProviderConfig("yaml-test-provider")
	require.NoError(t, err)
	assert.Equal(t, "YAML Test Provider", config.DisplayName)
	assert.Equal(t, "https://api.yamltest.com/v1/chat/completions", config.BaseURL)
	assert.Equal(t, "YAML_TEST_API_KEY", config.APIKeyEnvVar)
	assert.True(t, config.APIKeyRequired)
	assert.True(t, config.Features.Tools)
	assert.True(t, config.Features.Streaming)
	assert.False(t, config.Features.Vision)
	assert.Equal(t, "yaml-test-model", config.DefaultModel)
	assert.Equal(t, 3*time.Minute, config.DefaultTimeout)
}

func TestProviderRegistry_LoadDefaultConfigurations(t *testing.T) {
	registry := NewProviderRegistry()

	err := registry.LoadDefaultConfigurations()
	require.NoError(t, err)

	// Verify default providers are loaded
	expectedProviders := []ClientType{
		OpenAIClientType,
		DeepInfraClientType,
		OpenRouterClientType,
		OllamaLocalClientType,
		OllamaTurboClientType,
	}

	for _, providerType := range expectedProviders {
		config, err := registry.GetProviderConfig(providerType)
		require.NoError(t, err, "Provider %s should be loaded", providerType)
		assert.NotEmpty(t, config.DisplayName)
		assert.NotEmpty(t, config.BaseURL)
	}
}

func TestProviderRegistry_CalculateCost(t *testing.T) {
	registry := NewProviderRegistry()

	// Register a provider with cost configuration
	config := &ProviderConfig{
		Type:        "cost-test",
		DisplayName: "Cost Test Provider",
		BaseURL:     "https://api.costtest.com/v1/chat/completions",
		CostConfig: &CostConfig{
			Type: "tiered",
			Tiers: []CostTier{
				{ModelPattern: "expensive-model", InputPer1M: 10.0, OutputPer1M: 20.0},
				{ModelPattern: "*", InputPer1M: 1.0, OutputPer1M: 2.0},
			},
		},
	}
	err := registry.RegisterProvider(config)
	require.NoError(t, err)

	// Test cost calculation for expensive model
	cost := registry.CalculateCost("cost-test", 1000, 500, "expensive-model")
	expectedCost := (1000 * 10.0 / 1000000) + (500 * 20.0 / 1000000)
	assert.InDelta(t, expectedCost, cost, 0.0001)

	// Test cost calculation for default pattern
	cost = registry.CalculateCost("cost-test", 1000, 500, "some-other-model")
	expectedCost = (1000 * 1.0 / 1000000) + (500 * 2.0 / 1000000)
	assert.InDelta(t, expectedCost, cost, 0.0001)

	// Test cost calculation for unknown provider (should use fallback)
	cost = registry.CalculateCost("unknown-provider", 1000, 500, "any-model")
	expectedCost = (1000 * 0.001 / 1000) + (500 * 0.002 / 1000)
	assert.InDelta(t, expectedCost, cost, 0.0001)
}

func TestProviderRegistry_ValidationErrors(t *testing.T) {
	registry := NewProviderRegistry()

	tests := []struct {
		name   string
		config *ProviderConfig
		errMsg string
	}{
		{
			name:   "nil config",
			config: nil,
			errMsg: "provider config cannot be nil",
		},
		{
			name: "empty type",
			config: &ProviderConfig{
				DisplayName: "Test",
				BaseURL:     "https://api.test.com",
			},
			errMsg: "provider type is required",
		},
		{
			name: "empty display name",
			config: &ProviderConfig{
				Type:    "test",
				BaseURL: "https://api.test.com",
			},
			errMsg: "display name is required",
		},
		{
			name: "empty base URL",
			config: &ProviderConfig{
				Type:        "test",
				DisplayName: "Test",
			},
			errMsg: "base URL is required",
		},
		{
			name: "missing API key env var when required",
			config: &ProviderConfig{
				Type:           "test",
				DisplayName:    "Test",
				BaseURL:        "https://api.test.com",
				APIKeyRequired: true,
			},
			errMsg: "API key environment variable is required when API key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.RegisterProvider(tt.config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestProviderRegistry_ListProviders(t *testing.T) {
	registry := NewProviderRegistry()

	// Register some test providers
	configs := []*ProviderConfig{
		{Type: "provider1", DisplayName: "Provider 1", BaseURL: "https://api1.com"},
		{Type: "provider2", DisplayName: "Provider 2", BaseURL: "https://api2.com"},
		{Type: "provider3", DisplayName: "Provider 3", BaseURL: "https://api3.com"},
	}

	for _, config := range configs {
		err := registry.RegisterProvider(config)
		require.NoError(t, err)
	}

	providers := registry.ListProviders()
	assert.Len(t, providers, 3)
	assert.Contains(t, providers, ClientType("provider1"))
	assert.Contains(t, providers, ClientType("provider2"))
	assert.Contains(t, providers, ClientType("provider3"))
}
