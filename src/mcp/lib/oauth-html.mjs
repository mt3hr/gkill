// HTML template for the OAuth authorization login form.
// Renders a minimal, self-contained login page that submits credentials
// to POST /oauth/authorize. The password is SHA-256 hashed client-side
// before submission (matching gkill backend convention).

// scope → 同意画面の説明。以前のログイン画面は scope を hidden で送るだけで
// 「何に書き込み許可を与えるのか」を一切表示していなかった (2026-08-30 レビュー P0)。
// 未知の scope は認可側の検証で 400 になるのでここへは来ないが、来ても生の値を出す。
const SCOPE_DESCRIPTIONS = {
  "gkill:read": {
    label: "読み取り専用",
    detail: "記録の閲覧のみできます。追加・変更・削除はできません。",
    writable: false,
  },
  "gkill:write": {
    label: "書き込み",
    detail: "記録の追加・更新・削除・復元ができます。",
    writable: true,
  },
  "gkill:readwrite": {
    label: "読み書き",
    detail: "記録の閲覧に加えて、追加・更新・削除・復元ができます。",
    writable: true,
  },
};

/**
 * Render the authorization login page.
 * @param {object} params
 * @param {string} params.clientId
 * @param {string} [params.clientName] - DCR で登録された表示名 (無ければ空)。
 * @param {string} params.redirectUri
 * @param {string} params.state
 * @param {string} params.codeChallenge
 * @param {string} params.codeChallengeMethod
 * @param {string} params.scope
 * @param {string} [params.issuer] - 許可先のサーバ (公開 issuer URL)。
 * @param {string} [params.resource] - RFC 8707 resource indicator.
 * @param {string} [params.error] - Error message to display (e.g., "Invalid credentials").
 * @returns {string} HTML string.
 */
export function renderLoginPage({
  clientId,
  clientName,
  redirectUri,
  state,
  codeChallenge,
  codeChallengeMethod,
  scope,
  issuer,
  resource,
  error,
}) {
  const escHtml = (s) => String(s || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");

  const errorBlock = error
    ? `<div class="error">${escHtml(error)}</div>`
    : "";

  const scopeInfo = SCOPE_DESCRIPTIONS[scope] || { label: scope, detail: "", writable: false };
  const consentBlock = `<div class="consent">
    <div class="consent-row"><span class="consent-key">アプリ</span><span>${escHtml(clientName || "(名前未登録のクライアント)")}</span></div>
    <div class="consent-row"><span class="consent-key">許可する操作</span><span class="${scopeInfo.writable ? "consent-writable" : ""}">${escHtml(scopeInfo.label)} (${escHtml(scope)})</span></div>
    ${scopeInfo.detail ? `<div class="consent-detail">${escHtml(scopeInfo.detail)}</div>` : ""}
    ${issuer ? `<div class="consent-row"><span class="consent-key">接続先</span><span>${escHtml(issuer)}</span></div>` : ""}
  </div>`;

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
  ${errorBlock}
  ${consentBlock}
  <form id="loginForm" method="POST" action="/oauth/authorize">
    <input type="hidden" name="client_id" value="${escHtml(clientId)}">
    <input type="hidden" name="redirect_uri" value="${escHtml(redirectUri)}">
    <input type="hidden" name="state" value="${escHtml(state)}">
    <input type="hidden" name="code_challenge" value="${escHtml(codeChallenge)}">
    <input type="hidden" name="code_challenge_method" value="${escHtml(codeChallengeMethod)}">
    <input type="hidden" name="scope" value="${escHtml(scope)}">
    <input type="hidden" name="resource" value="${escHtml(resource)}">
    <input type="hidden" name="response_type" value="code">
    <input type="hidden" name="password_sha256" id="password_sha256" value="">
    <label for="user_id">ユーザーID</label>
    <input type="text" id="user_id" name="user_id" autocomplete="username" required autofocus>
    <label for="password">パスワード</label>
    <input type="password" id="password" name="password_raw" autocomplete="current-password">
    <button type="submit">ログイン</button>
  </form>
  <div class="client-info">Client: ${escHtml(clientId)}</div>
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
</html>`;
}

/**
 * Render the authorization success page with auto-redirect.
 * @param {object} params
 * @param {string} params.redirectUrl - URL to redirect to after delay.
 * @returns {string} HTML string.
 */
export function renderSuccessPage({ redirectUrl }) {
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
  window.location.href = ${JSON.stringify(redirectUrl)};
}, 1500);
</script>
</body>
</html>`;
}
