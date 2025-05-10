// Package common provides shared interfaces and utilities used throughout the gitbak application.
//
// This package contains core interfaces and common functionality that is reused across
// different components of the gitbak system. It serves as a central location for
// application-wide contracts that help standardize interactions between packages.
//
// # Core Components
//
// - Logger: Interface defining standardized logging methods used throughout the application
//
// # Logger Interface
//
// The Logger interface provides a standardized way for components to emit log messages
// at different levels of importance and visibility. It separates internal logging from
// user-facing messages, allowing for consistent handling of output across the application.
//
// # Usage
//
// The Logger interface is typically injected into components that need logging capabilities:
//
//	type MyComponent struct {
//	    logger common.Logger
//	    // other fields
//	}
//
//	func NewMyComponent(logger common.Logger) *MyComponent {
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
// # Design Principles
//
// This package follows these design principles:
//
// - Minimal Dependencies: The common package should have no dependencies on other internal packages
// - Interface-Based Design: Favors interfaces over concrete implementations
// - Separation of Concerns: Clearly separates user-facing and internal functionality
package common