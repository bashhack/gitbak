package git

import (
	"bytes"
	"testing"

	"github.com/bashhack/gitbak/internal/logger"
)

// TestDefaultInteractor tests the DefaultInteractor implementation
func TestDefaultInteractor(t *testing.T) {
	t.Parallel()
	log := logger.New(true, "", true)

	t.Run("DefaultInteractor constructor", func(t *testing.T) {
		interactor := NewDefaultInteractor(log)

		if interactor == nil {
			t.Fatal("NewDefaultInteractor returned nil")
		}

		if interactor.Logger != log {
			t.Errorf("Expected logger to be set, but was different instance")
		}

		if interactor.Reader == nil {
			t.Errorf("Expected Reader to be set, got nil")
		}

		if interactor.Writer == nil {
			t.Errorf("Expected Writer to be set, got nil")
		}
	})

	t.Run("PromptYesNo responds to yes", func(t *testing.T) {
		// Create a buffer to simulate user input
		input := bytes.NewBufferString("yes\n")
		output := &bytes.Buffer{}

		interactor := &DefaultInteractor{
			Reader: input,
			Writer: output,
			Logger: log,
		}

		result := interactor.PromptYesNo("Test question")

		if !result {
			t.Errorf("Expected true for 'yes' input, got false")
		}
	})

	t.Run("PromptYesNo responds to no", func(t *testing.T) {
		input := bytes.NewBufferString("no\n")
		output := &bytes.Buffer{}

		interactor := &DefaultInteractor{
			Reader: input,
			Writer: output,
			Logger: log,
		}

		result := interactor.PromptYesNo("Test question")

		if result {
			t.Errorf("Expected false for 'no' input, got true")
		}
	})

	t.Run("PromptYesNo handles error", func(t *testing.T) {
		// Create a buffer that will return an error on read
		errorReader := &errorReadCloser{}
		output := &bytes.Buffer{}

		// Create an interactor with our error reader
		interactor := &DefaultInteractor{
			Reader: errorReader,
			Writer: output,
			Logger: log,
		}

		result := interactor.PromptYesNo("Test question")

		if result {
			t.Errorf("Expected false when read fails, got true")
		}
	})
}

// TestNonInteractiveInteractor tests the NonInteractiveInteractor implementation
func TestNonInteractiveInteractor(t *testing.T) {
	t.Parallel()
	t.Run("NonInteractiveInteractor constructor", func(t *testing.T) {
		interactor := NewNonInteractiveInteractor()

		if interactor == nil {
			t.Fatal("NewNonInteractiveInteractor returned nil")
		}
	})

	t.Run("PromptYesNo always returns false", func(t *testing.T) {
		interactor := NewNonInteractiveInteractor()

		result1 := interactor.PromptYesNo("Question 1")
		result2 := interactor.PromptYesNo("Question 2")

		if result1 {
			t.Errorf("Expected false for any question, got true")
		}

		if result2 {
			t.Errorf("Expected false for any question, got true")
		}
	})
}

// errorReadCloser is a mock io.Reader that always returns an error
type errorReadCloser struct{}

func (e *errorReadCloser) Read(p []byte) (n int, err error) {
	return 0, bytes.ErrTooLarge // Return any error
}
