//go:build test
// +build test

package git

import (
	"context"
)

// RunSingleIteration is an exported version of the internal Gitbak logic for testing.
// It runs a single iteration of the gitbak process without the infinite loop.
// This function is only available in test builds.
func (g *Gitbak) RunSingleIteration(ctx context.Context) error {
	var err error
	g.originalBranch, err = g.getCurrentBranch(ctx)
	if err != nil {
		g.logger.Error("Failed to get current branch: %v", err)
		return err
	}

	if g.config.ContinueSession {
		if err := g.setupContinueSession(ctx); err != nil {
			return err
		}
	} else if g.config.CreateBranch {
		if err := g.setupNewBranchSession(ctx); err != nil {
			return err
		}
	} else {
		g.setupCurrentBranchSession(ctx)
	}

	g.displayStartupInfo()

	// Initialize commit counter based on commits count, exactly as in monitoringLoop
	// g.commitsCount is already set appropriately by setupContinueSession if we're in continue mode
	commitCounter := g.commitsCount + 1

	errorState := struct {
		consecutiveErrors int
		lastErrorMsg      string
	}{}

	// NOTE: We don't need to increment g.commitsCount here because
	// checkAndCommitChanges already increments it when creating a commit
	err = g.tryOperation(ctx, &errorState, func() error {
		var commitWasCreated bool
		return g.checkAndCommitChanges(ctx, commitCounter, &commitWasCreated)
	})

	return err
}

// TestRetryLoop is a test helper that runs multiple iterations of retry logic
// for testing purposes. This avoids duplicating retry logic in tests.
func (g *Gitbak) TestRetryLoop(ctx context.Context, iterations int) error {
	errorState := struct {
		consecutiveErrors int
		lastErrorMsg      string
	}{}

	commitCounter := g.commitsCount + 1

	// Simulate multiple rapid iterations
	for i := 0; i < iterations; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			opErr := g.tryOperation(ctx, &errorState, func() error {
				var commitWasCreated bool
				return g.checkAndCommitChanges(ctx, commitCounter, &commitWasCreated)
			})

			// If the operation hit max retries, bubble up the fatal error
			if opErr != nil && errorState.consecutiveErrors > g.config.MaxRetries {
				return opErr
			}
		}
	}
	return nil
}
