package main

import (
	"fmt"
	"html"
	"strings"
	"time"
)

// 画面ではこのプラグインの記録は data_type が git_commit_log なので native の GitCommitLogView が描く。
// ここの HTML は get_content_html（MCP の include_plugin_content や、型別データが引けなかったときの
// 予備の表示）用で、native と同じ「+追加行 / -削除行 / メッセージ」を先頭に置き、
// ハッシュ・author・日時・出どころ・ファイル別の行数を折り畳みで足す。

// commitHTMLHead はテーマ追従とiframe自動リサイズのための共通ヘッダ。
// 同梱プラグインで実績のある仕組みをそのまま使っている。
const commitHTMLHead = `<!DOCTYPE html>
<html><head><meta charset="utf-8">
<style>
:root {
  --bg: #ffffff;
  --text: #333333;
  --muted: #9ca3af;
  --add: limegreen;
  --del: crimson;
  --details-bg: #e9eaec;
  --details-color: #4b5563;
  --code-color: #1f2937;
  --scrollbar-thumb: #2672ed;
  --scrollbar-track: #e5e7eb;
}
[data-theme="dark"] {
  --bg: #212121;
  --text: #e0e0e0;
  --muted: #888888;
  --add: limegreen;
  --del: crimson;
  --details-bg: #383838;
  --details-color: #cccccc;
  --code-color: #d4d4d4;
  --scrollbar-thumb: #2672ed;
  --scrollbar-track: #424242;
}
html, body { height: auto; margin: 0; overflow: visible; }
body { font-family: sans-serif; padding: 12px; font-size: 14px;
  background: var(--bg); color: var(--text); }
.git_commit_addition { color: var(--add); }
.git_commit_deletion { color: var(--del); }
.git_commit_log_message { white-space: pre-line; word-break: break-word; line-height: 1.5; margin: 6px 0; }
.meta { font-size: 0.78em; color: var(--muted); margin-top: 6px; word-break: break-all; }
details { background: var(--details-bg); border-radius: 6px; margin: 6px 0; padding: 4px 8px; }
details > summary { cursor: pointer; font-size: 0.78em; color: var(--details-color);
  list-style: none; user-select: none; }
details > summary::-webkit-details-marker { display: none; }
details > summary::before { content: "\25B8 "; }
details[open] > summary::before { content: "\25BE "; }
.file-list, .source-list { margin: 6px 0 2px 0; padding: 0; list-style: none; }
.file-list li, .source-list li { font-size: 0.78em; margin: 2px 0; word-break: break-all;
  font-family: ui-monospace, Consolas, monospace; color: var(--code-color); }
.warn { font-size: 0.78em; color: var(--del); }
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

// renderCommitHTML は1コミットの詳細HTMLを組み立てる。
func renderCommitHTML(body commitBody) string {
	var sb strings.Builder
	sb.WriteString(commitHTMLHead)

	// native の git-commit-log-view.vue と同じ並び
	fmt.Fprintf(&sb, `<div><span class="git_commit_addition"> + %d </span><span class="git_commit_deletion"> - %d </span></div>`,
		body.Addition, body.Deletion)
	sb.WriteString(`<div class="git_commit_log_message">` + html.EscapeString(body.Message) + `</div>`)

	sb.WriteString(`<div class="meta">`)
	sb.WriteString(html.EscapeString(body.RepName) + ` · ` + html.EscapeString(body.Hash))
	sb.WriteString(`<br>` + html.EscapeString(body.AuthorName))
	if body.AuthorEmail != "" {
		sb.WriteString(` &lt;` + html.EscapeString(body.AuthorEmail) + `&gt;`)
	}
	sb.WriteString(` · ` + html.EscapeString(formatTime(body.committedAt())))
	sb.WriteString(`</div>`)

	if body.StatsError != "" {
		sb.WriteString(`<div class="warn">行数を集計できませんでした: ` + html.EscapeString(body.StatsError) + `</div>`)
	}

	if len(body.Files) != 0 {
		fmt.Fprintf(&sb, `<details><summary>変更したファイル %d</summary><ul class="file-list">`, len(body.Files))
		for _, file := range body.Files {
			fmt.Fprintf(&sb, `<li><span class="git_commit_addition">+%d</span> <span class="git_commit_deletion">-%d</span> %s</li>`,
				file.Addition, file.Deletion, html.EscapeString(file.Path))
		}
		sb.WriteString(`</ul></details>`)
	}

	if len(body.Sources) != 0 {
		fmt.Fprintf(&sb, `<details><summary>出どころ %d</summary><ul class="source-list">`, len(body.Sources))
		for _, source := range body.Sources {
			sb.WriteString(`<li>` + html.EscapeString(source.ArchivePath) + `!/` + html.EscapeString(source.GitDir) + `</li>`)
		}
		sb.WriteString(`</ul></details>`)
	}

	sb.WriteString(`</body></html>`)
	return sb.String()
}

// renderNotFoundHTML は見つからないときの表示。
func renderNotFoundHTML() string {
	return commitHTMLHead + `<div class="meta">このコミットはキャッシュにありません。取り込み中か、zip が指定から外れています。</div></body></html>`
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05 -07:00")
}
