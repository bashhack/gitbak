// Package logger provides logging facilities for the gitbak application.
//
// This package implements a simple, structured logging system with different
// log levels, colors for terminal output, and the ability to write logs to
// both the console and a file simultaneously. It defines both the logging interface
// and the standard implementation used throughout the application.
//
// # Core Components
//
// - Logger: The main interface for logging used throughout the application
// - DefaultLogger: Standard implementation that writes to console and/or file
//
// # Features
//
// - Multiple log levels (Info, Warning, Error, Success)
// - Colored output for terminal visibility
// - File logging with rotation
// - User-facing vs. debug-only messages
// - Conditional logging based on verbosity settings
//
// # Log Levels
//
// The logger supports the following distinct message types:
//
// - Info: General information messages
// - InfoToUser: Important information to display to the user
// - Warning: Warning messages for potential issues
// - WarningToUser: Important warnings to display to the user
// - Error: Error messages for failures
// - Success: Success messages for completed operations
// - StatusMessage: Current status updates
//
// # Usage
//
// Basic usage pattern:
//
//	// Create a new logger
//	logger := logger.New(true, "/path/to/log.file", true)
//
//	// Log different types of messages
//	logger.Info("Debug-only information: %v", details)
//	logger.InfoToUser("Important information: %v", userInfo)
//	logger.Warning("Potential issue: %v", warning)
//	logger.Error("An error occurred: %v", err)
//	logger.Success("Operation completed: %v", result)
//
// # Usage With Dependency Injection
//
// The Logger interface is typically injected into components that need logging capabilities:
//
//	type MyComponent struct {
//	    logger logger.Logger
//	    // other fields
//	}
//
//	func NewMyComponent(logger logger.Logger) *MyComponent {
//	    return &MyComponent{
//	        logger: logger,
//	    }
//	}
//
//	func (c *MyComponent) DoSomething() error {
//	    // Internal logging (debug information)
//	    c.logger.Info("Starting operation")
//
//	    // User-facing information
//	    c.logger.InfoToUser("Processing your request")
//
//	    // Success message shown to the user
//	    c.logger.Success("Operation completed successfully")
//
//	    return nil
//	}
//
// # Console Output
//
// Console output is formatted with colors and prefixes to distinguish different
// message types:
//
// - Info: White text with [INFO] prefix (only shown when verbose)
// - Warning: Yellow text with [WARNING] prefix
// - Error: Red text with [ERROR] prefix
// - Success: Green text with [SUCCESS] prefix
//
// Messages directed specifically to users (InfoToUser, WarningToUser) are
// always displayed regardless of verbosity settings.
//
// # File Logging
//
// When a log file is specified, all messages (regardless of verbosity settings)
// are written to the file. File logging includes timestamps and does not
// include ANSI color codes.
//
// # Resource Management
//
// The Logger interface provides a Close method that should be called before
// application termination to ensure all buffered logs are flushed to disk:
//
//	defer logger.Close()
//
// # Thread Safety
//
// The DefaultLogger implementation is safe for concurrent use by multiple
// goroutines. All logging methods can be called from different goroutines
// without additional synchronization.
package logger
