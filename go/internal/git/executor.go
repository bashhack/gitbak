package git

import (
	"bytes"
	"os/exec"

	"github.com/bashhack/gitbak/internal/errors"
)

// CommandExecutor defines an interface for executing commands
type CommandExecutor interface {
	// Execute runs a command and returns its exit code
	Execute(cmd *exec.Cmd) error

	// ExecuteWithOutput runs a command and returns its output and exit code
	ExecuteWithOutput(cmd *exec.Cmd) (string, error)
}

// ExecExecutor is the default implementation of CommandExecutor
// that delegates to the os/exec package
type ExecExecutor struct{}

// Execute implements CommandExecutor.Execute
func (e *ExecExecutor) Execute(cmd *exec.Cmd) error {
	err := cmd.Run()
	if err != nil {
		// Extract the executable
		operation := ""
		if len(cmd.Args) > 0 {
			operation = cmd.Args[0]
		}

		// Extract the arguments
		var args []string
		if len(cmd.Args) > 1 {
			args = cmd.Args[1:]
		}

		// Create a GitError that wraps the ErrGitOperationFailed sentinel error
		wrappedErr := errors.Wrap(errors.ErrGitOperationFailed, err.Error())
		return errors.NewGitError(operation, args, wrappedErr, "")
	}
	return nil
}

// ExecuteWithOutput implements CommandExecutor.ExecuteWithOutput
func (e *ExecExecutor) ExecuteWithOutput(cmd *exec.Cmd) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Extract the executable
		operation := ""
		if len(cmd.Args) > 0 {
			operation = cmd.Args[0]
		}

		// Extract the arguments
		var args []string
		if len(cmd.Args) > 1 {
			args = cmd.Args[1:]
		}

		// Create a GitError that wraps the ErrGitOperationFailed sentinel error
		wrappedErr := errors.Wrap(errors.ErrGitOperationFailed, err.Error())
		return "", errors.NewGitError(operation, args, wrappedErr, stderr.String())
	}

	return stdout.String(), nil
}

// NewExecExecutor creates a new ExecExecutor
func NewExecExecutor() *ExecExecutor {
	return &ExecExecutor{}
}
