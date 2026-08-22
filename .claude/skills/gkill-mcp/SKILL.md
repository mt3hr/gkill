---
name: gkill-mcp
description: "gkill の MCP サーバ（src/mcp/、read/write/readwrite の3種、stdio + HTTP OAuth 2.1）の約束。1リクエスト文脈は不変の requestContext を引数で流し server.current* に書かない、プラグイン本文は gkill_get_kyous の include_plugin_content でまとめて埋め込み同一プラグインへ並列に投げない、IDF ファイルの3経路（file_path / file_url / base64）とトランスポート種別ゲート、環境変数一覧を扱う。src/mcp/ を編集するとき、MCP ツールを追加・変更するとき必読。「他人のセッションに紐づく file_url が発行される」「ErrPluginBusy になる」の調査でも必読。"
---

# MCP サーバの不変条件

対象: `src/mcp/**`

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

### MCP Server — `src/mcp/`

3つのMCPサーバーを提供。いずれもstdio (ローカル) / HTTP (OAuth 2.1認証付きリモート) の2モード対応。

| Server | Tools | stdio | HTTP | Port |
|---|---|---|---|---|
| Read | 9 (get_kyous, get_mi_board_list, get_all_tag_names, get_all_rep_names, get_gps_log, get_application_config, get_idf_file, get_idf_file_path + plugin 1) | `npm run mcp:gkill-read` | `npm run mcp:gkill-read-http` | 8808 |
| Write | 24 (9 add + 1 submit_kftl + 1 delete + 9 update + 3 read convenience + plugin 1) | `npm run mcp:gkill-write` | `npm run mcp:gkill-write-http` | 8809 |
| ReadWrite | 29 (read 8 + write 20 + plugin 1) | `npm run mcp:gkill-readwrite` | `npm run mcp:gkill-readwrite-http` | 8810 |

**Plugin tools** — 3サーバ共通は `lib/plugin-tools.mjs` の `gkill_get_plugin_list` 1つだけ（読み取り専用。`post_plugin_config` は公開しない）。プラグインKyouの本文は gkill に保存されておらず、`get_kyous` が返すのはメタデータと `rep_name`/`kyou_id` を持つ `payload.kind="plugin"` だけ。本文は **`gkill_get_kyous` に `include_plugin_content:true` を渡して同じレスポンスに埋め込ませる**（1件ずつ取る `gkill_get_plugin_content` ツールは廃止。AIの往復が N+1 回になり非効率だったため）。インライン化は `plugin-tools.mjs` の `inlinePluginContents` が担当し、`/api/get_plugin_content_html` を rep ごとに直列・rep 間は並列（既定4）で叩き、`lib/html-text.mjs` でHTML→テキスト変換して `payload.content_text` に入れる。各ペイロードには `content_status`（`ok`/`truncated`/`skipped`/`error`）が付き、`ok` 以外なら本文は不完全。上限は1件4000文字（`plugin_content_max_text_length` で最大200000まで）・合計200000文字・20件・30秒。`plugin_content_format` で `text`（既定）/`html`/`both` を選べる。 同一プラグインへ並列に投げても速くならない理由は [ADR-0051](../../../documents/adr/0051-mcp-inline-plugin-content.md)。

> **同一プラグインへ並列に投げてはいけない。** プラグインの stdio は1本しかなく呼び出しは直列化されるので、同時発行しても速くならず順番待ちが伸びるだけ。待ちきれなかったぶんは `ErrPluginBusy` になる。インライン取得が実行中のリクエストを abort しないのも同じ配慮（abort 自体は現在プロセスに影響しないが、MCPサーバは古い gkill にも接続しうる）。デッドラインは「新しいリクエストを始めない」だけで実現している。Write専用サーバには `gkill_get_kyous` が無いため、プラグイン本文を読むには ReadWrite サーバを使う。
>
> （2026-08-06以前の gkill は30秒デッドラインを排他ロックの**前**に張っていたため、同時発行するとロック待ちで期限を食い潰し、期限切れ時の `Process.Kill()` でプラグインプロセスが回収されていた。現在は期限をスロット取得後に張るのでこの誤射はない。）

**Environment variables:** `GKILL_BASE_URL`, `GKILL_USER`, `GKILL_PASSWORD_SHA256`, `MCP_LOG` (default `info`), `GKILL_MCP_MAX_FILE_BYTES` (default 8MB — cap for `get_idf_file` base64), `GKILL_MCP_FILE_LINK_TTL_MS` (default 1h — HTTP-mode file URL token TTL), and for HTTP mode: `MCP_TRANSPORT`, `MCP_PORT`, `MCP_OAUTH_ISSUER`. Also read: `GKILL_PASSWORD` (plaintext alternative to the SHA256), `GKILL_SESSION_ID`, `GKILL_INSECURE`, `GKILL_LOCALE`, `GKILL_FETCH_TIMEOUT_MS`, `GKILL_HOME`.

**IDF file access for AI clients** (3 paths, preferred order): (1) `file_path` — `get_kyous` includes the absolute local path in IDF payloads for **stdio** clients (same machine); read it directly. (2) `file_url` / `file_url_full` — for **HTTP** clients, `get_kyous` includes a per-file, expiring, unguessable public URL served by the MCP server's own `GET /files/{token}` route (`FileLinkStore`, `lib/file-link-store.mjs`); fetch with no auth, any size (images: `file_url` is a thumbnail, `file_url_full` the original). (3) `gkill_get_idf_file` — base64 fallback, size-capped. Local paths never go to remote clients; the transport type (stdio vs HTTP) is the gate. gkill itself stays private — only the MCP server's public surface is used.

**HTTPモードの1リクエスト文脈は `server.current*` 共有フィールドに書かず、不変の `requestContext={sessionId,userId,remoteAddr}` を `handlePayload→handleMessage→handleToolCall/buildToolResult` へ引数で流す**（2026-08-21、監査 C-02）。以前は `_lastTokenUserId` と `server.currentSessionId/currentUserId/currentRemoteAddr` に書いて await をまたいで読んでいたため、並行リクエストで別要求の user/session が混線し、他人のセッションに紐づく file-link URL 発行や書き込みレコードの作成者誤記が起きた。`mcp-server-base.mjs` の入口で `requestContext ?? Object.freeze({...this.current*})` にフォールバックするので stdio と既存の直接呼び出しテストは無改修。**http-transport 側から `server.current*` への書き込みを復活させないこと**（渡し忘れは sessionId=null→gkill側401で顕在化し、静かな混線には戻らない）。OAuth は S256 必須・未登録 client_id は認可拒否（`oauth-server.mjs` の `_validateAuthorizeParams`）、公開ファイル配信は nosniff + CSP sandbox（Go 側 `withUserContentSecurityHeaders` のミラー）、`oauth-store.mjs` の保存は temp+rename の 0600。守るテストは `src/mcp/__tests__/http-transport.test.mjs`（Bearer 401 = C-01 回帰・並行分離 = C-02 回帰・M-06）。 却下案（AsyncLocalStorage 等）と渡し忘れが顕在化する理由は [ADR-0050](../../../documents/adr/0050-mcp-request-context-immutable.md)。

## 関連スキル

- [gkill-plugin](../gkill-plugin/SKILL.md) — プラグインの stdio 直列化（並列に投げても速くならない理由）
- [gkill-find-query](../gkill-find-query/SKILL.md) — 旧形式JSONキーの移行（`LEGACY_USE_FLAG_KEYS`）
- [gkill-go-backend](../gkill-go-backend/SKILL.md) — サーバ側の認可・セキュリティヘッダ

## 詳しい設計と却下案（ADR）

- [ADR-0050 リクエスト文脈は不変で引数渡し](../../../documents/adr/0050-mcp-request-context-immutable.md)
- [ADR-0051 プラグイン本文はインライン埋め込み](../../../documents/adr/0051-mcp-inline-plugin-content.md)
- [ADR-0052 カーソルは期間の端を進める](../../../documents/adr/0052-mcp-cursor-pushes-period-end.md)
