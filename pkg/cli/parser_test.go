package cli

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    *ParsedCommand
		wantErr bool
	}{
		{
			name: "Sync single package",
			args: []string{"-S", "python"},
			want: &ParsedCommand{
				Backend:   "nix",
				Operation: OpSync,
				Modifiers: Modifiers{},
				Targets:   []string{"python"},
			},
			wantErr: false,
		},
		{
			name: "Sync and sysupgrade",
			args: []string{"-Syu"},
			want: &ParsedCommand{
				Backend:   "nix",
				Operation: OpSync,
				Modifiers: Modifiers{Refresh: 1, SysUpgrade: true},
				Targets:   []string{},
			},
			wantErr: false,
		},
		{
			name: "Sync with double refresh and yay backend",
			args: []string{"--backend=yay", "-Syyu"},
			want: &ParsedCommand{
				Backend:   "yay",
				Operation: OpSync,
				Modifiers: Modifiers{Refresh: 2, SysUpgrade: true},
				Targets:   []string{},
			},
			wantErr: false,
		},
		{
			name: "Remove recursive and unneeded",
			args: []string{"-Rns", "neovim"},
			want: &ParsedCommand{
				Backend:   "nix",
				Operation: OpRemove,
				Modifiers: Modifiers{Unneeded: true, Recursive: true},
				Targets:   []string{"neovim"},
			},
			wantErr: false,
		},
		{
			name: "Remove cascade",
			args: []string{"-Rc", "bash"},
			want: &ParsedCommand{
				Backend:   "nix",
				Operation: OpRemove,
				Modifiers: Modifiers{Cascade: true},
				Targets:   []string{"bash"},
			},
			wantErr: false,
		},
		{
			name: "Query search",
			args: []string{"-Qs", "python"},
			want: &ParsedCommand{
				Backend:   "nix",
				Operation: OpQuery,
				Modifiers: Modifiers{Search: true},
				Targets:   []string{"python"},
			},
			wantErr: false,
		},
		{
			name: "Sync clean cache twice",
			args: []string{"-Scc"},
			want: &ParsedCommand{
				Backend:   "nix",
				Operation: OpSync,
				Modifiers: Modifiers{Clean: 2},
				Targets:   []string{},
			},
			wantErr: false,
		},
		{
			name: "Long flags sync refresh",
			args: []string{"--sync", "--refresh"},
			want: &ParsedCommand{
				Backend:   "nix",
				Operation: OpSync,
				Modifiers: Modifiers{Refresh: 1},
				Targets:   []string{},
			},
			wantErr: false,
		},
		{
			name: "Help flag standalone",
			args: []string{"--help"},
			want: &ParsedCommand{
				Backend:   "nix",
				Operation: OpUnknown,
				Modifiers: Modifiers{Help: true},
				Targets:   []string{},
			},
			wantErr: false,
		},
		{
			name: "Help flag with operation",
			args: []string{"-Sh"},
			want: &ParsedCommand{
				Backend:   "nix",
				Operation: OpSync,
				Modifiers: Modifiers{Help: true},
				Targets:   []string{},
			},
			wantErr: false,
		},
		{
			name:    "Missing operation",
			args:    []string{"python"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Operation flag not first",
			args:    []string{"-yS"}, // invalid in standard pacman too
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Unsupported short flag",
			args:    []string{"-Sz"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Unsupported long flag",
			args:    []string{"--sync", "--invalidflag"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Unsupported backend",
			args:    []string{"--backend=apt", "-S", "python"},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
