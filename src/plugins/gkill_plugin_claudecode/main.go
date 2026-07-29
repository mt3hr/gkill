package main

import (
	"context"
	_ "embed"
	"os"
	"slices"
	"strings"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// configKeySourceDirs は config.json に保存するデータソースフォルダのキー。
const configKeySourceDirs = "source_dirs"

// manifest.json をバイナリに埋め込み、--gkill-print-manifest で出力できるようにする。
// 配置スクリプトが manifest.json を用意できるようにするため。
// バイナリと manifest が必ず一致するので、別々に配る必要がない。
//
//go:embed manifest.json
var manifestJSON []byte

func extractPluginDir(args []string) string {
	for i, a := range args {
		if a == "--gkill-plugin-dir" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// sourcePatternsOf はデータソースのパターン一覧を取り出す。
//
// SDKはconfig.jsonをプロセス起動時に一度だけ読むが、この設定はプラグインフォルダの
// config.jsonを手で編集して変えるものなので、毎回読み直して即座に反映されるようにする。
// 読めなかったときは起動時に読んだ設定にフォールバックする。
func sourcePatternsOf(pluginDir string, cfg sdk.Config) []string {
	c := cfg
	if latest, err := sdk.LoadConfig(pluginDir); err == nil {
		c = latest
	}
	if c == nil {
		return parseSourcePatterns(nil)
	}
	return parseSourcePatterns(c[configKeySourceDirs])
}

// sourceOf は設定のパターンを実在するフォルダ・ファイルへ展開する。
func sourceOf(pluginDir string, cfg sdk.Config) expandedSource {
	return expandSourcePatterns(sourcePatternsOf(pluginDir, cfg))
}

func main() {
	// manifest.json の出力だけして終わる。sdk.Run より前に処理する
	// (sdk.Run の flag.Parse は知らないフラグを受け取るとエラー終了するため)
	if slices.Contains(os.Args[1:], "--gkill-print-manifest") {
		if _, err := os.Stdout.Write(manifestJSON); err != nil {
			os.Exit(1)
		}
		return
	}

	pluginDir := extractPluginDir(os.Args)

	sdk.Run(sdk.Handler{
		RepName: repName,

		FindKyous: func(_ context.Context, q sdk.Query, cfg sdk.Config) ([]sdk.Kyou, error) {
			turns, err := globalCache.GetTurns(pluginDir, sourceOf(pluginDir, cfg))
			if err != nil {
				return []sdk.Kyou{}, nil
			}

			var kyous []sdk.Kyou
			for _, t := range turns {
				relatedTime := unixToTime(t.RelatedTimeUnix)

				// カレンダーフィルタ
				if q.CalendarStartDate != nil && relatedTime.Before(*q.CalendarStartDate) {
					continue
				}
				if q.CalendarEndDate != nil && relatedTime.After(*q.CalendarEndDate) {
					continue
				}

				// ワードフィルタ（ターン単位）
				if !matchWordsText(t.SearchText, q) {
					continue
				}

				updateTime := unixToTime(t.UpdateTimeUnix)
				if updateTime.IsZero() {
					updateTime = relatedTime
				}

				kyous = append(kyous, sdk.Kyou{
					ID:          t.TurnID,
					RepName:     repName,
					DataType:    dataType,
					RelatedTime: relatedTime,
					CreateTime:  relatedTime,
					UpdateTime:  updateTime,
					CreateApp:   "gkill_plugin_claudecode",
					UpdateApp:   "gkill_plugin_claudecode",
				})
			}

			if q.Limit > 0 && len(kyous) > q.Limit {
				kyous = kyous[:q.Limit]
			}
			return kyous, nil
		},

		GetContentHTML: func(_ context.Context, kyouID string, cfg sdk.Config) (string, error) {
			t, err := globalCache.GetTurn(pluginDir, sourceOf(pluginDir, cfg), kyouID)
			if err != nil {
				return renderNotFoundHTML(), nil
			}
			return renderTurnHTML(t), nil
		},

		GetConfigHTML: func(_ context.Context, cfg sdk.Config) (string, error) {
			patterns := sourcePatternsOf(pluginDir, cfg)
			src := expandSourcePatterns(patterns)
			stats := globalCache.GetStats(pluginDir, src)
			return renderConfigHTML(pluginDir, stats, patterns, src), nil
		},

		PostConfig: func(_ context.Context, form map[string]string, cfg sdk.Config) (sdk.Config, error) {
			if cfg == nil {
				cfg = sdk.Config{}
			}
			if v, ok := form[configKeySourceDirs]; ok {
				cfg[configKeySourceDirs] = strings.TrimSpace(v)
			}
			// 対象から外れたフォルダのターンは、次回のスキャンで
			// 「消えたファイル」として自動的にキャッシュから削除される。
			return cfg, nil
		},
	})
}

// matchWordsText はワード検索条件にテキストが合致するかチェックする。
func matchWordsText(text string, q sdk.Query) bool {
	if len(q.Words) == 0 && len(q.NotWords) == 0 {
		return true
	}

	target := strings.ToLower(text)

	if len(q.Words) > 0 {
		if q.WordsAnd {
			for _, w := range q.Words {
				if !strings.Contains(target, strings.ToLower(w)) {
					return false
				}
			}
		} else {
			matched := false
			for _, w := range q.Words {
				if strings.Contains(target, strings.ToLower(w)) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}

	for _, w := range q.NotWords {
		if strings.Contains(target, strings.ToLower(w)) {
			return false
		}
	}
	return true
}

// タグによる絞り込みは行わない。
// gkillのタグ一覧(get_all_tag_names)にはプラグインが返したタグが載らないため、
// タグを付けるとrykvの既定の絞り込み「no tags」から漏れて何も表示されなくなる。
// Claude.ai/ChatGPTプラグインも同様にタグを扱わない。
// プロジェクト名やブランチ名は searchTextOf に含めてワード検索で引けるようにしている。
