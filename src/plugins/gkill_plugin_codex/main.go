package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// manifest.json をバイナリに埋め込み、--gkill-print-manifest で出力できるようにする。
// 配置スクリプトが manifest.json を用意できるようにするため。
// バイナリと manifest が必ず一致するので、別々に配る必要がない。
//
//go:embed manifest.json
var manifestJSON []byte

// latestConfig はハンドラが受け取った最新の設定。ビルダから読む。
var latestConfig atomic.Pointer[sdk.Config]

func rememberConfig(cfg sdk.Config) {
	if cfg == nil {
		return
	}
	latestConfig.Store(&cfg)
}

// configProviderOf はビルダに渡す「今の設定を返す関数」を作る。
func configProviderOf(pluginDir string) func() pluginConfig {
	return func() pluginConfig {
		var base sdk.Config
		if stored := latestConfig.Load(); stored != nil {
			base = *stored
		}
		return configOf(pluginDir, base)
	}
}

func extractPluginDir(args []string) string {
	for i, arg := range args {
		if arg == "--gkill-plugin-dir" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
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

	// 既定の config.json の出力だけして終わる。配置スクリプトが用意できるようにするため。
	// 通常はプラグイン起動時に自動生成されるので、これを使う必要はない。
	if slices.Contains(os.Args[1:], "--gkill-print-config") {
		if err := printDefaultConfig(); err != nil {
			os.Exit(1)
		}
		return
	}

	pluginDir := extractPluginDir(os.Args)
	provider := configProviderOf(pluginDir)

	sdk.Run(sdk.Handler{
		RepName:       repName,
		DefaultConfig: defaultConfig(),

		// ハンドラは全部「ビルダを起こして、今キャッシュにあるぶんを即返す」。
		// 初回が空なのは仕様。取り込みはバックグラウンドで進む。
		FindKyous: func(_ context.Context, q sdk.Query, cfg sdk.Config) ([]sdk.Kyou, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			// 単語で絞るなら LIMIT を SQL へ押し込まない。
			// 絞る前に切ると、後段のフィルタで落ちたぶん取りこぼす。
			hasWordFilter := len(q.Words) != 0 || len(q.NotWords) != 0
			rows, err := globalCache.QueryKyous(pluginDir, q.CalendarStartDate, q.CalendarEndDate, q.Limit, hasWordFilter)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: find kyous: %v\n", appName, err)
				return []sdk.Kyou{}, nil
			}

			kyous := make([]sdk.Kyou, 0, len(rows))
			for _, row := range rows {
				// スレッド名は search_text に焼いていないので、照合時に連結する
				if !matchWordsText(row.SearchText+"\n"+row.Title, q) {
					continue
				}
				kyous = append(kyous, kyouOf(row))
				if q.Limit > 0 && len(kyous) >= q.Limit {
					break
				}
			}
			return kyous, nil
		},

		// SDKの既定実装は FindKyous を全件やり直して線形探索するので必ず自前で持つ。
		// 一覧は行数ぶんこれを呼ぶ。
		GetKyou: func(_ context.Context, id string, cfg sdk.Config) (*sdk.Kyou, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			row, err := globalCache.QueryKyou(pluginDir, id)
			if err != nil {
				return nil, nil
			}
			kyou := kyouOf(row)
			return &kyou, nil
		},

		GetContentHTML: func(_ context.Context, kyouID string, cfg sdk.Config) (string, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			built, err := globalCache.QueryBody(pluginDir, kyouID)
			if err != nil {
				return renderNotFoundHTML(), nil
			}
			return renderMessageHTML(built), nil
		},

		// ここでファイルを走査してはいけない。IsAlive(5秒)と同じスロットに並ぶ。
		GetConfigHTML: func(_ context.Context, cfg sdk.Config) (string, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			config := configOf(pluginDir, cfg)
			stats := globalCache.Stats(pluginDir)
			return renderConfigHTML(pluginDir, stats, config.Patterns, expandPatterns(config.Patterns)), nil
		},

		PostConfig: func(_ context.Context, form map[string]string, cfg sdk.Config) (sdk.Config, error) {
			if cfg == nil {
				cfg = sdk.Config{}
			}
			if v, ok := form[configKeySourceDirs]; ok {
				// 設定画面のテキストエリアは1行1指定。config.json には配列で書き戻す
				// (1行の文字列でも ParseSourcePatterns は読めるが、配列のほうが手で編集しやすい)。
				cfg[configKeySourceDirs] = splitSourceDirsForm(v)
			}
			if v, ok := form[configKeySubagentMode]; ok {
				cfg[configKeySubagentMode] = parseSubagentMode(v)
			}
			rememberConfig(cfg)
			// 対象から外れたファイルは、次の走査で「消えたファイル」として
			// 自動的にキャッシュから削除される
			globalBuilder.Kick()
			return cfg, nil
		},
	})
}

// kyouOf はキャッシュの1行を gkill へ返す Kyou にする。
//
// タグは付けない。gkill 1.1.7 以降は manifest.json に "provides": ["tag"] を書けば
// プラグインのタグもタグ一覧(get_all_tag_names)に載るようになったが、
// 同梱の gkill_plugin_claudecode と揃えて宣言していない。
// プロジェクト名・ブランチ・モデル名は search_text に入れてワード検索で引ける。
func kyouOf(row kyouRow) sdk.Kyou {
	relatedTime := unixToTime(row.RelatedUnix)
	updateTime := unixToTime(row.UpdateUnix)
	if updateTime.IsZero() {
		updateTime = relatedTime
	}
	return sdk.Kyou{
		ID:           row.ID,
		RepName:      repName,
		DataType:     dataType,
		RelatedTime:  relatedTime,
		CreateTime:   relatedTime,
		UpdateTime:   updateTime,
		CreateApp:    appName,
		UpdateApp:    appName,
		CreateDevice: row.Originator,
		UpdateDevice: row.Originator,
	}
}

// unixToTime はUNIX秒を時刻にする。0はゼロ値のまま返す。
func unixToTime(unix int64) time.Time {
	if unix == 0 {
		return time.Time{}
	}
	return time.Unix(unix, 0).UTC()
}

// matchWordsText はワード検索条件にテキストが合致するかチェックする。
func matchWordsText(text string, q sdk.Query) bool {
	if len(q.Words) == 0 && len(q.NotWords) == 0 {
		return true
	}

	target := strings.ToLower(text)

	if len(q.Words) > 0 {
		if q.WordsAnd {
			for _, word := range q.Words {
				if !strings.Contains(target, strings.ToLower(word)) {
					return false
				}
			}
		} else {
			matched := false
			for _, word := range q.Words {
				if strings.Contains(target, strings.ToLower(word)) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}

	for _, word := range q.NotWords {
		if strings.Contains(target, strings.ToLower(word)) {
			return false
		}
	}
	return true
}
