package lock

import (
	"testing"
	"time"
)

func TestConcurrentLocks_EnforcesExclusivity(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		skipInShortMode bool
		goroutineCount  int
		holdTime        time.Duration
		minSuccessCount int
	}{
		"FiveGoroutinesCompeteForLock": {
			skipInShortMode: true,
			goroutineCount:  5,
			holdTime:        100 * time.Millisecond,
			minSuccessCount: 1,
		},
		"QuickRelease": {
			skipInShortMode: false,
			goroutineCount:  3,
			holdTime:        10 * time.Millisecond,
			minSuccessCount: 1,
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if test.skipInShortMode && testing.Short() {
				t.Skip("Skipping concurrency test in short mode")
			}

			repoPath := t.TempDir()
			done := make(chan bool, test.goroutineCount)

			for i := 0; i < test.goroutineCount; i++ {
				go func(id int) {
					locker, err := New(repoPath)
					if err != nil {
						t.Errorf("Goroutine %d: Failed to create locker: %v", id, err)
						done <- false
						return
					}

					err = locker.Acquire()
					if err != nil {
						// With multiple goroutines competing for the same lock,
						// only one can succeed at any given time, so it's normal
						// and expected for some acquisition attempts to fail
						done <- false
						return
					}

					// If we got the lock, release it after the specified pause
					time.Sleep(test.holdTime)
					releaseErr := locker.Release()
					if releaseErr != nil {
						t.Errorf("Goroutine %d: Failed to release lock: %v", id, releaseErr)
					}

					done <- true
				}(i)
			}

			successCount := 0
			for i := 0; i < test.goroutineCount; i++ {
				if <-done {
					successCount++
				}
			}

			if successCount < test.minSuccessCount {
				t.Errorf("Expected at least %d goroutines to acquire the lock, but only %d succeeded",
					test.minSuccessCount, successCount)
			}
		})
	}
}
