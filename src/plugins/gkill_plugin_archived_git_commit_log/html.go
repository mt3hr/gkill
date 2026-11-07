package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk/configview"
)

// maxShownRepos は設定画面に並べるリポジトリの最大件数。
const maxShownRepos = 100

// renderConfigHTML は設定画面のHTMLを返す。
//
// 描画は SDK の configview が1か所で行う（ADR-0316）。ここでは zip の Git リポジトリ固有の文言と統計を
// configview.Page へ写すだけで、head・保存スクリプト・進捗・一覧の体裁は持たない。
//
// キャッシュの状態(cache_meta と repo 表)しか読まない。ここで zip を開いてはいけない ――
// この画面は IsAlive(5秒)と同じスロットに並ぶので、走査すると殺される。
func renderConfigHTML(pluginDir string, stats cacheStats, patterns []string) string {
	page := configview.Page{
		Title: "アーカイブ Git コミットログプラグイン",
		Progress: configview.Progress{
			State: stats.BuildState, Done: stats.BuildDone, Total: stats.BuildTotal, Unit: "リポジトリ",
		},
		Stats: []configview.Row{
			{Label: "対象リポジトリ数", Value: configview.Count(stats.TargetRepoCount)},
			{Label: "取り込み済みリポジトリ", Value: configview.Count(stats.RepoCount)},
			{Label: "rep 名の数", Value: configview.Count(stats.RepNameCount)},
			{Label: "コミット数", Value: configview.Count(stats.CommitCount)},
			{Label: "最終スキャン", Value: configview.FormatTime(stats.LastScan)},
			{Label: "キャッシュDB", Value: stats.CacheDBPath, Code: true},
		},
		Patterns: patterns,
		Form: &configview.Form{Fields: []configview.Field{{
			Key:   "source_dirs",
			Label: "Git リポジトリを固めた zip の実パスか、zip を置いたフォルダ（1行に1つ。ワイルドカードも可。保存すると次の検索から反映されます）",
			Value: strings.Join(patterns, "\n"),
			Rows:  6,
		}}},
		ConfigFile: &configview.ConfigFile{
			Path:   filepath.Join(pluginDir, "config.json"),
			Sample: configview.SampleJSON(defaultConfig()),
			Notes: []string{configview.SourceDirsNote(
				"zip の実パスを書くとその zip だけを、フォルダを書くと配下の `*.zip` を再帰的に読みます",
				"zip は展開せず中の `.git` だけを読むので、`.git` の無い zip は読み飛ばします。"+
					"同じリポジトリが複数の zip に入っていても、コミットは1件ずつです。"+
					"`max_git_dir_mb` は1リポジトリの `.git` の合計サイズの上限(MB)です。",
			)},
		},
	}

	if stats.StatsErrorCount > 0 {
		page.Notices = append(page.Notices, configview.Notice{
			Title: fmt.Sprintf("行数を集計できなかったコミットが **%s 件**あります（コミット自体は表示されます）。",
				configview.Count(stats.StatsErrorCount)),
		})
	}
	if len(stats.SourceProblems) > 0 {
		n := configview.Notice{Title: "走査で見つかった問題:"}
		for _, problem := range stats.SourceProblems {
			n.Items = append(n.Items, configview.Item{Code: problem.Path, Text: problem.Message})
		}
		page.Notices = append(page.Notices, n)
	}
	unreadable := configview.Notice{Title: "読めなかったリポジトリ:"}
	for _, repo := range stats.Repos {
		if repo.Note == "" {
			continue
		}
		unreadable.Items = append(unreadable.Items, configview.Item{Code: repo.ArchivePath + "!/" + repo.GitDir, Text: repo.Note})
	}
	if len(unreadable.Items) > 0 {
		page.Notices = append(page.Notices, unreadable)
	}
	page.Notices = append(page.Notices, configview.ErrorNotice("取り込み時のエラー:", stats.BuildError))
	if stats.Err != nil {
		page.Notices = append(page.Notices, configview.ErrorNotice("キャッシュを開けませんでした:", stats.Err.Error()))
	}

	if len(stats.Repos) > 0 {
		repos := configview.Section{
			Title: fmt.Sprintf("取り込んだリポジトリ (%s)", configview.Count(len(stats.Repos))),
			Max:   maxShownRepos,
		}
		for _, repo := range stats.Repos {
			repos.Items = append(repos.Items, configview.Item{
				Label: repo.RepName,
				Code:  repo.ArchivePath + "!/" + repo.GitDir,
				Text:  fmt.Sprintf("%s コミット", configview.Count(repo.CommitCount)),
			})
		}
		page.Sections = append(page.Sections, repos)
	}

	if stats.CommitCount > 0 {
		page.Status = &configview.Status{
			Loaded: true,
			Headline: fmt.Sprintf("**%s リポジトリ・%s コミット**を読み込んでいます",
				configview.Count(stats.RepNameCount), configview.Count(stats.CommitCount)),
		}
	} else {
		page.Status = &configview.Status{
			Headline: "まだ読み込まれていません。",
			Lines: []string{
				"Git リポジトリを固めた zip の実パスか、zip を置いたフォルダを、下の「設定」で指定してください。",
				"取り込みはバックグラウンドで進むので、指定した直後は0件のことがあります。",
			},
		}
	}
	return configview.Render(page)
}
