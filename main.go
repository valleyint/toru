package main

import (
	"fmt"
	"os"
	"path/filepath"

	"toru/pkg/backend"
	"toru/pkg/backend/lix"
	"toru/pkg/backend/nix"
	"toru/pkg/backend/yay"
	"toru/pkg/cache"
	"toru/pkg/cli"
	"toru/pkg/repology"
)

func init() {
	// Register the available backends
	backend.Register(lix.New())
	backend.Register(nix.New())
	backend.Register(yay.New())
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: toru [options] <operation> [targets...]")
		os.Exit(1)
	}

	// 1. Parse Arguments
	cmd, err := cli.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	// Early exit for help command
	if cmd.Modifiers.Help && cmd.Operation == cli.OpUnknown {
		fmt.Println("Toru: The Universal Arch Linux Package Wrapper")
		fmt.Println("Usage: toru [options] <operation> [targets...]")
		os.Exit(0)
	}

	// 2. Initialize Cache Store
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home dir: %v\n", err)
		os.Exit(1)
	}
	dbPath := filepath.Join(homeDir, ".toru.db")
	store, err := cache.NewStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing cache store: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	// 3. Initialize Repology Client
	repoClient := repology.NewClient("toru/1.0 (https://github.com/toru/toru)")

	// 4. Determine Backend Runner
	runner, err := backend.Get(cmd.Backend)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading backend: %v\n", err)
		os.Exit(1)
	}

	// Keep a copy of the original targets for our internal database state tracking
	// before they get replaced by the translated Nix attributes.
	archTargets := make([]string, len(cmd.Targets))
	copy(archTargets, cmd.Targets)

	// 5. Translation Phase
	// Only Nix and Lix require Repology translation. Yay is 1:1 since it's native Arch.
	if len(cmd.Targets) > 0 && (cmd.Backend == "nix" || cmd.Backend == "lix") {
		translatedTargets := make([]string, 0, len(cmd.Targets))
		targetRepo := "nix_unstable"

		for _, archName := range cmd.Targets {
			// Check Cache
			translated, _, err := store.GetTranslation(archName, targetRepo)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Cache read error: %v\n", err)
			}

			if translated == "" {
				fmt.Printf("=> Translating '%s' via Repology...\n", archName)
				trans, master, err := repoClient.Fetch(archName, targetRepo)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: Translation failed for '%s': %v\n", archName, err)
					os.Exit(1)
				}
				translated = trans
				
				if err := store.SaveTranslation(archName, targetRepo, trans, master); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to save translation to cache: %v\n", err)
				}
			}
			
			translatedTargets = append(translatedTargets, translated)
		}
		// Overwrite the CLI targets with the translated names for the runner
		cmd.Targets = translatedTargets
	}

	// 6. Execute Native Backend Command
	if err := runner.Execute(cmd); err != nil {
		fmt.Fprintf(os.Stderr, "Execution failed: %v\n", err)
		os.Exit(1)
	}

	// 7. Post-Execution State Tracking
	// (Note: The advanced PATH binary diffing and Parity Sync logic would hook in here).
	if cmd.Operation == cli.OpSync && len(archTargets) > 0 {
		for _, target := range archTargets {
			// By default, a user typing `toru -S <pkg>` implies an Explicit installation.
			store.MarkInstalled(target, cmd.Backend, true, false)
		}
	} else if cmd.Operation == cli.OpRemove && len(archTargets) > 0 {
		for _, target := range archTargets {
			store.RemovePackage(target, cmd.Backend)
		}
	}
}
