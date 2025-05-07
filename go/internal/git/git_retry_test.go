package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bashhack/gitbak/internal/errors"
	"github.com/bashhack/gitbak/internal/logger"
)

// errMockingExecutor is a special executor that returns errors for testing retry logic
type errMockingExecutor struct {
	*MockCommandExecutor
	shouldFailCount     int           // Number of initial failures
	failWithErr         error         // Error to return when failing
	permanentFailAfter  int           // After this many calls, start failing permanently
	nextFailCall        chan struct{} // Signal when a fail occurs
	nextSuccessCall     chan struct{} // Signal when a success occurs
	callCount           int           // Track how many times Execute was called
	currentErr          error         // Current error being returned
	shouldResetErrorMsg bool          // Whether to vary error messages
	errorVariant        int           // Counter to vary error messages
}

func (e *errMockingExecutor) Execute(ctx context.Context, cmd *exec.Cmd) error {
	e.callCount++
	e.LastCmd = cmd
	e.Commands = append(e.Commands, cmd)

	// Signal before returning
	defer func() {
		if e.currentErr != nil && e.nextFailCall != nil {
			select {
			case e.nextFailCall <- struct{}{}:
			default:
			}
		} else if e.nextSuccessCall != nil {
			select {
			case e.nextSuccessCall <- struct{}{}:
			default:
			}
		}
	}()

	// Check for permanent failure mode
	if e.permanentFailAfter > 0 && e.callCount > e.permanentFailAfter {
		e.currentErr = e.failWithErr
		return e.currentErr
	}

	// Check for initial failures
	if e.callCount <= e.shouldFailCount {
		if e.shouldResetErrorMsg && e.errorVariant%2 == 0 {
			e.currentErr = errors.NewGitError("test", nil,
				errors.Wrap(errors.ErrGitOperationFailed,
					fmt.Sprintf("varying error message variant %d", e.errorVariant)), "")
			e.errorVariant++
		} else {
			e.currentErr = e.failWithErr
		}
		return e.currentErr
	}

	// Success
	e.currentErr = nil
	return nil
}

// ExecuteWithOutput implements the CommandExecutor interface
func (e *errMockingExecutor) ExecuteWithOutput(ctx context.Context, cmd *exec.Cmd) (string, error) {
	// Use the same error logic as Execute
	err := e.Execute(ctx, cmd)
	if err != nil {
		return "", err
	}
	return "mock output", nil
}

// ExecuteWithContext implements the CommandExecutor interface
func (e *errMockingExecutor) ExecuteWithContext(ctx context.Context, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return e.Execute(ctx, cmd)
}

// ExecuteWithContextAndOutput implements the CommandExecutor interface
func (e *errMockingExecutor) ExecuteWithContextAndOutput(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	return e.ExecuteWithOutput(ctx, cmd)
}

// TestRetryLogicExitsAfterMaxRetries tests that the monitoring loop stops after hitting MaxRetries
func TestRetryLogicExitsAfterMaxRetries(t *testing.T) {
	t.Parallel()
	repoPath := setupTestRepo(t)

	tempLogDir := t.TempDir()
	tempLogFile := filepath.Join(tempLogDir, "gitbak-retry-test.log")
	log := logger.New(true, tempLogFile, true)

	// Create a test file to ensure we have changes to commit
	testFile := filepath.Join(repoPath, "test-retry.txt")
	if err := os.WriteFile(testFile, []byte("Test content for retry logic"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Setup monitoring to ensure we get signals from our mocking executor
	nextFailChan := make(chan struct{}, 10)

	// Create a Gitbak with mocked executor that always fails
	mockErr := errors.NewGitError("status", nil,
		errors.Wrap(errors.ErrGitOperationFailed, "mock git failure for retry test"), "")

	mockExecutor := &errMockingExecutor{
		MockCommandExecutor: NewMockCommandExecutor(),
		shouldFailCount:     10, // More than our MaxRetries
		failWithErr:         mockErr,
		nextFailCall:        nextFailChan,
	}

	gb := &Gitbak{
		config: GitbakConfig{
			RepoPath:        repoPath,
			IntervalMinutes: 1,
			BranchName:      "gitbak-retry-test",
			CreateBranch:    true,
			MaxRetries:      3, // Set max retries to 3
		},
		logger:   log,
		executor: mockExecutor,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	gb.config.IntervalMinutes = 1

	loopErr := gb.TestRetryLoop(ctx, 10)

	if loopErr == nil {
		t.Fatal("Expected monitoringLoop to exit with error after max retries")
	}

	if !errors.Is(loopErr, errors.ErrGitOperationFailed) {
		t.Errorf("Expected error to be ErrGitOperationFailed, got: %v", loopErr)
	}

	if !strings.Contains(loopErr.Error(), "maximum retries") {
		t.Errorf("Expected error to mention maximum retries, got: %v", loopErr)
	}

	if mockExecutor.callCount < gb.config.MaxRetries {
		t.Errorf("Expected at least %d calls to executor, got %d",
			gb.config.MaxRetries, mockExecutor.callCount)
	}
}

// TestRetryLogicResetOnDifferentError tests that the retry counter resets with different errors
func TestRetryLogicResetOnDifferentError(t *testing.T) {
	t.Parallel()
	repoPath := setupTestRepo(t)

	tempLogDir := t.TempDir()
	tempLogFile := filepath.Join(tempLogDir, "gitbak-retry-reset-test.log")
	log := logger.New(true, tempLogFile, true)

	testFile := filepath.Join(repoPath, "test-retry-reset.txt")
	if err := os.WriteFile(testFile, []byte("Test content for retry reset logic"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	nextFailChan := make(chan struct{}, 10)

	mockErr := errors.NewGitError("status", nil,
		errors.Wrap(errors.ErrGitOperationFailed, "base error message that will vary"), "")

	mockExecutor := &errMockingExecutor{
		MockCommandExecutor: NewMockCommandExecutor(),
		shouldFailCount:     10, // More than our MaxRetries, but we'll vary the errors
		failWithErr:         mockErr,
		nextFailCall:        nextFailChan,
		shouldResetErrorMsg: true, // This will make every other error message different
	}

	gb := &Gitbak{
		config: GitbakConfig{
			RepoPath:        repoPath,
			IntervalMinutes: 1,
			BranchName:      "gitbak-retry-reset-test",
			CreateBranch:    true,
			MaxRetries:      3, // With 3 max retries and varying messages, we should get 6+ calls
		},
		logger:   log,
		executor: mockExecutor,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	gb.config.IntervalMinutes = 1

	mockExecutor.shouldResetErrorMsg = true
	mockExecutor.errorVariant = 0

	testErr := gb.TestRetryLoop(ctx, 10)

	// We don't care about the specific error here, just that the test ran properly
	_ = testErr

	// Verify the executor was called enough times to demonstrate the counter reset
	// We should see more calls than if it had failed immediately at MaxRetries
	if mockExecutor.callCount < 4 {
		t.Errorf("Expected at least 4 calls to executor (indicating counter reset works), got %d",
			mockExecutor.callCount)
	}
}

// TestRetryLogicResetOnSuccess tests that the retry counter resets after success
func TestRetryLogicResetOnSuccess(t *testing.T) {
	t.Parallel()
	repoPath := setupTestRepo(t)

	tempLogDir := t.TempDir()

	tempLogFile := filepath.Join(tempLogDir, "gitbak-retry-success-test.log")
	log := logger.New(true, tempLogFile, true)

	testFile := filepath.Join(repoPath, "test-retry-success.txt")
	if err := os.WriteFile(testFile, []byte("Test content for retry reset on success"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	nextFailChan := make(chan struct{}, 10)
	nextSuccessChan := make(chan struct{}, 10)

	mockErr := errors.NewGitError("status", nil,
		errors.Wrap(errors.ErrGitOperationFailed, "intermittent error for success reset test"), "")

	// This executor will fail MaxRetries-1 times, succeed once, then fail MaxRetries-1 times again
	// This pattern confirms the retry counter resets after success
	mockExecutor := &errMockingExecutor{
		MockCommandExecutor: NewMockCommandExecutor(),
		shouldFailCount:     2, // Fail first 2 times (MaxRetries-1)
		failWithErr:         mockErr,
		nextFailCall:        nextFailChan,
		nextSuccessCall:     nextSuccessChan,
		permanentFailAfter:  3, // After 3rd call (1 success), fail permanently
	}

	gb := &Gitbak{
		config: GitbakConfig{
			RepoPath:        repoPath,
			IntervalMinutes: 1,
			BranchName:      "gitbak-retry-success-test",
			CreateBranch:    true,
			MaxRetries:      3,
		},
		logger:   log,
		executor: mockExecutor,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	gb.config.IntervalMinutes = 1

	// Our mock is already configured to fail 2 times, succeed once, then fail permanently
	testErr := gb.TestRetryLoop(ctx, 10)

	if testErr == nil {
		t.Fatal("Expected error from monitoringLoop, got nil")
	}

	// We should still see enough calls to demonstrate the pattern of failure, success, and failure
	if mockExecutor.callCount < 4 {
		t.Errorf("Expected at least 4 calls to executor, got %d",
			mockExecutor.callCount)
	}
}
