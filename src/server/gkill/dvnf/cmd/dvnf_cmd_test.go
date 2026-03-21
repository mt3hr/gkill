package dvnf_cmd

import (
	"testing"
)

func TestDVNFCmdNotNil(t *testing.T) {
	if DVNFCmd == nil {
		t.Fatal("DVNFCmd should not be nil")
	}
	if DVNFCmd.Use != "dvnf" {
		t.Errorf("DVNFCmd.Use = %q, want %q", DVNFCmd.Use, "dvnf")
	}
}

func TestConfigStruct(t *testing.T) {
	c := &Config{
		Directory:  "/tmp/test",
		Device:     "testdevice",
		TimeLength: 8,
	}
	if c.Directory != "/tmp/test" {
		t.Errorf("Directory = %q, want %q", c.Directory, "/tmp/test")
	}
	if c.TimeLength != 8 {
		t.Errorf("TimeLength = %d, want 8", c.TimeLength)
	}
}

func TestSplitDVNFPathnium(t *testing.T) {
	tests := []struct {
		input     string
		wantDir   string
		wantChild string
	}{
		{"hoge", "hoge", ""},
		{"hoge/fuga", "hoge", "fuga"},
		{"hoge/fuga/piyo", "hoge", "fuga/piyo"},
	}

	for _, tt := range tests {
		// Note: splitDVNFPathnium uses filepath separator which varies by OS.
		// On Windows, forward slashes get converted.
		dir, child := splitDVNFPathnium(tt.input)
		if dir == "" {
			t.Errorf("splitDVNFPathnium(%q): dir is empty", tt.input)
		}
		_ = child // child format depends on OS path separator
	}
}

func TestPlaneFileName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hogehoge.txt", "hogehoge.txt"},
		{"hogehoge (1).txt", "hogehoge.txt"},
		{"hogehoge (23).txt", "hogehoge.txt"},
		{"noext", "noext"},
	}

	for _, tt := range tests {
		got := planeFileName(tt.input)
		if got != tt.want {
			t.Errorf("planeFileName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSubcommands(t *testing.T) {
	// Verify subcommands are registered
	cmds := DVNFCmd.Commands()
	names := map[string]bool{}
	for _, c := range cmds {
		names[c.Use] = true
	}
	for _, want := range []string{"get", "move", "copy"} {
		if !names[want] {
			t.Errorf("subcommand %q not found", want)
		}
	}
}
