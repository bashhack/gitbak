package git

import (
	"github.com/bashhack/gitbak/internal/common"
)

// setupTestGitbak creates a Gitbak instance for testing with default mocks
func setupTestGitbak(config GitbakConfig, logger common.Logger) *Gitbak {
	executor := NewExecExecutor()

	var interactor UserInteractor
	if config.NonInteractive {
		interactor = NewNonInteractiveInteractor()
	} else {
		interactor = NewDefaultInteractor(logger)
	}

	return NewGitbakWithDeps(config, logger, executor, interactor)
}
