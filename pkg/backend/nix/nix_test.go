package nix

import (
	"reflect"
	"testing"
	"toru/pkg/cli"
)

func TestNixRunner(t *testing.T) {
	tests := []struct {
		name         string
		cmd          *cli.ParsedCommand
		expectedCmds [][]string
		wantErr      bool
	}{
		{
			name: "Install packages",
			cmd: &cli.ParsedCommand{
				Operation: cli.OpSync,
				Targets:   []string{"python", "go"},
			},
			expectedCmds: [][]string{
				{"nix-env", "-iA", "nixpkgs.python", "nixpkgs.go"},
			},
			wantErr: false,
		},
		{
			name: "System upgrade",
			cmd: &cli.ParsedCommand{
				Operation: cli.OpSync,
				Modifiers: cli.Modifiers{SysUpgrade: true},
			},
			expectedCmds: [][]string{
				{"nix-channel", "--update"},
				{"nix-env", "-u"},
			},
			wantErr: false,
		},
		{
			name: "Clean cache",
			cmd: &cli.ParsedCommand{
				Operation: cli.OpSync,
				Modifiers: cli.Modifiers{Clean: 1},
			},
			expectedCmds: [][]string{
				{"nix-collect-garbage"},
			},
			wantErr: false,
		},
		{
			name: "Remove package",
			cmd: &cli.ParsedCommand{
				Operation: cli.OpRemove,
				Targets:   []string{"neovim"},
			},
			expectedCmds: [][]string{
				{"nix-env", "-e", "neovim"},
			},
			wantErr: false,
		},
		{
			name: "Query search",
			cmd: &cli.ParsedCommand{
				Operation: cli.OpQuery,
				Modifiers: cli.Modifiers{Search: true},
				Targets:   []string{"python"},
			},
			expectedCmds: [][]string{
				{"nix-env", "-qaP", ".*python.*"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := New()
			var gotCmds [][]string

			// Mock the execution
			runner.ExecFunc = func(name string, args ...string) error {
				fullCmd := append([]string{name}, args...)
				gotCmds = append(gotCmds, fullCmd)
				return nil
			}

			err := runner.Execute(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotCmds, tt.expectedCmds) {
				t.Errorf("Execute() got = %v, want %v", gotCmds, tt.expectedCmds)
			}
		})
	}
}
