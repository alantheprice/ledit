package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/alantheprice/ledit/pkg/agent"
	"github.com/alantheprice/ledit/pkg/agent_api"
	"github.com/alantheprice/ledit/pkg/agent_config"
)

// ProviderCommand implements the /provider slash command
type ProviderCommand struct{}

// Name returns the command name
func (p *ProviderCommand) Name() string {
	return "provider"
}

// Description returns the command description
func (p *ProviderCommand) Description() string {
	return "Show current provider status and switch providers"
}

// Execute runs the provider command
func (p *ProviderCommand) Execute(args []string, chatAgent *agent.Agent) error {
	configManager := chatAgent.GetConfigManager()

	// If no arguments, show current status
	if len(args) == 0 {
		return p.showProviderStatus(configManager, chatAgent)
	}

	// Handle subcommands
	switch args[0] {
	case "list":
		return p.listProviders(configManager)
	case "select":
		return p.selectProvider(configManager, chatAgent)
	case "status":
		return p.showProviderStatus(configManager, chatAgent)
	default:
		// Try to set provider directly by name
		return p.setProvider(args[0], configManager, chatAgent)
	}
}

// showProviderStatus displays current provider information
func (p *ProviderCommand) showProviderStatus(configManager *config.Manager, chatAgent *agent.Agent) error {
	fmt.Println("\n🔧 Provider Status:")
	fmt.Println("===================")

	// Show current active provider
	currentProvider := chatAgent.GetProviderType()
	currentModel := chatAgent.GetModel()
	fmt.Printf("✅ **Active Provider**: %s\n", api.GetProviderName(currentProvider))
	fmt.Printf("🤖 **Current Model**: %s\n", currentModel)
	fmt.Println()

	// Show status of all providers
	status := configManager.GetProviderStatus()
	fmt.Println("📋 All Providers:")
	fmt.Println("------------------")

	for providerType, info := range status {
		icon := "❌"
		if info.Available {
			icon = "✅"
		}

		lastUsedIcon := ""
		if info.IsLastUsed {
			lastUsedIcon = " 🌟"
		}

		fmt.Printf("%s **%s**%s\n", icon, info.Name, lastUsedIcon)
		fmt.Printf("   Model: %s\n", info.CurrentModel)

		if info.EnvVar != "" {
			envStatus := "❌ Not set"
			if info.Available && providerType != api.OllamaClientType {
				envStatus = "✅ Available"
			}
			fmt.Printf("   API Key (%s): %s\n", info.EnvVar, envStatus)
		} else {
			if providerType == api.OllamaClientType {
				if info.Available {
					fmt.Printf("   Status: ✅ Running\n")
				} else {
					fmt.Printf("   Status: ❌ Not running\n")
				}
			}
		}
		fmt.Println()
	}

	fmt.Println("Usage:")
	fmt.Println("  /provider                    - Show this status")
	fmt.Println("  /provider list              - List available providers only")
	fmt.Println("  /provider select            - Interactive provider selection")
	fmt.Println("  /provider <provider_name>   - Switch to specific provider")

	return nil
}

// listProviders shows only available providers
func (p *ProviderCommand) listProviders(configManager *config.Manager) error {
	available := configManager.ListAvailableProviders()

	if len(available) == 0 {
		fmt.Println("❌ No providers are currently available.")
		fmt.Println("Please set up an API key or start Ollama.")
		return nil
	}

	fmt.Println("\n✅ Available Providers:")
	fmt.Println("=======================")

	for i, provider := range available {
		name := api.GetProviderName(provider)
		model := configManager.GetModelForProvider(provider)
		fmt.Printf("%d. **%s** - %s\n", i+1, name, model)
	}

	return nil
}

// selectProvider allows interactive provider selection
func (p *ProviderCommand) selectProvider(configManager *config.Manager, chatAgent *agent.Agent) error {
	available := configManager.ListAvailableProviders()

	if len(available) == 0 {
		fmt.Println("❌ No providers are currently available.")
		fmt.Println("Please set up an API key or start Ollama.")
		return nil
	}

	fmt.Println("\n🎯 Select a Provider:")
	fmt.Println("=====================")

	for i, provider := range available {
		name := api.GetProviderName(provider)
		model := configManager.GetModelForProvider(provider)

		current := ""
		if provider == chatAgent.GetProviderType() {
			current = " (current)"
		}

		fmt.Printf("%d. **%s**%s - %s\n", i+1, name, current, model)
	}

	// Get user selection
	fmt.Print("\nEnter provider number (1-" + strconv.Itoa(len(available)) + ") or 'cancel': ")

	// Temporarily disable Esc monitoring during user input
	chatAgent.DisableEscMonitoring()
	defer chatAgent.EnableEscMonitoring()

	var input string
	fmt.Scanln(&input)

	input = strings.TrimSpace(input)
	if input == "cancel" || input == "" {
		fmt.Println("Provider selection cancelled.")
		return nil
	}

	// Parse selection
	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(available) {
		return fmt.Errorf("invalid selection. Please enter a number between 1 and %d", len(available))
	}

	selectedProvider := available[selection-1]
	return p.switchToProvider(selectedProvider, configManager, chatAgent)
}

// setProvider sets a specific provider by name
func (p *ProviderCommand) setProvider(providerName string, configManager *config.Manager, chatAgent *agent.Agent) error {
	// Convert name to provider type
	provider, err := config.GetProviderFromConfigName(strings.ToLower(providerName))
	if err != nil {
		return fmt.Errorf("unknown provider '%s'. Available: deepinfra, ollama, cerebras, openrouter, groq, deepseek", providerName)
	}

	// Check if provider is available
	available := configManager.ListAvailableProviders()
	isAvailable := false
	for _, p := range available {
		if p == provider {
			isAvailable = true
			break
		}
	}

	if !isAvailable {
		return fmt.Errorf("provider %s is not currently available. Use '/provider list' to see available providers", api.GetProviderName(provider))
	}

	return p.switchToProvider(provider, configManager, chatAgent)
}

// switchToProvider performs the actual provider switch
func (p *ProviderCommand) switchToProvider(provider api.ClientType, configManager *config.Manager, chatAgent *agent.Agent) error {
	// Get the configured model for this provider
	model := configManager.GetModelForProvider(provider)

	fmt.Printf("🔄 Switching to %s with model %s...\n", api.GetProviderName(provider), model)

	// Clear model caches to ensure fresh model lists for the new provider
	api.ClearModelCaches()

	// Persist the provider selection to configuration
	err := configManager.SetProviderAndModel(provider, model)
	if err != nil {
		return fmt.Errorf("failed to persist provider selection: %w", err)
	}

	// Switch the agent to use the new provider and model immediately
	err = chatAgent.SetModel(model)
	if err != nil {
		return fmt.Errorf("failed to switch to provider %s: %w", api.GetProviderName(provider), err)
	}

	fmt.Printf("✅ Provider switched to: %s\n", api.GetProviderName(provider))
	fmt.Printf("🤖 Using model: %s\n", model)

	return nil
}
