package lock

import (
	"os"
	"strconv"
	"testing"
)

// GetLockFile returns the path to the lock file - for testing only
func (l *Locker) GetLockFile() string {
	return l.lockFile
}

// TestStaleLockerRecovery is a platform-agnostic test for stale lock recovery functionality
// It avoids file descriptor issues that can occur with platform-specific behaviors
func TestStaleLockerRecovery(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupContent string
		expectError  bool
	}{
		"NonExistentPID": {
			setupContent: "999999",
			expectError:  false,
		},
		"InvalidPIDFormat": {
			setupContent: "not-a-pid",
			expectError:  false,
		},
		"EmptyLockFile": {
			setupContent: "",
			expectError:  false,
		},
		"CurrentPID": {
			setupContent: strconv.Itoa(os.Getpid()),
			expectError:  false,
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

			lockFilePath := locker.GetLockFile()

			err = os.WriteFile(lockFilePath, []byte(test.setupContent), 0666)
			if err != nil {
				t.Fatalf("Failed to create lock file: %v", err)
			}

			t.Cleanup(func() {
				_ = locker.Release()
			})

			// Try to acquire the lock (should handle the stale lock)
			err = locker.Acquire()

			if test.expectError && err == nil {
				t.Error("Expected an error but got none")
			} else if !test.expectError && err != nil {
				t.Errorf("Expected to acquire lock despite stale lock file, got error: %v", err)
			}
		})
	}
}
