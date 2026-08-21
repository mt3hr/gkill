package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"sync/atomic"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// configKeySourceDirs は config.json に保存するデータソースフォルダのキー。
const configKeySourceDirs = "source_dirs"

// JSONにはコメントが書けないので、読み飛ばされるキーで書式を書き残す。
// parseSourcePatterns は source_dirs しか見ないため、残っていても害はない。
const (
	configKeyComment           = "_comment"
	configKeyExampleSourceDirs = "_example_source_dirs"
)

// defaultConfig は config.json が無いときに書き出す既定設定。
// pluginDir に依存しない値にすること(--gkill-print-config から同じものを出すため)。
// 既定値は defaultSourceDir() と同じ場所を ~ 表記で書く。
func defaultConfig() sdk.Config {
	return sdk.Config{
		configKeyComment: "source_dirs にフォルダかファイルのパスを書きます。" +
			"* ** ? [] のワイルドカード、先頭の ~ と環境変数($HOME など)が使えます。" +
			"フォルダを指定すると再帰的に走査してセッションログ(.jsonl)を探し、" +
			"ファイルを直接指定するとその中身をそのまま読みます。" +
			"空にすると ~/.claude/projects を見ます。" +
			"編集は次の検索から反映されます(gkillの再起動は不要)。" +
			"_ で始まるキーは説明用なので消して構いません。",
		configKeyExampleSourceDirs: []string{
			"~/.claude/projects",
			"D:/backup/ClaudeCode_*/**/*.jsonl",
		},
		configKeySourceDirs: []string{"~/.claude/projects"},
	}
}

// printDefaultConfig は既定の config.json を標準出力に書く。--gkill-print-config 用。
func printDefaultConfig() error {
	data, err := json.MarshalIndent(defaultConfig(), "", "  ")
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(data, '\n'))
	return err
}

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

// latestConfig はハンドラが受け取った最新の設定。ビルダから読む。
var latestConfig atomic.Pointer[sdk.Config]

func rememberConfig(cfg sdk.Config) {
	if cfg == nil {
		return
	}
	latestConfig.Store(&cfg)
}

// sourceProviderOf はビルダに渡す「今のデータソースを返す関数」を作る。
// 設定画面や config.json の編集を反映できるよう、呼ばれるたびに読み直す。
func sourceProviderOf(pluginDir string) func() expandedSource {
	return func() expandedSource {
		var base sdk.Config
		if stored := latestConfig.Load(); stored != nil {
			base = *stored
		}
		return sourceOf(pluginDir, base)
	}
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

	// 既定の config.json の出力だけして終わる。配置スクリプトが用意できるようにするため。
	// 通常はプラグイン起動時に自動生成されるので、これを使う必要はない。
	if slices.Contains(os.Args[1:], "--gkill-print-config") {
		if err := printDefaultConfig(); err != nil {
			os.Exit(1)
		}
		return
	}

	pluginDir := extractPluginDir(os.Args)
	provider := sourceProviderOf(pluginDir)

	sdk.Run(sdk.Handler{
		RepName:       repName,
		DefaultConfig: defaultConfig(),

		// ハンドラは全部「ビルダを起こして、今キャッシュにあるぶんを即返す」。
		// 初回が空なのは仕様。取り込みはバックグラウンドで進む。
		FindKyous: func(_ context.Context, q sdk.Query, cfg sdk.Config) ([]sdk.Kyou, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			messages, err := globalCache.GetMessages(pluginDir)
			if err != nil {
				return []sdk.Kyou{}, nil
			}

			var kyous []sdk.Kyou
			for _, t := range messages {
				relatedTime := unixToTime(t.RelatedTimeUnix)

				// カレンダーフィルタ
				if q.CalendarStartDate != nil && relatedTime.Before(*q.CalendarStartDate) {
					continue
				}
				if q.CalendarEndDate != nil && relatedTime.After(*q.CalendarEndDate) {
					continue
				}

				// ワードフィルタ（発言単位）
				if !matchWordsText(t.SearchText, q) {
					continue
				}

				updateTime := unixToTime(t.UpdateTimeUnix)
				if updateTime.IsZero() {
					updateTime = relatedTime
				}

				kyous = append(kyous, sdk.Kyou{
					ID:          t.MessageID,
					RepName:     repName,
					DataType:    dataType,
					RelatedTime: relatedTime,
					CreateTime:  relatedTime,
					UpdateTime:  updateTime,
					CreateApp:   appName,
					UpdateApp:   appName,
				})
			}

			if q.Limit > 0 && len(kyous) > q.Limit {
				kyous = kyous[:q.Limit]
			}
			return kyous, nil
		},

		GetContentHTML: func(_ context.Context, kyouID string, cfg sdk.Config) (string, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			t, err := globalCache.GetMessage(pluginDir, kyouID)
			if err != nil {
				return renderNotFoundHTML(), nil
			}
			return renderMessageHTML(t), nil
		},

		// ここでファイルを走査してはいけない。IsAlive(5秒)と同じスロットに並ぶ。
		GetConfigHTML: func(_ context.Context, cfg sdk.Config) (string, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			patterns := sourcePatternsOf(pluginDir, cfg)
			src := expandSourcePatterns(patterns)
			stats := globalCache.GetStats(pluginDir)
			return renderConfigHTML(pluginDir, stats, patterns, src), nil
		},

		PostConfig: func(_ context.Context, form map[string]string, cfg sdk.Config) (sdk.Config, error) {
			if cfg == nil {
				cfg = sdk.Config{}
			}
			if v, ok := form[configKeySourceDirs]; ok {
				// 設定画面のテキストエリアは1行1指定。config.json には配列で書き戻す
				// (1行の文字列でも parseSourcePatterns は読めるが、配列のほうが手で編集しやすい)。
				cfg[configKeySourceDirs] = splitSourceDirsForm(v)
			}
			rememberConfig(cfg)
			// 対象から外れたフォルダのターンは、次回のスキャンで
			// 「消えたファイル」として自動的にキャッシュから削除される。
			globalBuilder.Kick()
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

// splitSourceDirsForm は設定画面のテキストエリア(1行1指定)を config.json 用の配列にする。
// 空行は落とす。展開(~ や環境変数)はしない —— 書いたとおりを保存し、
// 読み出し時に parseSourcePatterns が展開する。
func splitSourceDirsForm(v string) []string {
	dirs := make([]string, 0)
	for line := range strings.SplitSeq(v, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" {
			continue
		}
		dirs = append(dirs, line)
	}
	return dirs
}
