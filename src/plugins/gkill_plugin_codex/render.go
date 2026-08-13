package main

import (
	"fmt"
	"html"
	"slices"
	"strings"
	"time"
)

// maxHTMLBytes は1KyouのHTMLの上限。
// gkill側の読み取りバッファは32MBだが、そこまで大きいものは表示できないので手前で打ち切る。
const maxHTMLBytes = 4 * 1024 * 1024

// turnHTMLHead はテーマ追従とiframe自動リサイズのための共通ヘッダ。
// 同梱プラグインで実績のある仕組みをそのまま使っている。
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
.plan-body { font-size: 0.8em; white-space: pre-wrap; word-break: break-word;
  line-height: 1.45; margin: 4px 0; }
.patch-list { margin: 6px 0 2px 0; padding: 0; list-style: none; }
.patch-list li { font-size: 0.78em; margin: 2px 0; word-break: break-all;
  font-family: ui-monospace, Consolas, monospace; color: var(--code-color); }
.patch-add { color: #2e7d32; }
.patch-del { color: #c62828; }
[data-theme="dark"] .patch-add { color: #81c784; }
[data-theme="dark"] .patch-del { color: #ef9a9a; }
.ide-list { margin: 4px 0 2px 0; padding: 0; list-style: none; }
.ide-list li { font-size: 0.78em; margin: 2px 0; word-break: break-all;
  font-family: ui-monospace, Consolas, monospace; color: var(--code-color); }
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

// renderMessageHTML は1Kyouの詳細HTMLを組み立てる。
func renderMessageHTML(m message) string {
	var sb strings.Builder
	sb.WriteString(turnHTMLHead)

	if m.Title != "" {
		sb.WriteString(`<div class="conv-title">`)
		sb.WriteString(html.EscapeString(m.Title))
		sb.WriteString(`</div>`)
	}
	renderChips(&sb, m)

	class, sender := "assistant", "Codex"
	if m.Role == roleHuman {
		class, sender = "human", "あなた"
	}
	sb.WriteString(`<div class="msg `)
	sb.WriteString(class)
	sb.WriteString(`"><div class="sender">`)
	sb.WriteString(sender)
	sb.WriteString(`</div>`)
	sb.WriteString(html.EscapeString(m.Text))

	// IDEの前置きは本文の「後ろ」に畳んで置く。
	// rykv は一覧の行にこのHTMLをそのまま描くので、
	// 前に出すとどの行も「開いているタブ一覧」で埋まって読めなくなる。
	renderIDEContext(&sb, m.IDEContext)

	if summary := renderSummaryLine(m.Items); summary != "" {
		sb.WriteString(`<div class="summary-line">`)
		sb.WriteString(summary)
		sb.WriteString(`</div>`)
	}
	for _, item := range m.Items {
		if sb.Len() > maxHTMLBytes {
			sb.WriteString(`<div class="truncated">(以降は長いため省略しました)</div>`)
			break
		}
		renderItem(&sb, item)
	}

	ts := ""
	if !m.RelatedTime.IsZero() {
		ts = m.RelatedTime.Local().Format("2006-01-02 15:04")
	}
	sb.WriteString(`<div class="ts">`)
	sb.WriteString(html.EscapeString(ts))
	sb.WriteString(`</div></div>`)

	sb.WriteString(`</body></html>`)
	return sb.String()
}

func renderChips(sb *strings.Builder, m message) {
	chips := []string{m.Project, m.Branch, m.Model}
	if duration := formatDuration(m.DurationMs); duration != "" {
		chips = append(chips, duration)
	}
	chips = dedupeStrings(chips)
	if len(chips) == 0 {
		return
	}
	sb.WriteString(`<div class="chips">`)
	for _, chip := range chips {
		sb.WriteString(`<span class="chip">`)
		sb.WriteString(html.EscapeString(chip))
		sb.WriteString(`</span>`)
	}
	sb.WriteString(`</div>`)
}

// formatDuration は所要時間を「所要 3分12秒」の形にする。1秒未満は出さない。
func formatDuration(ms int64) string {
	if ms < 1000 {
		return ""
	}
	seconds := ms / 1000
	if seconds < 60 {
		return fmt.Sprintf("所要 %d秒", seconds)
	}
	return fmt.Sprintf("所要 %d分%d秒", seconds/60, seconds%60)
}

// renderIDEContext はIDEの前置きを折りたたみで描く。
func renderIDEContext(sb *strings.Builder, ideCtx *ideContext) {
	if ideCtx == nil {
		return
	}
	label := "🖥 IDEのコンテキスト"
	switch {
	case ideCtx.ActiveFile != "" && len(ideCtx.OpenTabs) != 0:
		label = fmt.Sprintf("🖥 IDEのコンテキスト (Active: %s / タブ %d件)", ideCtx.ActiveFile, len(ideCtx.OpenTabs))
	case ideCtx.ActiveFile != "":
		label = "🖥 IDEのコンテキスト (Active: " + ideCtx.ActiveFile + ")"
	case len(ideCtx.OpenTabs) != 0:
		label = fmt.Sprintf("🖥 IDEのコンテキスト (タブ %d件)", len(ideCtx.OpenTabs))
	}
	sb.WriteString(`<details><summary>`)
	sb.WriteString(html.EscapeString(label))
	sb.WriteString(`</summary><ul class="ide-list">`)
	for _, tab := range ideCtx.OpenTabs {
		sb.WriteString(`<li>`)
		sb.WriteString(html.EscapeString(tab.Name))
		if tab.Path != "" {
			sb.WriteString(`: `)
			sb.WriteString(html.EscapeString(tab.Path))
		}
		sb.WriteString(`</li>`)
	}
	sb.WriteString(`</ul></details>`)
}

// renderSummaryLine は応答に含まれるツール実行回数・thinking件数・変更ファイル数の1行サマリを作る。
func renderSummaryLine(items []turnItem) string {
	counts := map[string]int{}
	var order []string
	thinking := 0
	files := 0
	for _, item := range items {
		switch item.Kind {
		case blockTools:
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
		case blockThinking:
			thinking += len(item.Thinking)
		case blockPatch:
			files += len(item.Files)
		}
	}
	if len(order) == 0 && thinking == 0 && files == 0 {
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
	if files > 0 {
		parts = append(parts, fmt.Sprintf("変更 %dファイル", files))
	}
	return strings.Join(parts, " · ")
}

// renderItem は応答本文の要素1つを描画する。
func renderItem(sb *strings.Builder, item turnItem) {
	switch item.Kind {
	case blockText:
		sb.WriteString(`<div>`)
		sb.WriteString(html.EscapeString(item.Text))
		sb.WriteString(`</div>`)

	case blockNotice:
		sb.WriteString(`<div class="notice">`)
		sb.WriteString(html.EscapeString(item.Text))
		sb.WriteString(`</div>`)

	case blockThinking:
		fmt.Fprintf(sb, `<details><summary>💭 thinking (%d)</summary>`, len(item.Thinking))
		for _, thinking := range item.Thinking {
			sb.WriteString(`<div class="thinking-body">`)
			sb.WriteString(html.EscapeString(thinking))
			sb.WriteString(`</div>`)
		}
		sb.WriteString(`</details>`)

	case blockPlan:
		sb.WriteString(`<details><summary>📋 計画</summary><div class="plan-body">`)
		sb.WriteString(html.EscapeString(item.Text))
		sb.WriteString(`</div></details>`)

	case blockPatch:
		renderPatch(sb, item.Files)

	case blockTools:
		renderTools(sb, item.Tools)

	case blockSpawn:
		if item.Agent != nil {
			renderSubAgent(sb, item.Agent)
		}
	}
}

// renderPatch は変更したファイルの一覧を描く。unified diff は持っていない。
func renderPatch(sb *strings.Builder, files []patchFile) {
	if len(files) == 0 {
		return
	}
	fmt.Fprintf(sb, `<details><summary>📝 変更したファイル (%d)</summary><ul class="patch-list">`, len(files))
	for _, file := range files {
		sb.WriteString(`<li>`)
		sb.WriteString(html.EscapeString(file.Path))
		if file.Type != "" {
			sb.WriteString(` (`)
			sb.WriteString(html.EscapeString(file.Type))
			sb.WriteString(`)`)
		}
		if file.Added > 0 {
			fmt.Fprintf(sb, ` <span class="patch-add">+%d</span>`, file.Added)
		}
		if file.Removed > 0 {
			fmt.Fprintf(sb, ` <span class="patch-del">-%d</span>`, file.Removed)
		}
		sb.WriteString(`</li>`)
	}
	sb.WriteString(`</ul></details>`)
}

// renderTools は連続したツール呼び出しを1つの折りたたみにまとめる。
// サブエージェントを起こしたツールだけは、その中にさらに会話を折りたたんで入れる。
func renderTools(sb *strings.Builder, tools []toolCall) {
	if len(tools) == 0 {
		return
	}
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

// toolGroupLabel は "exec ×12  apply_patch ×3" のようなラベルを作る。
func toolGroupLabel(tools []toolCall) string {
	names, counts := toolCounts(tools)
	var parts []string
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("%s ×%d", html.EscapeString(name), counts[name]))
	}
	return strings.Join(parts, "  ")
}

// toolGroupLabelPlain はエスケープ前のツール集計ラベルを返す。
func toolGroupLabelPlain(tools []toolCall) string {
	names, counts := toolCounts(tools)
	var parts []string
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("%s ×%d", name, counts[name]))
	}
	return "🔧 " + strings.Join(parts, "  ")
}

func toolCounts(tools []toolCall) ([]string, map[string]int) {
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
	return order, counts
}

// renderSubAgent はサブエージェントの会話を折りたたみで描画する。
func renderSubAgent(sb *strings.Builder, agent *subAgent) {
	label := agent.Nickname
	if label == "" {
		label = "Agent"
	}
	if agent.AgentPath != "" {
		label += " " + agent.AgentPath
	}
	sb.WriteString(`<details><summary>🤖 `)
	sb.WriteString(html.EscapeString(label))
	sb.WriteString(`</summary>`)
	if agent.Prompt != "" {
		sb.WriteString(`<div class="agent-prompt">`)
		sb.WriteString(html.EscapeString(agent.Prompt))
		sb.WriteString(`</div>`)
	}
	for _, item := range agent.Items {
		if sb.Len() > maxHTMLBytes {
			sb.WriteString(`<div class="truncated">(以降は長いため省略しました)</div>`)
			break
		}
		switch item.Kind {
		case blockText, blockPlan:
			sb.WriteString(`<div class="agent-body">`)
			sb.WriteString(html.EscapeString(item.Text))
			sb.WriteString(`</div>`)
		case blockTools:
			sb.WriteString(`<div class="notice">`)
			sb.WriteString(html.EscapeString(toolGroupLabelPlain(item.Tools)))
			sb.WriteString(`</div>`)
		case blockPatch:
			sb.WriteString(`<div class="notice">`)
			sb.WriteString(html.EscapeString(fmt.Sprintf("📝 変更 %dファイル", len(item.Files))))
			sb.WriteString(`</div>`)
		}
	}
	sb.WriteString(`</details>`)
}

// renderNotFoundHTML は対象が見つからなかったときのHTML。
func renderNotFoundHTML() string {
	return turnHTMLHead + `<p>見つかりません</p></body></html>`
}

// formatTime は設定画面表示用に時刻を整形する。
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04:05")
}
