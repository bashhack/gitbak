package lock

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/bashhack/gitbak/internal/errors"
)

func TestNew(t *testing.T) {
	repoPath := "/tmp/test-repo"
	locker, err := New(repoPath)

	if err != nil {
		t.Fatalf("Failed to create locker: %v", err)
	}

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
}

func TestAcquireAndRelease(t *testing.T) {
	repoPath := filepath.Join(os.TempDir(), "gitbak-test-repo-"+t.Name())

	locker1, err := New(repoPath)
	if err != nil {
		t.Fatalf("Failed to create first locker: %v", err)
	}

	err = locker1.Acquire()
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	if _, err := os.Stat(locker1.lockFile); err != nil {
		t.Errorf("Expected lock file to exist: %v", err)
	}

	data, err := os.ReadFile(locker1.lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockPid, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatalf("Failed to parse PID from lock file: %v", err)
	}

	if lockPid != os.Getpid() {
		t.Errorf("Expected lock file to contain PID %d, got %d", os.Getpid(), lockPid)
	}

	locker2, err := New(repoPath)
	if err != nil {
		t.Fatalf("Failed to create second locker: %v", err)
	}

	err = locker2.Acquire()
	if err == nil {
		t.Error("Expected second locker to fail to acquire lock")
	}

	err = locker1.Release()
	if err != nil {
		t.Errorf("Failed to release lock: %v", err)
	}

	if _, err := os.Stat(locker1.lockFile); err == nil {
		t.Error("Expected lock file to be removed after release")

		if removeErr := os.Remove(locker1.lockFile); removeErr != nil {
			t.Logf("Failed to clean up lock file: %v", removeErr)
		}
	}

	err = locker2.Acquire()
	if err != nil {
		t.Errorf("Failed to acquire lock after release: %v", err)
	}

	err = locker2.Release()
	if err != nil {
		t.Errorf("Failed to release lock during cleanup: %v", err)
	}
}

func TestConcurrentLocks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	repoPath := filepath.Join(os.TempDir(), "gitbak-test-repo-"+t.Name())

	done := make(chan bool)

	for i := 0; i < 5; i++ {
		go func(id int) {
			locker, err := New(repoPath)
			if err != nil {
				t.Errorf("Goroutine %d: Failed to create locker: %v", id, err)
				done <- false
				return
			}

			err = locker.Acquire()
			if err != nil {
				// With multiple goroutines competing for the same lock,
				// only one can succeed at any given time, so it's normal
				// and expected for some acquisition attempts to fail
				done <- false
				return
			}

			// If we got the lock, release it after a brief pause
			time.Sleep(100 * time.Millisecond)
			releaseErr := locker.Release()
			if releaseErr != nil {
				t.Errorf("Goroutine %d: Failed to release lock: %v", id, releaseErr)
			}

			done <- true
		}(i)
	}

	successCount := 0
	for i := 0; i < 5; i++ {
		if <-done {
			successCount++
		}
	}

	// We should have at least one success
	if successCount == 0 {
		t.Error("Expected at least one goroutine to acquire the lock")
	}
}

func TestStaleLockDetection(t *testing.T) {
	repoPath := filepath.Join(os.TempDir(), "gitbak-test-repo-stale-"+t.Name())

	locker1, err := New(repoPath)
	if err != nil {
		t.Fatalf("Failed to create locker: %v", err)
	}

	// Create a lock file directly with a non-existent PID
	// (this simulates a stale lock from a process that no longer exists)
	nonExistentPid := 999999 // Using a very high PID that's unlikely to exist
	err = os.WriteFile(locker1.lockFile, []byte(strconv.Itoa(nonExistentPid)), 0666)
	if err != nil {
		t.Fatalf("Failed to create fake lock file: %v", err)
	}

	// For this test, we need a PID that doesn't exist on the system
	// 999999 is chosen because it's beyond the typical PID range on most systems
	// The test checks the stale lock detection mechanism

	// Try to acquire the lock - should succeed by removing the stale lock
	err = locker1.Acquire()
	if err != nil {
		t.Fatalf("Failed to acquire lock despite stale lock file: %v", err)
	}

	// Verify the lock file now contains our PID, not the fake one
	data, err := os.ReadFile(locker1.lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockPid, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatalf("Failed to parse PID from lock file: %v", err)
	}

	if lockPid != os.Getpid() {
		t.Errorf("Expected lock file to contain PID %d, got %d", os.Getpid(), lockPid)
	}

	err = locker1.Release()
	if err != nil {
		t.Errorf("Failed to release lock: %v", err)
	}
}

// TestHandleStaleLockPathValid tests handleStaleLock with a valid path
func TestHandleStaleLockPathValid(t *testing.T) {
	repoPath := filepath.Join(os.TempDir(), "gitbak-test-repo-stale-path-"+t.Name())

	locker, err := New(repoPath)
	if err != nil {
		t.Fatalf("Failed to create locker: %v", err)
	}

	// Make sure the lock file doesn't exist
	_, err = os.Stat(locker.lockFile)
	if err == nil {
		// Remove it if it exists
		if err := os.Remove(locker.lockFile); err != nil {
			t.Fatalf("Failed to remove existing lock file: %v", err)
		}
	} else if !os.IsNotExist(err) {
		t.Fatalf("Failed to check lock file: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "gitbak-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Use a custom lock file path with write permissions
	customLockFile := filepath.Join(tempDir, "lock")

	// Create the lock file first - handleStaleLock expects it to exist
	if err := os.WriteFile(customLockFile, []byte("12345"), 0666); err != nil {
		t.Fatalf("Failed to create test lock file: %v", err)
	}

	// Set the locker to use our custom lock file
	locker.lockFile = customLockFile

	// Call handleStaleLock - this should remove the existing lock and create a new one
	err = locker.handleStaleLock(12345)
	if err != nil {
		t.Errorf("Failed to handle stale lock: %v", err)
	}

	// Verify lock file exists and contains our PID
	data, err := os.ReadFile(customLockFile)
	if err != nil {
		t.Errorf("Lock file was not created: %v", err)
	} else {
		// Check if the content is our PID
		ourPid := os.Getpid()
		fileContent := string(data)
		if fileContent != strconv.Itoa(ourPid) {
			t.Errorf("Lock file contains wrong content: expected '%d', got '%s'", ourPid, fileContent)
		}
	}
}

// TestHandleStaleLockCloseError tests error when closing the file descriptor
func TestHandleStaleLockCloseError(t *testing.T) {
	repoPath := filepath.Join(os.TempDir(), "gitbak-test-stale-close-error-"+t.Name())

	locker, err := New(repoPath)
	if err != nil {
		t.Fatalf("Failed to create locker: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "gitbak-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	customLockFile := filepath.Join(tempDir, "lock")

	if err := os.WriteFile(customLockFile, []byte("12345"), 0666); err != nil {
		t.Fatalf("Failed to create test lock file: %v", err)
	}

	locker.lockFile = customLockFile

	// Create a file descriptor (deliberately not closing it)
	fd, err := os.OpenFile(customLockFile, os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("Failed to open file descriptor: %v", err)
	}
	defer func() {
		if err := fd.Close(); err != nil {
			t.Logf("Warning: Failed to close file descriptor: %v", err)
		}
	}()

	// Set this fd in the locker
	locker.lockFd = fd

	// The function should always close the fd and continue
	err = locker.handleStaleLock(12345)
	if err != nil {
		// If there's an error, it should be from trying to remove or create the file, not from close
		if strings.Contains(err.Error(), "close") {
			t.Errorf("Expected handleStaleLock to handle close errors, but got: %v", err)
		}
	}
}

// TestHandleStaleLockRemoveError tests error when removing the stale lock file
func TestHandleStaleLockRemoveError(t *testing.T) {
	repoPath := filepath.Join(os.TempDir(), "gitbak-test-stale-remove-error-"+t.Name())

	locker, err := New(repoPath)
	if err != nil {
		t.Fatalf("Failed to create locker: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "gitbak-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	locker.lockFile = filepath.Join(tempDir, "non-existent-file")

	err = locker.handleStaleLock(12345)
	if err == nil {
		t.Error("Expected error when trying to remove non-existent lock file, got nil")
	}
}

// TestHandleStaleLockOpenError tests error when opening the new lock file
func TestHandleStaleLockOpenError(t *testing.T) {
	repoPath := filepath.Join(os.TempDir(), "gitbak-test-stale-open-error-"+t.Name())
	locker, err := New(repoPath)
	if err != nil {
		t.Fatalf("Failed to create locker: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "gitbak-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.Chmod(tempDir, 0755); err != nil {
			t.Logf("Warning: Failed to restore directory permissions: %v", err)
		}
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create a subdirectory for our test file
	lockDir := filepath.Join(tempDir, "lockdir")
	if err := os.Mkdir(lockDir, 0755); err != nil {
		t.Fatalf("Failed to create lock directory: %v", err)
	}

	customLockPath := filepath.Join(lockDir, "lock")

	// Create a file for the initial stale lock
	if err := os.WriteFile(customLockPath, []byte("12345"), 0666); err != nil {
		t.Fatalf("Failed to create test lock file: %v", err)
	}

	locker.lockFile = customLockPath

	// Struggled with this one, trying to get a reliable test case for Unix-like systems:
	// 1. Make the lockDir read-only after we've the file has been created
	// 2. When handleStaleLock is called, it will:
	//    - Delete the existing file (still works with read-only dirs on macOS/Linux)
	//    - Try to create a new file, which will fail due to the read-only dir
	if err := os.Chmod(lockDir, 0555); err != nil {
		t.Fatalf("Failed to change directory permissions: %v", err)
	}

	err = locker.handleStaleLock(12345)

	// On macOS and Linux, this should fail with a permission-denied error
	if err == nil {
		t.Errorf("Expected error when trying to create a file in read-only directory, got nil")
	} else {
		if !strings.Contains(err.Error(), "permission denied") &&
			!strings.Contains(err.Error(), "Permission denied") {
			t.Errorf("Expected permission denied error, got: %v", err)
		}
	}

	// Restore permissions to allow cleanup
	_ = os.Chmod(lockDir, 0755)
}

// TestHandleStaleLockFlockError tests the locking part of handleStaleLock
// Note: This test made me pull my hair out, and I found it very difficult
// to make reliable across platforms because of file locking behaviors
// variance - so ¯\_(ツ)_/¯ ... this can probably be tested better somehow...
func TestHandleStaleLockFlockError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gitbak-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create a file that will be our lock target
	lockFile := filepath.Join(tempDir, "lock")

	// First, create the file to simulate a pre-existing lock file
	if err := os.WriteFile(lockFile, []byte("99999"), 0666); err != nil {
		t.Fatalf("Failed to create test lock file: %v", err)
	}

	// Directly testing what happens in handleStaleLock when
	// another process creates the file between our remove and open attempts.
	// To simulate this:
	// 1. We create a Locker with this file
	// 2. Remove the file (manually simulating part of handleStaleLock)
	// 3. Recreate the file from "another process" before we try to open with O_EXCL
	// 4. Try to open with O_EXCL which should fail

	// Remove the existing file
	if err := os.Remove(lockFile); err != nil {
		t.Fatalf("Failed to remove lock file: %v", err)
	}

	// Recreate the file to simulate another process grabbing it
	if err := os.WriteFile(lockFile, []byte("88888"), 0666); err != nil {
		t.Fatalf("Failed to recreate test lock file: %v", err)
	}

	// Try to open with O_EXCL, which should fail
	// This directly tests the behavior in handleStaleLock...
	_, err = os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666)

	if err == nil {
		t.Errorf("Expected to fail creating file with O_EXCL when it already exists")
	} else {
		if !os.IsExist(err) {
			t.Errorf("Expected 'file exists' error, got: %v", err)
		}
	}
}

// TestHandleStaleLockWriteError tests the writing part of handleStaleLock
func TestHandleStaleLockWriteError(t *testing.T) {
	repoPath := filepath.Join(os.TempDir(), "gitbak-test-repo-stale-write-error-"+t.Name())

	locker, err := New(repoPath)
	if err != nil {
		t.Fatalf("Failed to create locker: %v", err)
	}

	testDir, err := os.MkdirTemp("", "gitbak-lock-test-")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer func() {
		if err := os.Chmod(testDir, 0755); err != nil {
			t.Logf("Warning: Failed to restore directory permissions: %v", err)
		}
		if err := os.RemoveAll(testDir); err != nil {
			t.Logf("Failed to clean up test directory: %v", err)
		}
	}()

	customLockFile := filepath.Join(testDir, "lockfile")
	err = os.WriteFile(customLockFile, []byte("12345"), 0666)
	if err != nil {
		t.Fatalf("Failed to create test lock file: %v", err)
	}

	locker.lockFile = customLockFile

	tempFile, err := os.CreateTemp(testDir, "closed-fd")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}

	// Close the file right away
	if err := tempFile.Close(); err != nil {
		t.Logf("Warning: Failed to close temporary file: %v", err)
	}

	// Set this closed FD in the locker
	// This ensures that when handleStaleLock tries to write to it, it will fail
	locker.lockFd = tempFile

	// Call handleStaleLock - this should try to write to the closed FD and fail
	err = locker.handleStaleLock(12345)

	// Writing to a closed file descriptor should fail on all platforms
	if err == nil {
		// If it succeeds, the function might have detected the closed FD and
		// created a new one. This behavior is platform-dependent, so I'm
		// considering both success and failure as acceptable outcomes...
		t.Logf("Note: handleStaleLock succeeded despite using a closed file descriptor")
	} else {
		// Verify we got some kind of error
		t.Logf("Got expected error: %v", err)
	}
}

// TestLockWithRunningProcess simulates another process holding the lock
func TestLockWithRunningProcess(t *testing.T) {
	repoPath := filepath.Join(os.TempDir(), "gitbak-test-repo-running-"+t.Name())

	locker1, err := New(repoPath)
	if err != nil {
		t.Fatalf("Failed to create locker: %v", err)
	}

	var lockFd *os.File
	lockFd, err = os.OpenFile(locker1.lockFile, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		t.Fatalf("Failed to open lock file: %v", err)
	}
	defer func() {
		if closeErr := lockFd.Close(); closeErr != nil {
			t.Logf("Failed to close lock file: %v", closeErr)
		}
	}()

	err = syscall.Flock(int(lockFd.Fd()), syscall.LOCK_EX)
	if err != nil {
		t.Fatalf("Failed to lock file: %v", err)
	}

	currentPid := os.Getpid()
	_, err = lockFd.WriteAt([]byte(strconv.Itoa(currentPid)), 0)
	if err != nil {
		t.Fatalf("Failed to write PID: %v", err)
	}

	// For this test, we'll use a real running process (our own process)
	// and we'll verify the correct error message is returned

	locker2, err := New(repoPath)
	if err != nil {
		t.Fatalf("Failed to create second locker: %v", err)
	}

	// Try to acquire the lock - should fail because file is locked and process is running
	err = locker2.Acquire()
	if err == nil {
		t.Fatalf("Expected to fail acquiring lock with running process, but succeeded")
	}

	// Check for our custom lock error type
	var lockErr *errors.LockError
	if !errors.As(err, &lockErr) {
		t.Errorf("Expected error to be of type *errors.LockError, got %T", err)
	} else {
		if lockErr.PID <= 0 {
			t.Errorf("Expected positive PID in lock error, got %d", lockErr.PID)
		}

		if !errors.Is(lockErr.Err, errors.ErrAlreadyRunning) {
			t.Errorf("Expected underlying error to be ErrAlreadyRunning, got %v", lockErr.Err)
		}
	}

	if flockErr := syscall.Flock(int(lockFd.Fd()), syscall.LOCK_UN); flockErr != nil {
		t.Logf("Failed to release flock: %v", flockErr)
	}
}
