package lock

import (
	gitbakErrors "github.com/bashhack/gitbak/pkg/errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestNew_ValidatesProperties(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupPath    string
		expectError  bool
		errorMessage string
		validation   func(t *testing.T, locker *Locker)
	}{
		"ValidPath": {
			setupPath:   t.TempDir(),
			expectError: false,
			validation: func(t *testing.T, locker *Locker) {
				if locker == nil {
					t.Fatal("Expected non-nil locker")
				}

				if locker.pid != os.Getpid() {
					t.Errorf("Expected PID %d, got %d", os.Getpid(), locker.pid)
				}

				if !filepath.IsAbs(locker.lockFile) {
					t.Errorf("Expected absolute lock file path, got %s", locker.lockFile)
				}

				if locker.acquired {
					t.Error("Expected locker to not be acquired by default")
				}
			},
		},
		"EmptyPath": {
			setupPath:   "",
			expectError: false,
			validation: func(t *testing.T, locker *Locker) {
				if locker == nil {
					t.Fatal("Expected non-nil locker")
				}
				// The function hashes even empty paths
				if locker.lockFile == "" {
					t.Error("Expected non-empty lock file path")
				}
			},
		},
		"RelativePath": {
			setupPath:   "relative/path",
			expectError: false,
			validation: func(t *testing.T, locker *Locker) {
				if !filepath.IsAbs(locker.lockFile) {
					t.Errorf("Expected absolute lock file path, got %s", locker.lockFile)
				}
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			locker, err := New(test.setupPath)

			if test.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if test.errorMessage != "" && !strings.Contains(err.Error(), test.errorMessage) {
					t.Errorf("Expected error to contain '%s', got: %v", test.errorMessage, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if test.validation != nil {
					test.validation(t, locker)
				}
			}
		})
	}
}

// TestTryCreateLock_HandlesErrorCases tests various error scenarios for tryCreateLock
func TestTryCreateLock_HandlesErrorCases(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	tests := map[string]struct {
		setup          func(t *testing.T, tempDir string) (string, *Locker)
		expectedError  bool
		errorPredicate func(error) bool
		errorSubstring string
		customTest     func(t *testing.T, locker *Locker) error // Optional custom test function
	}{
		"ExistingFile": {
			setup: func(t *testing.T, tempDir string) (string, *Locker) {
				lockPath := filepath.Join(tempDir, "existing.lock")
				if err := os.WriteFile(lockPath, []byte("12345"), 0666); err != nil {
					t.Fatalf("Failed to create test lock file: %v", err)
				}

				return lockPath, &Locker{
					lockFile: lockPath,
					pid:      os.Getpid(),
				}
			},
			expectedError:  true,
			errorPredicate: os.IsExist,
			errorSubstring: "exists",
		},
		"DirectoryAtPath": {
			setup: func(t *testing.T, tempDir string) (string, *Locker) {
				dirPath := filepath.Join(subDir, "dir-instead-of-file")

				if err := os.Mkdir(dirPath, 0755); err != nil {
					t.Fatalf("Failed to create directory at lock path: %v", err)
				}

				tempFile, err := os.CreateTemp(tempDir, "temp-fd")
				if err != nil {
					t.Fatalf("Failed to create temporary file: %v", err)
				}

				if err := tempFile.Close(); err != nil {
					t.Logf("Warning: Failed to close temporary file: %v", err)
				}

				locker := &Locker{
					lockFile: dirPath,
					lockFd:   tempFile,
					pid:      os.Getpid(),
				}

				return dirPath, locker
			},
			expectedError:  true,
			errorSubstring: "file exists",
		},
		"NonExistentPath": {
			setup: func(t *testing.T, tempDir string) (string, *Locker) {
				nonExistentPath := filepath.Join(tempDir, "nonexistent", "lock.file")

				return nonExistentPath, &Locker{
					lockFile: nonExistentPath,
					pid:      os.Getpid(),
				}
			},
			expectedError:  true,
			errorSubstring: "failed to create lock file",
		},
		"AbsoluteNonExistentPath": {
			setup: func(t *testing.T, tempDir string) (string, *Locker) {
				// Use an absolute path that definitely doesn't exist
				invalidPath := "/this/path/does/not/exist/lockfile.lock"

				return invalidPath, &Locker{
					lockFile: invalidPath,
					pid:      os.Getpid(),
				}
			},
			expectedError:  true,
			errorSubstring: "failed to create lock file",
			errorPredicate: func(err error) bool {
				var lockErr *gitbakErrors.LockError
				return gitbakErrors.As(err, &lockErr)
			},
		},
		"ClosedFD": {
			setup: func(t *testing.T, tempDir string) (string, *Locker) {
				lockPath := filepath.Join(tempDir, "flock-error.lock")

				file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0666)
				if err != nil {
					t.Fatalf("Failed to create lock file: %v", err)
				}

				locker := &Locker{
					lockFile: lockPath,
					lockFd:   file,
					pid:      os.Getpid(),
				}

				// Explicitly close the file to create the error scenario
				if err := file.Close(); err != nil {
					t.Fatalf("Failed to close file: %v", err)
				}

				// This test differs from others in that it directly tests acquireFlock
				// rather than tryCreateLock, since we need to set up a specific state
				t.Cleanup(func() {
					// Ensure the file is cleaned up
					if err := os.Remove(lockPath); err != nil {
						t.Logf("Failed to remove test lock file during cleanup: %v", err)
					}
				})

				return lockPath, locker
			},
			expectedError: true,
			// We're calling acquireFlock directly in the test
			// We'll handle this special case in the main test loop
			customTest: func(t *testing.T, locker *Locker) error {
				return locker.acquireFlock()
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			lockPath, locker := test.setup(t, tempDir)

			var err error
			if test.customTest != nil {
				err = test.customTest(t, locker)
			} else {
				err = locker.tryCreateLock()
			}

			if locker.lockFd != nil {
				if closeErr := locker.lockFd.Close(); closeErr != nil {
					t.Logf("Warning: Failed to close file descriptor: %v", closeErr)
				}
				locker.lockFd = nil
			}

			if test.expectedError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else {
					if test.errorSubstring != "" && !strings.Contains(err.Error(), test.errorSubstring) {
						t.Errorf("Expected error to contain '%s', got: %v", test.errorSubstring, err)
					}

					if test.errorPredicate != nil && !test.errorPredicate(err) {
						t.Errorf("Error does not satisfy the predicate: %v", err)
					}
				}
			} else if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if _, statErr := os.Stat(lockPath); statErr == nil {
				if removeErr := os.RemoveAll(lockPath); removeErr != nil {
					t.Logf("Warning: Failed to clean up lock path: %v", removeErr)
				}
			}
		})
	}
}

// TestResetAndWritePid tests the resetAndWritePid function
func TestResetAndWritePid_Succeeds(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "reset-test.lock")

	if err := os.WriteFile(lockPath, []byte("old-content"), 0666); err != nil {
		t.Fatalf("Failed to create test lock file: %v", err)
	}

	file, err := os.OpenFile(lockPath, os.O_RDWR, 0666)
	if err != nil {
		t.Fatalf("Failed to open test lock file: %v", err)
	}

	locker := &Locker{
		lockFile: lockPath,
		lockFd:   file,
		pid:      os.Getpid(),
	}

	err = locker.resetAndWritePid()
	if err != nil {
		t.Errorf("resetAndWritePid failed: %v", err)
	}

	if err := file.Close(); err != nil {
		t.Logf("Warning: Failed to close file: %v", err)
	}

	// Verify file was truncated and contains our PID
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	expectedContent := []byte(strconv.Itoa(os.Getpid()))
	if string(data) != string(expectedContent) {
		t.Errorf("Expected lock file to contain PID %d, got '%s'", os.Getpid(), string(data))
	}
}

// TestResetAndWritePidErrorWithPipe tests the error path when truncating a file fails using pipe
func TestResetAndWritePid_ErrorsOnPipe(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "reset-pid-error.lock")

	readFd, writeFd, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	if err := writeFd.Close(); err != nil {
		t.Fatalf("Failed to close write pipe: %v", err)
	}

	locker := &Locker{
		lockFile: lockPath,
		lockFd:   readFd,
		pid:      os.Getpid(),
		acquired: true,
	}

	err = locker.resetAndWritePid()

	if err == nil {
		t.Error("Expected an error when truncating a pipe, got nil")
	} else {
		t.Logf("Got expected error from resetAndWritePid: %v", err)
	}

	if err := readFd.Close(); err != nil {
		t.Logf("Warning: Failed to close read pipe: %v", err)
	}
}

// TestWritePidToLockFileError tests writePidToLockFile with a closed file
func TestWritePidToLockFile_ErrorsOnClosedFD(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "write-error.lock")

	locker := &Locker{
		lockFile: lockPath,
		pid:      os.Getpid(),
	}

	file, err := os.CreateTemp(tempDir, "temp-fd")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}

	if err := file.Close(); err != nil {
		t.Logf("Warning: Failed to close temporary file: %v", err)
	}

	locker.lockFd = file

	err = locker.writePidToLockFile()
	if err == nil {
		t.Error("Expected error when writing PID to closed file descriptor")
	}
}
