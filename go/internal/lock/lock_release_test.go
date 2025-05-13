package lock

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReleaseWithRemoveError tests what happens when file removal fails
// due to a read-only parent directory.
func TestRelease_HandlesReadOnlyError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	t.Parallel()
	rootTempDir := t.TempDir()

	readOnlyDir := filepath.Join(rootTempDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0755); err != nil {
		t.Fatalf("Failed to create readonly directory: %v", err)
	}

	lockPath := filepath.Join(readOnlyDir, "locked-file.lock")

	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("Failed to create lock file: %v", err)
	}

	locker := &Locker{
		lockFile: lockPath,
		lockFd:   lockFile,
		pid:      os.Getpid(),
		acquired: true,
	}

	if err := os.Chmod(readOnlyDir, 0555); err != nil {
		t.Fatalf("Failed to make directory read-only: %v", err)
	}
	defer func() {
		if err := os.Chmod(readOnlyDir, 0755); err != nil {
			t.Logf("Warning: Failed to restore directory permissions: %v", err)
		}
	}()

	err = locker.Release()

	if locker.lockFd != nil {
		t.Error("Expected lockFd to be nil regardless of remove error")
	}

	if locker.acquired {
		t.Error("Expected acquired flag to be reset regardless of remove error")
	}
	if err != nil {
		t.Logf("Got expected error from Release with read-only parent dir: %v", err)

		if _, statErr := os.Stat(lockPath); statErr != nil {
			t.Errorf("Expected lock file to still exist when removal fails, got: %v", statErr)
		}
	} else {
		// If no error, log and check if file was actually removed
		// (macOS might allow this operation in some cases)
		if _, statErr := os.Stat(lockPath); os.IsNotExist(statErr) {
			t.Logf("Note: File was successfully removed despite read-only parent on this platform")
		} else if statErr != nil {
			t.Errorf("Unexpected stat error: %v", statErr)
		} else {
			t.Logf("Note: File still exists but no error was returned")
		}
	}
}

// TestRelease_HandlesVariousScenarios tests the Release method in different scenarios
func TestRelease_HandlesVariousScenarios(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup          func(t *testing.T) *Locker
		expectedError  bool
		fileCheckAfter bool // Check if a file exists after release
		expectedExists bool // Whether a file should exist after release
	}{
		"NilFileDescriptor": {
			setup: func(t *testing.T) *Locker {
				repoPath := t.TempDir()
				locker, err := New(repoPath)
				if err != nil {
					t.Fatalf("Failed to create locker: %v", err)
				}

				locker.lockFd = nil
				return locker
			},
			expectedError:  false,
			fileCheckAfter: false, // No file to check
		},
		"NormalFile": {
			setup: func(t *testing.T) *Locker {
				tempDir := t.TempDir()
				lockPath := filepath.Join(tempDir, "test-lock.lock")

				file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
				if err != nil {
					t.Fatalf("Failed to create lock file: %v", err)
				}

				return &Locker{
					lockFile: lockPath,
					lockFd:   file,
					pid:      os.Getpid(),
					acquired: true,
				}
			},
			expectedError:  false,
			fileCheckAfter: true,
			expectedExists: false, // File should be removed
		},
		"MissingFile": {
			setup: func(t *testing.T) *Locker {
				tempDir := t.TempDir()
				lockPath := filepath.Join(tempDir, "nonexistent-lock.lock")

				file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
				if err != nil {
					t.Fatalf("Failed to create lock file: %v", err)
				}

				locker := &Locker{
					lockFile: lockPath,
					lockFd:   file,
					pid:      os.Getpid(),
					acquired: true,
				}

				// Remove the file before Release is called
				if err := os.Remove(lockPath); err != nil {
					t.Fatalf("Failed to remove lock file: %v", err)
				}

				return locker
			},
			expectedError:  false,
			fileCheckAfter: true,
			expectedExists: false, // File already doesn't exist
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			locker := test.setup(t)

			lockPath := locker.lockFile
			err := locker.Release()

			if test.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if locker.lockFd != nil {
				t.Error("Expected lockFd to be nil after Release")
			}

			if locker.acquired {
				t.Error("Expected acquired flag to be reset after Release")
			}

			if test.fileCheckAfter && lockPath != "" {
				_, statErr := os.Stat(lockPath)
				fileExists := !os.IsNotExist(statErr)

				if fileExists != test.expectedExists {
					if test.expectedExists {
						t.Errorf("Expected lock file to still exist but it was removed")
					} else {
						t.Errorf("Expected lock file to be removed, but it still exists")

						if removeErr := os.Remove(lockPath); removeErr != nil {
							t.Logf("Warning: Failed to clean up lock file: %v", removeErr)
						}
					}
				}
			}
		})
	}
}
