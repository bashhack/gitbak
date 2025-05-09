package git

import (
	"context"
	"os/exec"
)

// MockCommandExecutor is a simple mock of the CommandExecutor interface
// that doesn't actually execute anything but just records calls.
type MockCommandExecutor struct {
	ExitCode            int
	Output              string
	LastCmd             *exec.Cmd
	Commands            []*exec.Cmd
	ExecuteFn           func(ctx context.Context, cmd *exec.Cmd) error
	ExecuteWithOutputFn func(ctx context.Context, cmd *exec.Cmd) (string, error)
}

// Execute implements the CommandExecutor interface
func (m *MockCommandExecutor) Execute(ctx context.Context, cmd *exec.Cmd) error {
	m.LastCmd = cmd
	m.Commands = append(m.Commands, cmd)

	if m.ExecuteFn != nil {
		return m.ExecuteFn(ctx, cmd)
	}

	return nil
}

// ExecuteWithOutput implements the CommandExecutor interface
func (m *MockCommandExecutor) ExecuteWithOutput(ctx context.Context, cmd *exec.Cmd) (string, error) {
	m.LastCmd = cmd
	m.Commands = append(m.Commands, cmd)

	if m.ExecuteWithOutputFn != nil {
		return m.ExecuteWithOutputFn(ctx, cmd)
	}

	return m.Output, nil
}

// ExecuteWithContext implements the CommandExecutor interface
func (m *MockCommandExecutor) ExecuteWithContext(ctx context.Context, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return m.Execute(ctx, cmd)
}

// ExecuteWithContextAndOutput implements the CommandExecutor interface
func (m *MockCommandExecutor) ExecuteWithContextAndOutput(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	return m.ExecuteWithOutput(ctx, cmd)
}

// NewMockCommandExecutor creates a new mock executor
func NewMockCommandExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{
		ExitCode: 0,
		Output:   "",
		Commands: make([]*exec.Cmd, 0),
	}
}
