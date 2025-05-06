package lock

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/bashhack/gitbak/internal/errors"
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
	if err := l.tryCreateLock(); err == nil {
		return nil
	}

	return l.tryAcquireExistingLock()
}

// tryCreateLock attempts to create and lock a new lock file
func (l *Locker) tryCreateLock() error {
	var err error

	// O_EXCL with O_CREATE ensures the file is created atomically
	l.lockFd, err = os.OpenFile(l.lockFile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666)
	if err != nil {
		return errors.NewLockError(l.lockFile, 0,
			errors.Wrap(errors.ErrLockAcquisitionFailure, "failed to create lock file"))
	}

	if err = l.acquireFlock(); err != nil {
		l.closeFileDescriptor()
		return errors.NewLockError(l.lockFile, 0,
			errors.Wrap(errors.ErrLockAcquisitionFailure, "failed to acquire lock on newly created lock file"))
	}

	if err = l.writePidToLockFile(); err != nil {
		releaseErr := l.Release()
		if releaseErr != nil {
			// Log the release error but return the original error
			return errors.Wrap(err, fmt.Sprintf("failed to write PID and failed to release lock: %v", releaseErr))
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
		return errors.NewLockError(l.lockFile, 0,
			errors.Wrap(errors.ErrLockAcquisitionFailure, "failed to open existing lock file"))
	}

	if err = l.acquireFlock(); err != nil {
		l.closeFileDescriptor()

		if errors.Is(err, syscall.EWOULDBLOCK) {
			return l.handleBlockedLock()
		}

		return errors.NewLockError(l.lockFile, 0,
			errors.Wrap(errors.ErrLockAcquisitionFailure, "failed to acquire lock"))
	}

	if err = l.resetAndWritePid(); err != nil {
		releaseErr := l.Release()
		if releaseErr != nil {
			// Log the release error but return the original error
			return errors.Wrap(err, fmt.Sprintf("failed to reset/write PID and failed to release lock: %v", releaseErr))
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
		return errors.NewLockError(l.lockFile, 0,
			errors.Wrap(pidErr, "another gitbak instance is running, but couldn't identify its PID"))
	}

	if isProcessRunning(otherPid) {
		return errors.NewLockError(l.lockFile, otherPid, errors.ErrAlreadyRunning)
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
		return errors.NewLockError(l.lockFile, l.pid,
			errors.Wrap(err, "failed to truncate lock file"))
	}

	return l.writePidToLockFile()
}

// writePidToLockFile writes PID to the lock file
func (l *Locker) writePidToLockFile() error {
	_, err := l.lockFd.WriteAt([]byte(strconv.Itoa(l.pid)), 0)
	if err != nil {
		return errors.NewLockError(l.lockFile, l.pid,
			errors.Wrap(err, "failed to write PID to lock file"))
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
		return errors.NewLockError(l.lockFile, otherPid,
			errors.Wrapf(fmt.Errorf("permission denied"), "found stale lock file from PID %d, but failed to remove it", otherPid))
	}

	var err error
	l.lockFd, err = os.OpenFile(l.lockFile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666)
	if err != nil {
		if os.IsExist(err) {
			return errors.NewLockError(l.lockFile, 0,
				errors.Wrap(errors.ErrLockAcquisitionFailure, "another gitbak instance took the lock immediately after we removed the stale lock"))
		}
		return errors.NewLockError(l.lockFile, 0,
			errors.Wrap(errors.ErrLockAcquisitionFailure, "failed to open lock file after removing stale lock"))
	}

	if err = l.acquireFlock(); err != nil {
		l.closeFileDescriptor()
		return errors.NewLockError(l.lockFile, 0,
			errors.Wrap(errors.ErrLockAcquisitionFailure, "failed to acquire lock even after removing stale lock"))
	}

	if err = l.writePidToLockFile(); err != nil {
		releaseErr := l.Release()
		if releaseErr != nil {
			// Log the release error but return the original error
			return errors.Wrap(err, fmt.Sprintf("failed to write PID and failed to release lock: %v", releaseErr))
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
		return 0, errors.Wrap(err, "failed to read lock file")
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, errors.Wrap(err, "invalid PID in lock file")
	}

	return pid, nil
}

// Release releases the lock if it was acquired
func (l *Locker) Release() error {
	if l.lockFd == nil {
		return nil
	}

	var err error

	if flockErr := syscall.Flock(int(l.lockFd.Fd()), syscall.LOCK_UN); flockErr != nil {
		err = errors.NewLockError(l.lockFile, l.pid,
			errors.Wrap(errors.ErrLockAcquisitionFailure, "failed to release lock"))
	}

	if closeErr := l.lockFd.Close(); closeErr != nil && err == nil {
		err = errors.NewLockError(l.lockFile, l.pid,
			errors.Wrap(errors.ErrLockAcquisitionFailure, "failed to close lock file"))
	}

	l.lockFd = nil
	l.acquired = false

	// Note: We don't remove the lock file as another process might need the PID
	if removeErr := os.Remove(l.lockFile); removeErr != nil && !os.IsNotExist(removeErr) && err == nil {
		err = errors.NewLockError(l.lockFile, l.pid,
			errors.Wrap(errors.ErrLockAcquisitionFailure, "failed to remove lock file"))
	}

	return err
}
