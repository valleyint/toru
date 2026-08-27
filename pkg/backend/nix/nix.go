package nix

import (
	"fmt"
	"os"
	"os/exec"
	"toru/pkg/cli"
)

// NixRunner implements the backend.Runner interface for Nix.
type NixRunner struct {
	ExecFunc func(name string, args ...string) error
}

// New returns a new NixRunner instance.
func New() *NixRunner {
	n := &NixRunner{}
	n.ExecFunc = n.defaultRunCommand
	return n
}

// Name returns the backend identifier.
func (n *NixRunner) Name() string {
	return "nix"
}

func (n *NixRunner) defaultRunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %s %v: %w", name, args, err)
	}
	return nil
}

func (n *NixRunner) runCommand(name string, args ...string) error {
	return n.ExecFunc(name, args...)
}

// Execute translates the parsed pacman command into Nix equivalents.
func (n *NixRunner) Execute(cmd *cli.ParsedCommand) error {
	// Note: cmd.Targets are assumed to have been translated into the target backend
	// nomenclature before this function is called.

	switch cmd.Operation {
	case cli.OpSync:
		return n.executeSync(cmd)
	case cli.OpRemove:
		return n.executeRemove(cmd)
	case cli.OpQuery:
		return n.executeQuery(cmd)
	case cli.OpDatabase:
		return n.executeDatabase(cmd)
	case cli.OpUpgrade:
		return n.executeUpgrade(cmd)
	case cli.OpFiles:
		return n.executeFiles(cmd)
	case cli.OpDeptest:
		return n.executeDeptest(cmd)
	case cli.OpUnknown:
		if cmd.Modifiers.Help {
			fmt.Println("Toru backend: nix")
			fmt.Println("Provides pacman-compatible wrapper for nix-env")
			return nil
		}
		return fmt.Errorf("no valid operation specified")
	default:
		return fmt.Errorf("operation %s is currently unsupported by the nix backend", cmd.Operation)
	}
}

func (n *NixRunner) executeSync(cmd *cli.ParsedCommand) error {
	// Handle full system update: pacman -Syu
	if cmd.Modifiers.SysUpgrade {
		fmt.Println("Updating Nix channels...")
		if err := n.runCommand("nix-channel", "--update"); err != nil {
			return err
		}
		fmt.Println("Updating Nix packages...")
		if err := n.runCommand("nix-env", "-u"); err != nil {
			return err
		}
		if len(cmd.Targets) == 0 {
			return nil
		}
	}

	// Handle cache clean: pacman -Sc
	if cmd.Modifiers.Clean > 0 {
		fmt.Println("Collecting Nix garbage...")
		if err := n.runCommand("nix-collect-garbage"); err != nil {
			return err
		}
		if len(cmd.Targets) == 0 {
			return nil
		}
	}

	// Handle information: pacman -Si
	if cmd.Modifiers.Info {
		if len(cmd.Targets) == 0 {
			return fmt.Errorf("no targets specified for info")
		}
		args := append([]string{"-qaP", "--description"}, cmd.Targets...)
		return n.runCommand("nix-env", args...)
	}

	// Handle installation: pacman -S pkg1 pkg2
	if len(cmd.Targets) > 0 {
		args := []string{"-iA"}
		for _, pkg := range cmd.Targets {
			args = append(args, fmt.Sprintf("nixpkgs.%s", pkg))
		}
		return n.runCommand("nix-env", args...)
	}

	return nil
}

func (n *NixRunner) executeRemove(cmd *cli.ParsedCommand) error {
	if len(cmd.Targets) == 0 {
		return fmt.Errorf("no targets specified for remove")
	}

	// Nix handles dependencies functionally, so Cascade/Unneeded flags (-Rc, -Rns)
	// don't map directly to the uninstall command. We just uninstall the package,
	// and a later `nix-collect-garbage` cleans up unused deps.
	args := append([]string{"-e"}, cmd.Targets...)
	return n.runCommand("nix-env", args...)
}

func (n *NixRunner) executeQuery(cmd *cli.ParsedCommand) error {
	// -Qs <pkg>
	if cmd.Modifiers.Search {
		args := []string{"-qaP"}
		if len(cmd.Targets) > 0 {
			// Nix-env -qaP interprets arguments as regex
			for _, pkg := range cmd.Targets {
				args = append(args, fmt.Sprintf(".*%s.*", pkg))
			}
		}
		return n.runCommand("nix-env", args...)
	}

	// -Qi <pkg>
	if cmd.Modifiers.Info {
		args := append([]string{"-qaP", "--description"}, cmd.Targets...)
		return n.runCommand("nix-env", args...)
	}

	// -Q list installed
	args := append([]string{"-q"}, cmd.Targets...)
	return n.runCommand("nix-env", args...)
}

func (n *NixRunner) executeDatabase(cmd *cli.ParsedCommand) error {
	fmt.Println("Warning: Nix does not support pacman database manipulation (e.g. changing install reason).")
	return nil
}

func (n *NixRunner) executeUpgrade(cmd *cli.ParsedCommand) error {
	// pacman -U <file> maps to nix-env -i <file> or nix-env -i -f <file>
	if len(cmd.Targets) == 0 {
		return fmt.Errorf("no targets specified for upgrade")
	}
	args := append([]string{"-i", "-f"}, cmd.Targets...)
	return n.runCommand("nix-env", args...)
}

func (n *NixRunner) executeFiles(cmd *cli.ParsedCommand) error {
	// pacman -F <file> searches for packages containing the file
	// Nix uses nix-locate for this, which requires nix-index
	if len(cmd.Targets) == 0 {
		// pacman -Fy maps to nix-index
		if cmd.Modifiers.Refresh > 0 {
			fmt.Println("Updating Nix index...")
			return n.runCommand("nix-index")
		}
		return fmt.Errorf("no targets specified for files")
	}
	
	args := append([]string{"-w"}, cmd.Targets...) // -w searches for whole words
	err := n.runCommand("nix-locate", args...)
	if err != nil {
		return fmt.Errorf("nix-locate failed (make sure nix-index is installed and populated): %w", err)
	}
	return nil
}

func (n *NixRunner) executeDeptest(cmd *cli.ParsedCommand) error {
	// pacman -T <pkg> checks if dependencies are satisfied
	// Nix guarantees dependencies are satisfied if a package is installed.
	fmt.Println("Nix backend: All requested dependencies are functionally guaranteed by the store.")
	return nil
}
