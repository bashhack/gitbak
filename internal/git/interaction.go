package git

import (
	"bufio"
	"io"
	"os"
	"strings"

	"github.com/bashhack/gitbak/internal/logger"
)

// UserInteractor defines an interface for interacting with the user
type UserInteractor interface {
	// PromptYesNo asks the user a yes/no question and returns their response
	PromptYesNo(question string) bool
}

// DefaultInteractor is the standard implementation of UserInteractor
// that reads from stdin and writes to stdout
type DefaultInteractor struct {
	Reader io.Reader
	Writer io.Writer
	Logger logger.Logger
}

// NewDefaultInteractor creates a new DefaultInteractor
func NewDefaultInteractor(logger logger.Logger) *DefaultInteractor {
	return &DefaultInteractor{
		Reader: os.Stdin,
		Writer: os.Stdout,
		Logger: logger,
	}
}

// PromptYesNo asks the user a yes/no question and returns their response
func (i *DefaultInteractor) PromptYesNo(question string) bool {
	i.Logger.StatusMessage("%s (y/n): ", question)

	reader := bufio.NewReader(i.Reader)
	answer, err := reader.ReadString('\n')
	if err != nil {
		// On error, default to 'no'
		return false
	}

	answer = strings.TrimSpace(answer)
	return strings.HasPrefix(strings.ToLower(answer), "y")
}

// NonInteractiveInteractor always returns default values without prompting
type NonInteractiveInteractor struct{}

// NewNonInteractiveInteractor creates a new NonInteractiveInteractor
func NewNonInteractiveInteractor() *NonInteractiveInteractor {
	return &NonInteractiveInteractor{}
}

// PromptYesNo always returns false without prompting
func (i *NonInteractiveInteractor) PromptYesNo(question string) bool {
	return false
}
