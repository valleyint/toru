package lix

import (
	"fmt"
	"os"
	"os/exec"
	"toru/pkg/cli"
)

// LixRunner implements the backend.Runner interface for Lix/Flakes.
type LixRunner struct {
	ExecFunc func(name string, args ...string) error
}

// New returns a new LixRunner instance.
func New() *LixRunner {
	l := &LixRunner{}
	l.ExecFunc = l.defaultRunCommand
	return l
}

// Name returns the backend identifier.
func (l *LixRunner) Name() string {
	return "lix"
}

func (l *LixRunner) defaultRunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %s %v: %w", name, args, err)
	}
	return nil
}

func (l *LixRunner) runCommand(name string, args ...string) error {
	return l.ExecFunc(name, args...)
}

// Execute translates the parsed pacman command into Lix/Flake equivalents.
func (l *LixRunner) Execute(cmd *cli.ParsedCommand) error {
	if cmd.Modifiers.Help {
		return l.runCommand("nix", "profile", "--help")
	}

	switch cmd.Operation {
	case cli.OpSync:
		return l.executeSync(cmd)
	case cli.OpRemove:
		return l.executeRemove(cmd)
	case cli.OpQuery:
		return l.executeQuery(cmd)
	case cli.OpDatabase:
		return l.executeDatabase(cmd)
	case cli.OpFiles:
		return l.executeFiles(cmd)
	case cli.OpDeptest:
		return l.executeDeptest(cmd)
	case cli.OpUpgrade:
		return l.executeUpgrade(cmd)
	default:
		return fmt.Errorf("unsupported lix operation: %v", cmd.Operation)
	}
}

func (l *LixRunner) executeSync(cmd *cli.ParsedCommand) error {
	// Syncing databases (-Sy) is a no-op in flakes since the registry is live
	if cmd.Modifiers.Refresh > 0 && !cmd.Modifiers.SysUpgrade && len(cmd.Targets) == 0 {
		fmt.Println("Lix backend: Registry is always live. Databases are up to date.")
		return nil
	}

	// System Upgrade (-Syu)
	if cmd.Modifiers.SysUpgrade {
		fmt.Println("Lix backend: Upgrading profile...")
		if err := l.runCommand("nix", "profile", "upgrade", ".*"); err != nil {
			return err
		}
	}

	// Clean cache (-Sc)
	if cmd.Modifiers.Clean > 0 {
		fmt.Println("Lix backend: Collecting garbage...")
		if err := l.runCommand("nix", "store", "gc"); err != nil {
			return err
		}
	}

	// Search (-Ss)
	if cmd.Modifiers.Search && len(cmd.Targets) > 0 {
		args := append([]string{"search", "nixpkgs"}, cmd.Targets...)
		return l.runCommand("nix", args...)
	}

	// Install packages
	if len(cmd.Targets) > 0 && !cmd.Modifiers.Search {
		args := []string{"profile", "install"}
		for _, pkg := range cmd.Targets {
			args = append(args, fmt.Sprintf("nixpkgs#%s", pkg))
		}
		return l.runCommand("nix", args...)
	}

	return nil
}

func (l *LixRunner) executeRemove(cmd *cli.ParsedCommand) error {
	if len(cmd.Targets) == 0 {
		return fmt.Errorf("no targets specified for removal")
	}

	args := append([]string{"profile", "remove"}, cmd.Targets...)
	return l.runCommand("nix", args...)
}

func (l *LixRunner) executeQuery(cmd *cli.ParsedCommand) error {
	// Check for out-of-date packages (-Qu)
	if cmd.Modifiers.SysUpgrade {
		fmt.Println("Lix backend: Checking for updates (dry-run)...")
		return l.runCommand("nix", "profile", "upgrade", ".*", "--dry-run")
	}

	// Search installed packages (-Qs)
	if cmd.Modifiers.Search {
		return l.runCommand("nix", "profile", "list")
	}

	return l.runCommand("nix", "profile", "list")
}

func (l *LixRunner) executeDatabase(cmd *cli.ParsedCommand) error {
	fmt.Println("Warning: Flakes do not natively support pacman database manipulation outside of declarative config.")
	return nil
}

func (l *LixRunner) executeFiles(cmd *cli.ParsedCommand) error {
	if len(cmd.Targets) > 0 {
		args := append([]string{"locate"}, cmd.Targets...)
		return l.runCommand("nix", args...)
	}
	return nil
}

func (l *LixRunner) executeDeptest(cmd *cli.ParsedCommand) error {
	fmt.Println("Lix backend: Dependencies are structurally guaranteed.")
	return nil
}

func (l *LixRunner) executeUpgrade(cmd *cli.ParsedCommand) error {
	if len(cmd.Targets) == 0 {
		return fmt.Errorf("no targets specified for local upgrade")
	}
	args := []string{"profile", "install"}
	// Treat targets as local flake directories or tarballs
	for _, pkg := range cmd.Targets {
		args = append(args, pkg) 
	}
	return l.runCommand("nix", args...)
}
