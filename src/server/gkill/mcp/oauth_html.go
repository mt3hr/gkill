package mcp

// OAuth 認可ログインフォームの HTML（旧 oauth-html.mjs）。
// POST /oauth/authorize へ資格情報を送る最小限の自己完結ページ。パスワードは送信前に
// クライアント側で SHA-256 にする（gkill 本体の慣習に合わせる）。

import (
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// scopeDescription は scope → 同意画面の説明。以前のログイン画面は scope を hidden で送るだけで
// 「何に書き込み許可を与えるのか」を一切表示していなかった (2026-08-30 レビュー P0)。
// 未知の scope は認可側の検証で 400 になるのでここへは来ないが、来ても生の値を出す。
type scopeDescription struct {
	Label    string
	Detail   string
	Writable bool
}

var scopeDescriptions = map[string]scopeDescription{
	"gkill:read": {
		Label:    "読み取り専用",
		Detail:   "記録の閲覧のみできます。追加・変更・削除はできません。",
		Writable: false,
	},
	"gkill:write": {
		Label:    "書き込み",
		Detail:   "記録の追加・更新・削除・復元ができます。",
		Writable: true,
	},
	"gkill:readwrite": {
		Label:    "読み書き",
		Detail:   "記録の閲覧に加えて、追加・更新・削除・復元ができます。",
		Writable: true,
	},
}

// escHtml は & < > " をエスケープする（Node 版と同じ順序・同じ4文字）。
func escHtml(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return replacer.Replace(s)
}

// LoginPageParams はログインページの内容。
type LoginPageParams struct {
	ClientID            string
	ClientName          string
	RedirectURI         string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Scope               string
	Issuer              string
	Resource            string
	Error               string
}

// RenderLoginPage は認可ログインページを描画する。
func RenderLoginPage(p LoginPageParams) string {
	errorBlock := ""
	if p.Error != "" {
		errorBlock = `<div class="error">` + escHtml(p.Error) + `</div>`
	}

	scopeInfo, ok := scopeDescriptions[p.Scope]
	if !ok {
		scopeInfo = scopeDescription{Label: p.Scope, Detail: "", Writable: false}
	}
	clientName := p.ClientName
	if clientName == "" {
		clientName = "(名前未登録のクライアント)"
	}
	writableClass := ""
	if scopeInfo.Writable {
		writableClass = "consent-writable"
	}
	detailBlock := ""
	if scopeInfo.Detail != "" {
		detailBlock = `<div class="consent-detail">` + escHtml(scopeInfo.Detail) + `</div>`
	}
	issuerBlock := ""
	if p.Issuer != "" {
		issuerBlock = `<div class="consent-row"><span class="consent-key">接続先</span><span>` + escHtml(p.Issuer) + `</span></div>`
	}
	consentBlock := `<div class="consent">
    <div class="consent-row"><span class="consent-key">アプリ</span><span>` + escHtml(clientName) + `</span></div>
    <div class="consent-row"><span class="consent-key">許可する操作</span><span class="` + writableClass + `">` + escHtml(scopeInfo.Label) + ` (` + escHtml(p.Scope) + `)</span></div>
    ` + detailBlock + `
    ` + issuerBlock + `
  </div>`

	return `<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>gkill — ログイン</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: Roboto, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    background: #212121; color: #ffffffde;
    display: flex; align-items: center; justify-content: center;
    min-height: 100vh; padding: 16px;
  }
  .card {
    background: #212121; border-radius: 4px; padding: 32px;
    width: 100%; max-width: 400px;
  }
  .welcome { font-size: x-large; margin-bottom: 16px; text-align: center; }
  h1 {
    font-size: 1.25rem; margin-bottom: 4px; color: #ffffffde;
    text-align: center; font-weight: 500;
  }
  .subtitle {
    font-size: 0.85rem; color: #999999; margin-bottom: 24px;
    text-align: center;
  }
  .error {
    background: #7a0117; color: #ef9a9a; border: 1px solid #a0334d;
    border-radius: 4px; padding: 10px 14px; margin-bottom: 16px; font-size: 0.9rem;
  }
  .consent {
    background: #2a2a2a; border: 1px solid #444; border-radius: 4px;
    padding: 12px 14px; margin-bottom: 20px; font-size: 0.85rem;
  }
  .consent-row { display: flex; gap: 12px; margin-bottom: 6px; word-break: break-all; }
  .consent-row:last-child { margin-bottom: 0; }
  .consent-key { color: #999999; flex: 0 0 6.5em; }
  .consent-writable { color: #ffb74d; font-weight: 500; }
  .consent-detail { color: #bbbbbb; margin: 2px 0 8px; }
  label { display: block; font-size: 0.75rem; color: #999999; margin-bottom: 4px; }
  input[type="text"], input[type="password"] {
    width: 100%; padding: 10px 12px; border: 1px solid #555; border-radius: 4px;
    background: #212121; color: #ffffffde; font-size: 1rem; margin-bottom: 16px;
    outline: none; transition: border-color 0.2s;
  }
  input:focus { border-color: #2672ed; }
  button {
    width: 100%; padding: 10px; border: none; border-radius: 4px;
    background: #2672ed; color: #ffffff; font-size: 0.875rem; font-weight: 500;
    cursor: pointer; transition: background 0.2s; text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  button:hover { background: #1e5fc7; }
  button:disabled { background: #555; cursor: not-allowed; }
  .client-info { font-size: 0.75rem; color: #666; margin-top: 16px; text-align: center; }
</style>
</head>
<body>
<div class="card">
  <div class="welcome">⭐️</div>
  <h1>gkill</h1>
  <div class="subtitle">MCP OAuth ログイン</div>
  ` + errorBlock + `
  ` + consentBlock + `
  <form id="loginForm" method="POST" action="/oauth/authorize">
    <input type="hidden" name="client_id" value="` + escHtml(p.ClientID) + `">
    <input type="hidden" name="redirect_uri" value="` + escHtml(p.RedirectURI) + `">
    <input type="hidden" name="state" value="` + escHtml(p.State) + `">
    <input type="hidden" name="code_challenge" value="` + escHtml(p.CodeChallenge) + `">
    <input type="hidden" name="code_challenge_method" value="` + escHtml(p.CodeChallengeMethod) + `">
    <input type="hidden" name="scope" value="` + escHtml(p.Scope) + `">
    <input type="hidden" name="resource" value="` + escHtml(p.Resource) + `">
    <input type="hidden" name="response_type" value="code">
    <input type="hidden" name="password_sha256" id="password_sha256" value="">
    <label for="user_id">ユーザーID</label>
    <input type="text" id="user_id" name="user_id" autocomplete="username" required autofocus>
    <label for="password">パスワード</label>
    <input type="password" id="password" name="password_raw" autocomplete="current-password">
    <button type="submit">ログイン</button>
  </form>
  <div class="client-info">Client: ` + escHtml(p.ClientID) + `</div>
</div>
<script>
document.getElementById("loginForm").addEventListener("submit", async function(e) {
  e.preventDefault();
  const form = e.target;
  const pw = form.password_raw.value;
  if (pw) {
    const encoded = new TextEncoder().encode(pw);
    const hashBuf = await crypto.subtle.digest("SHA-256", encoded);
    const hashHex = Array.from(new Uint8Array(hashBuf)).map(b => b.toString(16).padStart(2, "0")).join("");
    form.password_sha256.value = hashHex;
  }
  form.password_raw.removeAttribute("name");
  form.submit();
});
</script>
</body>
</html>`
}

// jsStringLiteral は <script> の中へ埋め込んでよい JS 文字列リテラルを返す。
// JSON 文字列にしたうえで、HTML 側が解釈しうる < > と JS の行区切り U+2028 / U+2029 を \uXXXX へ逃がす。
// redirect_uri は動的登録したクライアントが名乗る値なので、"</script>" を含めても script 要素が
// 早期に閉じない形にしておく（CodeQL go/reflected-xss の指摘箇所）。URL の & はそのまま（script 内では無害）。
func jsStringLiteral(s string) string {
	// 置換先は "バックスラッシュ + u + 4桁" の6文字。ソース上の \u エスケープは編集ツールが1文字へ
	// 畳むことがあるので、バックスラッシュは文字コードから組み立てる。
	backslash := string(rune(92))
	replacer := strings.NewReplacer(
		"<", backslash+"u003c",
		">", backslash+"u003e",
		string(rune(0x2028)), backslash+"u2028",
		string(rune(0x2029)), backslash+"u2029",
	)
	return replacer.Replace(jsonobj.MarshalString(s))
}

// RenderSuccessPage は自動リダイレクト付きの認可成功ページを描画する。
func RenderSuccessPage(redirectURL string) string {
	return `<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>gkill — ログイン成功</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: Roboto, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    background: #212121; color: #ffffffde;
    display: flex; align-items: center; justify-content: center;
    min-height: 100vh; padding: 16px;
  }
  .card {
    background: #212121; border-radius: 4px; padding: 32px;
    width: 100%; max-width: 400px; text-align: center;
  }
  .icon { font-size: 48px; margin-bottom: 16px; }
  h1 { font-size: 1.25rem; margin-bottom: 8px; color: #ffffffde; font-weight: 500; }
  .message { font-size: 0.9rem; color: #999999; margin-bottom: 24px; }
  .progress-bar {
    width: 100%; height: 3px; background: #333; border-radius: 2px; overflow: hidden;
  }
  .progress-bar-fill {
    height: 100%; background: #2672ed; border-radius: 2px;
    animation: progress 1.5s ease-in-out forwards;
  }
  @keyframes progress {
    from { width: 0%; }
    to { width: 100%; }
  }
</style>
</head>
<body>
<div class="card">
  <div class="icon">⭐️</div>
  <h1>ログイン成功</h1>
  <div class="message">リダイレクト中...</div>
  <div class="progress-bar"><div class="progress-bar-fill"></div></div>
</div>
<script>
setTimeout(function() {
  window.location.href = ` + jsStringLiteral(redirectURL) + `;
}, 1500);
</script>
</body>
</html>`
}
