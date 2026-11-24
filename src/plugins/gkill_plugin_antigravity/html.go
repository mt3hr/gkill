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
// 描画は SDK の configview が1か所で行う（ADR-0316）。ここでは Antigravity 固有の文言と統計を
// configview.Page へ写すだけで、head・保存スクリプト・進捗・一覧の体裁は持たない。
// 統計はキャッシュから即答した値だけを渡す（ここでファイルを走査すると IsAlive の5秒を超える）。
func renderConfigHTML(pluginDir string, stats cacheStats, patterns []string, src sdk.ExpandedSource) string {
	page := configview.Page{
		Title: "Antigravity チャットログプラグイン",
		Progress: configview.Progress{
			State: stats.BuildState, Done: stats.BuildDone, Total: stats.BuildTotal, Unit: "会話",
		},
		Stats: []configview.Row{
			{Label: "対象ファイル数", Value: configview.Count(stats.FileCount)},
			{Label: "会話数", Value: configview.Count(stats.ConversationCount)},
			{Label: "Kyou数", Value: configview.Count(stats.KyouCount)},
			{Label: "最終スキャン", Value: configview.FormatTime(stats.LastScan)},
			{Label: "キャッシュDB", Value: stats.CacheDBPath, Code: true},
		},
		Notices: []configview.Notice{
			configview.MissingPatterns(src.Missing),
			missingConversationIDNotice(stats.MissingIDFiles),
			shadowedNotice(stats.ShadowedCount),
			droppedLinesNotice(stats.DroppedLines),
			configview.ErrorNotice("取り込み時のエラー:", stats.BuildError),
			configview.ErrorNotice("キャッシュDBを開けません:", errorTextOf(stats.Err)),
		},
		Patterns: patterns,
		Expanded: &configview.Expanded{Dirs: src.Dirs, Files: src.Files},
		Form: &configview.Form{Fields: []configview.Field{{
			Key:   configKeySourceDirs,
			Label: "チャットログの置き場所（1行に1つ。フォルダかファイル。ワイルドカードも可。保存すると次の検索から反映されます）",
			Value: strings.Join(patterns, "\n"),
		}}},
		ConfigFile: &configview.ConfigFile{
			Path:   filepath.Join(pluginDir, "config.json"),
			Sample: configview.SampleJSON(defaultConfig()),
			Notes: []string{configview.SourceDirsNote(
				"マッチしたフォルダは再帰的に走査し、マッチしたファイルはそのまま対象にします",
				"読むのは `"+transcriptFileName+"`（会話の本文）と `"+summariesFileName+"`（会話タイトルとプロジェクト名。任意）だけです。",
				"`"+truncatedTranscriptFileName+"` と `chunks` フォルダの中は、同じ会話の写しなので読みません。",
				"キーごと省略するか空にすると `~/.gemini/antigravity` を使います。",
			)},
		},
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
				"Antigravity のチャットログが入ったフォルダを、下の「設定」で指定してください。",
				"会話IDはファイル名ではなくフォルダ名に入っているので、" +
					"`" + transcriptFileName + "` は会話IDのフォルダに入れたまま置いてください。",
				"取り込みはバックグラウンドで進むので、指定した直後は0件のことがあります。",
			},
		}
	}
	return configview.Render(page)
}

// missingConversationIDNotice は会話IDのフォルダに入っていないため読まなかったファイルを並べる。
func missingConversationIDNotice(paths []string) configview.Notice {
	if len(paths) == 0 {
		return configview.Notice{}
	}
	items := make([]configview.Item, 0, len(paths))
	for _, path := range paths {
		items = append(items, configview.Item{Code: path})
	}
	return configview.Notice{
		Title: "会話IDのフォルダに入っていないため読みませんでした:",
		Text: "会話IDは `" + transcriptFileName + "` の中には入っていないので、" +
			"会話IDのフォルダ（`<会話ID>/" + transcriptFileName + "`）に入れたまま置いてください。",
		Items: items,
	}
}

// shadowedNotice は同じ会話が複数の場所にあって使わなかった件数を出す。
func shadowedNotice(count int) configview.Notice {
	if count == 0 {
		return configview.Notice{}
	}
	return configview.Notice{
		Title: fmt.Sprintf("同じ会話のファイルが %s 件、複数の場所で見つかりました", configview.Count(count)),
		Text:  "ログは追記のみなので、会話ごとに一番大きいファイルだけを読んでいます（記録は二重になりません）。",
	}
}

// droppedLinesNotice は読めなかった行の数を出す。
func droppedLinesNotice(count int) configview.Notice {
	if count == 0 {
		return configview.Notice{}
	}
	return configview.Notice{
		Title: fmt.Sprintf("読めなかった行が %s 件あります", configview.Count(count)),
		Text:  "JSON として読めない行と、長すぎる行は飛ばしています。ほかの行は取り込まれています。",
	}
}

func errorTextOf(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
