package main

import (
	"testing"
)

func TestServerCmdNotNil(t *testing.T) {
	if ServerCmd == nil {
		t.Fatal("ServerCmd should not be nil")
	}
	if ServerCmd.Use != "gkill_server" {
		t.Errorf("ServerCmd.Use = %q, want %q", ServerCmd.Use, "gkill_server")
	}
}

func TestServerCmdHasSubcommands(t *testing.T) {
	cmds := ServerCmd.Commands()
	if len(cmds) == 0 {
		t.Error("ServerCmd should have subcommands")
	}
}
