package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// Logger defines the common logging interface used throughout the application.
// It provides a standardized way to emit log messages at different levels of importance,
// with a clear separation between internal (debug) logs and user-facing messages.
//
// The interface is designed to handle both internal logging needs (Info, Warning, Error)
// and user communication (InfoToUser, WarningToUser, Success, StatusMessage).
type Logger interface {
	// Private logging methods (typically written only to log file)

	// Info logs an informational message for debugging purposes.
	// These messages are typically only written to log files and are not shown to users
	// unless verbose mode is enabled.
	//
	// The format string follows fmt.Printf style formatting.
	Info(format string, args ...interface{})

	// Warning logs a warning message for debugging purposes.
	// These messages indicate potential issues that are not critical failures.
	// They are typically only written to log files and are not shown to users
	// unless verbose mode is enabled.
	//
	// The format string follows fmt.Printf style formatting.
	Warning(format string, args ...interface{})

	// Error logs an error message for debugging purposes.
	// These messages indicate operational failures or errors that occurred
	// during program execution. They are typically written to log files and
	// may also be shown to users depending on the logger implementation.
	//
	// The format string follows fmt.Printf style formatting.
	Error(format string, args ...interface{})

	// User-facing logging methods (typically written to both file and stdout)

	// InfoToUser logs an informational message intended for users.
	// These messages are always shown to users regardless of verbose settings,
	// and are also written to log files.
	//
	// The format string follows fmt.Printf style formatting.
	InfoToUser(format string, args ...interface{})

	// WarningToUser logs a warning message intended for users.
	// These messages highlight important issues that users should be aware of,
	// and are always shown regardless of verbose settings.
	//
	// The format string follows fmt.Printf style formatting.
	WarningToUser(format string, args ...interface{})

	// Success logs a success message to the user.
	// These messages indicate successful completion of operations and are
	// typically styled differently (e.g., green text) to stand out.
	//
	// The format string follows fmt.Printf style formatting.
	Success(format string, args ...interface{})

	// StatusMessage logs a status message to the user.
	// These messages provide information about the current state of the application
	// and are always shown to users. They are typically used for displaying
	// configuration information, progress updates, and other operational status.
	//
	// The format string follows fmt.Printf style formatting.
	StatusMessage(format string, args ...interface{})

	// Close ensures any buffered data is written and closes open log file handles.
	// This should be called before the application exits to ensure all logs are properly saved.
	Close() error
}

// DefaultLogger provides structured logging capability and implements the Logger interface
type DefaultLogger struct {
	mu      sync.Mutex
	logger  *slog.Logger
	enabled bool
	logFile string
	verbose bool
	stdout  io.Writer
	stderr  io.Writer
	file    *os.File // Store file handle for closing
}

// New creates a new Logger instance
func New(enabled bool, logFile string, verbose bool) Logger {
	return NewWithOutput(enabled, logFile, verbose, os.Stdout, os.Stderr)
}

// NewWithOutput creates a DefaultLogger with custom output writers
func NewWithOutput(enabled bool, logFile string, verbose bool, stdout, stderr io.Writer) *DefaultLogger {
	var logger *slog.Logger

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	var file *os.File

	if enabled {
		logDir := filepath.Dir(logFile)
		if logDir != "." {
			err := os.MkdirAll(logDir, 0755)
			if err != nil {
				_, _ = fmt.Fprintf(stderr, "⚠️ Failed to create log directory: %v\n", err)
			}
		}

		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			file = f
			fileHandler := slog.NewTextHandler(f, opts)
			logger = slog.New(fileHandler)
			_, _ = fmt.Fprintf(stdout, "🔍 Debug logging enabled. Logs will be written to: %s\n", logFile)

			logger.Info("gitbak debug logging started")
		} else {
			// Fallback to standard logger
			logger = slog.New(slog.NewTextHandler(stderr, opts))
			_, _ = fmt.Fprintf(stderr, "⚠️ Failed to open log file: %v, using stderr instead\n", err)
		}
	} else {
		// Setup non-file logger
		logger = slog.New(slog.NewTextHandler(stderr, opts))
	}

	return &DefaultLogger{
		logger:  logger,
		enabled: enabled,
		logFile: logFile,
		verbose: verbose,
		stdout:  stdout,
		stderr:  stderr,
		file:    file,
	}
}

// Info logs an informational message (file only)
func (l *DefaultLogger) Info(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.enabled {
		return
	}

	msg := fmt.Sprintf(format, args...)
	l.logger.Info(msg)
}

// InfoToUser logs an informational message to both file and stdout
func (l *DefaultLogger) InfoToUser(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)

	if l.enabled {
		l.logger.Info(msg)
	}

	_, _ = fmt.Fprintf(l.stdout, "ℹ️  %s\n", msg)
}

// Success logs a success message to both file and stdout
func (l *DefaultLogger) Success(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)

	if l.enabled {
		l.logger.Info(msg)
	}

	_, _ = fmt.Fprintf(l.stdout, "✅ %s\n", msg)
}

// Warning logs a warning message
func (l *DefaultLogger) Warning(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)

	if l.enabled {
		l.logger.Warn(msg)
	}

	// Always show the message to the user when verbose is on,
	// regardless of whether file logging is enabled
	if l.verbose {
		_, _ = fmt.Fprintf(l.stdout, "⚠️  %s\n", msg)
	}
}

// WarningToUser logs a warning message to both file and stdout
func (l *DefaultLogger) WarningToUser(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)

	if l.enabled {
		l.logger.Warn(msg)
	}

	_, _ = fmt.Fprintf(l.stdout, "⚠️  %s\n", msg)
}

// Error logs an error message
func (l *DefaultLogger) Error(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)

	if l.enabled {
		l.logger.Error(msg)
	}

	// Always show errors to the user regardless of debug status
	_, _ = fmt.Fprintf(l.stderr, "❌ %s\n", msg)
}

// StatusMessage prints a status message to stdout only (no logging)
func (l *DefaultLogger) StatusMessage(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintln(l.stdout, msg)
}

// Close ensures any buffered data is written and closes open log file handles
func (l *DefaultLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		// Sync ensures any buffered data is flushed to disk before closing
		if err := l.file.Sync(); err != nil {
			return err
		}
		return l.file.Close()
	}
	return nil
}

// SetStdout sets a custom writer for user-facing stdout messages only.
// NOTE: This does not affect where structured log messages from slog are directed.
// This method is thread-safe and is primarily intended for testing.
func (l *DefaultLogger) SetStdout(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stdout = w
}

// SetStderr sets a custom writer for user-facing stderr messages only.
// NOTE: This does not affect where structured log messages from slog are directed.
// This method is thread-safe and is primarily intended for testing.
func (l *DefaultLogger) SetStderr(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stderr = w
}
