package lix

import (
	"reflect"
	"testing"
	"toru/pkg/cli"
)

func TestLixRunner(t *testing.T) {
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
				{"nix", "profile", "install", "nixpkgs#python", "nixpkgs#go"},
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
				{"nix", "profile", "upgrade", ".*"},
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
				{"nix", "store", "gc"},
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
				{"nix", "profile", "remove", "neovim"},
			},
			wantErr: false,
		},
		{
			name: "Query updates (dry run)",
			cmd: &cli.ParsedCommand{
				Operation: cli.OpQuery,
				Modifiers: cli.Modifiers{SysUpgrade: true},
			},
			expectedCmds: [][]string{
				{"nix", "profile", "upgrade", ".*", "--dry-run"},
			},
			wantErr: false,
		},
		{
			name: "Sync databases no-op",
			cmd: &cli.ParsedCommand{
				Operation: cli.OpSync,
				Modifiers: cli.Modifiers{Refresh: 1},
				Targets:   []string{},
			},
			expectedCmds: nil, // no command executed
			wantErr:      false,
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
			
			if len(gotCmds) == 0 && len(tt.expectedCmds) == 0 {
				return // both are empty/nil
			}

			if !reflect.DeepEqual(gotCmds, tt.expectedCmds) {
				t.Errorf("Execute() got = %v, want %v", gotCmds, tt.expectedCmds)
			}
		})
	}
}
