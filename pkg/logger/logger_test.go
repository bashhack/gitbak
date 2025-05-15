package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	logFile := filepath.Join(tempDir, "test.log")

	logger := New(false, logFile, true)
	defer func() {
		if err := logger.Close(); err != nil {
			t.Logf("Failed to close logger: %v", err)
		}
	}() // ensure buffers flushed & handle released
	if logger == nil {
		t.Fatal("Expected non-nil logger with debug disabled")
	}

	if _, err := os.Stat(logFile); err == nil {
		t.Error("Expected no log file to be created when debug is disabled")
	}

	logger = New(true, logFile, true)
	if logger == nil {
		t.Fatal("Expected non-nil logger with debug enabled")
	}

	if _, err := os.Stat(logFile); err != nil {
		t.Errorf("Expected log file to be created when debug is enabled: %v", err)
	}

	if err := logger.Close(); err != nil {
		t.Fatalf("Failed to close logger: %v", err)
	}

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "gitbak debug logging started") {
		t.Error("Expected initial message to be logged")
	}
}

func TestLogging(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	logFile := filepath.Join(tempDir, "test.log")

	logger := New(true, logFile, true)

	logger.Info("Test info message")

	logger.Warning("Test warning message")

	logger.Error("Test error message")

	if err := logger.Close(); err != nil {
		t.Fatalf("Failed to close logger: %v", err)
	}

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	if !strings.Contains(logContent, "Test info message") {
		t.Error("Expected info message to be logged")
	}

	if !strings.Contains(logContent, "Test warning message") {
		t.Error("Expected warning message to be logged")
	}

	if !strings.Contains(logContent, "Test error message") {
		t.Error("Expected error message to be logged")
	}

	// ...Logger is already closed from previous explicit close

	if err := os.Remove(logFile); err != nil && !os.IsNotExist(err) {
		t.Logf("Failed to remove log file: %v", err)
	}

	logger = New(false, logFile, true)
	defer func() {
		if err := logger.Close(); err != nil {
			t.Logf("Failed to close logger: %v", err)
		}
	}()

	logger.Info("This should not be logged")
	logger.Warning("This should not be logged")
	logger.Error("This should not be logged")

	if _, err := os.Stat(logFile); err == nil {
		t.Error("Expected no log file to be created when debug is disabled")
	}
}

func TestUserMessages(t *testing.T) {
	tempDir := t.TempDir()

	logFile := filepath.Join(tempDir, "test.log")

	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}

	logger := New(true, logFile, true).(*DefaultLogger)
	logger.SetStdout(stdoutBuf)
	logger.SetStderr(stderrBuf)
	defer func() {
		if err := logger.Close(); err != nil {
			t.Logf("Failed to close logger: %v", err)
		}
	}()

	t.Run("InfoToUser", func(t *testing.T) {
		stdoutBuf.Reset()
		logger.InfoToUser("Test info to user: %s", "message")
		output := stdoutBuf.String()

		if !strings.Contains(output, "ℹ️") || !strings.Contains(output, "Test info to user: message") {
			t.Errorf("InfoToUser did not produce expected output, got: %s", output)
		}

		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		if !strings.Contains(string(content), "Test info to user: message") {
			t.Error("InfoToUser message was not written to log file")
		}
	})

	t.Run("Success", func(t *testing.T) {
		stdoutBuf.Reset()
		logger.Success("Success message: %s", "completed")
		output := stdoutBuf.String()

		if !strings.Contains(output, "✅") || !strings.Contains(output, "Success message: completed") {
			t.Errorf("Success did not produce expected output, got: %s", output)
		}

		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		if !strings.Contains(string(content), "Success message: completed") {
			t.Error("Success message was not written to log file")
		}
	})

	t.Run("WarningToUser", func(t *testing.T) {
		stdoutBuf.Reset()
		logger.WarningToUser("Warning to user: %s", "be careful")
		output := stdoutBuf.String()

		if !strings.Contains(output, "⚠️") || !strings.Contains(output, "Warning to user: be careful") {
			t.Errorf("WarningToUser did not produce expected output, got: %s", output)
		}

		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		if !strings.Contains(string(content), "Warning to user: be careful") {
			t.Error("WarningToUser message was not written to log file")
		}
	})

	t.Run("StatusMessage", func(t *testing.T) {
		stdoutBuf.Reset()
		logger.StatusMessage("Status: %s", "in progress")
		output := stdoutBuf.String()

		if !strings.Contains(output, "Status: in progress") {
			t.Errorf("StatusMessage did not produce expected output, got: %s", output)
		}

		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		if strings.Contains(string(content), "Status: in progress") {
			t.Error("StatusMessage should not write to log file")
		}
	})

	t.Run("With debug disabled", func(t *testing.T) {
		if err := os.Remove(logFile); err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to remove log file: %v", err)
		}

		disabledLogger := New(false, logFile, true).(*DefaultLogger)
		disabledLogger.SetStdout(stdoutBuf)
		disabledLogger.SetStderr(stderrBuf)
		defer func() {
			if err := logger.Close(); err != nil {
				t.Logf("Failed to close logger: %v", err)
			}
		}()

		stdoutBuf.Reset()
		disabledLogger.InfoToUser("Info with logging disabled")
		disabledLogger.Success("Success with logging disabled")
		disabledLogger.WarningToUser("Warning with logging disabled")
		disabledLogger.StatusMessage("Status with logging disabled")

		output := stdoutBuf.String()
		if !strings.Contains(output, "Info with logging disabled") ||
			!strings.Contains(output, "Success with logging disabled") ||
			!strings.Contains(output, "Warning with logging disabled") ||
			!strings.Contains(output, "Status with logging disabled") {
			t.Errorf("User messages not printed to stdout with logging disabled, got: %s", output)
		}

		if _, err := os.Stat(logFile); err == nil {
			t.Error("Expected no log file to be created when debug is disabled")
		}
	})
}
