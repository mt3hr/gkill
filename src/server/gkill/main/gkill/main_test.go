package main

import (
	"testing"
)

func TestAppCmdNotNil(t *testing.T) {
	if AppCmd == nil {
		t.Fatal("AppCmd should not be nil")
	}
	if AppCmd.Use != "gkill" {
		t.Errorf("AppCmd.Use = %q, want %q", AppCmd.Use, "gkill")
	}
}

func TestAppCmdHasSubcommands(t *testing.T) {
	cmds := AppCmd.Commands()
	if len(cmds) == 0 {
		t.Error("AppCmd should have subcommands")
	}
}
