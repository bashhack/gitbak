package git

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/bashhack/gitbak/internal/errors"
)

// CommandExecutor defines an interface for executing commands
type CommandExecutor interface {
	// Execute runs a command and returns its exit code
	Execute(ctx context.Context, cmd *exec.Cmd) error

	// ExecuteWithOutput runs a command and returns its output and exit code
	ExecuteWithOutput(ctx context.Context, cmd *exec.Cmd) (string, error)

	// ExecuteWithContext runs a command with context and returns its exit code
	ExecuteWithContext(ctx context.Context, name string, args ...string) error

	// ExecuteWithContextAndOutput runs a command with context and returns its output and exit code
	ExecuteWithContextAndOutput(ctx context.Context, name string, args ...string) (string, error)
}

// ExecExecutor is the default implementation of CommandExecutor
// that delegates to the os/exec package
type ExecExecutor struct{}

// NewExecExecutor creates a new ExecExecutor
func NewExecExecutor() *ExecExecutor {
	return &ExecExecutor{}
}

// handleExecutionError creates a standardized GitError from a command execution error
func (e *ExecExecutor) handleExecutionError(operation string, args []string, err error, stderr string) error {
	wrappedErr := errors.Wrap(err, "git operation failed")
	return errors.NewGitError(operation, args, wrappedErr, stderr)
}

// extractCommandInfo extracts the operation name and arguments from a command
func (e *ExecExecutor) extractCommandInfo(cmd *exec.Cmd) (string, []string) {
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

	return operation, args
}

// prepareCommandWithContext creates a new command with context and copies properties from the original command
func (e *ExecExecutor) prepareCommandWithContext(ctx context.Context, cmd *exec.Cmd) *exec.Cmd {
	cmdWithContext := exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	cmdWithContext.Stdin = cmd.Stdin
	cmdWithContext.Env = cmd.Env
	cmdWithContext.Dir = cmd.Dir
	return cmdWithContext
}

// Execute implements CommandExecutor.Execute
func (e *ExecExecutor) Execute(ctx context.Context, cmd *exec.Cmd) error {
	cmdWithContext := e.prepareCommandWithContext(ctx, cmd)
	cmdWithContext.Stdout = cmd.Stdout
	cmdWithContext.Stderr = cmd.Stderr

	err := cmdWithContext.Run()
	if err != nil {
		operation, args := e.extractCommandInfo(cmdWithContext)
		return e.handleExecutionError(operation, args, err, "")
	}
	return nil
}

// ExecuteWithOutput implements CommandExecutor.ExecuteWithOutput
func (e *ExecExecutor) ExecuteWithOutput(ctx context.Context, cmd *exec.Cmd) (string, error) {
	cmdWithContext := e.prepareCommandWithContext(ctx, cmd)

	// Copy existing stdout/stderr if set, otherwise create new buffers
	var stdout, stderr bytes.Buffer
	if cmd.Stdout != nil {
		cmdWithContext.Stdout = cmd.Stdout
	} else {
		cmdWithContext.Stdout = &stdout
	}

	if cmd.Stderr != nil {
		cmdWithContext.Stderr = cmd.Stderr
	} else {
		cmdWithContext.Stderr = &stderr
	}

	err := cmdWithContext.Run()
	if err != nil {
		operation, args := e.extractCommandInfo(cmdWithContext)
		return "", e.handleExecutionError(operation, args, err, stderr.String())
	}

	return stdout.String(), nil
}

// ExecuteWithContext implements CommandExecutor.ExecuteWithContext
func (e *ExecExecutor) ExecuteWithContext(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)

	err := cmd.Run()
	if err != nil {
		return e.handleExecutionError(name, args, err, "")
	}
	return nil
}

// ExecuteWithContextAndOutput implements CommandExecutor.ExecuteWithContextAndOutput
func (e *ExecExecutor) ExecuteWithContextAndOutput(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", e.handleExecutionError(name, args, err, stderr.String())
	}

	return stdout.String(), nil
}
