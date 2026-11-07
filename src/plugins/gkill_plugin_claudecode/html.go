package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk/configview"
)

// renderConfigHTML は設定画面のHTMLを返す。
//
// 描画は SDK の configview が1か所で行う（ADR-0316）。ここでは Claude Code 固有の文言と統計を
// configview.Page へ写すだけで、head・保存スクリプト・進捗・一覧の体裁は持たない。
// 設定の保存は gkill 本体の設定ダイアログが postMessage で肩代わりし、config.json を直接編集する経路も残る。
func renderConfigHTML(pluginDir string, stats cacheStats, patterns []string, src expandedSource) string {
	page := configview.Page{
		Title: "Claude Code チャットログプラグイン",
		Progress: configview.Progress{
			State: stats.BuildState, Done: stats.BuildDone, Total: stats.BuildTotal, Unit: "セッション",
		},
		Stats: []configview.Row{
			{Label: "対象ファイル数", Value: configview.Count(stats.FileCount)},
			{Label: "Kyou数", Value: configview.Count(stats.MessageCount)},
			{Label: "最終スキャン", Value: configview.FormatUnix(stats.LastScanUnix)},
			{Label: "キャッシュDB", Value: sdk.CacheDBPath(pluginDir), Code: true},
		},
		Notices: []configview.Notice{
			configview.MissingPatterns(src.Missing),
			configview.ErrorNotice("スキャン時のエラー:", stats.LastScanError),
		},
		Patterns: patterns,
		Expanded: &configview.Expanded{Dirs: src.Dirs, Files: src.Files},
		Form: &configview.Form{Fields: []configview.Field{{
			Key:   "source_dirs",
			Label: "セッションログの置き場所（1行に1つ。フォルダかファイル。ワイルドカードも可。保存すると次の検索から反映されます）",
			Value: strings.Join(patterns, "\n"),
		}}},
		ConfigFile: &configview.ConfigFile{
			Path:   filepath.Join(pluginDir, "config.json"),
			Sample: configview.SampleJSON(defaultConfig()),
			Notes: []string{configview.SourceDirsNote(
				"マッチしたフォルダは再帰的に走査し、マッチしたファイルはそのまま対象にします",
				"キーごと省略するか空にすると `"+defaultSourceDir()+"` を使います。",
			)},
		},
	}

	if stats.MessageCount > 0 {
		page.Status = &configview.Status{
			Loaded:   true,
			Headline: fmt.Sprintf("**%s 件**を読み込んでいます", configview.Count(stats.MessageCount)),
		}
	} else {
		page.Status = &configview.Status{
			Headline: "まだ読み込まれていません。",
			Lines: []string{
				"Claude Code のセッションログが入ったフォルダを、下の「設定」で指定してください。",
				"取り込みはバックグラウンドで進むので、指定した直後は0件のことがあります。",
			},
		}
	}
	return configview.Render(page)
}
