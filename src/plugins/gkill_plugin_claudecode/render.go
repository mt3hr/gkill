package main

import (
	"fmt"
	"html"
	"slices"
	"strings"
	"time"
)

// maxHTMLBytes は1ターンのHTMLの上限。
// gkill側の読み取りバッファは32MBだが、そこまで大きいものは表示できないので手前で打ち切る。
const maxHTMLBytes = 4 * 1024 * 1024

// turnHTMLHead はテーマ追従とiframe自動リサイズのための共通ヘッダ。
// Claude.aiプラグインで実績のある仕組みをそのまま使っている。
const turnHTMLHead = `<!DOCTYPE html>
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
  --chip-bg: #e5e7eb;
  --chip-color: #4b5563;
  --details-bg: #e9eaec;
  --details-color: #4b5563;
  --code-color: #1f2937;
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
  --chip-bg: #3a3a3a;
  --chip-color: #cccccc;
  --details-bg: #383838;
  --details-color: #cccccc;
  --code-color: #d4d4d4;
  --scrollbar-thumb: #2672ed;
  --scrollbar-track: #424242;
}
html, body { height: auto; margin: 0; overflow: visible; }
body { font-family: sans-serif; padding: 12px; font-size: 14px;
  background: var(--bg); color: var(--text); }
.conv-title { font-size: 0.85em; color: var(--title-color); margin-bottom: 6px; }
.chips { margin-bottom: 8px; }
.chip { display: inline-block; background: var(--chip-bg); color: var(--chip-color);
  border-radius: 10px; padding: 1px 8px; font-size: 0.7em; margin-right: 4px; }
.msg { padding: 8px 12px; border-radius: 8px; white-space: pre-wrap;
  word-break: break-word; line-height: 1.5; margin-bottom: 8px; }
.human { background: var(--msg-human-bg); }
.assistant { background: var(--msg-assistant-bg); }
.sender { font-size: 0.75em; color: var(--sender-color); margin-bottom: 4px; }
.ts { font-size: 0.7em; color: var(--ts-color); margin-top: 4px; }
.summary-line { font-size: 0.75em; color: var(--sender-color); margin-bottom: 6px; }
details { background: var(--details-bg); border-radius: 6px; margin: 6px 0; padding: 4px 8px; }
details > summary { cursor: pointer; font-size: 0.78em; color: var(--details-color);
  list-style: none; user-select: none; }
details > summary::-webkit-details-marker { display: none; }
details > summary::before { content: "\25B8 "; }
details[open] > summary::before { content: "\25BE "; }
.tool-list { margin: 6px 0 2px 0; padding: 0; list-style: none; }
.tool-list li { font-size: 0.78em; margin: 2px 0; word-break: break-all;
  font-family: ui-monospace, Consolas, monospace; color: var(--code-color); }
.tool-name { font-weight: bold; }
.thinking-body, .agent-body { font-size: 0.8em; white-space: pre-wrap;
  word-break: break-word; line-height: 1.45; margin: 4px 0; }
.notice { font-size: 0.75em; color: var(--sender-color); font-style: italic; margin: 4px 0; }
.agent-prompt { font-size: 0.78em; white-space: pre-wrap; word-break: break-word;
  border-left: 2px solid var(--chip-bg); padding-left: 6px; margin: 4px 0; }
.truncated { font-size: 0.75em; color: var(--ts-color); }
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

// renderTurnHTML は1ターンの詳細HTMLを組み立てる。
func renderTurnHTML(t turn) string {
	var sb strings.Builder
	sb.WriteString(turnHTMLHead)

	if t.SessionTitle != "" {
		sb.WriteString(`<div class="conv-title">`)
		sb.WriteString(html.EscapeString(t.SessionTitle))
		sb.WriteString(`</div>`)
	}
	if t.Project != "" || t.Branch != "" {
		sb.WriteString(`<div class="chips">`)
		for _, c := range []string{t.Project, t.Branch} {
			if c == "" {
				continue
			}
			sb.WriteString(`<span class="chip">`)
			sb.WriteString(html.EscapeString(c))
			sb.WriteString(`</span>`)
		}
		sb.WriteString(`</div>`)
	}

	// 人間の発言
	sb.WriteString(`<div class="msg human"><div class="sender">あなた</div>`)
	sb.WriteString(html.EscapeString(t.Prompt))
	sb.WriteString(`</div>`)

	// Claudeの応答
	sb.WriteString(`<div class="msg assistant"><div class="sender">Claude</div>`)
	if summary := renderSummaryLine(t.Items); summary != "" {
		sb.WriteString(`<div class="summary-line">`)
		sb.WriteString(summary)
		sb.WriteString(`</div>`)
	}
	for _, item := range t.Items {
		if sb.Len() > maxHTMLBytes {
			sb.WriteString(`<div class="truncated">(以降は長いため省略しました)</div>`)
			break
		}
		renderItem(&sb, item)
	}
	ts := ""
	if !t.RelatedTime.IsZero() {
		ts = t.RelatedTime.Local().Format("2006-01-02 15:04")
	}
	sb.WriteString(`<div class="ts">`)
	sb.WriteString(ts)
	sb.WriteString(`</div></div>`)

	sb.WriteString(`</body></html>`)
	return sb.String()
}

// renderSummaryLine はターン全体のツール実行回数とthinking件数の1行サマリを作る。
func renderSummaryLine(items []turnItem) string {
	counts := map[string]int{}
	var order []string
	thinking := 0
	for _, item := range items {
		switch item.Kind {
		case "tools":
			for _, tc := range item.Tools {
				name := tc.Name
				if name == "" {
					name = "(不明)"
				}
				if _, ok := counts[name]; !ok {
					order = append(order, name)
				}
				counts[name]++
			}
		case "thinking":
			thinking += len(item.Thinking)
		}
	}
	if len(order) == 0 && thinking == 0 {
		return ""
	}
	slices.SortFunc(order, func(a, b string) int {
		if counts[a] != counts[b] {
			return counts[b] - counts[a]
		}
		return strings.Compare(a, b)
	})

	var parts []string
	for _, name := range order {
		parts = append(parts, fmt.Sprintf("%s ×%d", html.EscapeString(name), counts[name]))
	}
	if thinking > 0 {
		parts = append(parts, fmt.Sprintf("thinking ×%d", thinking))
	}
	return strings.Join(parts, " · ")
}

// renderItem はターン本文の要素1つを描画する。
func renderItem(sb *strings.Builder, item turnItem) {
	switch item.Kind {
	case "text":
		sb.WriteString(`<div>`)
		sb.WriteString(html.EscapeString(item.Text))
		sb.WriteString(`</div>`)

	case "notice":
		sb.WriteString(`<div class="notice">`)
		sb.WriteString(html.EscapeString(item.Text))
		sb.WriteString(`</div>`)

	case "thinking":
		fmt.Fprintf(sb, `<details><summary>💭 thinking (%d)</summary>`, len(item.Thinking))
		for _, th := range item.Thinking {
			sb.WriteString(`<div class="thinking-body">`)
			sb.WriteString(html.EscapeString(th))
			sb.WriteString(`</div>`)
		}
		sb.WriteString(`</details>`)

	case "tools":
		renderTools(sb, item.Tools)
	}
}

// renderTools は連続したツール実行を1つの折りたたみにまとめる。
// サブエージェントを起動したツールだけは、その中にさらに会話を折りたたんで入れる。
func renderTools(sb *strings.Builder, tools []toolCall) {
	sb.WriteString(`<details><summary>🔧 `)
	sb.WriteString(toolGroupLabel(tools))
	sb.WriteString(`</summary><ul class="tool-list">`)
	for _, tc := range tools {
		sb.WriteString(`<li><span class="tool-name">`)
		sb.WriteString(html.EscapeString(tc.Name))
		sb.WriteString(`</span>`)
		if tc.Summary != "" {
			sb.WriteString(` `)
			sb.WriteString(html.EscapeString(tc.Summary))
		}
		sb.WriteString(`</li>`)
	}
	sb.WriteString(`</ul>`)
	for _, tc := range tools {
		if tc.Agent != nil {
			renderSubAgent(sb, tc.Agent)
		}
	}
	sb.WriteString(`</details>`)
}

// toolGroupLabel は "Bash ×12  Edit ×5" のようなラベルを作る。
func toolGroupLabel(tools []toolCall) string {
	counts := map[string]int{}
	var order []string
	for _, tc := range tools {
		name := tc.Name
		if name == "" {
			name = "(不明)"
		}
		if _, ok := counts[name]; !ok {
			order = append(order, name)
		}
		counts[name]++
	}
	var parts []string
	for _, name := range order {
		parts = append(parts, fmt.Sprintf("%s ×%d", html.EscapeString(name), counts[name]))
	}
	return strings.Join(parts, "  ")
}

// renderSubAgent はサブエージェントの会話を折りたたみで描画する。
func renderSubAgent(sb *strings.Builder, sa *subAgent) {
	label := sa.AgentType
	if label == "" {
		label = "Agent"
	}
	if sa.Description != "" {
		label += "「" + sa.Description + "」"
	}
	sb.WriteString(`<details><summary>🤖 `)
	sb.WriteString(html.EscapeString(label))
	sb.WriteString(`</summary>`)
	if sa.Prompt != "" {
		sb.WriteString(`<div class="agent-prompt">`)
		sb.WriteString(html.EscapeString(sa.Prompt))
		sb.WriteString(`</div>`)
	}
	for _, item := range sa.Items {
		if sb.Len() > maxHTMLBytes {
			sb.WriteString(`<div class="truncated">(以降は長いため省略しました)</div>`)
			break
		}
		switch item.Kind {
		case "text":
			sb.WriteString(`<div class="agent-body">`)
			sb.WriteString(html.EscapeString(item.Text))
			sb.WriteString(`</div>`)
		case "tools":
			sb.WriteString(`<div class="notice">`)
			sb.WriteString(html.EscapeString(toolGroupLabelPlain(item.Tools)))
			sb.WriteString(`</div>`)
		}
	}
	sb.WriteString(`</details>`)
}

// toolGroupLabelPlain はエスケープ前のツール集計ラベルを返す。
func toolGroupLabelPlain(tools []toolCall) string {
	counts := map[string]int{}
	var order []string
	for _, tc := range tools {
		name := tc.Name
		if name == "" {
			name = "(不明)"
		}
		if _, ok := counts[name]; !ok {
			order = append(order, name)
		}
		counts[name]++
	}
	var parts []string
	for _, name := range order {
		parts = append(parts, fmt.Sprintf("%s ×%d", name, counts[name]))
	}
	return "🔧 " + strings.Join(parts, "  ")
}

// renderNotFoundHTML はターンが見つからなかったときのHTML。
func renderNotFoundHTML() string {
	return turnHTMLHead + `<p>ターンが見つかりません</p></body></html>`
}

// formatUnix は設定画面表示用に時刻を整形する。
func formatUnix(unix int64) string {
	if unix == 0 {
		return "-"
	}
	return time.Unix(unix, 0).Local().Format("2006-01-02 15:04:05")
}
