package main

import (
	"slices"
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

// TestServerCmdRegistersSubcommands はサーバ版に登録するサブコマンドの集合を固定する。
// 「1つでもあれば緑」の検査だと、init() の AddCommand を1行落としても（mcp や generate_plugin_cache が
// 呼べなくなっても）気付けない。mcp はサーバ版だけ（デスクトップ版 gkill には無い）。
func TestServerCmdRegistersSubcommands(t *testing.T) {
	want := []string{
		"idf", "dvnf", "version",
		"generate_thumb_cache", "generate_video_cache", "optimize",
		"update_cache", "clear_cache", "generate_plugin_cache",
		"reset_password", "add_tag", "mcp",
	}
	got := []string{}
	for _, c := range ServerCmd.Commands() {
		got = append(got, c.Name())
	}
	for _, name := range want {
		if !slices.Contains(got, name) {
			t.Errorf("サブコマンド %q が gkill_server に登録されていない: %v", name, got)
		}
	}
	if len(got) != len(want) {
		t.Errorf("サブコマンドの数 = %d, want %d（増やしたらこの表と資料へ足す）: %v", len(got), len(want), got)
	}
}
