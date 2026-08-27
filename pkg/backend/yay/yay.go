package yay

import (
	"fmt"
	"os"
	"os/exec"
	"toru/pkg/cli"
)

// YayRunner implements the backend.Runner interface for Yay.
type YayRunner struct {
	ExecFunc func(args ...string) error
}

// New returns a new YayRunner instance.
func New() *YayRunner {
	y := &YayRunner{}
	y.ExecFunc = y.defaultRunCommand
	return y
}

// Name returns the backend identifier.
func (y *YayRunner) Name() string {
	return "yay"
}

// Execute translates the parsed command back into native yay/pacman syntax.
func (y *YayRunner) Execute(cmd *cli.ParsedCommand) error {
	var args []string

	// Reconstruct the primary operation
	switch cmd.Operation {
	case cli.OpSync:
		args = append(args, "-S")
	case cli.OpRemove:
		args = append(args, "-R")
	case cli.OpQuery:
		args = append(args, "-Q")
	case cli.OpDatabase:
		args = append(args, "-D")
	case cli.OpUpgrade:
		args = append(args, "-U")
	case cli.OpFiles:
		args = append(args, "-F")
	case cli.OpDeptest:
		args = append(args, "-T")
	default:
		return fmt.Errorf("unknown operation: %s", cmd.Operation)
	}

	// Reconstruct modifiers
	modifiers := ""
	for i := 0; i < cmd.Modifiers.Refresh; i++ {
		modifiers += "y"
	}
	if cmd.Modifiers.SysUpgrade {
		modifiers += "u"
	}
	for i := 0; i < cmd.Modifiers.Clean; i++ {
		modifiers += "c"
	}
	if cmd.Modifiers.Search {
		modifiers += "s"
	}
	if cmd.Modifiers.Info {
		modifiers += "i"
	}
	if cmd.Modifiers.Recursive {
		modifiers += "s"
	}
	if cmd.Modifiers.Unneeded {
		modifiers += "n"
	}
	if cmd.Modifiers.Cascade {
		modifiers += "c"
	}
	if cmd.Modifiers.Quiet {
		modifiers += "q"
	}
	if cmd.Modifiers.Help {
		modifiers += "h"
	}

	// Combine primary operation with short flag modifiers
	if modifiers != "" {
		args[0] = args[0] + modifiers
	}

	// Append targets
	args = append(args, cmd.Targets...)

	return y.runCommand(args...)
}

func (y *YayRunner) defaultRunCommand(args ...string) error {
	cmd := exec.Command("yay", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run yay %v: %w", args, err)
	}
	return nil
}

func (y *YayRunner) runCommand(args ...string) error {
	return y.ExecFunc(args...)
}
