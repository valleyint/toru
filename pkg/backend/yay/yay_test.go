package yay

import (
	"reflect"
	"testing"
	"toru/pkg/cli"
)

func TestYayRunner(t *testing.T) {
	tests := []struct {
		name         string
		cmd          *cli.ParsedCommand
		expectedCmds [][]string
		wantErr      bool
	}{
		{
			name: "Install single package",
			cmd: &cli.ParsedCommand{
				Operation: cli.OpSync,
				Targets:   []string{"python"},
			},
			expectedCmds: [][]string{
				{"yay", "-S", "python"},
			},
			wantErr: false,
		},
		{
			name: "System upgrade with double refresh",
			cmd: &cli.ParsedCommand{
				Operation: cli.OpSync,
				Modifiers: cli.Modifiers{Refresh: 2, SysUpgrade: true},
			},
			expectedCmds: [][]string{
				{"yay", "-Syyu"},
			},
			wantErr: false,
		},
		{
			name: "Remove recursive unneeded",
			cmd: &cli.ParsedCommand{
				Operation: cli.OpRemove,
				Modifiers: cli.Modifiers{Recursive: true, Unneeded: true},
				Targets:   []string{"bash"},
			},
			expectedCmds: [][]string{
				{"yay", "-Rsn", "bash"}, // order matches struct iteration: s then n
			},
			wantErr: false,
		},
		{
			name: "Database operation",
			cmd: &cli.ParsedCommand{
				Operation: cli.OpDatabase,
				Targets:   []string{"testpkg"},
			},
			expectedCmds: [][]string{
				{"yay", "-D", "testpkg"},
			},
			wantErr: false,
		},
		{
			name: "Sync with info",
			cmd: &cli.ParsedCommand{
				Operation: cli.OpSync,
				Modifiers: cli.Modifiers{Info: true},
				Targets:   []string{"htop"},
			},
			expectedCmds: [][]string{
				{"yay", "-Si", "htop"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := New()
			var gotCmds [][]string

			// Mock the execution
			runner.ExecFunc = func(args ...string) error {
				fullCmd := append([]string{"yay"}, args...)
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
