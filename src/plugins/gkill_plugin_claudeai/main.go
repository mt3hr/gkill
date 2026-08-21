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

const repName = "Claude.ai"
const dataType = "claude_conversation"

// configKeySourceDirs は config.json に書くデータソースの指定。
// 未設定ならプラグインフォルダ自身を見る(従来どおりの配置で動く)。
const configKeySourceDirs = "source_dirs"

// JSONにはコメントが書けないので、読み飛ばされるキーで書式を書き残す。
// parseSourcePatterns は source_dirs しか見ないため、残っていても害はない。
const (
	configKeyComment           = "_comment"
	configKeyExampleSourceDirs = "_example_source_dirs"
)

// defaultConfig は config.json が無いときに書き出す既定設定。
// pluginDir に依存しない値にすること(--gkill-print-config から同じものを出すため)。
func defaultConfig() sdk.Config {
	return sdk.Config{
		configKeyComment: "source_dirs にフォルダかファイルのパスを書きます。" +
			"* ** ? [] のワイルドカード、先頭の ~ と環境変数($HOME など)が使えます。" +
			"フォルダを指定すると再帰的に走査して conversations.json を探し、" +
			"ファイルを直接指定するとその中身をそのまま読みます。" +
			"空にするとこのプラグインのフォルダを見ます。" +
			"編集は次の検索から反映されます(gkillの再起動は不要)。" +
			"_ で始まるキーは説明用なので消して構いません。",
		configKeyExampleSourceDirs: []string{
			"~/Kyou/ClaudeAIExport",
			"D:/Dropbox/claude_export/**/conversations*.json",
		},
		configKeySourceDirs: []string{},
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

func extractpluginDir(args []string) string {
	for i, a := range args {
		if a == "--gkill-plugin-dir" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// sourcePatternsOf はデータソースの指定を取り出す。
// SDKはconfig.jsonを起動時に一度しか読まないが、この設定は手で書き換えるものなので
// 毎回読み直して即座に反映されるようにする。
func sourcePatternsOf(pluginDir string, cfg sdk.Config) []string {
	c := cfg
	if latest, err := sdk.LoadConfig(pluginDir); err == nil {
		c = latest
	}
	if c == nil {
		return parseSourcePatterns(nil, pluginDir)
	}
	return parseSourcePatterns(c[configKeySourceDirs], pluginDir)
}

// sourceOf は設定を実在するフォルダ・ファイルへ展開する。
func sourceOf(pluginDir string, cfg sdk.Config) expandedSource {
	return expandSourcePatterns(sourcePatternsOf(pluginDir, cfg))
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

	pluginDir := extractpluginDir(os.Args)
	provider := sourceProviderOf(pluginDir)

	sdk.Run(sdk.Handler{
		RepName:       repName,
		DefaultConfig: defaultConfig(),

		// ハンドラは全部「ビルダを起こして、今キャッシュにあるぶんを即返す」。
		// 初回が空なのは仕様。取り込みはバックグラウンドで進む。
		FindKyous: func(ctx context.Context, q sdk.Query, cfg sdk.Config) ([]sdk.Kyou, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			msgs, err := globalCache.GetMessages(pluginDir)
			if err != nil {
				return []sdk.Kyou{}, nil
			}

			var kyous []sdk.Kyou
			for _, msg := range msgs {
				relatedTime := unixToTimeFromCache(msg.RelatedTimeUnix)

				// カレンダーフィルタ
				if q.CalendarStartDate != nil && relatedTime.Before(*q.CalendarStartDate) {
					continue
				}
				if q.CalendarEndDate != nil && relatedTime.After(*q.CalendarEndDate) {
					continue
				}

				// ワードフィルタ（メッセージ単位）
				if !matchWordsText(msg.Text, q) {
					continue
				}

				createTime := unixToTimeFromCache(msg.CreateTimeUnix)
				updateTime := unixToTimeFromCache(msg.UpdateTimeUnix)
				if updateTime.IsZero() {
					updateTime = createTime
				}

				k := sdk.Kyou{
					ID:          msg.MsgID,
					RepName:     repName,
					DataType:    dataType,
					RelatedTime: relatedTime,
					CreateTime:  createTime,
					UpdateTime:  updateTime,
					CreateApp:   "gkill_plugin_claudeai",
					UpdateApp:   "gkill_plugin_claudeai",
				}
				kyous = append(kyous, k)
			}

			if q.Limit > 0 && len(kyous) > q.Limit {
				kyous = kyous[:q.Limit]
			}

			return kyous, nil
		},

		GetContentHTML: func(ctx context.Context, kyouID string, cfg sdk.Config) (string, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			convTitle, msg, err := globalCache.GetMsgByID(pluginDir, kyouID)
			if err != nil {
				return "<html><body><p>メッセージが見つかりません</p></body></html>", nil
			}
			return renderSingleMsgHTML(convTitle, msg), nil
		},

		// ここでファイルを走査してはいけない。IsAlive(5秒)と同じスロットに並ぶ。
		GetConfigHTML: func(ctx context.Context, cfg sdk.Config) (string, error) {
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
			globalBuilder.Kick()
			return cfg, nil
		},
	})
}

// matchWordsText はワード検索条件にメッセージテキストが合致するかチェックする。
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

// renderSingleMsgHTML は1メッセージのみのHTMLを生成する。
// テーマはpostMessage経由で親ページから受け取り動的に切り替える。
func renderSingleMsgHTML(convTitle string, msg cachedMessage) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html><head><meta charset="utf-8">
<style>
:root {
  --bg: #ffffff;
  --text: #333333;
  --msg-human-bg: #dbeafe;
  --msg-assistant-bg: #f3f4f6;
  --sender-color: #6b7280;
  --ts-color: #9ca3af;
  --title-color: #9ca3af;
  --scrollbar-thumb: #2672ed;
  --scrollbar-track: #e5e7eb;
}
[data-theme="dark"] {
  --bg: #212121;
  --text: #e0e0e0;
  --msg-human-bg: #1a3557;
  --msg-assistant-bg: #2d2d2d;
  --sender-color: #aaaaaa;
  --ts-color: #888888;
  --title-color: #888888;
  --scrollbar-thumb: #2672ed;
  --scrollbar-track: #424242;
}
html, body { height: auto; margin: 0; overflow: visible; }
body { font-family: sans-serif; padding: 12px; font-size: 14px;
  background: var(--bg); color: var(--text); }
.conv-title { font-size: 0.85em; color: var(--title-color); margin-bottom: 8px; }
.msg { padding: 8px 12px; border-radius: 8px; white-space: pre-wrap;
  word-break: break-word; line-height: 1.5; }
.human { background: var(--msg-human-bg); }
.assistant { background: var(--msg-assistant-bg); }
.sender { font-size: 0.75em; color: var(--sender-color); margin-bottom: 4px; }
.ts { font-size: 0.7em; color: var(--ts-color); margin-top: 4px; }
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: var(--scrollbar-track); }
::-webkit-scrollbar-thumb { background: var(--scrollbar-thumb); border-radius: 3px; }
</style>
<script>
(function() {
  function notifySize() {
    window.parent.postMessage({
      gkill_iframe_size: {
        width: document.documentElement.scrollWidth,
        height: document.documentElement.scrollHeight
      }
    }, '*');
  }
  window.addEventListener('message', function(e) {
    if (e.data && e.data.gkill_theme) {
      document.documentElement.setAttribute('data-theme', e.data.gkill_theme);
      setTimeout(notifySize, 10);
    }
  });
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', notifySize);
  } else {
    notifySize();
  }
  if (window.ResizeObserver) {
    new ResizeObserver(notifySize).observe(document.documentElement);
  }
})();
</script>
</head><body>`)

	if convTitle != "" {
		sb.WriteString(`<div class="conv-title">`)
		sb.WriteString(htmlEscape(convTitle))
		sb.WriteString(`</div>`)
	}

	text := strings.TrimSpace(msg.Text)
	class := "assistant"
	senderLabel := "Claude"
	if msg.Sender == "human" {
		class = "human"
		senderLabel = "あなた"
	}
	ts := ""
	if msg.RelatedTimeUnix != 0 {
		ts = unixToTimeFromCache(msg.RelatedTimeUnix).Format("2006-01-02 15:04")
	}
	sb.WriteString(`<div class="msg `)
	sb.WriteString(class)
	sb.WriteString(`"><div class="sender">`)
	sb.WriteString(htmlEscape(senderLabel))
	sb.WriteString(`</div>`)
	sb.WriteString(htmlEscape(text))
	sb.WriteString(`<div class="ts">`)
	sb.WriteString(ts)
	sb.WriteString(`</div></div>`)

	sb.WriteString(`</body></html>`)
	return sb.String()
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&#34;")
	return s
}

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
