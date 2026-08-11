package main

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"
)

// metricHTMLHead はテーマ追従とiframe自動リサイズのための共通ヘッダ。
// 他のプラグインで実績のある仕組みをそのまま使っている。
const metricHTMLHead = `<!DOCTYPE html>
<html><head><meta charset="utf-8">
<style>
:root {
  --bg: #ffffff;
  --text: #333333;
  --card-bg: #f3f4f6;
  --label-color: #6b7280;
  --sub-color: #9ca3af;
  --chip-bg: #e5e7eb;
  --chip-color: #4b5563;
  --bar: #2672ed;
  --bar-track: #e5e7eb;
  --details-bg: #e9eaec;
  --details-color: #4b5563;
  --scrollbar-thumb: #2672ed;
  --scrollbar-track: #e5e7eb;
}
[data-theme="dark"] {
  --bg: #212121;
  --text: #e0e0e0;
  --card-bg: #2d2d2d;
  --label-color: #aaaaaa;
  --sub-color: #888888;
  --chip-bg: #3a3a3a;
  --chip-color: #cccccc;
  --bar: #2672ed;
  --bar-track: #424242;
  --details-bg: #383838;
  --details-color: #cccccc;
  --scrollbar-thumb: #2672ed;
  --scrollbar-track: #424242;
}
html, body { height: auto; margin: 0; overflow: visible; }
body { font-family: sans-serif; padding: 12px; font-size: 14px;
  background: var(--bg); color: var(--text); }
.date { font-size: 0.85em; color: var(--sub-color); margin-bottom: 2px; }
.card { background: var(--card-bg); border-radius: 8px; padding: 10px 14px; }
.title { font-size: 0.9em; color: var(--label-color); }
.value { font-size: 1.9em; font-weight: bold; line-height: 1.2; margin: 2px 0; }
.unit { font-size: 0.5em; font-weight: normal; color: var(--label-color); margin-left: 4px; }
.stats { font-size: 0.75em; color: var(--sub-color); margin-top: 2px; }
.chips { margin-top: 6px; }
.chip { display: inline-block; background: var(--chip-bg); color: var(--chip-color);
  border-radius: 10px; padding: 1px 8px; font-size: 0.7em; margin-right: 4px; }
.hours { margin-top: 10px; }
.hours-label { font-size: 0.72em; color: var(--sub-color); margin-bottom: 3px; }
.bars { display: flex; align-items: flex-end; gap: 2px; height: 48px; }
.bar-cell { flex: 1; background: var(--bar-track); border-radius: 2px 2px 0 0;
  display: flex; align-items: flex-end; height: 100%; }
.bar { width: 100%; background: var(--bar); border-radius: 2px 2px 0 0; min-height: 1px; }
.hour-axis { display: flex; justify-content: space-between;
  font-size: 0.65em; color: var(--sub-color); margin-top: 2px; }
details { background: var(--details-bg); border-radius: 6px; margin: 8px 0 0 0; padding: 4px 8px; }
details > summary { cursor: pointer; font-size: 0.75em; color: var(--details-color);
  list-style: none; user-select: none; }
details > summary::-webkit-details-marker { display: none; }
details > summary::before { content: "\25B8 "; }
details[open] > summary::before { content: "\25BE "; }
.paths { font-size: 0.7em; word-break: break-all; margin: 4px 0 0 0; padding: 0; list-style: none; }
.paths li { margin: 2px 0; font-family: ui-monospace, Consolas, monospace; }
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
  document.addEventListener('toggle', function() { setTimeout(notifySize, 10); }, true);
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
</head><body>`

// weekdayNames は日付の横に出す曜日。
var weekdayNames = [...]string{"日", "月", "火", "水", "木", "金", "土"}

// renderMetricHTML は1日1指標のカードを組み立てる。
func renderMetricHTML(metric dailyMetric) string {
	def := metricByKey[metric.MetricKey]

	var sb strings.Builder
	sb.WriteString(metricHTMLHead)

	weekday := ""
	if parsed, err := time.Parse("2006-01-02", metric.DateLocal); err == nil {
		weekday = "(" + weekdayNames[int(parsed.Weekday())] + ")"
	}
	fmt.Fprintf(&sb, `<div class="date">%s %s</div>`, html.EscapeString(metric.DateLocal), weekday)

	sb.WriteString(`<div class="card">`)
	fmt.Fprintf(&sb, `<div class="title">%s</div>`, html.EscapeString(metric.Title))
	fmt.Fprintf(&sb, `<div class="value">%s`, html.EscapeString(metric.NumValue))
	if metric.Unit != "" {
		fmt.Fprintf(&sb, `<span class="unit">%s</span>`, html.EscapeString(metric.Unit))
	}
	sb.WriteString(`</div>`)

	// 合計・件数の指標に最小/最大を出しても意味が無いので、平均系だけに出す
	stats := []string{}
	if def.Agg == aggMean || def.Agg == aggMin || def.Agg == aggMax {
		stats = append(stats, fmt.Sprintf("最小 %s / 最大 %s",
			trimFloat(metric.MinValue), trimFloat(metric.MaxValue)))
	}
	if metric.SampleCount > 1 {
		stats = append(stats, fmt.Sprintf("サンプル %s件", withThousandSeparator(metric.SampleCount)))
	}
	if len(stats) != 0 {
		fmt.Fprintf(&sb, `<div class="stats">%s</div>`, html.EscapeString(strings.Join(stats, " · ")))
	}

	devices := splitNonEmptyLines(metric.Devices)
	if len(devices) != 0 {
		sb.WriteString(`<div class="chips">`)
		for _, device := range devices {
			fmt.Fprintf(&sb, `<span class="chip">%s</span>`, html.EscapeString(device))
		}
		sb.WriteString(`</div>`)
	}

	// 時刻別のミニ棒グラフ。サンプルが1件以下のときは出しても意味が無い
	if metric.SampleCount > 1 {
		sb.WriteString(renderHourBars(metric, def))
	}
	sb.WriteString(`</div>`)

	paths := splitNonEmptyLines(metric.SourcePaths)
	if len(paths) != 0 {
		fmt.Fprintf(&sb, `<details><summary>取り込み元 (%d件)</summary><ul class="paths">`, len(paths))
		for _, path := range paths {
			fmt.Fprintf(&sb, `<li>%s</li>`, html.EscapeString(path))
		}
		sb.WriteString(`</ul></details>`)
	}

	sb.WriteString(`</body></html>`)
	return sb.String()
}

// renderHourBars は時刻別の棒グラフを組み立てる。
// 合計・件数の指標はそのまま、平均系は件数で割った値を使う。
func renderHourBars(metric dailyMetric, def metricDef) string {
	sums := parseFloatVector(metric.HourSums)
	counts := parseIntVector(metric.HourCounts)
	if len(sums) != 24 || len(counts) != 24 {
		return ""
	}

	values := make([]float64, 24)
	maxValue := 0.0
	for i := range 24 {
		value := sums[i]
		if def.Agg == aggMean || def.Agg == aggMin || def.Agg == aggMax {
			if counts[i] > 0 {
				value = sums[i] / float64(counts[i])
			} else {
				value = 0
			}
		}
		if def.Agg == aggCount {
			value = float64(counts[i])
		}
		values[i] = value
		if value > maxValue {
			maxValue = value
		}
	}
	if maxValue <= 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(`<div class="hours"><div class="hours-label">時刻別</div><div class="bars">`)
	for i := range 24 {
		percent := values[i] / maxValue * 100
		if percent < 0 {
			percent = 0
		}
		fmt.Fprintf(&sb, `<div class="bar-cell" title="%d時"><div class="bar" style="height:%.1f%%"></div></div>`, i, percent)
	}
	sb.WriteString(`</div><div class="hour-axis"><span>0</span><span>6</span><span>12</span><span>18</span><span>23</span></div></div>`)
	return sb.String()
}

// renderNotFoundHTML は見つからなかったときのHTML。
func renderNotFoundHTML() string {
	return metricHTMLHead + `<div class="stats">この記録は取り込み済みのデータに見つかりませんでした。設定画面で取り込み状況を確認してください。</div></body></html>`
}

// trimFloat は末尾の余分な0を落とした表示用の数値にする。
func trimFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
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

// splitNonEmptyLines は改行区切りを空要素抜きで分解する。
func splitNonEmptyLines(value string) []string {
	values := []string{}
	for line := range strings.SplitSeq(value, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			values = append(values, line)
		}
	}
	return values
}
