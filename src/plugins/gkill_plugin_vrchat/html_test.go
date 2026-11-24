package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// 設定画面の枠（見出し・フォーム・CSS）は SDK の configview が組み、そちらのテストが見る。
// ここで見るのは、このプラグインの統計と問題がどの文言で画面へ写るかだけ。
func TestRenderConfigHTML(t *testing.T) {
	t.Run("未読み込み・folding・警告あり", func(t *testing.T) {
		html := renderConfigHTML("/example/plugin", cacheStats{
			BuildState:     "folding",
			DirtyCount:     2,
			DroppedLines:   5,
			BuildError:     "sample build error",
			Err:            errors.New("sample open error"),
			SourceProblems: []string{"/example/missing"},
			CacheDBPath:    "/example/cache.db",
		}, []string{"/example/vrchat"}, sdk.ExpandedSource{})

		for _, want := range []string{
			"まだ読み込まれていません。",
			"作り直し待ち 2 回ぶんの起動",
			"長すぎて読み飛ばした行が",
			"sample build error",
			"sample open error",
		} {
			if !strings.Contains(html, want) {
				t.Errorf("HTML に %q が含まれていない", want)
			}
		}
	})

	t.Run("読み込み済み", func(t *testing.T) {
		html := renderConfigHTML("/example/plugin", cacheStats{
			TargetFileCount: 1,
			FileCount:       1,
			SessionCount:    1,
			WorldCount:      2,
			PlayerCount:     3,
			EventCount:      1,
			VideoCount:      1,
			RecordCount:     7,
			LastScan:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			CacheDBPath:     "/example/cache.db",
		}, []string{"/example/vrchat"}, sdk.ExpandedSource{Dirs: []string{"/example/vrchat"}})

		if !strings.Contains(html, "7 件") {
			t.Error("読み込み済み件数が HTML に出ていない")
		}
		if strings.Contains(html, "まだ読み込まれていません。") {
			t.Error("読み込み済みなのに未読み込み見出しが出ている")
		}
	})
}
