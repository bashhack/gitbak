package lock

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestAcquireRelease_HandlesMultipleLockers(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup      func(t *testing.T) (string, *Locker, *Locker)
		operations func(t *testing.T, repoPath string, locker1, locker2 *Locker)
	}{
		"SequentialAcquireRelease": {
			setup: func(t *testing.T) (string, *Locker, *Locker) {
				repoPath := t.TempDir()

				locker1, err := New(repoPath)
				if err != nil {
					t.Fatalf("Failed to create first locker: %v", err)
				}

				locker2, err := New(repoPath)
				if err != nil {
					t.Fatalf("Failed to create second locker: %v", err)
				}

				return repoPath, locker1, locker2
			},
			operations: func(t *testing.T, repoPath string, locker1, locker2 *Locker) {
				if err := locker1.Acquire(); err != nil {
					t.Fatalf("Failed to acquire lock with first locker: %v", err)
				}

				if _, err := os.Stat(locker1.lockFile); err != nil {
					t.Errorf("Expected lock file to exist: %v", err)
				}

				// Create a copy of the lock file to avoid file descriptor issues
				tempDir := t.TempDir()
				tempLockCopy := filepath.Join(tempDir, "lock-copy.tmp")
				if err := copyFile(locker1.lockFile, tempLockCopy); err != nil {
					t.Fatalf("Failed to create copy of lock file: %v", err)
				}

				data, err := os.ReadFile(tempLockCopy)
				if err != nil {
					t.Fatalf("Failed to read lock file copy: %v", err)
				}

				lockPid, err := strconv.Atoi(string(data))
				if err != nil {
					t.Fatalf("Failed to parse PID from lock file copy: %v", err)
				}

				if lockPid != os.Getpid() {
					t.Errorf("Expected lock file to contain PID %d, got %d", os.Getpid(), lockPid)
				}

				// The second locker should fail to acquire...
				if err := locker2.Acquire(); err == nil {
					t.Error("Expected second locker to fail to acquire lock")

					if releaseErr := locker2.Release(); releaseErr != nil {
						t.Logf("Failed to release unexpected lock: %v", releaseErr)
					}
				}

				// First locker releases
				if err := locker1.Release(); err != nil {
					t.Errorf("Failed to release lock: %v", err)
				}

				// Verify the lock file is removed
				if _, err := os.Stat(locker1.lockFile); err == nil {
					t.Error("Expected lock file to be removed after release")

					if removeErr := os.Remove(locker1.lockFile); removeErr != nil {
						t.Logf("Failed to clean up lock file: %v", removeErr)
					}
				}

				// The second locker should now be able to acquire
				if err := locker2.Acquire(); err != nil {
					t.Errorf("Failed to acquire lock after release: %v", err)
				}

				// Clean up
				if err := locker2.Release(); err != nil {
					t.Errorf("Failed to release lock during cleanup: %v", err)
				}
			},
		},
		"ConcurrentRelockAfterRelease": {
			setup: func(t *testing.T) (string, *Locker, *Locker) {
				repoPath := t.TempDir()

				locker1, err := New(repoPath)
				if err != nil {
					t.Fatalf("Failed to create first locker: %v", err)
				}

				locker2, err := New(repoPath)
				if err != nil {
					t.Fatalf("Failed to create second locker: %v", err)
				}

				return repoPath, locker1, locker2
			},
			operations: func(t *testing.T, repoPath string, locker1, locker2 *Locker) {
				// First acquire with locker1
				if err := locker1.Acquire(); err != nil {
					t.Fatalf("Failed to acquire initial lock: %v", err)
				}

				// Release with locker1
				if err := locker1.Release(); err != nil {
					t.Errorf("Failed to release initial lock: %v", err)
				}

				// Create a synchronization channel
				ready := make(chan struct{})
				done := make(chan bool, 2)

				// Both lockers try to acquire concurrently
				go func() {
					<-ready
					if err := locker1.Acquire(); err != nil {
						t.Logf("Locker1 failed to acquire: %v", err)
						done <- false
						return
					}
					done <- true
				}()

				go func() {
					<-ready
					if err := locker2.Acquire(); err != nil {
						t.Logf("Locker2 failed to acquire: %v", err)
						done <- false
						return
					}
					done <- true
				}()

				// Signal both goroutines to start
				close(ready)

				// Wait for results
				success1 := <-done
				success2 := <-done

				// Only one should succeed
				if success1 && success2 {
					t.Error("Both lockers acquired the lock, expected only one to succeed")
				}

				if !success1 && !success2 {
					t.Error("Both lockers failed to acquire, expected one to succeed")
				}

				// Clean up - release whichever locker got the lock
				if success1 {
					if err := locker1.Release(); err != nil {
						t.Errorf("Failed to release locker1: %v", err)
					}
				}

				if success2 {
					if err := locker2.Release(); err != nil {
						t.Errorf("Failed to release locker2: %v", err)
					}
				}
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			repoPath, locker1, locker2 := test.setup(t)
			test.operations(t, repoPath, locker1, locker2)
		})
	}
}

func TestAcquire_RecoversStaleLock(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupLockContent string
		expectedError    bool
		verifyPid        bool // Whether to verify the PID in the lock file
	}{
		"NonExistentPID": {
			setupLockContent: "999999", // Using a very high PID that's unlikely to exist
			expectedError:    false,
			verifyPid:        true,
		},
		"InvalidPIDFormat": {
			setupLockContent: "not-a-pid",
			expectedError:    false,
			verifyPid:        true,
		},
		"EmptyLockFile": {
			setupLockContent: "",
			expectedError:    false,
			verifyPid:        true,
		},
		"CurrentPID": {
			setupLockContent: strconv.Itoa(os.Getpid()),
			expectedError:    false, // Should succeed because it's our own PID
			verifyPid:        true,
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			repoPath := t.TempDir()

			locker, err := New(repoPath)
			if err != nil {
				t.Fatalf("Failed to create locker: %v", err)
			}

			err = os.WriteFile(locker.lockFile, []byte(test.setupLockContent), 0666)
			if err != nil {
				t.Fatalf("Failed to create fake lock file: %v", err)
			}

			// Register cleanup to ensure file descriptors are properly closed
			t.Cleanup(func() {
				// Ensure the lock is released
				if locker.lockFd != nil {
					if err := locker.Release(); err != nil {
						t.Logf("Warning: Failed to release lock during cleanup: %v", err)
					}
				}
			})

			err = locker.Acquire()

			if test.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("Failed to acquire lock despite stale lock file: %v", err)
				}

				if !locker.acquired {
					t.Error("Lock was not marked as acquired")
				}

				if test.verifyPid {
					expectedPID := os.Getpid()

					// Get current PID from a lock file before releasing
					// We're verifying directly from the lock file descriptor
					var pidFromLock int

					if locker.lockFd != nil {
						currentPos, err := locker.lockFd.Seek(0, io.SeekCurrent)
						if err != nil {
							t.Fatalf("Failed to get current file position: %v", err)
						}

						_, err = locker.lockFd.Seek(0, io.SeekStart)
						if err != nil {
							t.Fatalf("Failed to seek to beginning of file: %v", err)
						}

						pidBytes, err := io.ReadAll(locker.lockFd)
						if err != nil {
							t.Fatalf("Failed to read PID from file: %v", err)
						}

						pidFromLock, err = strconv.Atoi(string(pidBytes))
						if err != nil {
							t.Fatalf("Failed to parse PID from file: %v", err)
						}

						_, err = locker.lockFd.Seek(currentPos, io.SeekStart)
						if err != nil {
							t.Fatalf("Failed to restore file position: %v", err)
						}
					}

					if err := locker.Release(); err != nil {
						t.Fatalf("Failed to release lock: %v", err)
					}

					if pidFromLock != expectedPID {
						t.Errorf("Expected lock file to contain PID %d, got %d", expectedPID, pidFromLock)
					}
				} else {
					// Release the lock if we're not verifying the PID
					if err := locker.Release(); err != nil {
						t.Fatalf("Failed to release lock: %v", err)
					}
				}
			}
		})
	}
}

// Helper function to copy a file
func copyFile(src, dst string) error {
	// Check if the source file is a regular file
	fileInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}
	if !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("source file is not a regular file: %s", src)
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Write to the destination file
	return os.WriteFile(dst, data, 0666)
}

// TestTryAcquireExistingLock_HandlesErrors tests error scenarios for tryAcquireExistingLock
func TestTryAcquireExistingLock_HandlesErrors(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	tests := map[string]struct {
		setup          func(t *testing.T) *Locker
		expectedError  bool
		errorSubstring string
		cleanup        func(t *testing.T)
	}{
		"MissingFile": {
			setup: func(t *testing.T) *Locker {
				nonExistentDir := filepath.Join(tempDir, "nonexistent")
				lockPath := filepath.Join(nonExistentDir, "missing.lock")

				return &Locker{
					lockFile: lockPath,
					pid:      os.Getpid(),
				}
			},
			expectedError: true,
		},
		"AlreadyLocked": {
			setup: func(t *testing.T) *Locker {
				lockPath := filepath.Join(tempDir, "already-locked.lock")

				// Create and acquire first lock
				firstLocker, err := New(lockPath)
				if err != nil {
					t.Fatalf("Failed to create first locker: %v", err)
				}

				err = firstLocker.Acquire()
				if err != nil {
					t.Fatalf("Failed to acquire first lock: %v", err)
				}

				// Store the first locker in the test context for cleanup
				t.Cleanup(func() {
					err := firstLocker.Release()
					if err != nil {
						t.Logf("Warning: Failed to release first lock: %v", err)
					}
				})

				// Create the second locker that will fail to acquire
				secondLocker, err := New(lockPath)
				if err != nil {
					t.Fatalf("Failed to create second locker: %v", err)
				}

				return secondLocker
			},
			expectedError: true,
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			locker := test.setup(t)

			err := locker.tryAcquireExistingLock()

			if test.expectedError {
				if err == nil {
					t.Error("Expected error but got nil")

					// If no error but expected one, try to release
					// the lock for cleanup
					_ = locker.Release()
				} else if test.errorSubstring != "" && !strings.Contains(err.Error(), test.errorSubstring) {
					t.Errorf("Expected error containing '%s', got: %v", test.errorSubstring, err)
				} else {
					t.Logf("Got expected error: %v", err)
				}
			} else if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if test.cleanup != nil {
				test.cleanup(t)
			}
		})
	}
}

// TestAcquireFlock tests the acquireFlock method directly
func TestAcquireFlock_Succeeds(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "flock-test.lock")

	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Cleanup(func() {
		if file != nil {
			if err := file.Close(); err != nil {
				if !strings.Contains(err.Error(), "file already closed") {
					t.Logf("Warning: Failed to close file: %v", err)
				}
			}
			file = nil
		}
	})

	locker := &Locker{
		lockFile: lockPath,
		lockFd:   file,
		pid:      os.Getpid(),
	}

	err = locker.acquireFlock()
	if err != nil {
		t.Errorf("Failed to acquire flock: %v", err)
	}

	// Explicitly close the file after the test is done to avoid file descriptor leaks
	// This is important for cross-platform compatibility
	if err := file.Close(); err != nil {
		// Only log if it's not already closed
		if !strings.Contains(err.Error(), "file already closed") {
			t.Logf("Warning: Failed to close file in cleanup: %v", err)
		}
	}
	file = nil
}

func TestReadLockFilePid_HandlesFormats(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "pid_test.lock")

	l := &Locker{
		lockFile: lockPath,
	}

	tests := map[string]struct {
		fileContent    string
		createFile     bool
		expectedPID    int
		expectedError  bool
		errorSubstring string
	}{
		"MissingFile": {
			createFile:     false,
			expectedPID:    0,
			expectedError:  true,
			errorSubstring: "no such file",
		},
		"ValidPID": {
			fileContent:   "12345",
			createFile:    true,
			expectedPID:   12345,
			expectedError: false,
		},
		"InvalidFormat": {
			fileContent:    "not-a-pid",
			createFile:     true,
			expectedPID:    0,
			expectedError:  true,
			errorSubstring: "invalid syntax",
		},
		"EmptyFile": {
			fileContent:    "",
			createFile:     true,
			expectedPID:    0,
			expectedError:  true,
			errorSubstring: "invalid syntax",
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			if _, err := os.Stat(lockPath); err == nil {
				if err := os.Remove(lockPath); err != nil {
					t.Fatalf("Failed to clean up lock file: %v", err)
				}
			}

			if test.createFile {
				if err := os.WriteFile(lockPath, []byte(test.fileContent), 0644); err != nil {
					t.Fatalf("Failed to write test lock file: %v", err)
				}
			}

			pid, err := l.readLockFilePid()

			if pid != test.expectedPID {
				t.Errorf("Expected PID %d, got %d", test.expectedPID, pid)
			}

			if test.expectedError && err == nil {
				t.Error("Expected error but got none")
			} else if !test.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if test.expectedError && err != nil && test.errorSubstring != "" {
				if !strings.Contains(err.Error(), test.errorSubstring) {
					t.Errorf("Expected error containing '%s', got: '%v'", test.errorSubstring, err)
				}
			}
		})
	}
}
