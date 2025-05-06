package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temporary directory: %v", err)
		}
	}()

	logFile := filepath.Join(tempDir, "test.log")

	logger := New(false, logFile, true)
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

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "gitbak debug logging started") {
		t.Error("Expected initial message to be logged")
	}
}

func TestLogging(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temporary directory: %v", err)
		}
	}()

	logFile := filepath.Join(tempDir, "test.log")

	logger := New(true, logFile, true)

	logger.Info("Test info message")

	logger.Warning("Test warning message")

	logger.Error("Test error message")

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

	if err := os.Remove(logFile); err != nil && !os.IsNotExist(err) {
		t.Logf("Failed to remove log file: %v", err)
	}
	logger = New(false, logFile, true)

	logger.Info("This should not be logged")
	logger.Warning("This should not be logged")
	logger.Error("This should not be logged")

	if _, err := os.Stat(logFile); err == nil {
		t.Error("Expected no log file to be created when debug is disabled")
	}
}
