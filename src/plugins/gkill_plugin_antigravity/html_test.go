package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// 設定画面の枠（見出し・フォーム・CSS）は SDK の configview が組み、そちらのテストが見る。
// ここで見るのは、このプラグインの統計と問題がどの文言で画面へ写るかだけ。
func TestRenderConfigHTML(t *testing.T) {
	t.Run("未読み込み時は案内と警告を出す", func(t *testing.T) {
		html := renderConfigHTML("/example/plugin", cacheStats{
			BuildState:     "idle",
			CacheDBPath:    "/example/cache.db",
			MissingIDFiles: []string{"/example/no-uuid/transcript_full.jsonl"},
			ShadowedCount:  2,
			DroppedLines:   3,
			BuildError:     "sample build error",
			Err:            errors.New("sample open error"),
		}, []string{"/example/brain"}, sdk.ExpandedSource{
			Missing: []string{"/example/missing"},
		})

		for _, want := range []string{
			"まだ読み込まれていません。",
			"会話IDのフォルダに入っていないため読みませんでした:",
			"同じ会話のファイルが 2 件、複数の場所で見つかりました",
			"読めなかった行が 3 件あります",
			"sample build error",
			"sample open error",
		} {
			if !strings.Contains(html, want) {
				t.Errorf("HTML に %q が含まれていない", want)
			}
		}
	})

	t.Run("読み込み済み時は件数と統計を出す", func(t *testing.T) {
		html := renderConfigHTML("/example/plugin", cacheStats{
			FileCount:         2,
			ConversationCount: 1,
			KyouCount:         4,
			LastScan:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			CacheDBPath:       "/example/cache.db",
		}, []string{"/example/brain"}, sdk.ExpandedSource{
			Dirs: []string{"/example/brain"},
		})

		if !strings.Contains(html, "4 件") {
			t.Error("読み込み済み件数が HTML に出ていない")
		}
		if strings.Contains(html, "まだ読み込まれていません。") {
			t.Error("読み込み済みなのに未読み込み見出しが出ている")
		}
	})
}
