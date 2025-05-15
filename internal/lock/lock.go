package lock

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	gitbakErrors "github.com/bashhack/gitbak/internal/errors"
)

// Locker prevents concurrent gitbak instances using file locks
type Locker struct {
	lockFile string
	lockFd   *os.File
	pid      int
	acquired bool
}

// New creates a Locker for the specified repository path
func New(repoPath string) (*Locker, error) {
	if runtime.GOOS == "windows" {
		return nil, gitbakErrors.NewLockError("", 0,
			gitbakErrors.Wrap(gitbakErrors.ErrLockAcquisitionFailure,
				"gitbak currently only supports Unix-like operating systems (Linux, macOS, BSD). "+
					"Windows support is not available at this time."))
	}

	repoHash := fmt.Sprintf("%x", sha256.Sum256([]byte(repoPath)))[:16]
	lockFile := filepath.Join(os.TempDir(), fmt.Sprintf("gitbak-%s.lock", repoHash))

	return &Locker{
		lockFile: lockFile,
		pid:      os.Getpid(),
		acquired: false,
	}, nil
}

// Acquire tries to acquire the lock
func (l *Locker) Acquire() error {
	err := l.tryCreateLock()
	if err == nil {
		return nil
	} else if os.IsExist(err) {
		// Only try to acquire an existing lock if the error is specifically about the file already existing
		return l.tryAcquireExistingLock()
	}

	// For other errors, return immediately without trying to acquire an existing lock
	return err
}

// tryCreateLock attempts to create and lock a new lock file
func (l *Locker) tryCreateLock() error {
	var err error

	// O_EXCL with O_CREATE ensures the file is created atomically
	l.lockFd, err = os.OpenFile(l.lockFile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666)
	if err != nil {
		// Pass through the original error so os.IsExist() can detect it
		if os.IsExist(err) {
			return err
		}
		return gitbakErrors.NewLockError(l.lockFile, 0,
			gitbakErrors.Wrap(err, "failed to create lock file"))
	}

	if err = l.acquireFlock(); err != nil {
		l.closeFileDescriptor()
		return gitbakErrors.NewLockError(l.lockFile, 0,
			gitbakErrors.Wrap(err, "failed to acquire lock on newly created lock file"))
	}

	if err = l.writePidToLockFile(); err != nil {
		releaseErr := l.Release()
		if releaseErr != nil {
			// Log the release error but return the original error
			return gitbakErrors.Wrap(err, fmt.Sprintf("failed to write PID and failed to release lock: %v", releaseErr))
		}
		return err
	}

	l.acquired = true
	return nil
}

// tryAcquireExistingLock acquires a lock on an existing lock file
func (l *Locker) tryAcquireExistingLock() error {
	var err error
	l.lockFd, err = os.OpenFile(l.lockFile, os.O_RDWR, 0666)
	if err != nil {
		return gitbakErrors.NewLockError(l.lockFile, 0,
			gitbakErrors.Wrap(err, "failed to open existing lock file"))
	}

	if err = l.acquireFlock(); err != nil {
		l.closeFileDescriptor()

		// Hedging bets here and checking either EWOULDBLOCK or EAGAIN,
		// Per GNU docs ...
		//     Portability Note: In many older Unix systems ...
		//     [EWOULDBLOCK was] a distinct error code different from EAGAIN.
		//     To make your program portable, you should check for both codes
		//     and treat them the same.
		// Ref: https://www.gnu.org/savannah-checkouts/gnu/libc/manual/html_node/Error-Codes.html
		if gitbakErrors.Is(err, syscall.EWOULDBLOCK) || gitbakErrors.Is(err, syscall.EAGAIN) {
			return l.handleBlockedLock()
		}

		return gitbakErrors.NewLockError(l.lockFile, 0,
			gitbakErrors.Wrap(err, "failed to acquire lock"))
	}

	if err = l.resetAndWritePid(); err != nil {
		releaseErr := l.Release()
		if releaseErr != nil {
			// Log the release error but return the original error
			return gitbakErrors.Wrap(err, fmt.Sprintf("failed to reset/write PID and failed to release lock: %v", releaseErr))
		}
		return err
	}

	l.acquired = true
	return nil
}

// handleBlockedLock handles locks held by another process
// and attempts to recover from stale locks
func (l *Locker) handleBlockedLock() error {
	otherPid, pidErr := l.readLockFilePid()
	if pidErr != nil {
		return gitbakErrors.NewLockError(l.lockFile, 0,
			gitbakErrors.Wrap(pidErr, "another gitbak instance is running, but couldn't identify its PID"))
	}

	if isProcessRunning(otherPid) {
		return gitbakErrors.NewLockError(l.lockFile, otherPid, gitbakErrors.ErrAlreadyRunning)
	}

	return l.handleStaleLock(otherPid)
}

// acquireFlock gets an exclusive non-blocking lock
func (l *Locker) acquireFlock() error {
	return syscall.Flock(int(l.lockFd.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

// resetAndWritePid clears the file and writes the current PID
func (l *Locker) resetAndWritePid() error {
	if err := l.lockFd.Truncate(0); err != nil {
		return gitbakErrors.NewLockError(l.lockFile, l.pid,
			gitbakErrors.Wrap(err, "failed to truncate lock file"))
	}

	return l.writePidToLockFile()
}

// writePidToLockFile writes PID to the lock file
func (l *Locker) writePidToLockFile() error {
	_, err := l.lockFd.WriteAt([]byte(strconv.Itoa(l.pid)), 0)
	if err != nil {
		return gitbakErrors.NewLockError(l.lockFile, l.pid,
			gitbakErrors.Wrap(err, "failed to write PID to lock file"))
	}
	return nil
}

// closeFileDescriptor closes the lock file descriptor
func (l *Locker) closeFileDescriptor() {
	if l.lockFd != nil {
		_ = l.lockFd.Close()
		l.lockFd = nil
	}
}

// isProcessRunning checks if a process exists using signal 0
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// handleStaleLock removes and recreates a stale lock
func (l *Locker) handleStaleLock(otherPid int) error {
	l.closeFileDescriptor()

	if err := os.Remove(l.lockFile); err != nil {
		return gitbakErrors.NewLockError(l.lockFile, otherPid,
			gitbakErrors.Wrap(err, fmt.Sprintf("found stale lock file from PID %d, but failed to remove it", otherPid)))
	}

	var err error
	l.lockFd, err = os.OpenFile(l.lockFile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666)
	if err != nil {
		if os.IsExist(err) {
			return gitbakErrors.NewLockError(l.lockFile, 0,
				gitbakErrors.Wrap(err, "another gitbak instance took the lock immediately after we removed the stale lock"))
		}
		return gitbakErrors.NewLockError(l.lockFile, 0,
			gitbakErrors.Wrap(err, "failed to open lock file after removing stale lock"))
	}

	if err = l.acquireFlock(); err != nil {
		l.closeFileDescriptor()
		return gitbakErrors.NewLockError(l.lockFile, 0,
			gitbakErrors.Wrap(err, "failed to acquire lock even after removing stale lock"))
	}

	if err = l.writePidToLockFile(); err != nil {
		releaseErr := l.Release()
		if releaseErr != nil {
			// Log the release error but return the original error
			return gitbakErrors.Wrap(err, fmt.Sprintf("failed to write PID and failed to release lock: %v", releaseErr))
		}
		return err
	}

	l.acquired = true
	return nil
}

// readLockFilePid reads and parses the PID from the lock file
func (l *Locker) readLockFilePid() (int, error) {
	data, err := os.ReadFile(l.lockFile)
	if err != nil {
		return 0, gitbakErrors.Wrap(err, "failed to read lock file")
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, gitbakErrors.Wrap(err, "invalid PID in lock file")
	}

	return pid, nil
}

// Release releases the lock if it was acquired
func (l *Locker) Release() error {
	if l.lockFd == nil {
		return nil
	}

	var err error

	// First, try to verify if the file descriptor is valid
	// We do this by getting file stats, a safer operation than unlocking
	fd := l.lockFd.Fd()
	var stat syscall.Stat_t
	if statErr := syscall.Fstat(int(fd), &stat); statErr != nil {
		// If we can't even stat the file, it's definitely broken...
		err = gitbakErrors.NewLockError(l.lockFile, l.pid,
			gitbakErrors.Wrap(statErr, "failed to stat lock file - file descriptor is invalid"))
	} else {
		// We can stat the file, but let's also try to write to it to check if it's valid...
		// this handles cases where the file descriptor might appear valid but actual operations fail
		_, writeErr := l.lockFd.WriteAt([]byte{}, 0)
		if writeErr != nil {
			err = gitbakErrors.NewLockError(l.lockFile, l.pid,
				gitbakErrors.Wrap(writeErr, "failed to write to lock file - file descriptor is invalid"))
		} else {
			// Attempt to unlock the file
			if flockErr := syscall.Flock(int(fd), syscall.LOCK_UN); flockErr != nil {
				err = gitbakErrors.NewLockError(l.lockFile, l.pid,
					gitbakErrors.Wrap(flockErr, "failed to release lock"))
			}
		}
	}

	// Always try to close the file descriptor, even if previous operations failed
	if closeErr := l.lockFd.Close(); closeErr != nil && err == nil {
		err = gitbakErrors.NewLockError(l.lockFile, l.pid,
			gitbakErrors.Wrap(closeErr, "failed to close lock file"))
	}

	l.lockFd = nil
	l.acquired = false

	// Always try to remove the lock file, regardless of previous errors
	// This ensures we clean up as much as possible even if there were errors
	// Only report the error if there were no previous errors
	if removeErr := os.Remove(l.lockFile); removeErr != nil && !os.IsNotExist(removeErr) && err == nil {
		err = gitbakErrors.NewLockError(l.lockFile, l.pid,
			gitbakErrors.Wrap(removeErr, "failed to remove lock file"))
	}

	return err
}
