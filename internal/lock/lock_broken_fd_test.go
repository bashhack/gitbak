package lock

import (
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
)

// TestDirectBrokenFD tests Release with a file descriptor that's explicitly invalidated
func TestDirectBrokenFD(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "direct-badfd.lock")

	err := os.WriteFile(lockPath, []byte(strconv.Itoa(os.Getpid())), 0600)
	if err != nil {
		t.Fatalf("failed to create lock file: %v", err)
	}

	f, err := os.OpenFile(lockPath, os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("failed to open lock file: %v", err)
	}

	locker := &Locker{
		lockFile: lockPath,
		lockFd:   f,
		pid:      os.Getpid(),
		acquired: true,
	}

	// Test without breaking the file descriptor first - should succeed
	err = locker.Release()
	if err != nil {
		t.Fatalf("Expected no error from Release with valid file, got: %v", err)
	}

	// Now create a new file for the broken fd test
	lockPath = filepath.Join(tempDir, "direct-badfd2.lock")
	err = os.WriteFile(lockPath, []byte(strconv.Itoa(os.Getpid())), 0600)
	if err != nil {
		t.Fatalf("failed to create second lock file: %v", err)
	}

	// Open the file
	f, err = os.OpenFile(lockPath, os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("failed to open second lock file: %v", err)
	}

	// Get the file descriptor number and close it at the syscall level
	fd := f.Fd()

	// Close the file, which should invalidate the descriptor
	if err := f.Close(); err != nil {
		t.Logf("Warning: failed to close file: %v", err)
	}

	// Also try to close at syscall level to ensure the fd is broken
	_ = syscall.Close(int(fd))

	// Create a new locker with a broken file descriptor
	invalidLocker := &Locker{
		lockFile: lockPath,
		lockFd:   f, // This file is already closed, so the descriptor is invalid
		pid:      os.Getpid(),
		acquired: true,
	}

	// Now Release should produce an error because we explicitly invalidated the descriptor
	err = invalidLocker.Release()
	if err == nil {
		t.Error("Expected error from Release with invalid file descriptor, but got none")
	} else {
		t.Logf("Got expected error from Release with invalid file descriptor: %v", err)
	}
}
