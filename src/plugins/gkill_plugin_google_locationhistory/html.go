package main

import (
	"fmt"
	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
	"html"
	"sort"
	"strconv"
	"strings"
	"time"
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
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #212121; --text: #e0e0e0; --muted: #999999; --border: #555555;
    --field-bg: #2d2d2d; --ok-bg: #1e3a26; --ok-border: #44aa66;
    --warn-bg: #3a2e1e; --warn-border: #cc8844; --code-bg: #383838;
    --btn-bg: #2672ed; --btn-color: #ffffff;
  }
}
[data-theme="dark"] {
  --bg: #212121; --text: #e0e0e0; --muted: #999999; --border: #555555;
  --field-bg: #2d2d2d; --ok-bg: #1e3a26; --ok-border: #44aa66;
  --warn-bg: #3a2e1e; --warn-border: #cc8844; --code-bg: #383838;
  --btn-bg: #2672ed; --btn-color: #ffffff;
}
body { font-family: sans-serif; margin: 16px; background: var(--bg); color: var(--text); }
h2 { font-size: 1.1em; margin-top: 0; }
h3 { font-size: 0.95em; margin-bottom: 4px; }
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
textarea, input[type=text] { width: 100%; box-sizing: border-box;
  font-family: ui-monospace, Consolas, monospace; font-size: 0.85em;
  background: var(--field-bg); color: var(--text);
  border: 1px solid var(--border); border-radius: 4px; padding: 6px; }
label { font-size: 0.85em; display: block; margin-top: 6px; }
button { background: var(--btn-bg); color: var(--btn-color); border: none;
  border-radius: 4px; padding: 6px 14px; font-size: 0.9em; cursor: pointer; margin-right: 8px; }
button:disabled { opacity: 0.6; cursor: default; }
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
	sb.WriteString(`<h2>Google ロケーション履歴 プラグイン</h2>`)
	sb.WriteString(`<div class="hint">読み込んだ位置情報は gkill の位置情報ログとして扱われ、地図と地図の絞り込みで使えます。記録（Kyou）は作りません。</div>`)

	sb.WriteString(`<table>`)
	writeConfigRow(&sb, "取り込んだ点", withThousandSeparator(stats.TotalPoints))
	writeConfigRow(&sb, "フィルタ後", withThousandSeparator(stats.FilteredPoints))
	writeConfigRow(&sb, "重複除去後", withThousandSeparator(stats.UniquePoints))
	if stats.OldestUnixMilli != 0 {
		writeConfigRow(&sb, "期間", formatMilli(stats.OldestUnixMilli)+" 〜 "+formatMilli(stats.NewestUnixMilli))
	}
	writeConfigRow(&sb, "最終スキャン", formatUnix(stats.LastScanUnix))
	writeConfigRow(&sb, "キャッシュDB", sdk.CacheDBPath(pluginDir))
	sb.WriteString(`</table>`)

	if len(stats.FileCountByFormat) != 0 {
		sb.WriteString(`<h3>形式別のファイル数</h3><table><tr><th>形式</th><th>ファイル</th></tr>`)
		formatIDs := make([]string, 0, len(stats.FileCountByFormat))
		for formatID := range stats.FileCountByFormat {
			formatIDs = append(formatIDs, formatID)
		}
		sort.Strings(formatIDs)
		for _, formatID := range formatIDs {
			label := formatID
			if format, exist := formatByID[formatID]; exist {
				label = format.Label
				if format.Parse == nil {
					label += "（未対応）"
				}
			}
			fmt.Fprintf(&sb, `<tr><td>%s</td><td>%d</td></tr>`,
				html.EscapeString(label), stats.FileCountByFormat[formatID])
		}
		sb.WriteString(`</table>`)
	}

	if len(stats.PointsBySource) != 0 {
		sb.WriteString(`<h3>測位の出所</h3><table><tr><th>出所</th><th>点</th></tr>`)
		sources := make([]string, 0, len(stats.PointsBySource))
		for source := range stats.PointsBySource {
			sources = append(sources, source)
		}
		sort.Strings(sources)
		for _, source := range sources {
			fmt.Fprintf(&sb, `<tr><td>%s</td><td>%s</td></tr>`,
				html.EscapeString(source), withThousandSeparator(stats.PointsBySource[source]))
		}
		sb.WriteString(`</table>`)
	}

	if len(stats.UnsupportedFiles) != 0 {
		sb.WriteString(`<div class="warn"><strong>この形式は検出しましたが、まだ読めません</strong><ul>`)
		for i, path := range stats.UnsupportedFiles {
			if i >= maxShownExpanded {
				fmt.Fprintf(&sb, `<li>… ほか %d 件</li>`, len(stats.UnsupportedFiles)-i)
				break
			}
			fmt.Fprintf(&sb, `<li>%s</li>`, html.EscapeString(path))
		}
		sb.WriteString(`</ul></div>`)
	}
	if len(stats.ScanErrors) != 0 {
		sb.WriteString(`<div class="warn"><strong>読み込みに失敗したファイル</strong><ul>`)
		for i, message := range stats.ScanErrors {
			if i >= maxShownExpanded {
				break
			}
			fmt.Fprintf(&sb, `<li>%s</li>`, html.EscapeString(message))
		}
		sb.WriteString(`</ul></div>`)
	}
	if len(config.Source.Missing) != 0 {
		sb.WriteString(`<div class="warn"><strong>次の指定は何にもマッチしませんでした</strong><ul>`)
		for _, missing := range config.Source.Missing {
			fmt.Fprintf(&sb, `<li>%s</li>`, html.EscapeString(missing))
		}
		sb.WriteString(`</ul></div>`)
	}

	for _, problem := range stats.Problems {
		fmt.Fprintf(&sb, `<div class="warn"><strong>%s</strong><div>%s</div></div>`,
			html.EscapeString(problem.Path), html.EscapeString(problem.Message))
	}

	sb.WriteString(`<h3>取り込み元の書き出し</h3>`)
	if len(stats.Exports) == 0 {
		sb.WriteString(`<p>ZIPがまだ見つかっていません。取り込み元のフォルダに Google Takeout の ZIP を展開せずそのまま置いてください。</p>`)
	} else {
		sb.WriteString(`<table><tr><th>フォルダ</th><th>ZIP数</th><th>書き出し日時</th><th>対象エントリ</th></tr>`)
		shown := 0
		for _, export := range stats.Exports {
			if shown >= maxShownExpanded {
				break
			}
			fmt.Fprintf(&sb, `<tr><td>%s</td><td>%d</td><td>%s</td><td>%d</td></tr>`,
				html.EscapeString(export.Dir), export.ArchiveCount,
				html.EscapeString(formatUnix(export.NewestUnix)), export.EntryCount)
			shown++
		}
		sb.WriteString(`</table>`)
		if len(stats.Exports) > shown {
			fmt.Fprintf(&sb, `<p>… ほか %d 件</p>`, len(stats.Exports)-shown)
		}
		sb.WriteString(`<p>書き出しが複数あっても、同じ時刻・同じ座標の点は1つにまとめるので二重にはなりません。` +
			`古い書き出しにしか無い期間もそのまま残ります。</p>`)
	}

	sb.WriteString(`<h3>設定</h3>`)
	sb.WriteString(`<label>取り込み元（1行に1つ。ZIPを置いたフォルダ、またはZIPそのもの。ワイルドカードも可）</label>`)
	fmt.Fprintf(&sb, `<textarea id="gkill_source_dirs" rows="4">%s</textarea>`,
		html.EscapeString(strings.Join(config.Patterns, "\n")))
	sb.WriteString(`<label>精度の上限（メートル。これより粗い測位は捨てる。0以下で無効）</label>`)
	fmt.Fprintf(&sb, `<input type="text" id="gkill_accuracy_max_meters" value="%d">`, config.AccuracyMaxMeters)
	sb.WriteString(`<label>測位の出所（カンマ区切り。空なら絞らない。GPS / WIFI / WIFI_ONLY / CELL / UNKNOWN / FITBIT）</label>`)
	fmt.Fprintf(&sb, `<input type="text" id="gkill_sources" value="%s">`,
		html.EscapeString(strings.Join(config.Sources, ", ")))
	sb.WriteString(`<label>返す点数の上限</label>`)
	fmt.Fprintf(&sb, `<input type="text" id="gkill_max_points" value="%d">`, config.MaxPoints)
	fmt.Fprintf(&sb, `<label><input type="checkbox" id="gkill_include_fitbit_gps"%s> ワークアウトのトラックを含める</label>`,
		checkedAttr(config.IncludeFitbitGPS))
	fmt.Fprintf(&sb, `<label><input type="checkbox" id="gkill_visit_points"%s> 滞在地・移動区間の端点も点として出す（生の測位より粗い）</label>`,
		checkedAttr(config.VisitPoints))
	sb.WriteString(`<p><button id="gkill_save">保存</button><span id="gkill_save_result" class="hint"></span></p>`)

	sb.WriteString(`<details><summary>config.json を直接編集する</summary>`)
	sb.WriteString(`<div class="hint">この画面で保存する代わりに、プラグインフォルダの <code>config.json</code> を直接書き換えても構いません。次の取得から反映されます。</div>`)
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

func checkedAttr(checked bool) string {
	if checked {
		return " checked"
	}
	return ""
}

// withThousandSeparator は3桁区切りにする。
func withThousandSeparator(value int) string {
	digits := strconv.Itoa(value)
	negative := strings.HasPrefix(digits, "-")
	digits = strings.TrimPrefix(digits, "-")

	var sb strings.Builder
	for i, r := range digits {
		if i != 0 && (len(digits)-i)%3 == 0 {
			sb.WriteByte(',')
		}
		sb.WriteRune(r)
	}
	if negative {
		return "-" + sb.String()
	}
	return sb.String()
}

func formatUnix(unix int64) string {
	if unix == 0 {
		return ""
	}
	return time.Unix(unix, 0).Local().Format("2006-01-02 15:04:05")
}

func formatMilli(unixMilli int64) string {
	if unixMilli == 0 {
		return ""
	}
	return time.UnixMilli(unixMilli).Local().Format("2006-01-02")
}

// configSaveScript は設定ダイアログ(親)へ保存を依頼するスクリプト。
const configSaveScript = `<script>
(function () {
  var sourceDirs = document.getElementById('gkill_source_dirs');
  var accuracy = document.getElementById('gkill_accuracy_max_meters');
  var sources = document.getElementById('gkill_sources');
  var maxPoints = document.getElementById('gkill_max_points');
  var includeFitbit = document.getElementById('gkill_include_fitbit_gps');
  var visitPoints = document.getElementById('gkill_visit_points');
  var btn = document.getElementById('gkill_save');
  var out = document.getElementById('gkill_save_result');
  if (!sourceDirs || !btn || !out) { return; }
  btn.addEventListener('click', function () {
    btn.disabled = true;
    out.textContent = '保存中…';
    parent.postMessage({ gkill_plugin_config: {
      source_dirs: sourceDirs.value,
      accuracy_max_meters: accuracy ? accuracy.value : '',
      sources: sources ? sources.value : '',
      max_points: maxPoints ? maxPoints.value : '',
      include_fitbit_gps: (includeFitbit && includeFitbit.checked) ? '1' : '',
      visit_points: (visitPoints && visitPoints.checked) ? '1' : ''
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
