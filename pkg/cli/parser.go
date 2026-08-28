package cli

import (
	"fmt"
	"strings"
)

// OperationType represents the major mode pacman is running in.
type OperationType string

const (
	OpSync     OperationType = "SYNC"     // -S, --sync
	OpRemove   OperationType = "REMOVE"   // -R, --remove
	OpQuery    OperationType = "QUERY"    // -Q, --query
	OpDatabase OperationType = "DATABASE" // -D, --database
	OpUpgrade  OperationType = "UPGRADE"  // -U, --upgrade
	OpFiles    OperationType = "FILES"    // -F, --files
	OpDeptest  OperationType = "DEPTEST"  // -T, --deptest
	OpUnknown  OperationType = "UNKNOWN"
)

// Modifiers holds all the optional flags that modify a primary operation.
type Modifiers struct {
	Refresh    int  // -y, --refresh (can be passed twice: -yy)
	SysUpgrade bool // -u, --sysupgrade
	Clean      int  // -c, --clean (in -S context, can be passed twice)
	Search     bool // -s, --search (in -S, -Q context)
	Info       bool // -i, --info
	Recursive  bool // -s, --recursive (in -R context)
	Unneeded   bool // -n, --unneeded (in -R context)
	Cascade    bool // -c, --cascade (in -R context)
	Quiet      bool // -q, --quiet
	List       bool // -l, --list
	Download   bool // -w, --downloadonly
	Groups     bool // -g, --groups
	Print      bool // -p, --print
	Help       bool // -h, --help
}

// ParsedCommand holds the fully structured CLI intent.
type ParsedCommand struct {
	Backend   string
	Operation OperationType
	Modifiers Modifiers
	Targets   []string
}

// Parse analyzes the CLI arguments and returns the ParsedCommand intent.
func Parse(args []string) (*ParsedCommand, error) {
	cmd := &ParsedCommand{
		Backend:   "lix", // Default backend
		Operation: OpUnknown,
		Modifiers: Modifiers{},
		Targets:   []string{},
	}

	for _, arg := range args {
		// Backend override
		if strings.HasPrefix(arg, "--backend=") {
			cmd.Backend = strings.TrimPrefix(arg, "--backend=")
			continue
		}

		// Handle long flags
		if strings.HasPrefix(arg, "--") {
			switch arg {
			case "--sync":
				cmd.Operation = OpSync
			case "--remove":
				cmd.Operation = OpRemove
			case "--query":
				cmd.Operation = OpQuery
			case "--database":
				cmd.Operation = OpDatabase
			case "--upgrade":
				cmd.Operation = OpUpgrade
			case "--files":
				cmd.Operation = OpFiles
			case "--deptest":
				cmd.Operation = OpDeptest
			case "--refresh":
				cmd.Modifiers.Refresh++
			case "--sysupgrade":
				cmd.Modifiers.SysUpgrade = true
			case "--clean":
				cmd.Modifiers.Clean++
			case "--search":
				cmd.Modifiers.Search = true
			case "--info":
				cmd.Modifiers.Info = true
			case "--recursive":
				cmd.Modifiers.Recursive = true
			case "--unneeded":
				cmd.Modifiers.Unneeded = true
			case "--cascade":
				cmd.Modifiers.Cascade = true
			case "--quiet":
				cmd.Modifiers.Quiet = true
			case "--list":
				cmd.Modifiers.List = true
			case "--downloadonly":
				cmd.Modifiers.Download = true
			case "--groups":
				cmd.Modifiers.Groups = true
			case "--print":
				cmd.Modifiers.Print = true
			case "--help":
				cmd.Modifiers.Help = true
			default:
				// We can ignore unknown long flags or error out. Let's error to be strict.
				return nil, fmt.Errorf("unsupported long flag: %s", arg)
			}
			continue
		}

		// Handle short flags grouping
		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			for i, char := range arg[1:] {
				switch char {
				case 'S':
					if i == 0 {
						cmd.Operation = OpSync
					} else {
						return nil, fmt.Errorf("operation flags like 'S' must be the first flag in a group")
					}
				case 'R':
					if i == 0 {
						cmd.Operation = OpRemove
					} else {
						return nil, fmt.Errorf("operation flags like 'R' must be the first flag in a group")
					}
				case 'Q':
					if i == 0 {
						cmd.Operation = OpQuery
					} else {
						return nil, fmt.Errorf("operation flags like 'Q' must be the first flag in a group")
					}
				case 'D':
					if i == 0 {
						cmd.Operation = OpDatabase
					}
				case 'U':
					if i == 0 {
						cmd.Operation = OpUpgrade
					}
				case 'F':
					if i == 0 {
						cmd.Operation = OpFiles
					}
				case 'T':
					if i == 0 {
						cmd.Operation = OpDeptest
					}
				case 'y':
					cmd.Modifiers.Refresh++
				case 'u':
					cmd.Modifiers.SysUpgrade = true
				case 'c':
					cmd.Modifiers.Clean++
				case 's':
					cmd.Modifiers.Search = true
				case 'n':
					cmd.Modifiers.Unneeded = true
				case 'i':
					cmd.Modifiers.Info = true
				case 'q':
					cmd.Modifiers.Quiet = true
				case 'l':
					cmd.Modifiers.List = true
				case 'w':
					cmd.Modifiers.Download = true
				case 'g':
					cmd.Modifiers.Groups = true
				case 'p':
					cmd.Modifiers.Print = true
				case 'h':
					cmd.Modifiers.Help = true
				default:
					return nil, fmt.Errorf("unsupported short flag: -%c in %s", char, arg)
				}
			}
			continue
		}

		// Treat as a positional argument (target)
		cmd.Targets = append(cmd.Targets, arg)
	}

	// Post-processing context specific flags
	if cmd.Operation == OpRemove {
		// In remove context, 'c' is cascade, 's' is recursive.
		// Since we captured them in Clean and Search due to short flag sharing, let's remap them.
		if cmd.Modifiers.Clean > 0 {
			cmd.Modifiers.Cascade = true
			cmd.Modifiers.Clean-- // Optionally zero it out if we assume -Rcc is just Cascade
		}
		if cmd.Modifiers.Search {
			cmd.Modifiers.Recursive = true
			cmd.Modifiers.Search = false
		}
	}

	if cmd.Operation == OpUnknown && !cmd.Modifiers.Help {
		return nil, fmt.Errorf("no primary operation specified (e.g., -S, -R, -Q)")
	}

	if cmd.Backend != "nix" && cmd.Backend != "yay" && cmd.Backend != "lix" {
		return nil, fmt.Errorf("unsupported backend: %s (must be 'nix', 'lix', or 'yay')", cmd.Backend)
	}

	return cmd, nil
}
