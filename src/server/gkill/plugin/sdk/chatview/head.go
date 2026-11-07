package chatview

// headHTML はテーマ追従と iframe 自動リサイズのための共通ヘッダ。
// 4本の中で最も広かった Codex プラグインの head をそのまま採り、コードブロック用の規則を足した。
//
// スクリプトが使う postMessage のキーは gkill_iframe_size（iframe → 親）と gkill_theme（親 → iframe）だけ。
// 親は最初のサイズ通知のときだけテーマを返す（毎回返すと 10ms 周期のピンポンになる）ので、
// テーマ受信時はこちらが setTimeout で測り直す。<details> の開閉（toggle）でも高さが変わるので capture で拾う。
// dblclick の転送はクライアント（plugin-html-view.vue）が本文の末尾にスクリプトを足すので、ここには書かない。
//
// iframe は allow-same-origin なしで動くため、CSS も JS もこの文書の中に閉じる（外部ファイルは読めない）。
const headHTML = `<!DOCTYPE html>
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
  --code-bg: #e9eaec;
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
  --code-bg: #1e1e1e;
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
.fold-list { margin: 4px 0 2px 0; padding: 0; list-style: none; }
.fold-list li { font-size: 0.78em; margin: 2px 0; word-break: break-all;
  font-family: ui-monospace, Consolas, monospace; color: var(--code-color); }
.answer { margin: 2px 0 6px 0; }
.commentary { font-size: 0.85em; white-space: pre-wrap; word-break: break-word;
  line-height: 1.45; margin: 4px 0; }
.question { border-left: 3px solid var(--sender-color); padding-left: 8px; margin: 6px 0; }
.question-label { font-size: 0.75em; color: var(--sender-color); margin-bottom: 2px; }
.truncated { font-size: 0.75em; color: var(--ts-color); }
.msg code { font-family: ui-monospace, Consolas, monospace; font-size: 0.9em;
  background: var(--code-bg); color: var(--code-color); padding: 1px 4px; border-radius: 3px; }
.msg pre { background: var(--code-bg); color: var(--code-color); border-radius: 6px;
  padding: 8px 10px; margin: 6px 0; white-space: pre; overflow: auto; max-height: 20em;
  font-size: 0.85em; line-height: 1.4; }
.msg pre code { background: none; padding: 0; font-size: inherit; }
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
