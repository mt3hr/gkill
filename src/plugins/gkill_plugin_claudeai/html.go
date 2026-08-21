package main

import (
	"encoding/json"
	"fmt"
	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
	"html"
	"path/filepath"
	"strings"
	"time"
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
  --ok-bg: #f0fff4;
  --ok-border: #44aa66;
  --warn-bg: #fff8f0;
  --warn-border: #cc8844;
  --code-bg: #eeeeee;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #212121;
    --text: #e0e0e0;
    --muted: #999999;
    --border: #555555;
    --ok-bg: #1e3a26;
    --ok-border: #44aa66;
    --warn-bg: #3a2e1e;
    --warn-border: #cc8844;
    --code-bg: #383838;
  }
}
body { font-family: sans-serif; margin: 16px; background: var(--bg); color: var(--text); }
h2 { font-size: 1.1em; margin-top: 0; }
h3 { font-size: 0.95em; margin-bottom: 4px; }
.ok { background: var(--ok-bg); border-left: 4px solid var(--ok-border);
  padding: 12px; border-radius: 4px; margin: 12px 0; }
.warn { background: var(--warn-bg); border-left: 4px solid var(--warn-border);
  padding: 12px; border-radius: 4px; margin: 12px 0; }
code { background: var(--code-bg); padding: 2px 6px; border-radius: 3px; font-size: 0.9em; }
ul, ol { margin: 4px 0; padding-left: 20px; font-size: 0.85em; }
pre { background: var(--code-bg); border: 1px solid var(--border); border-radius: 4px;
  padding: 8px; font-size: 0.85em; overflow-x: auto; }
.hint { font-size: 0.8em; color: var(--muted); margin-top: 4px; }
.tag { font-size: 0.75em; background: var(--code-bg); color: var(--muted);
  border-radius: 8px; padding: 0 6px; }
table { border-collapse: collapse; font-size: 0.85em; margin: 8px 0; }
td { padding: 2px 12px 2px 0; }
td.k { color: var(--muted); }
textarea { width: 100%; box-sizing: border-box; font-family: monospace; font-size: 0.85em;
  background: var(--field-bg); color: var(--text); border: 1px solid var(--border);
  border-radius: 4px; padding: 6px; }
button { background: var(--btn-bg); color: var(--btn-color); border: none; border-radius: 4px;
  padding: 6px 16px; font-size: 0.85em; cursor: pointer; margin-top: 6px; }
button:disabled { opacity: 0.6; cursor: default; }
#gkill_save_result { font-size: 0.85em; margin-left: 8px; color: var(--muted); }
</style>
</head><body>`

// configSaveScript は設定ダイアログ(親)へ保存を依頼するスクリプト。
const configSaveScript = `<script>
(function () {
  var ta = document.getElementById('gkill_source_dirs');
  var btn = document.getElementById('gkill_save');
  var out = document.getElementById('gkill_save_result');
  if (!ta || !btn || !out) { return; }
  btn.addEventListener('click', function () {
    btn.disabled = true;
    out.textContent = '保存中…';
    parent.postMessage({ gkill_plugin_config: { source_dirs: ta.value } }, '*');
  });
  window.addEventListener('message', function (e) {
    var r = e.data && e.data.gkill_plugin_config_result;
    if (!r) { return; }
    btn.disabled = false;
    out.textContent = r.ok ? '保存しました' : ('保存に失敗しました: ' + (r.error || ''));
  });
})();
</script>`

// maxShownExpanded は展開結果として設定画面に並べる最大件数。
const maxShownExpanded = 20

// renderConfigHTML は設定画面のHTMLを返す。
//
// gkillの設定ダイアログは設定HTMLを表示するだけで、保存(post_plugin_config)を呼ぶ導線が
// まだ無い。プラグイン側から本体を変更するわけにもいかないので、ここでは編集フォームを出さず、
// 現状の表示とconfig.jsonの編集手順の案内にとどめる。
func renderConfigHTML(pluginDir string, stats cacheStats, patterns []string, src expandedSource) string {
	var sb strings.Builder
	sb.WriteString(configHTMLHead)
	sb.WriteString(`<h2>Claude.ai チャット履歴プラグイン</h2>`)

	// 走査・読み込みはバックグラウンドのビルダが行う。ここでは絶対にファイルを読まない
	// (GetConfigHTML は IsAlive 5秒のスロットに並ぶため)。統計はキャッシュから即答する。
	if stats.MessageCount > 0 {
		sb.WriteString(`<div class="ok">`)
		fmt.Fprintf(&sb, `<p>✓ <strong>%d 件</strong>のメッセージ(会話 %d 件)を読み込んでいます</p>`,
			stats.MessageCount, stats.ConvCount)
		sb.WriteString(`<p>データを更新するには、Claude.ai から再エクスポートして ` +
			`<code>conversations.json</code> を置き換えてください。</p>`)
		sb.WriteString(`</div>`)
	} else {
		sb.WriteString(`<div class="warn">`)
		sb.WriteString(`<p><strong>まだ読み込まれていません。</strong></p>`)
		sb.WriteString(`<p>取り込みはバックグラウンドで進むので、指定した直後は0件のことがあります。</p>`)
		sb.WriteString(`<p>エクスポート手順:</p><ol>`)
		sb.WriteString(`<li>Claude.ai にログイン → 左下のアカウントアイコン → <strong>Settings</strong></li>`)
		sb.WriteString(`<li>「Privacy」→「<strong>Export data</strong>」をクリック</li>`)
		sb.WriteString(`<li>ZIPが届いたら解凍し、<code>conversations.json</code> を取り出す</li>`)
		sb.WriteString(`<li>下の「データソース」で指定したフォルダに置く</li>`)
		sb.WriteString(`</ol></div>`)
	}

	renderBuildProgress(&sb, stats)

	sb.WriteString(`<table>`)
	fmt.Fprintf(&sb, `<tr><td class="k">読み込んだファイル数</td><td>%d</td></tr>`, stats.FileCount)
	fmt.Fprintf(&sb, `<tr><td class="k">会話数</td><td>%d</td></tr>`, stats.ConvCount)
	fmt.Fprintf(&sb, `<tr><td class="k">メッセージ数</td><td>%d</td></tr>`, stats.MessageCount)
	fmt.Fprintf(&sb, `<tr><td class="k">最終スキャン</td><td>%s</td></tr>`,
		html.EscapeString(formatUnix(stats.LastScanUnix)))
	fmt.Fprintf(&sb, `<tr><td class="k">キャッシュDB</td><td><code>%s</code></td></tr>`,
		html.EscapeString(sdk.CacheDBPath(pluginDir)))
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

	sb.WriteString(`<h3>データソースを変える</h3>`)
	sb.WriteString(`<p>1行に1つ書いてください。保存すると次の検索から反映されます(gkillの再起動は不要)。</p>`)
	sb.WriteString(`<textarea id="gkill_source_dirs" rows="4" spellcheck="false">` +
		html.EscapeString(strings.Join(patterns, "\n")) + `</textarea>`)
	sb.WriteString(`<div><button type="button" id="gkill_save">保存</button>` +
		`<span id="gkill_save_result"></span></div>`)
	sb.WriteString(configSaveScript)

	sb.WriteString(`<p class="hint">このプラグインのフォルダ(<code>manifest.json</code> と同じ場所)の ` +
		`<code>config.json</code> を直接編集してもかまいません。` +
		`プラグインの起動時に無ければ自動で作られます。</p>`)
	sb.WriteString(`<p>編集するファイル: <code>` +
		html.EscapeString(filepath.Join(pluginDir, "config.json")) + `</code></p>`)
	sb.WriteString(`<pre>` + html.EscapeString(sampleConfigJSON()) + `</pre>`)
	sb.WriteString(`<div class="hint">` +
		`<code>source_dirs</code> は<strong>配列で複数指定</strong>できます(1つなら文字列でも可)。` +
		`ワイルドカード <code>*</code> <code>**</code> <code>?</code> <code>[]</code> が使えます — ` +
		`マッチしたフォルダは再帰的に走査して <code>conversations.json</code> を探し、` +
		`マッチしたファイルはそのまま読みます。` +
		`先頭の <code>~</code> と環境変数(<code>$HOME</code> など)も展開されます` +
		`(ただしgkillをWindowsサービスで動かしている場合は実行アカウントのホームになるため、絶対パスが確実です)。` +
		`キーごと省略するか空にすると、このプラグインのフォルダを見ます。` +
		`変更は次の検索から反映されます(gkillの再起動は不要)。</div>`)

	sb.WriteString(`</body></html>`)
	return sb.String()
}

// sampleConfigJSON は設定画面に出す config.json の記入例。
// 自動生成される内容そのものを見せる。
func sampleConfigJSON() string {
	data, err := json.MarshalIndent(defaultConfig(), "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// renderBuildProgress は取り込みの進捗を出す。
// 「初回は空が返る」のが仕様なので、進んでいることが見えないと壊れて見える。
func renderBuildProgress(sb *strings.Builder, stats cacheStats) {
	switch stats.BuildState {
	case "", "idle":
		return
	case "error":
		return // エラーは下の「スキャン時のエラー」欄で出す
	}
	sb.WriteString(`<div class="ok">`)
	switch stats.BuildState {
	case "scanning":
		sb.WriteString(`<p>データソースを走査しています…</p>`)
	case "ingesting":
		fmt.Fprintf(sb, `<p>取り込み中 %d / %d 会話</p>`, stats.BuildDone, stats.BuildTotal)
	default:
		sb.WriteString(`<p>処理中…</p>`)
	}
	sb.WriteString(`</div>`)
}

// formatUnix はUnix時刻を表示用文字列にする。0は「-」。
func formatUnix(unix int64) string {
	if unix <= 0 {
		return "-"
	}
	return time.Unix(unix, 0).Format("2006-01-02 15:04")
}
