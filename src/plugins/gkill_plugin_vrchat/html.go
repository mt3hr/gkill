package main

import (
	"fmt"
	"path/filepath"
	"strings"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk/configview"
)

// renderConfigHTML は設定画面のHTMLを返す。
//
// 描画は SDK の configview が1か所で行う（ADR-0316）。ここでは VRChat 固有の文言と統計を
// configview.Page へ写すだけ。
//
// キャッシュの状態(cache_meta)しか読まない。ここでファイルを走査してはいけない ――
// この画面は IsAlive(5秒)と同じスロットに並ぶので、走査すると殺される。
func renderConfigHTML(pluginDir string, stats cacheStats, patterns []string, src sdk.ExpandedSource) string {
	progress := configview.Progress{
		State: stats.BuildState, Done: stats.BuildDone, Total: stats.BuildTotal, Unit: "ファイル",
	}
	if stats.BuildState == "folding" {
		progress.Total = 0 // 記録の作り直しはファイル数ではなく起動の数で進む
		progress.Note = fmt.Sprintf("作り直し待ち %s 回ぶんの起動", configview.Count(stats.DirtyCount))
	}

	page := configview.Page{
		Title:    "VRChat ログプラグイン",
		Progress: progress,
		Stats: []configview.Row{
			{Label: "対象ファイル数", Value: configview.Count(stats.TargetFileCount)},
			{Label: "取り込み済みファイル", Value: configview.Count(stats.FileCount)},
			{Label: "起動の回数", Value: configview.Count(stats.SessionCount)},
			{Label: "ワールドの滞在", Value: configview.Count(stats.WorldCount)},
			{Label: "同じ部屋にいた人（のべ）", Value: configview.Count(stats.PlayerCount)},
			{Label: "アバターの切り替え", Value: configview.Count(stats.EventCount)},
			{Label: "動画", Value: configview.Count(stats.VideoCount)},
			{Label: "最終スキャン", Value: configview.FormatTime(stats.LastScan)},
			{Label: "キャッシュDB", Value: stats.CacheDBPath, Code: true},
		},
		Notices: []configview.Notice{
			configview.MissingPatterns(stats.SourceProblems),
			configview.ErrorNotice("取り込み時のエラー:", stats.BuildError),
		},
		Patterns: patterns,
		Expanded: &configview.Expanded{Dirs: src.Dirs, Files: src.Files},
		Form: &configview.Form{Fields: []configview.Field{{
			Key:   configKeySourceDirs,
			Label: "出力ログ（output_log_*.txt）の置き場所（1行に1つ。フォルダかファイル。ワイルドカードも可。保存すると次の検索から反映されます）",
			Value: strings.Join(patterns, "\n"),
		}}},
		ConfigFile: &configview.ConfigFile{
			Path:   filepath.Join(pluginDir, "config.json"),
			Sample: configview.SampleJSON(defaultConfig()),
			Notes: []string{configview.SourceDirsNote(
				"マッチしたフォルダは再帰的に走査し、`output_log_*.txt` を対象にします。"+
					"VRChat は古いログを数日で消すので、同期スクリプトなどで別の場所へ運んでいるなら、その場所も書いてください",
				"同じ起動のログが複数の場所にあるときは、一番大きいファイルから記録を作ります。",
			)},
		},
	}

	if stats.DroppedLines > 0 {
		page.Notices = append(page.Notices, configview.Notice{
			Title: fmt.Sprintf("長すぎて読み飛ばした行が **%s 行**あります。", configview.Count(stats.DroppedLines)),
			Text:  "拾う行（入室・人の出入り・アバター・動画・終了）はどれも短いので、通常は記録に影響しません。",
		})
	}
	if stats.Err != nil {
		page.Notices = append(page.Notices, configview.ErrorNotice("キャッシュを開けませんでした:", stats.Err.Error()))
	}

	if stats.RecordCount > 0 {
		page.Status = &configview.Status{
			Loaded:   true,
			Headline: fmt.Sprintf("**%s 件**を読み込んでいます", configview.Count(stats.RecordCount)),
		}
	} else {
		page.Status = &configview.Status{
			Headline: "まだ読み込まれていません。",
			Lines: []string{
				"VRChat の出力ログ（output_log_*.txt）が入ったフォルダを、下の「設定」で指定してください。",
				"取り込みはバックグラウンドで進むので、指定した直後は0件のことがあります。",
			},
		}
	}
	return configview.Render(page)
}
