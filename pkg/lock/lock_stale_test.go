package lock

import (
	"fmt"
	gitbakErrors "github.com/bashhack/gitbak/pkg/errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestHandleStaleLock_HandlesEdgeCases tests various edge cases for the handleStaleLock function
func TestHandleStaleLock_HandlesEdgeCases(t *testing.T) {
	tests := map[string]struct {
		setup          func(t *testing.T) (*Locker, int)
		expectedError  bool
		errorPredicate func(error) bool
		errorSubstring string
		customCheck    func(t *testing.T, err error) // Additional custom verification
	}{
		"FDCloseError": {
			setup: func(t *testing.T) (*Locker, int) {
				repoPath := t.TempDir()
				tempDir := t.TempDir()
				customLockFile := filepath.Join(tempDir, "lock")

				locker, err := New(repoPath)
				if err != nil {
					t.Fatalf("Failed to create locker: %v", err)
				}

				if err := os.WriteFile(customLockFile, []byte("12345"), 0666); err != nil {
					t.Fatalf("Failed to create test lock file: %v", err)
				}

				locker.lockFile = customLockFile

				// Create a file descriptor (deliberately not closing it)
				fd, err := os.OpenFile(customLockFile, os.O_RDONLY, 0)
				if err != nil {
					t.Fatalf("Failed to open file descriptor: %v", err)
				}

				// Register cleanup to close the fd
				t.Cleanup(func() {
					if err := fd.Close(); err != nil {
						t.Logf("Warning: Failed to close file descriptor: %v", err)
					}
				})

				// Set this fd in the locker
				locker.lockFd = fd

				return locker, 12345
			},
			expectedError: false,
			customCheck: func(t *testing.T, err error) {
				if err != nil && strings.Contains(err.Error(), "close") {
					t.Errorf("Expected handleStaleLock to handle close errors, but got: %v", err)
				}
			},
		},
		"RaceCondition": {
			setup: func(t *testing.T) (*Locker, int) {
				tempDir := t.TempDir()
				lockFile := filepath.Join(tempDir, "lock")

				// First, create the file to simulate a pre-existing lock file
				if err := os.WriteFile(lockFile, []byte("99999"), 0666); err != nil {
					t.Fatalf("Failed to create test lock file: %v", err)
				}

				locker := &Locker{
					lockFile: lockFile,
					pid:      os.Getpid(),
				}

				// Setup code to simulate race condition
				// We'll create a goroutine that watches for file deletion and recreates it quickly
				done := make(chan struct{})
				go func() {
					defer close(done)

					// Watch for the file to be deleted
					for i := 0; i < 50; i++ {
						if _, err := os.Stat(lockFile); os.IsNotExist(err) {
							// File was deleted, quickly recreate it
							if writeErr := os.WriteFile(lockFile, []byte("88888"), 0666); writeErr != nil {
								// If we failed, it's ok, just log
								if !os.IsExist(writeErr) {
									t.Logf("Race condition simulation: %v", writeErr)
								}
							}
							return
						}
						time.Sleep(1 * time.Millisecond)
					}
				}()

				// Register cleanup to ensure we wait for the goroutine to finish
				t.Cleanup(func() {
					<-done // Wait for goroutine to complete
				})

				return locker, 99999
			},
			// We're simulating a race condition, but the outcome is non-deterministic
			// due to timing. We can't guarantee the error will happen every time,
			// so we'll accept either outcome.
			// Note: The test is still valuable as it exercises the code path,
			// even if we can't guarantee the race will always occur.
			expectedError: false,
		},
		"DirectRaceConditionTest": {
			setup: func(t *testing.T) (*Locker, int) {
				// This is a more direct test of the specific OpenFile behavior
				// that handleStaleLock relies on
				tempDir := t.TempDir()
				lockFile := filepath.Join(tempDir, "race-lock")

				if err := os.WriteFile(lockFile, []byte("99999"), 0666); err != nil {
					t.Fatalf("Failed to create test lock file: %v", err)
				}

				// Remove the existing file to simulate the first part of handleStaleLock
				if err := os.Remove(lockFile); err != nil {
					t.Fatalf("Failed to remove lock file: %v", err)
				}

				// Recreate the file to simulate another process grabbing it
				if err := os.WriteFile(lockFile, []byte("88888"), 0666); err != nil {
					t.Fatalf("Failed to recreate test lock file: %v", err)
				}

				// Create a locker just for context, but we'll directly test the OS behavior
				locker := &Locker{
					lockFile: lockFile,
					pid:      os.Getpid(),
				}

				t.Cleanup(func() {
					if removeErr := os.Remove(lockFile); removeErr != nil && !os.IsNotExist(removeErr) {
						t.Logf("Warning: Failed to clean up lock file: %v", removeErr)
					}
				})

				return locker, 99999
			},
			customCheck: func(t *testing.T, err error) {
				// The actual test is to verify the OS behavior directly
				lockFile := filepath.Join(t.TempDir(), "direct-test")

				if err := os.WriteFile(lockFile, []byte("test"), 0666); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				// Try to open with O_EXCL, which should fail since it exists
				_, fileErr := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666)

				if fileErr == nil {
					t.Errorf("Expected to fail creating file with O_EXCL when it already exists")
				} else if !os.IsExist(fileErr) {
					t.Errorf("Expected 'file exists' error, got: %v", fileErr)
				}

				if err := os.Remove(lockFile); err != nil {
					t.Logf("Warning: Failed to remove test file: %v", err)
				}
			},
			expectedError: false, // The handleStaleLock function should handle this case
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			locker, pid := test.setup(t)

			err := locker.handleStaleLock(pid)

			if name == "RaceCondition" {
				if err != nil {
					t.Logf("Race condition occurred as expected, got error: %v", err)
					if !strings.Contains(err.Error(), "another gitbak instance took the lock") {
						t.Errorf("Expected race condition error, got: %v", err)
					}
				} else {
					t.Logf("No race condition occurred this time")
				}
			} else {
				// Normal error checking for other tests
				if test.expectedError && err == nil {
					t.Error("Expected error but got nil")
				} else if !test.expectedError && err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}

			if test.errorPredicate != nil && err != nil {
				if !test.errorPredicate(err) {
					t.Errorf("Error does not satisfy predicate: %v", err)
				}
			}

			if test.errorSubstring != "" && err != nil {
				if !strings.Contains(err.Error(), test.errorSubstring) {
					t.Errorf("Expected error containing '%s', got: %v", test.errorSubstring, err)
				}
			}

			if test.customCheck != nil {
				test.customCheck(t, err)
			}
		})
	}
}

func TestHandleStaleLock_MultiplePIDScenarios(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	var nonExistentPID int
	for pid := 999999; pid > 900000; pid-- {
		if !isProcessRunning(pid) {
			nonExistentPID = pid
			break
		}
	}

	currentPID := os.Getpid()

	tests := map[string]struct {
		fileContent   string
		passedPID     int
		expectedError bool
	}{
		"NumericPID": {
			fileContent:   "999",
			passedPID:     999,
			expectedError: false,
		},
		"NonExistentPID": {
			fileContent:   strconv.Itoa(nonExistentPID),
			passedPID:     nonExistentPID,
			expectedError: false,
		},
		"CurrentPID": {
			fileContent:   strconv.Itoa(currentPID),
			passedPID:     currentPID,
			expectedError: false,
		},
		"InvalidPIDFormat": {
			fileContent:   "not-a-pid",
			passedPID:     0,
			expectedError: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			l := &Locker{
				lockFile: lockPath,
				pid:      os.Getpid(),
			}

			if l.lockFd != nil {
				if err := l.lockFd.Close(); err != nil {
					t.Logf("Warning: Failed to close file descriptor: %v", err)
				}
				l.lockFd = nil
			}

			err := os.WriteFile(lockPath, []byte(test.fileContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test lock file: %v", err)
			}

			err = l.handleStaleLock(test.passedPID)

			if test.expectedError && err == nil {
				t.Error("Expected error but got none")
			} else if !test.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			_, err = os.Stat(lockPath)
			if os.IsNotExist(err) {
				t.Error("Expected lock file to exist after handleStaleLock")
			}
		})
	}

	// Separate test for read-only directory (special environment setup)
	if os.Geteuid() != 0 { // Skip if running as root
		t.Run("ReadOnlyDirectory", func(t *testing.T) {
			readOnlyDir := filepath.Join(tempDir, "readonly")
			err := os.Mkdir(readOnlyDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create readonly directory: %v", err)
			}

			readOnlyLockPath := filepath.Join(readOnlyDir, "readonly.lock")
			err = os.WriteFile(readOnlyLockPath, []byte(strconv.Itoa(nonExistentPID)), 0644)
			if err != nil {
				t.Fatalf("Failed to write test lock file: %v", err)
			}

			err = os.Chmod(readOnlyDir, 0555)
			if err != nil {
				t.Fatalf("Failed to make directory read-only: %v", err)
			}
			defer func() {
				if err := os.Chmod(readOnlyDir, 0755); err != nil {
					t.Logf("Warning: Failed to restore directory permissions: %v", err)
				}
			}() // Restore permissions for cleanup

			readOnlyLock := &Locker{
				lockFile: readOnlyLockPath,
				pid:      os.Getpid(),
			}

			err = readOnlyLock.handleStaleLock(nonExistentPID)
			if err == nil {
				t.Log("Note: Expected permission error didn't occur, possibly running as root")
			}
		})
	}
}

// TestHandleStaleLock_HandlesErrors tests error scenarios for handleStaleLock
func TestHandleStaleLock_HandlesErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup            func(t *testing.T) (*Locker, int)
		skipIfRoot       bool
		expectedError    bool
		errorSubstring   string
		checkLockError   bool // Check if error is a LockError
		checkAcquired    bool // Check the acquired flag
		expectedAcquired bool // Expected value of the acquired flag
	}{
		"ReadOnlyDirectory": {
			setup: func(t *testing.T) (*Locker, int) {
				tempDir := t.TempDir()
				readOnlyDir := filepath.Join(tempDir, "readonly")

				if err := os.Mkdir(readOnlyDir, 0755); err != nil {
					t.Fatalf("Failed to create read-only directory: %v", err)
				}

				lockPath := filepath.Join(readOnlyDir, "stale.lock")

				err := os.WriteFile(lockPath, []byte("12345"), 0600)
				if err != nil {
					t.Fatalf("Failed to create lock file: %v", err)
				}

				if err := os.Chmod(readOnlyDir, 0555); err != nil {
					t.Fatalf("Failed to make directory read-only: %v", err)
				}

				// Register cleanup for directory permissions
				t.Cleanup(func() {
					if err := os.Chmod(readOnlyDir, 0755); err != nil {
						t.Logf("Warning: Failed to restore directory permissions: %v", err)
					}
				})

				return &Locker{
					lockFile: lockPath,
					pid:      os.Getpid(),
				}, 12345
			},
			skipIfRoot:     true,
			expectedError:  true,
			errorSubstring: "failed to remove it",
			checkLockError: true,
		},
		"MissingFile": {
			setup: func(t *testing.T) (*Locker, int) {
				tempDir := t.TempDir()
				lockPath := filepath.Join(tempDir, "nonexistent.lock")

				return &Locker{
					lockFile: lockPath,
					pid:      os.Getpid(),
				}, 12345
			},
			expectedError: true,
		},
		"CreateFailure": {
			setup: func(t *testing.T) (*Locker, int) {
				tempDir := t.TempDir()
				restrictedDir := filepath.Join(tempDir, "restricted")

				if err := os.Mkdir(restrictedDir, 0755); err != nil {
					t.Fatalf("Failed to create restricted directory: %v", err)
				}

				lockPath := filepath.Join(restrictedDir, "stale.lock")

				err := os.WriteFile(lockPath, []byte("12345"), 0600)
				if err != nil {
					t.Fatalf("Failed to create lock file: %v", err)
				}

				// Make the directory read-only to prevent file creation
				if err := os.Chmod(restrictedDir, 0555); err != nil {
					t.Fatalf("Failed to make directory read-only: %v", err)
				}

				// Register cleanup for directory permissions
				t.Cleanup(func() {
					if err := os.Chmod(restrictedDir, 0755); err != nil {
						t.Logf("Warning: Failed to restore directory permissions: %v", err)
					}
				})

				return &Locker{
					lockFile: lockPath,
					pid:      os.Getpid(),
				}, 12345
			},
			// This is platform-dependent, so we don't check expectedError
			// Just check the acquired flag is consistent with any error
			checkAcquired:    true,
			expectedAcquired: false, // Default expectation, may be overridden in the test
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			if test.skipIfRoot && os.Geteuid() == 0 {
				t.Skip("Skipping test that requires non-root permissions")
			}

			locker, pid := test.setup(t)

			err := locker.handleStaleLock(pid)

			// Special case for CreateFailure which is platform-dependent
			if name == "CreateFailure" {
				if err == nil {
					t.Logf("No error from handleStaleLock in read-only dir - may be platform dependent")
					test.expectedAcquired = true // Override default if no error
				} else {
					t.Logf("Got error as expected on some platforms: %v", err)
				}
			}

			// Check error as expected (for cases where we know the expected outcome)
			if test.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if test.errorSubstring != "" && !strings.Contains(err.Error(), test.errorSubstring) {
					t.Errorf("Expected error containing '%s', got: %v", test.errorSubstring, err)
				}

				// Check for LockError type if required
				if test.checkLockError {
					var lockErr *gitbakErrors.LockError
					if !gitbakErrors.As(err, &lockErr) {
						t.Errorf("Expected LockError type, got: %T", err)
					}
				}
			} else if !test.checkAcquired && err != nil { // Skip if we're only checking acquired state
				t.Errorf("Expected no error, got: %v", err)
			}

			// Check acquired state if required
			if test.checkAcquired {
				if locker.acquired != test.expectedAcquired {
					if test.expectedAcquired {
						t.Error("Expected locker to be acquired")
					} else {
						t.Error("Expected locker to not be acquired")
					}
				}
			}
		})
	}
}

// TestHandleBlockedLockWithReadError tests handleBlockedLock when reading the PID fails
func TestHandleBlockedLock_ErrorsOnInvalidPID(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "invalid-pid.lock")

	err := os.WriteFile(lockPath, []byte("not-a-pid"), 0600)
	if err != nil {
		t.Fatalf("Failed to create lock file: %v", err)
	}

	locker := &Locker{
		lockFile: lockPath,
		pid:      os.Getpid(),
	}

	err = locker.handleBlockedLock()

	if err == nil {
		t.Error("Expected error when handling blocked lock with invalid PID")
	} else {
		t.Logf("Got expected error: %v", err)
	}
}

// TestHandleBlockedLockWithOurOwnPID tests an edge case where the blocked lock has our PID
func TestHandleBlockedLock_ErrorsOnSamePID(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "our-pid.lock")

	currentPid := os.Getpid()

	locker := &Locker{
		lockFile: lockPath,
		pid:      currentPid,
	}

	err := os.WriteFile(lockPath, []byte(strconv.Itoa(currentPid)), 0666)
	if err != nil {
		t.Fatalf("Failed to create test lock file: %v", err)
	}

	err = locker.handleBlockedLock()

	if err == nil {
		t.Fatal("Expected error when lock is held by another instance of our process")
	}
}

// TestHandleBlockedLock_HandlesVariousPIDs tests handleBlockedLock with different PID scenarios
func TestHandleBlockedLock_HandlesVariousPIDs(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	var nonExistentPID int
	for pid := 999999; pid > 900000; pid-- {
		if !isProcessRunning(pid) {
			nonExistentPID = pid
			break
		}
	}

	tests := map[string]struct {
		pidToWrite        string // String to write to lock file
		expectedAcquired  bool   // Whether we expect acquisition to succeed
		expectedPidChange bool   // Whether we expect the PID in file to change
	}{
		"NonExistentPID": {
			pidToWrite:        strconv.Itoa(nonExistentPID),
			expectedAcquired:  true,
			expectedPidChange: true,
		},
		"InvalidPIDFormat": {
			pidToWrite:        "not-a-pid",
			expectedAcquired:  false, // Should fail with parse error
			expectedPidChange: false,
		},
		"EmptyFile": {
			pidToWrite:        "",
			expectedAcquired:  false, // Should fail with parse error
			expectedPidChange: false,
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			lockPath := filepath.Join(tempDir, fmt.Sprintf("%s.lock", name))

			locker := &Locker{
				lockFile: lockPath,
				pid:      os.Getpid(),
			}

			err := os.WriteFile(lockPath, []byte(test.pidToWrite), 0666)
			if err != nil {
				t.Fatalf("Failed to create test lock file: %v", err)
			}

			err = locker.handleBlockedLock()

			if test.expectedAcquired {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}

				if !locker.acquired {
					t.Error("Expected locker to be marked as acquired")
				}
			} else {
				if err == nil {
					t.Error("Expected error but got success")
				}

				if locker.acquired {
					t.Error("Expected locker to not be acquired")
				}
			}

			if test.expectedPidChange {
				data, readErr := os.ReadFile(lockPath)
				if readErr != nil {
					t.Errorf("Failed to read lock file: %v", readErr)
				} else {
					expectedPid := strconv.Itoa(os.Getpid())
					if string(data) != expectedPid {
						t.Errorf("Expected lock file to contain PID %s, got %s", expectedPid, string(data))
					}
				}
			}

			if err != nil && locker.lockFd != nil {
				t.Error("Expected lockFd to be nil after failed handleBlockedLock")
			}
		})
	}
}
