package main

import (
	"slices"
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

// TestAppCmdRegistersSubcommands はデスクトップ版に登録するサブコマンドの集合を固定する。
// idf と mcp はサーバ版（gkill_server）だけの意図的な非対称。generate_plugin_cache は両方にある。
func TestAppCmdRegistersSubcommands(t *testing.T) {
	want := []string{
		"dvnf", "version",
		"generate_thumb_cache", "generate_video_cache", "optimize",
		"update_cache", "clear_cache", "generate_plugin_cache",
		"reset_password", "add_tag",
	}
	got := []string{}
	for _, c := range AppCmd.Commands() {
		got = append(got, c.Name())
	}
	for _, name := range want {
		if !slices.Contains(got, name) {
			t.Errorf("サブコマンド %q が gkill に登録されていない: %v", name, got)
		}
	}
	for _, serverOnly := range []string{"idf", "mcp"} {
		if slices.Contains(got, serverOnly) {
			t.Errorf("サーバ版だけのサブコマンド %q がデスクトップ版に登録されている", serverOnly)
		}
	}
	if len(got) != len(want) {
		t.Errorf("サブコマンドの数 = %d, want %d: %v", len(got), len(want), got)
	}
}
