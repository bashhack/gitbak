package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors that can be used with errors.Is() for error type checking
var (
	// ErrNotGitRepository indicates the target path is not a git repository
	ErrNotGitRepository = errors.New("not a git repository")

	// ErrLockAcquisitionFailure indicates a lock file could not be acquired
	ErrLockAcquisitionFailure = errors.New("failed to acquire lock")

	// ErrAlreadyRunning indicates another gitbak instance is running for this repo
	ErrAlreadyRunning = errors.New("another gitbak instance is already running for this repository")

	// ErrGitOperationFailed indicates a git command returned an error
	ErrGitOperationFailed = errors.New("git operation failed")

	// ErrInvalidConfiguration indicates an invalid or conflicting user configuration
	ErrInvalidConfiguration = errors.New("invalid configuration")
)

// New creates a new error with the given message.
// This is a convenience function that wraps errors.New.
func New(message string) error {
	return errors.New(message)
}

// Errorf creates a new formatted error.
// This is a convenience function that wraps fmt.Errorf.
func Errorf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

// Wrap wraps an error with a message for better context.
func Wrap(err error, message string) error {
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf wraps an error with a formatted message for better context.
func Wrapf(err error, format string, args ...interface{}) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

// Is reports whether target is in err's chain.
// This is a convenience function that wraps errors.Is.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target.
// This is a convenience function that wraps errors.As.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// GitError represents an error that occurred during a Git operation.
// It captures the command details, underlying error, and command output.
type GitError struct {
	Operation string
	Args      []string
	Err       error
	Output    string
}

// Error implements the error interface with a detailed, user-friendly error message.
func (e *GitError) Error() string {
	msg := fmt.Sprintf("git %s failed", e.Operation)
	if e.Output != "" {
		msg = fmt.Sprintf("%s: %s", msg, e.Output)
	}
	if e.Err != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.Err)
	}
	return msg
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *GitError) Unwrap() error {
	return e.Err
}

// NewGitError creates a new GitError with the given parameters.
func NewGitError(operation string, args []string, err error, output string) *GitError {
	return &GitError{
		Operation: operation,
		Args:      args,
		Err:       err,
		Output:    output,
	}
}

// LockError represents an error that occurred when interacting with file locks.
// It includes the lock file path, process ID if available, and underlying error.
type LockError struct {
	LockFile string
	PID      int
	Err      error
}

// Error implements the error interface with details about the lock file and process.
func (e *LockError) Error() string {
	if e.PID > 0 {
		return fmt.Sprintf("lock error with file %s (PID: %d): %v", e.LockFile, e.PID, e.Err)
	}
	return fmt.Sprintf("lock error with file %s: %v", e.LockFile, e.Err)
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *LockError) Unwrap() error {
	return e.Err
}

// NewLockError creates a new LockError with the given parameters.
func NewLockError(lockFile string, pid int, err error) *LockError {
	return &LockError{
		LockFile: lockFile,
		PID:      pid,
		Err:      err,
	}
}

// ConfigError represents an error in the application configuration.
// It includes the parameter name, its value if available, and the underlying error.
type ConfigError struct {
	Parameter string
	Value     interface{}
	Err       error
}

// Error implements the error interface with details about the invalid configuration.
func (e *ConfigError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("configuration error for %s = %v: %v", e.Parameter, e.Value, e.Err)
	}
	return fmt.Sprintf("configuration error for %s: %v", e.Parameter, e.Err)
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *ConfigError) Unwrap() error {
	return e.Err
}

// NewConfigError creates a new ConfigError with the given parameters.
func NewConfigError(parameter string, value interface{}, err error) *ConfigError {
	return &ConfigError{
		Parameter: parameter,
		Value:     value,
		Err:       err,
	}
}
