package main

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk/configview"
)

// renderConfigHTML は設定画面のHTMLを返す。
//
// 描画は SDK の configview が1か所で行う（ADR-0316）。ここでは位置情報固有の文言と統計を
// configview.Page へ写すだけで、head・保存スクリプト・一覧の体裁は持たない。
// 設定の保存は gkill 本体の設定ダイアログが postMessage で肩代わりする（欄のキーは PostConfig が読むキーと同じ。
// チェックボックスは '1' / ” で届く）。
func renderConfigHTML(pluginDir string, config pluginConfig, stats cacheStats) string {
	page := configview.Page{
		Title: "Google ロケーション履歴 プラグイン",
		Intro: "読み込んだ位置情報は gkill の位置情報ログとして扱われ、地図と地図の絞り込みで使えます。記録（Kyou）は作りません。",
		Stats: []configview.Row{
			{Label: "取り込んだ点", Value: configview.Count(stats.TotalPoints)},
			{Label: "フィルタ後", Value: configview.Count(stats.FilteredPoints)},
			{Label: "重複除去後", Value: configview.Count(stats.UniquePoints)},
		},
		Notices: []configview.Notice{
			listNotice("この形式は検出しましたが、まだ読めません", stats.UnsupportedFiles),
			listNotice("読み込みに失敗したファイル", stats.ScanErrors),
			configview.MissingPatterns(config.Source.Missing),
			sourceProblemsNotice(stats.Problems),
		},
		Sections: []configview.Section{
			countTableSection("形式別のファイル数", "形式", "ファイル", stats.FileCountByFormat, formatLabel),
			countTableSection("測位の出所", "出所", "点", stats.PointsBySource, func(s string) string { return s }),
			exportsSection(stats.Exports),
		},
		Form: &configview.Form{Fields: []configview.Field{
			{Key: configKeySourceDirs, Label: "取り込み元（1行に1つ。ZIPを置いたフォルダ、またはZIPそのもの。ワイルドカードも可）",
				Value: strings.Join(config.Patterns, "\n")},
			{Key: configKeyAccuracyMaxMeters, Kind: configview.FieldText, Label: "精度の上限（メートル。これより粗い測位は捨てる。0以下で無効）",
				Value: strconv.Itoa(config.AccuracyMaxMeters)},
			{Key: configKeySources, Kind: configview.FieldText, Label: "測位の出所（カンマ区切り。空なら絞らない。GPS / WIFI / WIFI_ONLY / CELL / UNKNOWN / FITBIT）",
				Value: strings.Join(config.Sources, ", ")},
			{Key: configKeyMaxPoints, Kind: configview.FieldText, Label: "返す点数の上限",
				Value: strconv.Itoa(config.MaxPoints)},
			{Key: configKeyIncludeFitbitGPS, Kind: configview.FieldCheckbox, Label: "ワークアウトのトラックを含める",
				Checked: config.IncludeFitbitGPS},
			{Key: configKeyVisitPoints, Kind: configview.FieldCheckbox, Label: "滞在地・移動区間の端点も点として出す（生の測位より粗い）",
				Checked: config.VisitPoints},
		}},
		ConfigFile: &configview.ConfigFile{
			Path:   filepath.Join(pluginDir, "config.json"),
			Sample: configview.SampleJSON(defaultConfig()),
		},
	}
	if stats.OldestUnixMilli != 0 {
		page.Stats = append(page.Stats, configview.Row{Label: "期間",
			Value: formatMilli(stats.OldestUnixMilli) + " 〜 " + formatMilli(stats.NewestUnixMilli)})
	}
	page.Stats = append(page.Stats,
		configview.Row{Label: "最終スキャン", Value: configview.FormatUnix(stats.LastScanUnix)},
		configview.Row{Label: "キャッシュDB", Value: sdk.CacheDBPath(pluginDir), Code: true},
	)
	return configview.Render(page)
}

// listNotice はパスやメッセージの一覧を警告にする。0件なら空。
func listNotice(title string, lines []string) configview.Notice {
	if len(lines) == 0 {
		return configview.Notice{}
	}
	n := configview.Notice{Title: title}
	for _, line := range lines {
		n.Items = append(n.Items, configview.Item{Code: line})
	}
	return n
}

// sourceProblemsNotice は走査で見つかった問題を警告にする。0件なら空。
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

// countTableSection は「名前 → 件数」の map をキー順に並べた表にする。0件なら節ごと出ない。
func countTableSection(title string, keyHeader string, countHeader string, counts map[string]int, label func(string) string) configview.Section {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	section := configview.Section{
		Title: title,
		Max:   -1,
		Table: &configview.Table{Header: []string{keyHeader, countHeader}},
	}
	for _, key := range keys {
		section.Table.Rows = append(section.Table.Rows, []string{label(key), configview.Count(counts[key])})
	}
	return section
}

// formatLabel は形式IDの表示名。未対応の形式には印を付ける。
func formatLabel(formatID string) string {
	format, exist := formatByID[formatID]
	if !exist {
		return formatID
	}
	label := format.Label
	if format.Parse == nil {
		label += "（未対応）"
	}
	return label
}

// exportsSection は取り込み元の書き出し（Takeout の世代）の表。
func exportsSection(exports []exportRow) configview.Section {
	section := configview.Section{
		Title: "取り込み元の書き出し",
		Empty: "ZIPがまだ見つかっていません。取り込み元のフォルダに Google Takeout の ZIP を展開せずそのまま置いてください。",
		Table: &configview.Table{Header: []string{"フォルダ", "ZIP数", "書き出し日時", "対象エントリ"}},
	}
	for _, export := range exports {
		section.Table.Rows = append(section.Table.Rows, []string{
			export.Dir, configview.Count(export.ArchiveCount),
			configview.FormatUnix(export.NewestUnix), configview.Count(export.EntryCount),
		})
	}
	if len(exports) != 0 {
		section.Note = "書き出しが複数あっても、同じ時刻・同じ座標の点は1つにまとめるので二重にはなりません。" +
			"古い書き出しにしか無い期間もそのまま残ります。"
	}
	return section
}

// formatMilli は Unix ミリ秒を日付にする。0 なら空。
func formatMilli(unixMilli int64) string {
	if unixMilli == 0 {
		return ""
	}
	return time.UnixMilli(unixMilli).Local().Format("2006-01-02")
}
