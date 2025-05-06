package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestWrap(t *testing.T) {
	originalErr := New("original error")
	wrappedErr := Wrap(originalErr, "wrapped message")

	if !Is(wrappedErr, originalErr) {
		t.Errorf("Expected wrapped error to match original, but it didn't")
	}

	expectedMsg := "wrapped message: original error"
	if wrappedErr.Error() != expectedMsg {
		t.Errorf("Expected message %q, got %q", expectedMsg, wrappedErr.Error())
	}
}

func TestWrapf(t *testing.T) {
	originalErr := New("original error")
	wrappedErr := Wrapf(originalErr, "wrapped message with %s", "format")

	if !Is(wrappedErr, originalErr) {
		t.Errorf("Expected wrapped error to match original, but it didn't")
	}

	expectedMsg := "wrapped message with format: original error"
	if wrappedErr.Error() != expectedMsg {
		t.Errorf("Expected message %q, got %q", expectedMsg, wrappedErr.Error())
	}
}

func TestGitError(t *testing.T) {
	err := errors.New("command failed")
	gitErr := NewGitError("pull", []string{"origin", "main"}, err, "Permission denied")

	expectedMsg := "git pull failed: Permission denied: command failed"
	if gitErr.Error() != expectedMsg {
		t.Errorf("Expected message %q, got %q", expectedMsg, gitErr.Error())
	}

	if !errors.Is(gitErr, err) {
		t.Errorf("Expected GitError.Unwrap() to return the original error")
	}
}

func TestLockError(t *testing.T) {
	err := errors.New("file not found")
	lockErr := NewLockError("/tmp/lock.file", 1234, err)

	expectedMsg := "lock error with file /tmp/lock.file (PID: 1234): file not found"
	if lockErr.Error() != expectedMsg {
		t.Errorf("Expected message %q, got %q", expectedMsg, lockErr.Error())
	}

	// Test with zero PID
	lockErr = NewLockError("/tmp/lock.file", 0, err)
	expectedMsg = "lock error with file /tmp/lock.file: file not found"
	if lockErr.Error() != expectedMsg {
		t.Errorf("Expected message %q, got %q", expectedMsg, lockErr.Error())
	}

	if !errors.Is(lockErr, err) {
		t.Errorf("Expected LockError.Unwrap() to return the original error")
	}
}

func TestConfigError(t *testing.T) {
	err := errors.New("invalid value")
	configErr := NewConfigError("interval", 0, err)

	expectedMsg := "configuration error for interval = 0: invalid value"
	if configErr.Error() != expectedMsg {
		t.Errorf("Expected message %q, got %q", expectedMsg, configErr.Error())
	}

	configErr = NewConfigError("branchName", nil, err)
	expectedMsg = "configuration error for branchName: invalid value"
	if configErr.Error() != expectedMsg {
		t.Errorf("Expected message %q, got %q", expectedMsg, configErr.Error())
	}

	if !errors.Is(configErr, err) {
		t.Errorf("Expected ConfigError.Unwrap() to return the original error")
	}
}

func TestErrorMatching(t *testing.T) {
	gitErr := NewGitError("status", nil, ErrNotGitRepository, "")

	if !Is(gitErr, ErrNotGitRepository) {
		t.Errorf("Expected gitErr to match ErrNotGitRepository")
	}

	var ge *GitError
	if !As(gitErr, &ge) {
		t.Errorf("Expected gitErr to match GitError type")
	}

	wrappedErr := Wrap(gitErr, "operation failed")

	if !Is(wrappedErr, ErrNotGitRepository) {
		t.Errorf("Expected wrappedErr to match ErrNotGitRepository")
	}

	if !As(wrappedErr, &ge) {
		t.Errorf("Expected wrappedErr to match GitError type")
	}
}

func TestErrorCases(t *testing.T) {
	t.Run("New creates errors", func(t *testing.T) {
		err := New("custom error")
		if err.Error() != "custom error" {
			t.Errorf("Expected error message 'custom error', got %s", err.Error())
		}
	})

	t.Run("Errorf formats errors", func(t *testing.T) {
		err := Errorf("formatted error: %d", 42)
		expected := "formatted error: 42"
		if err.Error() != expected {
			t.Errorf("Expected error message %q, got %q", expected, err.Error())
		}
	})
}

func ExampleWrap() {
	err := fmt.Errorf("original error")

	wrapped := Wrap(err, "context information")

	fmt.Println(wrapped)
	// Output: context information: original error
}

func ExampleNewGitError() {
	err := NewGitError("clone", []string{"https://github.com/example/repo.git"}, fmt.Errorf("connection failed"), "")

	fmt.Println(err)
	// Output: git clone failed: connection failed
}

func ExampleNewLockError() {
	err := NewLockError("/tmp/gitbak.lock", 1234, fmt.Errorf("permission denied"))

	fmt.Println(err)
	// Output: lock error with file /tmp/gitbak.lock (PID: 1234): permission denied
}

func ExampleNewConfigError() {
	err := NewConfigError("interval", -1, fmt.Errorf("must be positive"))

	fmt.Println(err)
	// Output: configuration error for interval = -1: must be positive
}
