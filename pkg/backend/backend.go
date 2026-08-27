package backend

import (
	"fmt"
	"toru/pkg/cli"
)

// Runner defines the interface for backend package managers.
type Runner interface {
	// Execute takes the parsed CLI intent, translates the packages via the
	// cache/Repology (if necessary), constructs the native command string,
	// and runs it using os/exec.
	Execute(cmd *cli.ParsedCommand) error

	// Name returns the identifier of the backend (e.g., "nix", "yay").
	Name() string
}

var registry = make(map[string]Runner)

// Register makes a backend available for use.
func Register(runner Runner) {
	registry[runner.Name()] = runner
}

// Get retrieves a backend by its name.
func Get(name string) (Runner, error) {
	if runner, exists := registry[name]; exists {
		return runner, nil
	}
	return nil, fmt.Errorf("backend %s not found", name)
}
