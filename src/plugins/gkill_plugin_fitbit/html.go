package main

import (
	"fmt"
	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
	"html"
	"strconv"
	"strings"
)

// configHTMLHead は設定画面の共通ヘッダ。
// テーマは postMessage でも通知されるが、届かない場合に備えてOSの配色にも追従させる。
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
  --bar: #2672ed;
  --bar-track: #e5e7eb;
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
    --bar: #2672ed;
    --bar-track: #424242;
  }
}
[data-theme="dark"] {
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
  --bar: #2672ed;
  --bar-track: #424242;
}
body { font-family: sans-serif; margin: 16px; background: var(--bg); color: var(--text); }
h2 { font-size: 1.1em; margin-top: 0; }
h3 { font-size: 0.95em; margin-bottom: 4px; }
.ok { background: var(--ok-bg); border-left: 4px solid var(--ok-border);
  padding: 12px; border-radius: 4px; margin: 12px 0; }
.warn { background: var(--warn-bg); border-left: 4px solid var(--warn-border);
  padding: 12px; border-radius: 4px; margin: 12px 0; }
code { background: var(--code-bg); padding: 2px 6px; border-radius: 3px; font-size: 0.9em; }
ul { margin: 4px 0; padding-left: 20px; font-size: 0.85em; }
pre { background: var(--code-bg); border: 1px solid var(--border); border-radius: 4px;
  padding: 8px; font-size: 0.85em; overflow-x: auto; }
.hint { font-size: 0.8em; color: var(--muted); margin-top: 4px; }
table { border-collapse: collapse; font-size: 0.85em; margin: 8px 0; }
td, th { padding: 2px 12px 2px 0; text-align: left; }
td.k, th { color: var(--muted); font-weight: normal; }
textarea, input[type=text] { width: 100%; box-sizing: border-box; font-family: ui-monospace, Consolas, monospace;
  font-size: 0.85em; background: var(--field-bg); color: var(--text);
  border: 1px solid var(--border); border-radius: 4px; padding: 6px; }
button { background: var(--btn-bg); color: var(--btn-color); border: none;
  border-radius: 4px; padding: 6px 14px; font-size: 0.9em; cursor: pointer; margin-right: 8px; }
button:disabled { opacity: 0.6; cursor: default; }
.progress { background: var(--bar-track); border-radius: 4px; height: 10px; overflow: hidden; margin-top: 6px; }
.progress > div { background: var(--bar); height: 100%; }
details { margin: 12px 0; }
details > summary { cursor: pointer; font-size: 0.9em; }
</style>
<script>
window.addEventListener('message', function (e) {
  if (e.data && e.data.gkill_theme) {
    document.documentElement.setAttribute('data-theme', e.data.gkill_theme);
  }
});
</script>
</head><body>`

// maxShownExpanded は展開結果として設定画面に並べる最大件数。
const maxShownExpanded = 20

// renderConfigHTML は設定画面のHTMLを返す。
//
// 設定の保存は gkill 本体の設定ダイアログが postMessage で肩代わりする。
// iframe は allow-same-origin なしで動くため自力では API を叩けない。
//
//	iframe → 親 : { gkill_plugin_config: { source_dirs: "...", ... } }
//	親 → iframe : { gkill_plugin_config_result: { ok: true } }
func renderConfigHTML(pluginDir string, config pluginConfig, stats cacheStats) string {
	var sb strings.Builder
	sb.WriteString(configHTMLHead)
	sb.WriteString(`<h2>Fitbit (Google Takeout) プラグイン</h2>`)

	// 取り込みの進捗。バックグラウンドで作っているので、ここでしか見えない。
	if stats.BuildState != "" && stats.BuildState != "idle" {
		sb.WriteString(`<div class="warn">`)
		fmt.Fprintf(&sb, `<strong>%s</strong>`, html.EscapeString(buildStateLabel(stats.BuildState)))
		if stats.BuildTotalFiles > 0 {
			percent := stats.BuildDoneFiles * 100 / stats.BuildTotalFiles
			fmt.Fprintf(&sb, ` %s / %s ファイル (%d%%)`,
				withThousandSeparator(stats.BuildDoneFiles), withThousandSeparator(stats.BuildTotalFiles), percent)
			fmt.Fprintf(&sb, `<div class="progress"><div style="width:%d%%"></div></div>`, percent)
		}
		sb.WriteString(`<div class="hint">初回は1〜2分かかることがあります。取り込みが終わった日から順に表示されます。</div>`)
		sb.WriteString(`</div>`)
	}
	if stats.BuildError != "" {
		fmt.Fprintf(&sb, `<div class="warn"><strong>エラー</strong><div class="hint">%s</div></div>`, html.EscapeString(stats.BuildError))
	}

	sb.WriteString(`<table>`)
	writeConfigRow(&sb, "対象ファイル数", withThousandSeparator(stats.TargetFileCount))
	writeConfigRow(&sb, "取り込み済みファイル", withThousandSeparator(stats.ScannedFileCount))
	writeConfigRow(&sb, "日数", withThousandSeparator(stats.DayCount))
	writeConfigRow(&sb, "指標", withThousandSeparator(stats.MetricCount))
	writeConfigRow(&sb, "記録数", withThousandSeparator(stats.KyouCount))
	writeConfigRow(&sb, "最終スキャン", formatUnix(stats.LastScanUnix))
	writeConfigRow(&sb, "タイムゾーン", stats.Timezone)
	writeConfigRow(&sb, "キャッシュDB", sdk.CacheDBPath(pluginDir))
	sb.WriteString(`</table>`)

	if len(config.Source.Missing) != 0 {
		sb.WriteString(`<div class="warn"><strong>次の指定は何にもマッチしませんでした</strong><ul>`)
		for _, missing := range config.Source.Missing {
			fmt.Fprintf(&sb, `<li>%s</li>`, html.EscapeString(missing))
		}
		sb.WriteString(`</ul></div>`)
	}

	for _, problem := range stats.Problems {
		fmt.Fprintf(&sb, `<div class="warn"><strong>%s</strong><div class="hint">%s</div></div>`,
			html.EscapeString(problem.Path), html.EscapeString(problem.Message))
	}

	sb.WriteString(`<h3>取り込み元の書き出し</h3>`)
	if len(stats.Exports) == 0 {
		sb.WriteString(`<div class="hint">ZIPがまだ見つかっていません。取り込み元のフォルダに Google Takeout の ZIP を展開せずそのまま置いてください。</div>`)
	} else {
		sb.WriteString(`<table><tr><th>採用順</th><th>フォルダ</th><th>ZIP数</th><th>書き出し日時</th><th>日数</th></tr>`)
		for _, export := range stats.Exports {
			label := "新しい"
			if export.Rank != 0 {
				// 日が重なったぶんだけ上位に譲る。重ならない日はこの書き出しの値が使われる
				label = fmt.Sprintf("%d 番目", export.Rank+1)
			}
			fmt.Fprintf(&sb, `<tr><td>%s</td><td>%s</td><td>%d</td><td>%s</td><td>%s</td></tr>`,
				label, html.EscapeString(export.Dir), export.ArchiveCount,
				html.EscapeString(formatUnix(export.NewestUnix)), withThousandSeparator(export.DayCount))
		}
		sb.WriteString(`</table>`)
		sb.WriteString(`<div class="hint">同じフォルダに置いた分割ZIP(-001 -002 …)は1つの書き出しとして合算します。` +
			`書き出しが複数あるときは、日が重なったぶんだけ新しいほうの値を使います(合算しません)。</div>`)
	}

	sb.WriteString(`<h3>設定</h3>`)
	sb.WriteString(`<div class="hint">取り込み元（1行に1つ。ZIPを置いたフォルダ、またはZIPそのもの。ワイルドカードも可）</div>`)
	fmt.Fprintf(&sb, `<textarea id="gkill_source_dirs" rows="4">%s</textarea>`,
		html.EscapeString(strings.Join(config.Patterns, "\n")))
	sb.WriteString(`<div class="hint">タイムゾーン（「この日はどの日か」の判定に使います）</div>`)
	fmt.Fprintf(&sb, `<input type="text" id="gkill_timezone" value="%s">`, html.EscapeString(config.Timezone))
	sb.WriteString(`<div class="hint">取り込む指標（1行に1つ。空欄なら全部。キーは下の一覧を参照）</div>`)
	fmt.Fprintf(&sb, `<textarea id="gkill_metrics" rows="3">%s</textarea>`,
		html.EscapeString(strings.Join(config.Metrics, "\n")))
	sb.WriteString(`<div class="hint">同時に読むファイル数（0 なら自動）</div>`)
	fmt.Fprintf(&sb, `<input type="text" id="gkill_scan_workers" value="%s">`, strconv.Itoa(config.ScanWorkers))
	sb.WriteString(`<p><button id="gkill_save">保存</button><span id="gkill_save_result" class="hint"></span></p>`)

	fmt.Fprintf(&sb, `<details><summary>指標一覧 (%d件)</summary><table><tr><th>キー</th><th>名前</th><th>単位</th><th>集計</th></tr>`, len(metricRegistry))
	for _, def := range metricRegistry {
		fmt.Fprintf(&sb, `<tr><td><code>%s</code></td><td>%s</td><td>%s</td><td>%s</td></tr>`,
			html.EscapeString(def.Key), html.EscapeString(def.Title),
			html.EscapeString(def.Unit), html.EscapeString(aggLabel(def.Agg)))
	}
	sb.WriteString(`</table></details>`)

	sb.WriteString(`<details><summary>config.json を直接編集する</summary>`)
	sb.WriteString(`<div class="hint">この画面で保存する代わりに、プラグインフォルダの <code>config.json</code> を直接書き換えても構いません。次の検索から反映されます。</div>`)
	fmt.Fprintf(&sb, `<pre>%s</pre></details>`, html.EscapeString(sampleConfigJSON()))

	sb.WriteString(configSaveScript)
	sb.WriteString(`</body></html>`)
	return sb.String()
}

func writeConfigRow(sb *strings.Builder, label string, value string) {
	if value == "" {
		value = "—"
	}
	fmt.Fprintf(sb, `<tr><td class="k">%s</td><td>%s</td></tr>`, html.EscapeString(label), html.EscapeString(value))
}

// buildStateLabel は取り込み状態の表示名。
func buildStateLabel(state string) string {
	switch state {
	case "scanning":
		return "フォルダを走査中"
	case "ingesting":
		return "取り込み中"
	case "folding":
		return "集計中"
	case "error":
		return "エラー"
	}
	return state
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

// configSaveScript は設定ダイアログ(親)へ保存を依頼するスクリプト。
const configSaveScript = `<script>
(function () {
  var sourceDirs = document.getElementById('gkill_source_dirs');
  var timezone = document.getElementById('gkill_timezone');
  var metrics = document.getElementById('gkill_metrics');
  var scanWorkers = document.getElementById('gkill_scan_workers');
  var btn = document.getElementById('gkill_save');
  var out = document.getElementById('gkill_save_result');
  if (!sourceDirs || !btn || !out) { return; }
  btn.addEventListener('click', function () {
    btn.disabled = true;
    out.textContent = '保存中…';
    parent.postMessage({ gkill_plugin_config: {
      source_dirs: sourceDirs.value,
      timezone: timezone ? timezone.value : '',
      metrics: metrics ? metrics.value : '',
      scan_workers: scanWorkers ? scanWorkers.value : ''
    } }, '*');
  });
  window.addEventListener('message', function (e) {
    var r = e.data && e.data.gkill_plugin_config_result;
    if (!r) { return; }
    btn.disabled = false;
    out.textContent = r.ok ? '保存しました' : ('保存に失敗しました: ' + (r.error || ''));
  });
})();
</script>`
