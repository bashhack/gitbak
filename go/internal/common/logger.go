package common

// Logger defines the common logging interface used throughout the application
type Logger interface {
	// Private logging methods (file only)

	// Info logs an informational message
	Info(format string, args ...interface{})

	// Warning logs a warning message
	Warning(format string, args ...interface{})

	// Error logs an error message
	Error(format string, args ...interface{})

	// User-facing logging methods (file + stdout)

	// InfoToUser logs an informational message to the user
	InfoToUser(format string, args ...interface{})

	// WarningToUser logs a warning message to the user
	WarningToUser(format string, args ...interface{})

	// Success logs a success message to the user
	Success(format string, args ...interface{})

	// StatusMessage logs a status message to the user
	StatusMessage(format string, args ...interface{})
}
