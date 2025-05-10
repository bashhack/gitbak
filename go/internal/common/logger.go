package common

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
}
