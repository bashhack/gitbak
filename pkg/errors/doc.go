// Package errors provides error handling utilities for the gitbak application.
//
// This package implements specialized error types and error handling functions
// to improve error management throughout the application. It focuses on
// providing rich context for errors while maintaining compatibility with
// the standard error handling practices.
//
// # Features
//
//   - Error wrapping with context
//   - Standardized error formatting
//
// # Usage
//
// Basic error wrapping:
//
//	if err != nil {
//	    return errors.Wrap(err, "failed to open file")
//	}
//
// Creating a new error:
//
//	if value < 0 {
//	    return errors.New("value must be non-negative")
//	}
//
// # Error Wrapping
//
// The package uses standard error wrapping conventions, allowing errors to be
// unwrapped and inspected using errors.Is and errors.As.
//
// # Error Formatting
//
// Errors created with this package provide consistent, formatted error messages
// that include:
//
//   - The error context (what operation was being attempted)
//   - The underlying error message
//   - Optional formatting with variable values
//
// # Compatibility
//
// The package is fully compatible with the standard library errors package
// and can be used as a drop-in replacement with additional functionality.
//
// # Thread Safety
//
// All types and functions in this package are safe for concurrent use
// by multiple goroutines.
package errors
