package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// Logger provides structured logging capability
type Logger struct {
	logger  *slog.Logger
	enabled bool
	logFile string
	verbose bool
	stdout  io.Writer
	stderr  io.Writer
}

// New creates a new Logger instance
func New(enabled bool, logFile string, verbose bool) *Logger {
	return NewWithOutput(enabled, logFile, verbose, os.Stdout, os.Stderr)
}

// NewWithOutput creates a Logger with custom output writers
func NewWithOutput(enabled bool, logFile string, verbose bool, stdout, stderr io.Writer) *Logger {
	var logger *slog.Logger

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

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

	return &Logger{
		logger:  logger,
		enabled: enabled,
		logFile: logFile,
		verbose: verbose,
		stdout:  stdout,
		stderr:  stderr,
	}
}

// Info logs an informational message (file only)
func (l *Logger) Info(format string, args ...interface{}) {
	if !l.enabled {
		return
	}

	msg := fmt.Sprintf(format, args...)
	l.logger.Info(msg)
}

// InfoToUser logs an informational message to both file and stdout
func (l *Logger) InfoToUser(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)

	if l.enabled {
		l.logger.Info(msg)
	}

	_, _ = fmt.Fprintf(l.stdout, "ℹ️  %s\n", msg)
}

// Success logs a success message to both file and stdout
func (l *Logger) Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)

	if l.enabled {
		l.logger.Info(msg)
	}

	_, _ = fmt.Fprintf(l.stdout, "✅ %s\n", msg)
}

// Warning logs a warning message
func (l *Logger) Warning(format string, args ...interface{}) {
	if !l.enabled {
		return
	}

	msg := fmt.Sprintf(format, args...)
	l.logger.Warn(msg)

	// Also print to console if verbose is enabled
	if l.verbose {
		_, _ = fmt.Fprintf(l.stdout, "⚠️  %s\n", msg)
	}
}

// WarningToUser logs a warning message to both file and stdout
func (l *Logger) WarningToUser(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)

	if l.enabled {
		l.logger.Warn(msg)
	}

	_, _ = fmt.Fprintf(l.stdout, "⚠️  %s\n", msg)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	if !l.enabled {
		return
	}

	msg := fmt.Sprintf(format, args...)
	l.logger.Error(msg)

	_, _ = fmt.Fprintf(l.stderr, "❌ %s\n", msg)
}

// StatusMessage prints a status message to stdout only (no logging)
func (l *Logger) StatusMessage(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintln(l.stdout, msg)
}
