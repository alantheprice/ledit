# Ledit - AI-Powered Code Editor & Orchestrator

`ledit` is a command-line tool that leverages Large Language Models (LLMs) to automate and assist in software development tasks. It can understand your entire workspace, generate code, orchestrate complex features, and ground its responses with live web search results.

## Overview

`ledit` is more than just a code generator. It's a development partner that can:

-   **Implement complex features**: Take a high-level prompt and break it down into a step-by-step plan of file changes.
-   **Intelligently use context**: Automatically determines which files in your workspace are relevant to a task, including either their full content or just a summary to optimize the context provided to the LLM.
-   **Self-correct**: When orchestrating changes, it can validate its own work, and if an error occurs, it retries with an understanding of the failure.
-   **Stay up-to-date**: Use real-time web search to ground its knowledge and answer questions about new technologies or libraries.
-   **Work with your tools**: Integrates with Git for automatic commits and respects your `.gitignore` files.

## Features

-   **Feature Orchestration**: Decomposes high-level feature requests into a detailed, executable plan.
-   **Smart Workspace Context**: Automatically builds and maintains an index of your workspace. An LLM selects the most relevant files to include as context for any given task.
-   **Leaked Credentials Check**: Automatically performs a check for potential credentials in files before sending the file to the workspace analysis process. This reduces the chance that sensitive credentials are sent to an llm via an api.
-   **Search Grounding**: Augments prompts with fresh information from the web using the `#SG "query"` directive.
-   **Interactive and Automated Modes**: Confirm each change manually, or run in a fully automated mode with `--skip-prompt`.
-   **Multi-Provider LLM Support**: Works with OpenAI, Groq, Gemini, Ollama, and more.
-   **Change Tracking**: Keeps a local history of all changes made.
-   **Git Integration**: Can automatically commit applied changes with generated messages.
-   **Self-Correction Loop**: In orchestration mode, it attempts to fix its own errors by analyzing validation failures and retrying.

## Installation

To get started with `ledit`, the preferred method is to install it via `go install`.

### Prerequisites

-   Go 1.20+
-   Git (for version control integration)

### From Source (Preferred Method)

Make sure you have Go installed and configured.

```bash
go install github.com/alantheprice/ledit@latest # Replace with the actual repository path
```

This will install the `ledit` executable in your `GOPATH/bin` directory (e.g., `~/go/bin` on Linux/macOS).

**Note on PATH:** If `ledit` is not found after installation, you may need to add your `GOPATH/bin` directory to your system's PATH environment variable. For example, you can add the following line to your shell's configuration file (e.g., `.bashrc`, `.zshrc`, or `.profile`):

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```
After adding this, restart your terminal or run `source ~/.bashrc` (or your respective config file) for the changes to take effect.

## Getting Started

Once installed, you can initialize `ledit` in your project directory and start using its powerful features.

```bash
# Initialize ledit in your project
ledit init

# Ask ledit to create a simple Python script
ledit code "Create a python script that prints 'Hello, World!'"

# Ask ledit a question about your workspace
ledit question "What does the main function in main.go do?"

# Generate a conventional commit message for staged changes
ledit commit

# View the history of changes made by ledit
ledit log

# Attempt to fix a problem in your code based on an error message
ledit fix "Error: undefined variable 'user_id' in main.go"

# Ignore a directory from workspace analysis
ledit ignore "dist/"

# For more detailed examples and a comprehensive guide, see the documentation:
# [Getting Started Guide](docs/GETTING_STARTED.md)
```

## Configuration

`ledit` is configured via a `config.json` file. It looks for this file first in `./.ledit/config.json` and then in `~/.ledit/config.json`. A default configuration is created on first run.

**API Keys** for services like OpenAI, Groq, JinaAI, etc., are stored securely in `~/.ledit/api_keys.json`. If a key is not found, `ledit` will prompt you to enter it.

### `config.json` settings

```json
{
  "EditingModel": "lambda-ai:deepseek-v3-0324",
  "SummaryModel": "lambda-ai:hermes3-8b",
  "OrchestrationModel": "lambda-ai:qwen25-coder-32b-instruct",
  "WorkspaceModel": "lambda-ai:qwen25-coder-32b-instruct",
  "OllamaServerURL": "http://localhost:11434",
  "TrackWithGit": false,
  "SkipPrompt": false,
  "EnableSecurityChecks": true,
  "UseGeminiSearchGrounding": false,
  "OrchestrationMaxAttempts": 6,
  "CodeStyle": {
    "FunctionSize": "Aim for smaller, single-purpose functions (under 50 lines).",
    "FileSize": "Prefer smaller files, breaking down large components into multiple files (under 500 lines).",
    "NamingConventions": "Use clear, descriptive names for variables, functions, and types. Follow Go conventions (camelCase for local, PascalCase for exported).",
    "ErrorHandling": "Handle errors explicitly, returning errors as the last return value. Avoid panics for recoverable errors.",
    "TestingApproach": "Write unit tests for all critical logic. Aim for high test coverage.",
    "Modularity": "Design components to be loosely coupled and highly cohesive.",
    "Readability": "Prioritize code readability and maintainability. Use comments where necessary to explain complex logic."
  }
}
```

-   **`EditingModel`**: The primary model for generating and modifying code.
-   **`SummaryModel`**: The model used for summarizing files for the workspace index.
-   **`OrchestrationModel`**: The model used to generate the high-level feature plan.
-   **`WorkspaceModel`**: The model used to select relevant files for context.
-   **`OllamaServerURL`**: The URL for your local Ollama server, if used.
-   **`TrackWithGit`**: If `true`, automatically commit changes to Git.
-   **`SkipPrompt`**: If `true`, bypasses all user confirmation prompts.
-   **`EnableSecurityChecks`**: If `true`, enables checks for potential credentials before sending files to LLM.
-   **`UseGeminiSearchGrounding`**: If `true`, enables experimental Gemini-powered search grounding.
-   **`OrchestrationMaxAttempts`**: The maximum number of retries for a failed orchestration step.
-   **`CodeStyle`**: Defines preferred code style guidelines for the project, influencing LLM generation.

## Usage and Commands

### Workspace Initialization

The first time you run `ledit` in a project, it will create a `.ledit` directory. This directory contains:

-   `workspace.json`: An index of your project's files, including summaries and exports, used for context selection.
-   `leditignore`: A file for patterns to ignore, in addition to `.gitignore`.
-   `config.json`: (Optional) Project-specific configuration.
-   `setup.sh` - Generated setup script
-   `validate.sh` - Generated validation script

The workspace index is automatically updated whenever you run a command, ensuring the context is always fresh.

### Basic Editing and Interaction

`ledit` provides several commands for direct code manipulation and interaction.

```bash
# Edit an existing file
ledit code "Add a function to reverse a string" -f path/to/your/file.go

# Create a new file (omit the -f flag)
ledit code "Create a python script that prints 'Hello, World!'"

# Start an interactive chat about your workspace
ledit question

# Generate a conventional commit message for staged changes
ledit commit

# View the history of changes made by ledit
ledit log

# Attempt to fix a problem in your code based on an error message
ledit fix "Error: undefined variable 'user_id' in main.go"

# Add a pattern to .ledit/leditignore
ledit ignore "temp_files/"
```

### Orchestration

**NOTE**: Currently the orchestration process should be considered in an alpha state and not ready for production use

For larger tasks, use the `process` command. This is the most powerful feature of `ledit`.

```bash
ledit process "Implement a REST API for a user model with create, read, and delete endpoints. Use Gin framework."
```

**The Orchestration Process:**

1.  **Analysis**: `ledit` analyzes your prompt and the current workspace.
2.  **Planning**: An LLM generates a JSON plan of all the required changes (new files, modifications to existing files).
3.  **Review**: The plan is presented to you for approval.
4.  **Execution**: `ledit` executes each step of the plan one by one.
    -   It generates the code for the change.
    -   It applies the change.
    -   For testable files, it follows a TDD-like approach.
    -   It may run validation or setup scripts.
5.  **Validation & Self-Correction**: If a step results in an error (e.g., a test fails), `ledit` will:
    -   Analyze the error message.
    -   Optionally perform a web search for solutions.
    -   Re-prompt the LLM with the error context to generate a fix.
    -   Retry the step up to 4 times before halting.

### Ignoring Files

To explicitly ignore files or directories from the workspace index, use the `ignore` command. By default, ledit will ignore based on a .gitignore file, if it exists and falls back to defaults otherwise.

```bash
ledit ignore "dist/"
ledit ignore "*.log"
```

This adds the pattern to the `.ledit/leditignore` file.

## Advanced Concepts: Prompting with Context

You can control the context provided to the LLM using special `#` directives in your prompts.

### `#<filepath>` - Include a File

To manually include the full content of a file in the context:

```bash
ledit code "Refactor the main function to use the helper functions from #./helpers.go" -f main.go
```

### `#WORKSPACE` / `#WS` - Smart Context

This is the recommended way to provide context for most tasks.

```bash
ledit code "Add user authentication using JWT. #WORKSPACE"
```

When `#WORKSPACE` is used, `ledit` performs a multi-step process:

1.  It provides an LLM with the summaries of all files in your project.
2.  The LLM identifies which files are relevant to your prompt.
3.  It decides whether to include the **full content** of a file or just its **summary**.
4.  This curated context is then used to perform the main task (e.g., orchestration or code generation).

This prevents overflowing the LLM's context window and focuses its attention on only the relevant parts of your codebase.

### `#SG \"query\"` - Search Grounding

To provide the LLM with up-to-date information from the web, use Search Grounding.

```bash
ledit code "Add the latest version of 'react-query' and its dependencies. #SG \"latest react-query version npm\"" -f package.json
```

The Search Grounding process:

1.  `ledit` performs a web search using the Jina AI API with your query.
2.  An LLM reviews the search results and selects the 1-3 most relevant URLs.
3.  `ledit` fetches the content from these URLs.
4.  Embeddings are used to extract the most relevant snippets of text from the web pages.
5.  This extracted text is prepended to your prompt, giving the main LLM the external context it needs.

This is particularly useful for tasks involving new libraries, APIs, or resolving complex errors. During orchestration retries, `ledit` automatically uses this feature to find solutions to validation errors.

## Supported LLM Providers

`ledit` supports a few OpenAI-compatible API's, including many open-source and self-hosted models. While we don't support every possible provider, we aim to cover a range of popular and open-compatible options. Additional providers can be added via a pull request. To specify a provider and model, use the format `<provider>:<model_name>` in your config or with the `-m` flag.

Current supported providers include:

-   **`openai`**: For OpenAI's models (e.g., `openai:gpt-4-turbo`)
-   **`groq`**: For Groq's fast inference models (e.g., `groq:llama3-70b-8192`)
-   **`gemini`**: For Google Gemini models (e.g., `gemini:gemini-pro`)
-   **`ollama`**: For local Ollama models (e.g., `ollama:llama3`)
-   **`lambda-ai`**: For Lambda AI models (e.g., `lambda-ai:deepseek-v3-0324`)
-   **`cerebras`**: For Cerebras models
-   **`deepseek`**: For Deepseek models (e.g., `deepseek:deepseek-coder`)

## Documentation

Explore the full capabilities of `ledit` with our detailed documentation:

-   [Getting Started](docs/GETTING_STARTED.md)
-   [Cheatsheet](docs/CHEATSHEET.md)
-   [Examples](docs/EXAMPLES.md)
-   [Tips and Tricks](docs/TIPS_AND_TRICKS.md)

## Contributing

We welcome contributions to `ledit`! Please see our [CONTRIBUTING.md](CONTRIBUTING.md) guide for more details on how to get involved.

## File Structure

### Key files maintained by ledit:

-   `.ledit/workspace.json` - Workspace analysis data
-   `.ledit/requirements.json` - Orchestration plans
-   `.ledit/config.json` - Project configuration
-   `.ledit/setup.sh` - Generated setup script
-   `.ledit/validate.sh` - Generated validation script

## License

This project is licensed under the [MIT License](LICENSE).

## Support and Community

If you encounter any issues or have questions, please open an issue on our [GitHub repository](https://github.com/alantheprice/ledit/issues).