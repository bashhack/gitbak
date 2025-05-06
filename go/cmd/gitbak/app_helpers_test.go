package main

import (
	"testing"
)

// TestMockLoggerMethods tests all the MockLogger methods
func TestMockLoggerMethods(t *testing.T) {
	mockLogger := &MockLogger{}

	t.Run("Warning", func(t *testing.T) {
		mockLogger.Warning("Test warning: %s", "message")

		if !mockLogger.WarningCalled {
			t.Error("Warning method did not set WarningCalled flag")
		}

		if mockLogger.LastMessage != "Test warning: message" {
			t.Errorf("Warning method did not set LastMessage correctly, got: %s", mockLogger.LastMessage)
		}
	})

	t.Run("InfoToUser", func(t *testing.T) {
		mockLogger.InfoToUser("Test info: %s", "user message")

		if !mockLogger.InfoToUserCalled {
			t.Error("InfoToUser method did not set InfoToUserCalled flag")
		}

		if mockLogger.LastMessage != "Test info: user message" {
			t.Errorf("InfoToUser method did not set LastMessage correctly, got: %s", mockLogger.LastMessage)
		}
	})

	t.Run("WarningToUser", func(t *testing.T) {
		mockLogger.WarningToUser("Test warning: %s", "user warning")

		if !mockLogger.WarningToUserCalled {
			t.Error("WarningToUser method did not set WarningToUserCalled flag")
		}

		if mockLogger.LastMessage != "Test warning: user warning" {
			t.Errorf("WarningToUser method did not set LastMessage correctly, got: %s", mockLogger.LastMessage)
		}
	})

	t.Run("Success", func(t *testing.T) {
		mockLogger.Success("Test success: %s", "completed")

		if !mockLogger.SuccessCalled {
			t.Error("Success method did not set SuccessCalled flag")
		}

		if mockLogger.LastMessage != "Test success: completed" {
			t.Errorf("Success method did not set LastMessage correctly, got: %s", mockLogger.LastMessage)
		}
	})

	t.Run("StatusMessage", func(t *testing.T) {
		mockLogger.StatusMessage("Test status: %s", "in progress")

		if !mockLogger.StatusCalled {
			t.Error("StatusMessage method did not set StatusCalled flag")
		}

		if mockLogger.LastMessage != "Test status: in progress" {
			t.Errorf("StatusMessage method did not set LastMessage correctly, got: %s", mockLogger.LastMessage)
		}
	})
}

// TestWithExit tests the WithExit helper function
func TestWithExit(t *testing.T) {
	app := NewTestApp()

	exitCalled := false
	exitCode := 0

	customExit := func(code int) {
		exitCalled = true
		exitCode = code
	}

	modifiedApp := WithExit(app, customExit)

	if modifiedApp != app {
		t.Error("WithExit did not return the same app instance")
	}

	modifiedApp.exit(42)

	if !exitCalled {
		t.Error("Exit function was not called")
	}

	if exitCode != 42 {
		t.Errorf("Exit function did not receive correct code, expected 42, got %d", exitCode)
	}
}
