package main

import (
	"encoding/json"
	"fmt"
	"html"
	"path/filepath"
	"strings"
)

// configHTMLHead は設定画面の共通ヘッダ。
// 設定ダイアログはテーマをpostMessageしてこないので、OSの配色設定に追従させる。
const configHTMLHead = `<!DOCTYPE html>
<html><head><meta charset="utf-8">
<style>
:root {
  --bg: #ffffff;
  --text: #333333;
  --muted: #888888;
  --border: #cccccc;
  --field-bg: #ffffff;
  --ok-bg: #f0fff4;
  --ok-border: #44aa66;
  --warn-bg: #fff8f0;
  --warn-border: #cc8844;
  --code-bg: #eeeeee;
  --btn-bg: #2672ed;
  --btn-color: #ffffff;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #212121;
    --text: #e0e0e0;
    --muted: #999999;
    --border: #555555;
    --field-bg: #2d2d2d;
    --ok-bg: #1e3a26;
    --ok-border: #44aa66;
    --warn-bg: #3a2e1e;
    --warn-border: #cc8844;
    --code-bg: #383838;
    --btn-bg: #2672ed;
    --btn-color: #ffffff;
  }
}
body { font-family: sans-serif; margin: 16px; background: var(--bg); color: var(--text); }
h2 { font-size: 1.1em; margin-top: 0; }
.ok { background: var(--ok-bg); border-left: 4px solid var(--ok-border);
  padding: 12px; border-radius: 4px; margin: 12px 0; }
.warn { background: var(--warn-bg); border-left: 4px solid var(--warn-border);
  padding: 12px; border-radius: 4px; margin: 12px 0; }
code { background: var(--code-bg); padding: 2px 6px; border-radius: 3px; font-size: 0.9em; }
h3 { font-size: 0.95em; margin-bottom: 4px; }
ul { margin: 4px 0; padding-left: 20px; font-size: 0.85em; }
pre { background: var(--code-bg); border: 1px solid var(--border); border-radius: 4px;
  padding: 8px; font-size: 0.85em; overflow-x: auto; }
.hint { font-size: 0.8em; color: var(--muted); margin-top: 4px; }
.tag { font-size: 0.75em; background: var(--code-bg); color: var(--muted);
  border-radius: 8px; padding: 0 6px; }
table { border-collapse: collapse; font-size: 0.85em; margin: 8px 0; }
td { padding: 2px 12px 2px 0; }
td.k { color: var(--muted); }
</style>
</head><body>`

// renderConfigHTML は設定画面のHTMLを返す。
//
// gkillの設定ダイアログは設定HTMLを表示するだけで、保存(post_plugin_config)を呼ぶ導線が
// まだ無い。プラグイン側から本体を変更するわけにもいかないので、ここでは編集フォームを出さず、
// 現状の表示とconfig.jsonの編集手順の案内にとどめる。
// (post_configコマンド自体はハンドラで実装済みなので、本体が対応すればそのまま動く)
func renderConfigHTML(pluginDir string, stats cacheStats, patterns []string, src expandedSource) string {
	var sb strings.Builder
	sb.WriteString(configHTMLHead)
	sb.WriteString(`<h2>Claude Code チャットログプラグイン</h2>`)

	if stats.MessageCount > 0 {
		sb.WriteString(`<div class="ok">`)
		fmt.Fprintf(&sb, `<p>✓ <strong>%d 件の発言</strong>を読み込んでいます</p>`, stats.MessageCount)
		sb.WriteString(`</div>`)
	} else {
		sb.WriteString(`<div class="warn">`)
		sb.WriteString(`<p>まだ発言が読み込まれていません。</p>`)
		sb.WriteString(`<p>Claude Code のセッションログが入ったフォルダを、下の手順で指定してください。</p>`)
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`<table>`)
	fmt.Fprintf(&sb, `<tr><td class="k">対象ファイル数</td><td>%d</td></tr>`, stats.FileCount)
	fmt.Fprintf(&sb, `<tr><td class="k">発言数</td><td>%d</td></tr>`, stats.MessageCount)
	fmt.Fprintf(&sb, `<tr><td class="k">最終スキャン</td><td>%s</td></tr>`,
		html.EscapeString(formatUnix(stats.LastScanUnix)))
	sb.WriteString(`</table>`)

	if len(src.Missing) > 0 {
		sb.WriteString(`<div class="warn"><p>次の指定は何にもマッチしませんでした:</p><ul>`)
		for _, d := range src.Missing {
			sb.WriteString(`<li><code>` + html.EscapeString(d) + `</code></li>`)
		}
		sb.WriteString(`</ul></div>`)
	}
	if stats.LastScanError != "" {
		sb.WriteString(`<div class="warn"><p>スキャン時のエラー:</p><p><code>` +
			html.EscapeString(stats.LastScanError) + `</code></p></div>`)
	}

	sb.WriteString(`<h3>設定されている指定</h3><ul>`)
	for _, p := range patterns {
		sb.WriteString(`<li><code>` + html.EscapeString(p) + `</code>`)
		if hasGlobMeta(p) {
			sb.WriteString(` <span class="tag">パターン</span>`)
		}
		sb.WriteString(`</li>`)
	}
	sb.WriteString(`</ul>`)

	fmt.Fprintf(&sb, `<h3>展開結果 (フォルダ %d / ファイル %d)</h3><ul>`, len(src.Dirs), len(src.Files))
	shown := 0
	for _, d := range src.Dirs {
		sb.WriteString(`<li><code>` + html.EscapeString(d) + `</code> (再帰的に走査)</li>`)
		shown++
	}
	for _, f := range src.Files {
		if shown >= maxShownExpanded {
			break
		}
		sb.WriteString(`<li><code>` + html.EscapeString(f) + `</code></li>`)
		shown++
	}
	if len(src.Dirs)+len(src.Files) > shown {
		fmt.Fprintf(&sb, `<li>ほか %d 件</li>`, len(src.Dirs)+len(src.Files)-shown)
	}
	sb.WriteString(`</ul>`)

	sb.WriteString(`<h3>指定を変える</h3>`)
	sb.WriteString(`<p>このプラグインのフォルダに <code>config.json</code> を置いてください。</p>`)
	sb.WriteString(`<p>配置先: <code>` + html.EscapeString(filepath.Join(pluginDir, "config.json")) + `</code></p>`)
	sb.WriteString(`<pre>` + html.EscapeString(sampleConfigJSON()) + `</pre>`)
	sb.WriteString(`<div class="hint">` +
		`<code>source_dirs</code> は<strong>配列で複数指定</strong>できます(1つなら文字列でも可)。` +
		`ワイルドカード <code>*</code> <code>**</code> <code>?</code> <code>[]</code> が使えます — ` +
		`マッチしたフォルダは再帰的に走査し、マッチしたファイルはそのまま対象にします。` +
		`先頭の <code>~</code> と環境変数(<code>$HOME</code> など)も展開されます` +
		`(ただしgkillをWindowsサービスで動かしている場合は実行アカウントのホームになるため、絶対パスが確実です)。` +
		`キーごと省略するか空にすると <code>` + html.EscapeString(defaultSourceDir()) + `</code> を使います。` +
		`変更は次の検索から反映されます(gkillの再起動は不要)。</div>`)

	sb.WriteString(`</body></html>`)
	return sb.String()
}

// maxShownExpanded は展開結果として設定画面に並べる最大件数。
const maxShownExpanded = 20

// sampleConfigJSON は設定画面に出す config.json の記入例。
func sampleConfigJSON() string {
	sample := map[string]any{
		configKeySourceDirs: []string{
			defaultSourceDir(),
			"~/DevPC/ClaudeCode_*",
		},
	}
	data, err := json.MarshalIndent(sample, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}
