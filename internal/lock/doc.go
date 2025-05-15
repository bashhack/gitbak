// Package lock provides file-based locking for the gitbak application.
//
// This package implements a simple file-based locking mechanism to ensure
// that only one instance of gitbak runs for a given repository at a time.
// It creates lock files containing the process ID and handles cleanup
// to prevent stale locks.
//
// # Core Components
//
// - Locker: Main type that manages lock files
//
// # Features
//
// - Repository-specific lock files
// - Process ID tracking to identify lock ownership
// - Stale lock detection and cleanup
// - Clean error messages for lock conflicts
//
// # Usage
//
// Basic usage pattern:
//
//	// Create a new file lock for a repository
//	lock, err := lock.New("/path/to/repo")
//	if err != nil {
//	    // Handle error
//	}
//
//	// Acquire the lock
//	if err := lock.Acquire(); err != nil {
//	    // Handle lock acquisition failure
//	    // Often means another instance is running
//	}
//
//	// Use the locked resource
//	// ...
//
//	// Release the lock when done
//	defer lock.Release()
//
// # Lock Files
//
// Lock files are created in the system's temporary directory with a name derived
// from the repository path. Each lock file contains the process ID of the locking
// process to facilitate ownership verification and cleanup.
//
// The lock file path follows the pattern:
//
//	/tmp/gitbak-<repo-hash>.lock
//
// Where <repo-hash> is a hash of the repository's absolute path.
//
// # Error Handling
//
// The package provides specific error types for common lock-related issues:
//
// - ErrLockExists: Another process holds the lock
// - ErrLockFailed: Failed to create or acquire the lock
//
// # Cleanup
//
// Lock files are automatically removed when the lock is released or when
// the owning process terminates normally. In case of abnormal termination,
// subsequent instances will detect and clean up stale locks.
//
// # Thread Safety
//
// The FileLock type is not designed to be used concurrently by multiple
// goroutines. A single instance should only be accessed from one goroutine
// at a time.
//
// # System Requirements
//
// This package relies on the ability to create and write to files in the
// system's temporary directory. It requires:
//
// - Write permissions to the temporary directory
// - A filesystem that supports exclusive file creation
// - OS-level process ID information
package lock
