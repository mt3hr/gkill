# ADR-0050: MCP HTTPモードの1リクエスト文脈は不変オブジェクトを引数で流す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-21 |
| Sources | `bb364253`（監査 C-02） / `.claude/skills/gkill-mcp/SKILL.md`「HTTPモードの1リクエスト文脈は server.current* 共有フィールドに書かず」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/mcp/lib/mcp-server-base.mjs` |

## Context

MCP サーバは HTTP モードで**複数のリクエストを並行に処理する**。ところが1リクエストの文脈（session / user / remoteAddr）を `server.currentSessionId` / `currentUserId` / `currentRemoteAddr` と `_lastTokenUserId` という**インスタンスの共有フィールドに書いて、`await` をまたいで読んでいた**。

並行リクエストで user / session が混線し、

- **他人のセッションに紐づく file-link URL が発行される**
- 書き込みレコードの**作成者が誤って記録される**

stdio モードは1リクエストずつなので露見しない。**HTTPモードでだけ、しかも並行時にだけ**起きる。

## Decision

不変の `requestContext = {sessionId, userId, remoteAddr}` を作り、`handlePayload` → `handleMessage` → `handleToolCall` / `buildToolResult` へ**引数で流す**。

`mcp-server-base.mjs` の入口で `requestContext ?? Object.freeze({...this.current*})` にフォールバックするので、stdio と既存の直接呼び出しテストは無改修。

## Rejected alternatives

- **共有フィールドに書いたまま、`await` をまたがないように書き換える** — `await` をまたがない保証を人間が維持することになる。1箇所でも `await` が入れば同じ混線が戻り、しかも**テストでは（並行しないので）緑のまま**。

- **リクエストごとにサーバインスタンスを作る** — OAuth のトークンストアや file-link ストアがインスタンスに載っている。作り直すと状態が失われる。

- **AsyncLocalStorage を使う** — 引数で流すのと効果は同じだが、渡し忘れが**実行時まで分からない**。引数にすると渡し忘れは `sessionId=null` → gkill 側401 で顕在化する（静かな混線には戻らない）。

## Consequences

**`http-transport` 側から `server.current*` への書き込みを復活させないこと。** フォールバックが残っているので、書き戻しても動いてしまう。

同じ監査で OAuth も固めた: S256 必須・未登録 `client_id` は認可拒否（`oauth-server.mjs` の `_validateAuthorizeParams`）、公開ファイル配信は nosniff ＋ CSP sandbox（Go 側 `withUserContentSecurityHeaders` のミラー）、`oauth-store.mjs` の保存は temp+rename の 0600。

## Evidence

実測なし — 脅威モデルからの判断（外部監査 C-02 の指摘）。

## Related tests

- `src/mcp/__tests__/http-transport.test.mjs`（Bearer 401 ＝ C-01 回帰 / **並行分離 ＝ C-02 回帰** / M-06）
- `src/mcp/__tests__/oauth-server.test.mjs`
- `src/mcp/__tests__/oauth-store.test.mjs`
