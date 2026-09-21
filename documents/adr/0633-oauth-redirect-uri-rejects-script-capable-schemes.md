# ADR-0633: OAuth の redirect_uri は「移動でスクリプトが走る」scheme を拒む

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-21 |
| Sources | CodeQL `go/reflected-xss`（`http_transport.go` の HTML 送出）を追った先で見つけた実際の経路。[gkill-mcp](../../.claude/skills/gkill-mcp/SKILL.md) の節「OAuth の redirect_uri」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/mcp/oauth_server.go`（`isSafeRedirectURI` / `HandleRegister` / `validateAuthorizeParams` / `HandleAuthorizePost`）/ `src/server/gkill/mcp/oauth_html.go`（`RenderSuccessPage`） |

## Context

MCP の HTTP モードは認可の最後に 302 ではなく**成功ページ**を返し、その中で
`window.location.href = <redirect_uri>` により自動遷移する。ページ側の埋め込みは
JSON 文字列リテラル化と `<` `>` の Unicode エスケープで `</script>` から抜け出せないようにしてあるが、
**遷移先の scheme は見ていなかった**。

redirect_uri の検証は2箇所にあり、どちらも「スキームのある絶対 URL か」だけを見ていた。

- 動的クライアント登録（RFC 7591、`HandleRegister`）: `redirect_uris` の各要素
- 認可要求（`validateAuthorizeParams`）: 登録済みの `redirect_uris` との一致

動的登録は仕様どおり**認証を要求しない**。したがって次の2手が通っていた。

1. 攻撃者が `redirect_uris: ["javascript:…"]` でクライアントを登録し、`client_id` を得る
2. 被害者にその `client_id` の認可 URL を踏ませ、gkill のログイン画面でログインさせる

成功ページが `window.location.href = "javascript:…"` を実行するので、
**gkill の生成元で任意のスクリプトが走り**、同じ URL に載った認可コードも攻撃者の手に渡る。
`data:` / `vbscript:` も同じ形で使える。

あわせて、成功ページを組む直前の `url.Parse` が戻り値のエラーを捨てており、
解析に失敗すると nil 参照でプロセスごと落ちる経路も残っていた。

## Decision

**redirect_uri は「ブラウザが移動しただけで、移動元のページの生成元でスクリプトが走りうる」
scheme を拒む。** 対象は `javascript` / `data` / `vbscript` / `blob` / `about` / `filesystem`
（大小無視）。判定は `isSafeRedirectURI` 1箇所に置き、**登録時と認可要求時の両方で掛ける**。
`url.Parse` のエラーは捨てず、400 を返す。

ネイティブアプリのカスタム scheme（`myapp://callback` / `com.example.app:/oauth2redirect`）は
正当な使い方なので通す。

## Rejected alternatives

- **http / https だけの許可リストにする** — 却下。ネイティブアプリのカスタム scheme が
  OAuth 2.1 で正当な redirect_uri であり、MCP クライアントには据え置き型でないものもある。
  壊す範囲が大きすぎる。
- **成功ページをやめて 302 を返す** — 却下しない（将来やってよい）が、これだけでは足りない。
  302 の Location に `javascript:` を入れてもブラウザは追わないので直接の実行は防げるが、
  「認可コードを任意の URL へ渡す」オープンリダイレクトは残る。scheme の判定は結局必要。
- **成功ページ側のエスケープだけで足りると見なす** — 却下。エスケープは
  「script 要素から抜け出せない」ことしか保証しない。`window.location.href` に入れた値は
  **文字列として正しくても実行される**ので、埋め込み方ではなく値そのものを見る必要がある。

## Consequences

- `javascript:` を redirect_uri に持つクライアントは登録できず、登録済みでも認可されない
- 認可要求側にも同じ判定を置くので、古い登録が残っていても成功ページの自動遷移は守られる
- 許可リストではなく拒否リストなので、新しい「移動で実行される」scheme が現れたら足す必要がある

## Evidence

判定の固定は `oauth_server_test.go` の `TestOAuthRejectsScriptCapableRedirectURI`。
`javascript:` / `JavaScript:`（大小違い）/ `data:` / `vbscript:` / `blob:` / `about:blank` を
登録でも認可でも 400 で拒み、`http://localhost/callback` /
`http://127.0.0.1:8080/cb` / `https://client.example/callback` / `myapp://callback` /
`com.example.app:/oauth2redirect` は通ることを表で確かめる。

## Related tests

- `src/server/gkill/mcp/oauth_server_test.go` の `TestOAuthRejectsScriptCapableRedirectURI`
  （登録で拒む / 認可要求で拒む / http・https・カスタム scheme は通る の3本）
- `src/server/gkill/mcp/oauth_html_test.go` の `TestRenderSuccessPageEscapesRedirectURLForScriptContext`
  （埋め込み側のエスケープ。この ADR の判定と合わせて2枚の防御になる）
