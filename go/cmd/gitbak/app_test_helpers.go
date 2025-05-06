package main

import (
	"context"
	"fmt"
	"github.com/bashhack/gitbak/internal/common"
	"github.com/bashhack/gitbak/internal/config"
)

// MockGitbaker implements a mock of the Gitbaker interface for testing
type MockGitbaker struct {
	SummaryCalled bool
	RunCalled     bool
	RunErr        error
}

func (m *MockGitbaker) PrintSummary() {
	m.SummaryCalled = true
}

func (m *MockGitbaker) Run(ctx context.Context) error {
	m.RunCalled = true
	return m.RunErr
}

// MockLocker implements the Locker interface for testing
type MockLocker struct {
	AcquireErr    error
	ReleaseErr    error
	AcquireCalled bool
	ReleaseCalled bool
	Released      bool
}

func (m *MockLocker) Acquire() error {
	m.AcquireCalled = true
	return m.AcquireErr
}

func (m *MockLocker) Release() error {
	m.ReleaseCalled = true
	m.Released = true
	return m.ReleaseErr
}

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	InfoCalled          bool
	InfoToUserCalled    bool
	WarningCalled       bool
	WarningToUserCalled bool
	ErrorCalled         bool
	SuccessCalled       bool
	StatusCalled        bool
	LastMessage         string
}

// Standard logging methods

// Info logs an info message
func (m *MockLogger) Info(format string, args ...interface{}) {
	m.InfoCalled = true
	m.LastMessage = fmt.Sprintf(format, args...)
}

// Warning logs an warning message
func (m *MockLogger) Warning(format string, args ...interface{}) {
	m.WarningCalled = true
	m.LastMessage = fmt.Sprintf(format, args...)
}

// Error logs an error message
func (m *MockLogger) Error(format string, args ...interface{}) {
	m.ErrorCalled = true
	m.LastMessage = fmt.Sprintf(format, args...)
}

// Enhanced user-facing logging methods

// InfoToUser logs an info message to the user
func (m *MockLogger) InfoToUser(format string, args ...interface{}) {
	m.InfoToUserCalled = true
	m.LastMessage = fmt.Sprintf(format, args...)
}

// WarningToUser logs a warning message to the user
func (m *MockLogger) WarningToUser(format string, args ...interface{}) {
	m.WarningToUserCalled = true
	m.LastMessage = fmt.Sprintf(format, args...)
}

// Success logs a success message
func (m *MockLogger) Success(format string, args ...interface{}) {
	m.SuccessCalled = true
	m.LastMessage = fmt.Sprintf(format, args...)
}

// StatusMessage logs a status message
func (m *MockLogger) StatusMessage(format string, args ...interface{}) {
	m.StatusCalled = true
	m.LastMessage = fmt.Sprintf(format, args...)
}

// Testing helper functions

// NewTestApp creates a new App with default test settings
func NewTestApp() *App {
	app := NewDefaultApp(config.VersionInfo{})

	app.exit = func(int) {}

	return app
}

// WithMockLocker adds a mock locker to the app
func WithMockLocker(app *App, mockLocker *MockLocker) *App {
	app.Locker = mockLocker
	return app
}

// WithMockLogger adds a mock logger to the app
func WithMockLogger(app *App, mockLogger common.Logger) *App {
	app.Logger = mockLogger
	return app
}

// WithIsRepository mocks the isRepository function
func WithIsRepository(app *App, fn func(string) bool) *App {
	app.isRepository = fn
	return app
}

// WithExecLookPath mocks the execLookPath function
func WithExecLookPath(app *App, fn func(string) (string, error)) *App {
	app.execLookPath = fn
	return app
}

// WithExit mocks the exit function
func WithExit(app *App, fn func(int)) *App {
	app.exit = fn
	return app
}
