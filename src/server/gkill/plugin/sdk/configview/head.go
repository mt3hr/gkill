package configview

// lightVars は明るい配色の CSS 変数。7本の設定画面にあった変数の和集合（field / btn / bar を含む）。
// Claude.ai 版が未定義の --field-bg / --btn-bg / --btn-color を参照して textarea が透明になっていたのは、
// 変数がここ1か所になったことで起きなくなる。
const lightVars = `  --bg: #ffffff;
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
`

// darkVars は暗い配色の CSS 変数。OS の配色（prefers-color-scheme）と gkill のテーマ（data-theme）の両方から参照する。
const darkVars = `  --bg: #212121;
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
`

// headHTML は設定画面の共通ヘッダ。
//
// 明暗は2段で決める。設定ダイアログ（use-plugin-config-dialog.ts）は iframe の load で gkill_theme を1回送るので、
// 届いたら data-theme が勝つ。届く前・届かない古い本体では OS の配色（prefers-color-scheme）に追従する。
// data-theme="light" が付いたら OS が暗くても明るい配色にする（gkill の見た目に合わせる）。
//
// iframe は allow-same-origin なしで動くため、CSS も JS もこの文書の中に閉じる（外部ファイルは読めない）。
// 本文（chatview）と違い、設定ダイアログは iframe の高さを自分で持つので gkill_iframe_size は送らない。
const headHTML = `<!DOCTYPE html>
<html><head><meta charset="utf-8">
<style>
:root {
` + lightVars + `}
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
` + darkVars + `  }
}
:root[data-theme="dark"] {
` + darkVars + `}
body { font-family: sans-serif; margin: 16px; background: var(--bg); color: var(--text); }
h2 { font-size: 1.1em; margin-top: 0; }
h3 { font-size: 0.95em; margin-bottom: 4px; }
p { margin: 6px 0; }
.ok { background: var(--ok-bg); border-left: 4px solid var(--ok-border);
  padding: 12px; border-radius: 4px; margin: 12px 0; }
.warn { background: var(--warn-bg); border-left: 4px solid var(--warn-border);
  padding: 12px; border-radius: 4px; margin: 12px 0; }
.ok p:first-child, .warn p:first-child { margin-top: 0; }
.ok p:last-child, .warn p:last-child { margin-bottom: 0; }
code { background: var(--code-bg); padding: 2px 6px; border-radius: 3px; font-size: 0.9em; word-break: break-all; }
ul, ol { margin: 4px 0; padding-left: 20px; font-size: 0.85em; }
pre { background: var(--code-bg); border: 1px solid var(--border); border-radius: 4px;
  padding: 8px; font-size: 0.85em; overflow-x: auto; }
.hint { font-size: 0.8em; color: var(--muted); margin-top: 4px; }
.tag { font-size: 0.75em; background: var(--code-bg); color: var(--muted);
  border-radius: 8px; padding: 0 6px; }
table { border-collapse: collapse; font-size: 0.85em; margin: 8px 0; }
td, th { padding: 2px 12px 2px 0; text-align: left; vertical-align: top; }
td.k, th { color: var(--muted); font-weight: normal; }
label { font-size: 0.85em; display: block; margin-top: 8px; }
textarea, input[type=text] { width: 100%; box-sizing: border-box;
  font-family: ui-monospace, Consolas, monospace; font-size: 0.85em;
  background: var(--field-bg); color: var(--text);
  border: 1px solid var(--border); border-radius: 4px; padding: 6px; margin-top: 2px; }
button { background: var(--btn-bg); color: var(--btn-color); border: none;
  border-radius: 4px; padding: 6px 16px; font-size: 0.9em; cursor: pointer; margin-top: 6px; }
button:disabled { opacity: 0.6; cursor: default; }
#gkill_save_result { font-size: 0.85em; margin-left: 8px; color: var(--muted); }
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
