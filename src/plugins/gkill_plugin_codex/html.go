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
// 描画は SDK の configview が1か所で行う（ADR-0316）。ここでは Codex 固有の文言と統計を
// configview.Page へ写すだけで、head・保存スクリプト・進捗・一覧の体裁は持たない。
//
// キャッシュの状態(cache_meta)しか読まない。ここでファイルを走査してはいけない ――
// この画面は IsAlive(5秒)と同じスロットに並ぶので、走査すると殺される。
func renderConfigHTML(pluginDir string, stats cacheStats, patterns []string, src sdk.ExpandedSource) string {
	progress := configview.Progress{
		State: stats.BuildState, Done: stats.BuildDone, Total: stats.BuildTotal, Unit: "ファイル",
	}
	if stats.BuildState == "folding" {
		progress.Total = 0 // 畳み直しはファイル数ではなくグループ数で進む
		progress.Note = fmt.Sprintf("畳み直し待ち %s グループ", configview.Count(stats.DirtyCount))
	}

	page := configview.Page{
		Title:    "Codex チャットログプラグイン",
		Progress: progress,
		Stats: []configview.Row{
			{Label: "対象ファイル数", Value: configview.Count(stats.TargetFileCount)},
			{Label: "取り込み済みファイル", Value: configview.Count(stats.FileCount)},
			{Label: "スレッド数", Value: fmt.Sprintf("%s (うちサブエージェント %s)",
				configview.Count(stats.ThreadCount), configview.Count(stats.SubAgentCount))},
			{Label: "Kyou数", Value: configview.Count(stats.KyouCount)},
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
			Key:   "source_dirs",
			Label: "ロールアウト JSONL の置き場所（1行に1つ。フォルダかファイル。ワイルドカードも可。保存すると次の検索から反映されます）",
			Value: strings.Join(patterns, "\n"),
		}}},
		ConfigFile: &configview.ConfigFile{
			Path:   filepath.Join(pluginDir, "config.json"),
			Sample: configview.SampleJSON(defaultConfig()),
			Notes: []string{configview.SourceDirsNote(
				"マッチしたフォルダは再帰的に走査し、マッチしたファイルはそのまま対象にします。"+
					"`session_index.jsonl` も指定しておくとスレッド名が表示されます",
				"`subagent_mode` は `fold`(既定)でサブエージェントの会話を親の応答に畳み込み、"+
					"`own_kyou` にすると独立したKyouにします。",
			)},
		},
	}

	if stats.DroppedLines > 0 {
		page.Notices = append(page.Notices, configview.Notice{
			Title: fmt.Sprintf("大きすぎて読み飛ばした行が **%s 行**あります。", configview.Count(stats.DroppedLines)),
			Text:  "ログの形式が変わった可能性があります。",
		})
	}
	if stats.UnknownKinds != "" {
		page.Notices = append(page.Notices, configview.Notice{
			Title: "知らない種類のレコードがありました: `" + stats.UnknownKinds + "`",
			Text:  "Codex 側の形式が増えたのかもしれません。取り込みは続いています。",
		})
	}
	if stats.RewriteWarning != "" {
		page.Notices = append(page.Notices, configview.Notice{
			Title: "ログが書き換えられた可能性があります:",
			Text:  "このスレッドのKyouは別のIDで作り直され、付けたタグやテキストが外れることがあります。",
			Items: []configview.Item{{Code: stats.RewriteWarning}},
		})
	}
	if stats.Err != nil {
		page.Notices = append(page.Notices, configview.ErrorNotice("キャッシュを開けませんでした:", stats.Err.Error()))
	}

	if stats.KyouCount > 0 {
		page.Status = &configview.Status{
			Loaded:   true,
			Headline: fmt.Sprintf("**%s 件**を読み込んでいます", configview.Count(stats.KyouCount)),
		}
	} else {
		page.Status = &configview.Status{
			Headline: "まだ読み込まれていません。",
			Lines: []string{
				"Codex のセッションログ(ロールアウトJSONL)が入ったフォルダを、下の「設定」で指定してください。",
				"取り込みはバックグラウンドで進むので、指定した直後は0件のことがあります。",
			},
		}
	}
	return configview.Render(page)
}
