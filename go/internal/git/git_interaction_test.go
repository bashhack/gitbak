package git

import (
	"bytes"
	"github.com/bashhack/gitbak/internal/logger"
	"io"
	"path/filepath"
	"testing"
)

// TestInteractionScenarios tests various interaction methods and behaviors
func TestInteractionScenarios(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFunc    func(t *testing.T) (UserInteractor, *bytes.Buffer)
		promptText   string
		expectedResp bool
	}{
		"DefaultInteractorYesResponse": {
			setupFunc: func(t *testing.T) (UserInteractor, *bytes.Buffer) {
				log := logger.New(true, "", true)
				reader := bytes.NewBufferString("yes\n")
				output := &bytes.Buffer{}

				interactor := &DefaultInteractor{
					Reader: reader,
					Writer: output,
					Logger: log,
				}

				return interactor, output
			},
			promptText:   "Do you want to continue?",
			expectedResp: true,
		},
		"DefaultInteractorNoResponse": {
			setupFunc: func(t *testing.T) (UserInteractor, *bytes.Buffer) {
				log := logger.New(true, "", true)
				reader := bytes.NewBufferString("no\n")
				output := &bytes.Buffer{}

				interactor := &DefaultInteractor{
					Reader: reader,
					Writer: output,
					Logger: log,
				}

				return interactor, output
			},
			promptText:   "Do you want to continue?",
			expectedResp: false,
		},
		"NonInteractiveInteractorResponse": {
			setupFunc: func(t *testing.T) (UserInteractor, *bytes.Buffer) {
				output := &bytes.Buffer{}
				interactor := NewNonInteractiveInteractor()
				return interactor, output
			},
			promptText:   "Any prompt should return false",
			expectedResp: false,
		},
		"RealGitbakNonInteractiveConfiguration": {
			setupFunc: func(t *testing.T) (UserInteractor, *bytes.Buffer) {
				repoPath := setupTestRepo(t)

				tempLogDir := t.TempDir()
				tempLogFile := filepath.Join(tempLogDir, "gitbak-test-prompt.log")
				log := logger.New(true, tempLogFile, true)

				output := &bytes.Buffer{}

				gb := setupTestGitbak(
					GitbakConfig{
						RepoPath:        repoPath,
						IntervalMinutes: 1,
						BranchName:      "gitbak-prompt-branch",
						CommitPrefix:    "[gitbak-prompt] Commit",
						CreateBranch:    true,
						Verbose:         true,
						ShowNoChanges:   true,
						ContinueSession: false,
						NonInteractive:  true,
					},
					log,
				)

				return gb.interactor, output
			},
			promptText:   "Test question?",
			expectedResp: false,
		},
		"DefaultInteractorErrorHandling": {
			setupFunc: func(t *testing.T) (UserInteractor, *bytes.Buffer) {
				log := logger.New(true, "", true)
				reader := &errorReadCloser{} // Custom implementation that always fails
				output := &bytes.Buffer{}

				interactor := &DefaultInteractor{
					Reader: reader,
					Writer: output,
					Logger: log,
				}

				return interactor, output
			},
			promptText:   "Should handle error gracefully",
			expectedResp: false, // Default to false on error
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			interactor, output := test.setupFunc(t)

			result := interactor.PromptYesNo(test.promptText)

			if result != test.expectedResp {
				t.Errorf("Expected response %v, got %v", test.expectedResp, result)
			}

			if _, isDefault := interactor.(*DefaultInteractor); isDefault && output.Len() > 0 {
				promptOutput := output.String()
				if len(promptOutput) == 0 {
					t.Errorf("Expected prompt to be written to output")
				}
			}
		})
	}
}

// errorReadCloser is a mock io.Reader that always returns an error
type errorReadCloser struct{}

func (e *errorReadCloser) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF // Simulate read error
}

// TestGitbakPromptYesNoScenarios directly tests the promptYesNo method on Gitbak struct
func TestGitbakPromptYesNoScenarios(t *testing.T) {
	t.Parallel()

	repoPath := setupTestRepo(t)
	tempLogDir := t.TempDir()
	tempLogFile := filepath.Join(tempLogDir, "gitbak-direct-prompt.log")
	log := logger.New(true, tempLogFile, true)
	defer func() {
		if err := log.Close(); err != nil {
			t.Logf("Failed to close log: %v", err)
		}
	}()

	gb := setupTestGitbak(
		GitbakConfig{
			RepoPath:        repoPath,
			IntervalMinutes: 1,
			BranchName:      "gitbak-direct-prompt-branch",
			CommitPrefix:    "[gitbak-direct-prompt] Commit",
			CreateBranch:    true,
			Verbose:         true,
			ShowNoChanges:   true,
			ContinueSession: false,
			NonInteractive:  true,
		},
		log,
	)

	result := gb.promptYesNo("Direct test for Gitbak.promptYesNo")

	if result != false {
		t.Errorf("Expected promptYesNo to return false in non-interactive mode, got %v", result)
	}
}
