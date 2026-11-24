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
// 描画は SDK の configview が1か所で行う（ADR-0316）。ここでは Gemini 固有の文言と統計を
// configview.Page へ写すだけで、head・保存スクリプト・進捗・一覧の体裁は持たない。
//
// 走査・読み込みはバックグラウンドのビルダが行う。ここでは絶対にファイルを読まない
// (GetConfigHTML は IsAlive 5秒のスロットに並ぶため)。統計はキャッシュから即答する。
func renderConfigHTML(pluginDir string, stats cacheStats, patterns []string, src sdk.ExpandedSource) string {
	page := configview.Page{
		Title: "Gemini 会話履歴プラグイン",
		Progress: configview.Progress{
			State: stats.BuildState, Done: stats.BuildDone, Total: stats.BuildTotal, Unit: "往復",
		},
		Stats: []configview.Row{
			{Label: "ZIP 内のアクティビティファイル数", Value: configview.Count(stats.FileCount)},
			{Label: "会話数", Value: configview.Count(stats.ConvCount)},
			{Label: "発言数", Value: configview.Count(stats.MessageCount)},
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
			Label: "取り込み元（1行に1つ。Google Takeout の ZIP を置いたフォルダ、または ZIP そのもの。ワイルドカードも可。保存すると次の検索から反映されます）",
			Value: strings.Join(patterns, "\n"),
		}}},
		ConfigFile: &configview.ConfigFile{
			Path:   filepath.Join(pluginDir, "config.json"),
			Sample: configview.SampleJSON(defaultConfig()),
			Notes: []string{configview.SourceDirsNote(
				"マッチしたフォルダは再帰的に走査して `*.zip` を探し、ZIP の中の "+
					"`マイ アクティビティ/Gemini アプリ/` 配下の JSON を読みます"+
					"（フォルダ名もファイル名もアカウントの言語で訳されるので、名前ではなく位置と中身で見分けます）。"+
					"ZIP を直接指定してもかまいません。"+
					"**ZIP は解凍しないでください**(解凍したフォルダは読みません)",
				"キーごと省略するか空にすると、このプラグインのフォルダを見ます。",
			)},
		},
	}

	if stats.MessageCount > 0 {
		page.Status = &configview.Status{
			Loaded: true,
			Headline: fmt.Sprintf("**%s 件**の発言(会話 %s 件)を読み込んでいます",
				configview.Count(stats.MessageCount), configview.Count(stats.ConvCount)),
			Lines: []string{"データを更新するには、Google Takeout から書き出し直して ZIP を置いてください" +
				"（古い ZIP を残しても、同じ往復は新しいほうだけを使います）。"},
		}
	} else {
		page.Status = &configview.Status{
			Headline: "まだ読み込まれていません。",
			Lines: []string{
				"取り込みはバックグラウンドで進むので、指定した直後は0件のことがあります。",
				"**Takeout のトップレベルの「Gemini」には会話が入りません**（Gem と予定アクションの設定だけです）。" +
					"会話は「マイ アクティビティ」の中にあります。",
				"エクスポート手順:",
			},
			Steps: []string{
				"`https://takeout.google.com` を開いて「選択をすべて解除」",
				"「**マイ アクティビティ**」にチェックを入れる",
				"その行の「すべてのアクティビティ データが含まれます」→「選択をすべて解除」→「**Gemini アプリ**」だけにチェック",
				"同じ行の「**複数の形式**」→「アクティビティの記録」を HTML から「**JSON**」に変える",
				"書き出しを作成し、届いた ZIP を**解凍せず**、そのまま下の「設定」で指定したフォルダに置く",
			},
		}
	}
	return configview.Render(page)
}

// sourceProblemsNotice は走査で見つかった問題（ZIP ではないものの指定・HTML 形式など）を警告にする。0件なら空。
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
