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
// 描画は SDK の configview が1か所で行う（ADR-0316）。ここでは Claude.ai 固有の文言と統計を
// configview.Page へ写すだけで、head・保存スクリプト・進捗・一覧の体裁は持たない。
//
// 走査・読み込みはバックグラウンドのビルダが行う。ここでは絶対にファイルを読まない
// (GetConfigHTML は IsAlive 5秒のスロットに並ぶため)。統計はキャッシュから即答する。
func renderConfigHTML(pluginDir string, stats cacheStats, patterns []string, src sdk.ExpandedSource) string {
	page := configview.Page{
		Title: "Claude.ai チャット履歴プラグイン",
		Progress: configview.Progress{
			State: stats.BuildState, Done: stats.BuildDone, Total: stats.BuildTotal, Unit: "会話",
		},
		Stats: []configview.Row{
			{Label: "ZIP 内の会話ファイル数", Value: configview.Count(stats.FileCount)},
			{Label: "会話数", Value: configview.Count(stats.ConvCount)},
			{Label: "メッセージ数", Value: configview.Count(stats.MessageCount)},
			{Label: "最終スキャン", Value: configview.FormatUnix(stats.LastScanUnix)},
			{Label: "キャッシュDB", Value: sdk.CacheDBPath(pluginDir), Code: true},
		},
		Notices: []configview.Notice{
			configview.MissingPatterns(src.Missing),
			sourceProblemsNotice(stats.SourceProblems),
			configview.ErrorNotice("スキャン時のエラー:", stats.LastScanError),
		},
		Patterns: patterns,
		Expanded: &configview.Expanded{Dirs: src.Dirs, Files: src.Files},
		Form: &configview.Form{Fields: []configview.Field{{
			Key:   "source_dirs",
			Label: "取り込み元（1行に1つ。エクスポート ZIP を置いたフォルダ、または ZIP そのもの。ワイルドカードも可。保存すると次の検索から反映されます）",
			Value: strings.Join(patterns, "\n"),
		}}},
		ConfigFile: &configview.ConfigFile{
			Path:   filepath.Join(pluginDir, "config.json"),
			Sample: configview.SampleJSON(defaultConfig()),
			Notes: []string{configview.SourceDirsNote(
				"マッチしたフォルダは再帰的に走査して `*.zip` を探し、ZIP の中の `conversations.json` を読みます。"+
					"ZIP を直接指定してもかまいません。**ZIP は解凍しないでください**(解凍したフォルダや JSON は読みません)",
				"キーごと省略するか空にすると、このプラグインのフォルダを見ます。",
			)},
		},
	}

	if stats.MessageCount > 0 {
		page.Status = &configview.Status{
			Loaded: true,
			Headline: fmt.Sprintf("**%s 件**のメッセージ(会話 %s 件)を読み込んでいます",
				configview.Count(stats.MessageCount), configview.Count(stats.ConvCount)),
			Lines: []string{"データを更新するには、Claude.ai から再エクスポートして ZIP を置き換えてください" +
				"（古い ZIP を残しても、同じ会話は新しいほうだけを使います）。"},
		}
	} else {
		page.Status = &configview.Status{
			Headline: "まだ読み込まれていません。",
			Lines: []string{
				"取り込みはバックグラウンドで進むので、指定した直後は0件のことがあります。",
				"エクスポート手順:",
			},
			Steps: []string{
				"Claude.ai にログイン → 左下のアカウントアイコン → **Settings**",
				"「Privacy」→「**Export data**」をクリック",
				"ZIP（`conversations-000.zip` など）が届いたら**解凍せず**、そのまま下の「設定」で指定したフォルダに置く",
				"ZIP の中の `conversations.json` を自動で読みます。解凍した JSON は読みません",
			},
		}
	}
	return configview.Render(page)
}

// sourceProblemsNotice は走査で見つかった問題（ZIP ではないものの指定など）を警告にする。0件なら空。
func sourceProblemsNotice(problems []sourceProblemRow) configview.Notice {
	if len(problems) == 0 {
		return configview.Notice{}
	}
	n := configview.Notice{Title: "走査で見つかった問題:"}
	for _, problem := range problems {
		n.Items = append(n.Items, configview.Item{Code: problem.Path, Text: problem.Message})
	}
	return n
}
