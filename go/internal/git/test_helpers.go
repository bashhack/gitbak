package git

import (
	"github.com/bashhack/gitbak/internal/common"
)

// setupTestGitbak creates a Gitbak instance for testing with default mocks
// In test context, we panic on validation errors since tests should be providing valid configs
func setupTestGitbak(config GitbakConfig, logger common.Logger) *Gitbak {
	executor := NewExecExecutor()

	var interactor UserInteractor
	if config.NonInteractive {
		interactor = NewNonInteractiveInteractor()
	} else {
		interactor = NewDefaultInteractor(logger)
	}

	gb, err := NewGitbakWithDeps(config, logger, executor, interactor)
	if err != nil {
		panic("Test setup failed: " + err.Error())
	}
	return gb
}
