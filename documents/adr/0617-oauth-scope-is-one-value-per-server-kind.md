# ADR-0617: OAuth の scope はサーバ種別ごとに1値

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-30 |
| Sources | 2026-08-30 の MCP レビュー P0（scope の矛盾広告と非照合受理）。[gkill-mcp](../../.claude/skills/gkill-mcp/SKILL.md) の節「OAuth の scope はサーバ種別ごとに1値で、正本は各エントリスクリプトの `START_SPEC` 1箇所。」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/mcp/gkill-read-server.mjs` / `src/mcp/gkill-write-server.mjs` / `src/mcp/gkill-readwrite-server.mjs`（`START_SPEC`）/ `src/mcp/lib/mcp-server-bootstrap.mjs`（`startMcpServer`）/ `src/mcp/lib/oauth-server.mjs`（`_validateAuthorizeParams` / `_handleAuthorizationCodeGrant` / `_handleRefreshTokenGrant`）/ `src/mcp/lib/http-transport.mjs`（Bearer 受理の scope 照合） |

## Context

MCP の HTTP モードは read / write / readwrite の3サーバが別ポート・別 OAuth 状態で動く。
scope の値は3箇所（protected-resource metadata・authorization-server metadata・受理側）に
現れるが、それぞれが別の場所で決まっていた。

実測（2026-08-30 レビュー）では ReadWrite サーバで
protected-resource が `gkill:readwrite`、authorization-server が `gkill:read` を広告する
**矛盾広告**になっており、さらに Bearer 受理は**トークンの存在しか見ていなかった**。
つまり scope 修正前の ReadWrite サーバが発行した `gkill:read` のトークンでも
書き込みツールが呼べた。利用者が「読み取りを許可した」つもりのトークンが
書き込み可能というのは、同意画面が何を表示していても同意の意味を失う。

## Decision

**scope はサーバ種別ごとに1値**（read=`gkill:read` / write=`gkill:write` /
readwrite=`gkill:readwrite`）とし、**正本は各エントリスクリプトの `START_SPEC` 1箇所**に置く。
bootstrap（`startMcpServer`）がその1値を `OAuthServer` へ渡し、
metadata の `scopes_supported`・authorize の既定値と不一致拒否（400）・
発行するコード / トークンへ刻む値・refresh の照合・Bearer 受理の照合が
すべて同じ1値から生成される。

- 認可要求の scope はこの1値との**厳密一致のみ**通す（省略は既定=この1値。
  空白区切りの複数要求も全要素一致のみ）
- 旧 scope が刻まれた認可コード / refresh token は `invalid_scope` で拒否し、
  refresh token は**失効させる**（再試行しても同じ失敗を繰り返すだけ）
- Bearer 受理の不一致は 401 ではなく **403 `insufficient_scope`**
  （トークンは本物だが権限が足りない = 再認可が要る、をクライアントへ伝える）
- `HttpTransport` は `oauthServer.scope` を読む。二重指定の不一致は起動時に throw

## Rejected alternatives

- **`scopes_supported` に3値を載せてクライアントに選ばせる** — サーバは自分の種別しか
  発行できないので、選べるように見せるだけ不正確。選ばれた「別の scope」を拒否するなら、
  最初から1値だけ広告するほうが正直で、コネクタ実装の分岐も減る
- **metadata の矛盾だけ直し、受理は従来どおりトークンの存在だけ見る** — 事故の本体は
  「read トークンで書き込めた」側。広告を直しても受理が照合しなければ何も守られない
- **既存トークンを移行救済で通す（旧 scope を暗黙に昇格）** — 「読み取りで認可した
  トークンが書き込める」状態を意図的に延命することになり、修正の目的と正面衝突する。
  失うものは1回の再認可の手間だけ
- **scope をテストごとに自前定義し続ける** — 修正そのものは正しくても、エントリスクリプトの
  宣言値を誰も読まないままでは同じ取り違えが再発する（実際に出荷まで漏れた）。
  `START_SPEC` を export して宣言値そのものをテストが固定する

## Consequences

- 2026-08-30 より前に発行されたトークン・別種サーバのトークンは 403 / `invalid_scope` になり、
  **クライアントの再認可が1回必要**（mcp-setup-guide.md のトラブルシューティングに記載）
- 認可ログイン画面は scope の意味（読み取り専用 / 書き込み / 読み書き）と書き込み可否の
  強調表示を出す。書き込み権限の同意が初めて見えるようになった
- scope 拒否は `token_scope_rejected`（token_scope / required_scope 付き）として
  アクセスログに残り、運用者が「再認可が必要なクライアントがいる」ことに気づける
- サーバ種別を増やす場合は `START_SPEC` の1行と `SCOPE_DESCRIPTIONS`（同意画面の説明）を
  足すだけでよい

## Evidence

- 2026-08-30 レビュー実測: ReadWrite サーバの protected-resource metadata が
  `gkill:readwrite`、authorization-server metadata が `gkill:read` を返す矛盾広告。
  Bearer 受理は scope 非照合で、`gkill:read` トークンから書き込みツールが呼べた
- 修正コミット 5b8da91b（認可と受理の両方で不一致拒否）、65c58bce 後の
  テスト補強で「3サーバの宣言値が1本も検証されていない」ことが判明し `START_SPEC` を導入

## Related tests

- `src/mcp/__tests__/start-spec.test.mjs`
  - 3サーバの `START_SPEC` 宣言値（scope / 既定ポート / file-link 可否）と相互の重複禁止、
    bootstrap が `spec.scope` を OAuthServer へ渡す配線
- `src/mcp/__tests__/oauth-server.test.mjs`
  - `OAuthServer — scope enforcement (P0)`（authorize GET / POST の不一致 400 と code 非発行、
    旧 scope の code / refresh token の `invalid_scope` 拒否と失効、省略時の既定）
  - `OAuthServer — consent display on the login page`（3 scope の表示・書き込み強調・エスケープ）
- `src/mcp/__tests__/http-transport.test.mjs`
  - `OAuth scope boundary (P0)`（両 metadata の一致広告、不一致トークンの 403 と
    `token_scope_rejected` ログ、一致トークンの通過、生成時の不一致 throw / scope 欠落 throw）
