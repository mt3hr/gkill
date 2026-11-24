package main

import (
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// 設定画面の枠（見出し・フォーム・CSS）は SDK の configview が組み、そちらのテストが見る。
// ここで見るのは、このプラグインの統計と問題がどの文言で画面へ写るかだけ。
func TestRenderConfigHTML(t *testing.T) {
	t.Run("未読み込み時は Takeout 手順と問題一覧を出す", func(t *testing.T) {
		html := renderConfigHTML("/example/plugin", cacheStats{
			BuildState:    "idle",
			LastScanError: "sample scan error",
			SourceProblems: []sourceProblemRow{
				{Path: "/example/bad.html", Message: htmlFormatMessage},
			},
		}, []string{"/example/takeout"}, sdk.ExpandedSource{
			Missing: []string{"/example/missing"},
		})

		for _, want := range []string{
			"まだ読み込まれていません。",
			"エクスポート手順:",
			"走査で見つかった問題:",
			htmlFormatMessage,
			"sample scan error",
		} {
			if !strings.Contains(html, want) {
				t.Errorf("HTML に %q が含まれていない", want)
			}
		}
	})

	t.Run("読み込み済み時は発言数と会話数を出す", func(t *testing.T) {
		html := renderConfigHTML("/example/plugin", cacheStats{
			FileCount:    1,
			ConvCount:    2,
			MessageCount: 6,
			LastScanUnix: 1704067200,
		}, []string{"/example/takeout"}, sdk.ExpandedSource{
			Dirs: []string{"/example/takeout"},
		})

		if !strings.Contains(html, "6 件") {
			t.Error("読み込み済み件数が HTML に出ていない")
		}
		if strings.Contains(html, "まだ読み込まれていません。") {
			t.Error("読み込み済みなのに未読み込み見出しが出ている")
		}
	})
}
