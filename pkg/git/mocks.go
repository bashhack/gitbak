package git

import (
	"context"
	"fmt"
	gitbakErrors "github.com/bashhack/gitbak/pkg/errors"
	"os/exec"
)

// MockInteractor is a mock implementation of the UserInteractor interface
// for testing user interaction scenarios.
type MockInteractor struct {
	// Response to return from PromptYesNo
	PromptYesNoResponse bool

	// Track the number of times PromptYesNo was called
	PromptYesNoCalled bool

	// Store the last prompt that was passed to PromptYesNo
	LastPrompt string
}

// PromptYesNo implements the UserInteractor interface
func (m *MockInteractor) PromptYesNo(prompt string) bool {
	m.PromptYesNoCalled = true
	m.LastPrompt = prompt
	return m.PromptYesNoResponse
}

// NewMockInteractor creates a new MockInteractor with default values
func NewMockInteractor(response bool) *MockInteractor {
	return &MockInteractor{
		PromptYesNoResponse: response,
	}
}

// MockCommandExecutor is a mock of the CommandExecutor interface
// that can be configured for different test scenarios.
type MockCommandExecutor struct {
	// Basic tracking
	ExitCode  int
	Output    string
	LastCmd   *exec.Cmd
	Commands  []*exec.Cmd
	CallCount int

	// Function hooks for customizing behavior
	ExecuteFn           func(ctx context.Context, cmd *exec.Cmd) error
	ExecuteWithOutputFn func(ctx context.Context, cmd *exec.Cmd) (string, error)

	// Git command tracking (for uncommitted changes tests)
	AddCalled    bool
	CommitCalled bool
	AddError     bool
	CommitError  bool

	// Error simulation and signaling (for retry tests)
	ShouldFailCount     int
	PermanentFailAfter  int
	FailWithErr         error
	CurrentErr          error
	ErrorVariant        int
	ShouldResetErrorMsg bool

	// Advanced retry test behavior properties
	// These are used directly by the retry tests
	UseAdvancedRetryBehavior bool

	// Channels for signaling (for retry tests)
	NextFailCall    chan struct{}
	NextSuccessCall chan struct{}
}

// Execute implements the CommandExecutor interface
func (m *MockCommandExecutor) Execute(ctx context.Context, cmd *exec.Cmd) error {
	m.CallCount++
	m.LastCmd = cmd
	m.Commands = append(m.Commands, cmd)

	// Signal before returning for retry tests
	defer func() {
		if m.CurrentErr != nil && m.NextFailCall != nil {
			select {
			case m.NextFailCall <- struct{}{}:
			default:
			}
		} else if m.NextSuccessCall != nil {
			select {
			case m.NextSuccessCall <- struct{}{}:
			default:
			}
		}
	}()

	// If we're using the advanced retry behavior
	if m.UseAdvancedRetryBehavior {
		// Check for permanent failure mode
		if m.PermanentFailAfter > 0 && m.CallCount > m.PermanentFailAfter {
			m.CurrentErr = m.FailWithErr
			return m.CurrentErr
		}

		// Check for initial failures
		if m.CallCount <= m.ShouldFailCount {
			if m.ShouldResetErrorMsg && m.ErrorVariant%2 == 0 {
				// Create a new error with a varying message for retry tests
				m.CurrentErr = gitbakErrors.NewGitError("test", nil,
					gitbakErrors.Wrap(gitbakErrors.ErrGitOperationFailed,
						fmt.Sprintf("varying error message variant %d", m.ErrorVariant)), "")
				m.ErrorVariant++
			} else {
				m.CurrentErr = m.FailWithErr
			}
			return m.CurrentErr
		}

		// Reset current error since we're past the failure count
		m.CurrentErr = nil
		return nil
	}

	// Simple error behavior for non-advanced retry tests
	if m.CallCount <= m.ShouldFailCount {
		m.CurrentErr = m.FailWithErr
		return m.CurrentErr
	}

	// Reset current error
	m.CurrentErr = nil

	// Custom function takes precedence
	if m.ExecuteFn != nil {
		return m.ExecuteFn(ctx, cmd)
	}

	return nil
}

// ExecuteWithOutput implements the CommandExecutor interface
func (m *MockCommandExecutor) ExecuteWithOutput(ctx context.Context, cmd *exec.Cmd) (string, error) {
	// Use the Execute method to handle error logic consistently
	err := m.Execute(ctx, cmd)
	if err != nil {
		return "", err
	}

	if m.ExecuteWithOutputFn != nil {
		return m.ExecuteWithOutputFn(ctx, cmd)
	}

	return m.Output, nil
}

// ExecuteWithContext implements the CommandExecutor interface
func (m *MockCommandExecutor) ExecuteWithContext(ctx context.Context, name string, args ...string) error {
	// For uncommitted changes test: check for git add/commit operations
	if name == "git" {
		// Handle the args differently based on how they are passed
		if len(args) > 0 {
			// For git operations via runGitCommand (which adds -C repoPath)
			if args[0] == "-C" && len(args) > 2 {
				switch args[2] {
				case "add":
					m.AddCalled = true
					if m.AddError {
						return fmt.Errorf("mock add error")
					}
				case "commit":
					m.CommitCalled = true
					if m.CommitError {
						return fmt.Errorf("mock commit error")
					}
				}
			} else {
				// Direct git operations without -C
				switch args[0] {
				case "add":
					m.AddCalled = true
					if m.AddError {
						return fmt.Errorf("mock add error")
					}
				case "commit":
					m.CommitCalled = true
					if m.CommitError {
						return fmt.Errorf("mock commit error")
					}
				}
			}
		}
	}

	cmd := exec.CommandContext(ctx, name, args...)
	return m.Execute(ctx, cmd)
}

// ExecuteWithContextAndOutput implements the CommandExecutor interface
func (m *MockCommandExecutor) ExecuteWithContextAndOutput(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return m.ExecuteWithOutput(ctx, cmd)
}

// NewMockCommandExecutor creates a new mock executor with default values
func NewMockCommandExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{
		// Basic tracking
		ExitCode:  0,
		Output:    "",
		Commands:  make([]*exec.Cmd, 0),
		CallCount: 0,

		// Git command tracking (for uncommitted changes tests)
		AddCalled:    false,
		CommitCalled: false,
		AddError:     false,
		CommitError:  false,

		// Error simulation (for retry tests) - all disabled by default
		ShouldFailCount:    0,
		PermanentFailAfter: 0,
		ErrorVariant:       0,
	}
}

// NewMockRetryExecutor creates a mock executor configured for retry tests
func NewMockRetryExecutor(failCount int, failWithErr error) *MockCommandExecutor {
	mock := NewMockCommandExecutor()
	mock.ShouldFailCount = failCount
	mock.FailWithErr = failWithErr
	mock.NextFailCall = make(chan struct{}, 1)
	mock.NextSuccessCall = make(chan struct{}, 1)
	return mock
}

// NewAdvancedMockRetryExecutor creates a mock executor with advanced retry behavior
// for complex testing scenarios like message variation and permanent failures
func NewAdvancedMockRetryExecutor(failCount int, failWithErr error) *MockCommandExecutor {
	mock := NewMockRetryExecutor(failCount, failWithErr)
	mock.UseAdvancedRetryBehavior = true
	return mock
}

// NewMockUncommittedChangesExecutor creates a mock for uncommitted changes tests
func NewMockUncommittedChangesExecutor(addError, commitError bool) *MockCommandExecutor {
	mock := NewMockCommandExecutor()
	mock.AddError = addError
	mock.CommitError = commitError
	return mock
}
