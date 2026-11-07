package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk/configview"
)

// renderConfigHTML は設定画面のHTMLを返す。
//
// 描画は SDK の configview が1か所で行う（ADR-0316）。ここでは fitbit 固有の文言と統計を
// configview.Page へ写すだけで、head・保存スクリプト・進捗・一覧の体裁は持たない。
// 設定の保存は gkill 本体の設定ダイアログが postMessage で肩代わりする（欄のキーは PostConfig が読むキーと同じ）。
func renderConfigHTML(pluginDir string, config pluginConfig, stats cacheStats) string {
	page := configview.Page{
		Title: "Fitbit (Google Takeout) プラグイン",
		// 取り込みの進捗。バックグラウンドで作っているので、ここでしか見えない。
		Progress: configview.Progress{
			State: stats.BuildState, Done: stats.BuildDoneFiles, Total: stats.BuildTotalFiles, Unit: "ファイル",
			Note: "初回は1〜2分かかることがあります。取り込みが終わった日から順に表示されます。",
		},
		Stats: []configview.Row{
			{Label: "対象ファイル数", Value: configview.Count(stats.TargetFileCount)},
			{Label: "取り込み済みファイル", Value: configview.Count(stats.ScannedFileCount)},
			{Label: "日数", Value: configview.Count(stats.DayCount)},
			{Label: "指標", Value: configview.Count(stats.MetricCount)},
			{Label: "記録数", Value: configview.Count(stats.KyouCount)},
			{Label: "最終スキャン", Value: configview.FormatUnix(stats.LastScanUnix)},
			{Label: "タイムゾーン", Value: stats.Timezone},
			{Label: "キャッシュDB", Value: sdk.CacheDBPath(pluginDir), Code: true},
		},
		Notices: []configview.Notice{
			configview.ErrorNotice("取り込み時のエラー:", stats.BuildError),
			configview.MissingPatterns(config.Source.Missing),
			sourceProblemsNotice(stats.Problems),
		},
		Sections: []configview.Section{
			exportsSection(stats.Exports),
			metricRegistrySection(),
		},
		Form: &configview.Form{Fields: []configview.Field{
			{Key: configKeySourceDirs, Label: "取り込み元（1行に1つ。ZIPを置いたフォルダ、またはZIPそのもの。ワイルドカードも可）",
				Value: strings.Join(config.Patterns, "\n")},
			{Key: configKeyTimezone, Kind: configview.FieldText, Label: "タイムゾーン（「この日はどの日か」の判定に使います）",
				Value: config.Timezone},
			{Key: configKeyMetrics, Label: "取り込む指標（1行に1つ。空欄なら全部。キーは上の「指標一覧」を参照）",
				Value: strings.Join(config.Metrics, "\n"), Rows: 3},
			{Key: configKeySecondaryDataSources, Label: "時計の行が無い日にだけ使うデータソース（1行に1つ。CSV の data source 列の値。" +
				"歩数の CSV にはスマホ(Phone Health Connect)と時計の行が同じ日に並ぶことがあり、両方を足すと2倍になります。" +
				"ここに書いたソースは時計の行がある日には使いません。空欄なら全部を合算します）",
				Value: strings.Join(config.SecondaryDataSources, "\n"), Rows: 3},
			{Key: configKeyScanWorkers, Kind: configview.FieldText, Label: "同時に読むファイル数（0 なら自動）",
				Value: strconv.Itoa(config.ScanWorkers)},
		}},
		ConfigFile: &configview.ConfigFile{
			Path:   filepath.Join(pluginDir, "config.json"),
			Sample: configview.SampleJSON(defaultConfig()),
		},
	}
	return configview.Render(page)
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

// exportsSection は取り込み元の書き出し（Takeout の世代）の表。
func exportsSection(exports []exportRow) configview.Section {
	section := configview.Section{
		Title: "取り込み元の書き出し",
		Empty: "ZIPがまだ見つかっていません。取り込み元のフォルダに Google Takeout の ZIP を展開せずそのまま置いてください。",
		Table: &configview.Table{Header: []string{"採用順", "フォルダ", "ZIP数", "書き出し日時", "日数"}},
	}
	for _, export := range exports {
		label := "新しい"
		if export.Rank != 0 {
			// 日が重なったぶんだけ上位に譲る。重ならない日はこの書き出しの値が使われる
			label = fmt.Sprintf("%d 番目", export.Rank+1)
		}
		section.Table.Rows = append(section.Table.Rows, []string{
			label, export.Dir, configview.Count(export.ArchiveCount),
			configview.FormatUnix(export.NewestUnix), configview.Count(export.DayCount),
		})
	}
	if len(exports) != 0 {
		section.Note = "同じフォルダに置いた分割ZIP(-001 -002 …)は1つの書き出しとして合算します。" +
			"書き出しが複数あるときは、日が重なったぶんだけ新しいほうの値を使います(合算しません)。"
	}
	return section
}

// metricRegistrySection は取り込める指標の一覧（折りたたみ）。設定欄 metrics のキーの参照先。
func metricRegistrySection() configview.Section {
	section := configview.Section{
		Title:     fmt.Sprintf("指標一覧 (%d件)", len(metricRegistry)),
		Collapsed: true,
		Max:       -1,
		Table:     &configview.Table{Header: []string{"キー", "名前", "単位", "集計"}},
	}
	for _, def := range metricRegistry {
		section.Table.Rows = append(section.Table.Rows, []string{def.Key, def.Title, def.Unit, aggLabel(def.Agg)})
	}
	return section
}

// aggLabel は集計方法の表示名。
func aggLabel(agg aggKind) string {
	switch agg {
	case aggSum:
		return "日合計"
	case aggMean:
		return "日平均"
	case aggMax:
		return "日最大"
	case aggMin:
		return "日最小"
	case aggLast:
		return "その日の値"
	case aggCount:
		return "件数"
	}
	return ""
}
