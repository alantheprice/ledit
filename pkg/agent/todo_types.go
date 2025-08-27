package agent

// ExecutionType defines the method for executing a todo
type ExecutionType int

const (
	ExecutionTypeAnalysis     ExecutionType = iota // Analysis-only, no code changes
	ExecutionTypeDirectEdit                        // Simple file edits
	ExecutionTypeCodeCommand                       // Complex code generation
	ExecutionTypeShellCommand                      // Shell commands for filesystem operations
	ExecutionTypeContinuation                      // Continuation to next phase of complex workflow
)

// String returns the string representation of ExecutionType
func (et ExecutionType) String() string {
	switch et {
	case ExecutionTypeAnalysis:
		return "analysis"
	case ExecutionTypeDirectEdit:
		return "direct_edit"
	case ExecutionTypeCodeCommand:
		return "code_command"
	case ExecutionTypeShellCommand:
		return "shell_command"
	case ExecutionTypeContinuation:
		return "continuation"
	default:
		return "unknown"
	}
}