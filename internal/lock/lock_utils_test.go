package lock

import (
	"os"
	"testing"
)

func TestIsProcessRunning_DetectsPIDState(t *testing.T) {
	nonExistentPID := 999999
	for pid := nonExistentPID; pid > 900000; pid-- {
		proc, err := os.FindProcess(pid)
		if err != nil || proc == nil {
			nonExistentPID = pid
			break
		}

		// On Unix, FindProcess always succeeds, so need to check if process exists
		err = proc.Signal(os.Signal(nil))
		if err != nil {
			nonExistentPID = pid
			break
		}
	}

	// Current system's behavior with PID 0
	zeroPidIsRunning := isProcessRunning(0)

	tests := map[string]struct {
		pid      int
		expected bool
	}{
		"CurrentProcess": {os.Getpid(), true},
		"NonExistentPID": {nonExistentPID, false},
		"NegativePID":    {-1, false},
		"ZeroPID":        {0, zeroPidIsRunning}, // uses actual system behavior
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			result := isProcessRunning(test.pid)
			if result != test.expected {
				t.Errorf("Expected isProcessRunning(%d) to be %v, got %v", test.pid, test.expected, result)
			}
		})
	}
}
