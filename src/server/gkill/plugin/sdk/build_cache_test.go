package sdk

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// runBuildCache は `--gkill-build-cache` 付きで起動されたときの本体。
// gkill 側（generate_plugin_cache）は stdout の結果行だけで成否を判定するので、
// 「何を stdout に書くか」「nil のとき」「失敗のとき」をここで固定する。

func TestRunBuildCache_NilHandlerPrintsNoCache(t *testing.T) {
	var out bytes.Buffer
	ok := runBuildCache(context.Background(), Handler{}, Config{}, &out)
	if !ok {
		t.Fatal("BuildCache が nil のプラグインは成功扱い（all 指定で赤くしない）")
	}
	if got := out.String(); got != BuildCacheResultNoCache+"\n" {
		t.Errorf("stdout = %q, want %q", got, BuildCacheResultNoCache+"\n")
	}
}

func TestRunBuildCache_SuccessPrintsBuilt(t *testing.T) {
	called := 0
	var gotCfg Config
	var gotUserID string
	h := Handler{
		BuildCache: func(ctx context.Context, cfg Config) error {
			called++
			gotCfg = cfg
			gotUserID, _ = ctx.Value(ctxKeyUserID{}).(string)
			return nil
		},
	}
	cfg := Config{"source_dirs": []string{"x"}}

	var out bytes.Buffer
	ok := runBuildCache(newCtx("testuser"), h, cfg, &out)
	if !ok {
		t.Fatal("成功なのに false")
	}
	if called != 1 {
		t.Errorf("BuildCache の呼び出し回数 = %d, want 1", called)
	}
	if gotUserID != "testuser" {
		t.Errorf("ctx の user id = %q, want testuser（stdio ループと同じ newCtx で渡す）", gotUserID)
	}
	if _, exist := gotCfg["source_dirs"]; !exist {
		t.Errorf("EnsureConfig の結果がそのまま渡っていない: %v", gotCfg)
	}
	if got := out.String(); got != BuildCacheResultBuilt+"\n" {
		t.Errorf("stdout = %q, want %q", got, BuildCacheResultBuilt+"\n")
	}
}

func TestRunBuildCache_ErrorGoesToStderrAndReturnsFalse(t *testing.T) {
	var logs bytes.Buffer
	SetLogWriter(&logs)
	t.Cleanup(func() { SetLogWriter(nil) })

	h := Handler{
		BuildCache: func(context.Context, Config) error { return errors.New("boom") },
	}
	var out bytes.Buffer
	ok := runBuildCache(context.Background(), h, Config{}, &out)
	if ok {
		t.Fatal("失敗なのに true")
	}
	// 失敗のときは結果行を出さない。出すと gkill 側が built と読んで成功にしてしまう。
	if out.Len() != 0 {
		t.Errorf("失敗時に stdout へ書いている: %q", out.String())
	}
	if !strings.Contains(logs.String(), "ERROR: build cache: boom") {
		t.Errorf("stderr にエラーが出ていない: %q", logs.String())
	}
}

// 結果行以外を stdout に書かないこと。gkill 側は stdout 全体の完全一致で判定するので、
// 1文字でも余計なものが混ざると「対応していない」扱いになる。
func TestRunBuildCache_WritesOnlyResultLineToStdout(t *testing.T) {
	var logs bytes.Buffer
	SetLogWriter(&logs)
	t.Cleanup(func() { SetLogWriter(nil) })

	h := Handler{
		BuildCache: func(context.Context, Config) error {
			LogWarn("skipped 1 file")
			return nil
		},
	}
	var out bytes.Buffer
	if !runBuildCache(context.Background(), h, Config{}, &out) {
		t.Fatal("成功なのに false")
	}
	if got := out.String(); got != BuildCacheResultBuilt+"\n" {
		t.Errorf("stdout に結果行以外が混ざっている: %q", got)
	}
	if !strings.Contains(logs.String(), "WARN: skipped 1 file") {
		t.Errorf("警告が stderr 側へ出ていない: %q", logs.String())
	}
}

// 同梱の配布対象プラグイン（src/plugins/gkill_plugin_*）が全部 Handler.BuildCache を配線していること。
// 1本でも欠けると、そのプラグインだけ generate_plugin_cache all で no_cache になり、
// キャッシュが作られないのにエラーも出ない。examples/ は雛形なので対象外。
func TestBundledPluginsWireBuildCache(t *testing.T) {
	// このファイルは src/server/gkill/plugin/sdk/ にあるので、4つ上が src/
	pluginsDir := filepath.Join("..", "..", "..", "..", "plugins")
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		t.Fatalf("src/plugins が読めない（レイアウトが変わったならこのテストを直す）: %v", err)
	}
	wired := regexp.MustCompile(`(?m)^\s*BuildCache:\s`)
	checked := 0
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "gkill_plugin_") {
			continue
		}
		mainPath := filepath.Join(pluginsDir, entry.Name(), "main.go")
		content, err := os.ReadFile(mainPath)
		if err != nil {
			t.Errorf("%s: main.go が読めない: %v", entry.Name(), err)
			continue
		}
		checked++
		if !wired.Match(content) {
			t.Errorf("%s: sdk.Handler に BuildCache が配線されていない（generate_plugin_cache で no_cache になる）", entry.Name())
		}
	}
	if checked < 7 {
		t.Fatalf("走査したプラグインが少なすぎる（走査の壊れ）: %d", checked)
	}
}
