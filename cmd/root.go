package cmd

import (
	"fmt"

	"github.com/alantheprice/ledit/pkg/config"
	"github.com/alantheprice/ledit/pkg/utils"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ledit",
	Short: "ledit is an AI-powered code editor CLI",
	Long:  `A command-line tool to process code with LLM instructions, track changes, and manage configurations.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger := utils.GetLogger(skipPrompt) // Get the logger instance

		// If no command is specified, show help
		if len(args) == 0 {
			cfg, err := config.LoadOrInitConfig(skipPrompt)
			if err != nil {
				logger.LogUserInteraction("Configuration not found. Please run 'ledit init'.")
				cmd.Help()
				return
			}
			logger.LogUserInteraction("Configuration loaded. Defaulting to 'code' command.")
			logger.LogUserInteraction("Usage: ledit code \"your instructions\" [-f <file-name>] [-m <model>] [--skip-prompt]")
			logger.LogUserInteraction(fmt.Sprintf("Editing Model: %s", cfg.EditingModel))
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(codeCmd)
	rootCmd.AddCommand(logCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(questionCmd)
	rootCmd.AddCommand(processCmd)
	rootCmd.AddCommand(fixCmd)
	rootCmd.AddCommand(commitCmd)
}
