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
| Read | 10 (get_kyous, get_mi_board_list, get_all_tag_names, get_all_rep_names, get_gps_log, get_application_config, get_rep_infos, get_idf_file, get_kyou_history + plugin 1) | `npm run mcp:gkill-read` | `npm run mcp:gkill-read-http` | 8808 |
| Write | 26 (9 add + 1 submit_kftl + 1 delete + 1 restore + 9 update + 4 read convenience + plugin 1) | `npm run mcp:gkill-write` | `npm run mcp:gkill-write-http` | 8809 |
| ReadWrite | 31 (read 9 + write 21 + plugin 1) | `npm run mcp:gkill-readwrite` | `npm run mcp:gkill-readwrite-http` | 8810 |

**Plugin tools** — 3サーバ共通は `lib/plugin-tools.mjs` の `gkill_get_plugin_list` 1つだけ（読み取り専用。`post_plugin_config` は公開しない）。プラグインKyouの本文は gkill に保存されておらず、`get_kyous` が返すのはメタデータと `rep_name`/`kyou_id` を持つ `payload.kind="plugin"` だけ。本文は **`gkill_get_kyous` に `include_plugin_content:true` を渡して同じレスポンスに埋め込ませる**（1件ずつ取る `gkill_get_plugin_content` ツールは廃止。AIの往復が N+1 回になり非効率だったため）。インライン化は `plugin-tools.mjs` の `inlinePluginContents` が担当し、`/api/get_plugin_content_html` を rep ごとに直列・rep 間は並列（既定4）で叩き、`lib/html-text.mjs` でHTML→テキスト変換して `payload.content_text` に入れる。各ペイロードには `content_status`（`ok`/`truncated`/`skipped`/`error`）が付き、`ok` 以外なら本文は不完全。上限は1件4000文字（`plugin_content_max_text_length` で最大200000まで）・合計200000文字・20件・30秒。`plugin_content_format` で `text`（既定）/`html`/`both` を選べる。 同一プラグインへ並列に投げても速くならない理由は [ADR-0051](../../../documents/adr/0051-mcp-inline-plugin-content.md)。

> **同一プラグインへ並列に投げてはいけない。** プラグインの stdio は1本しかなく呼び出しは直列化されるので、同時発行しても速くならず順番待ちが伸びるだけ。待ちきれなかったぶんは `ErrPluginBusy` になる。インライン取得が実行中のリクエストを abort しないのも同じ配慮（abort 自体は現在プロセスに影響しないが、MCPサーバは古い gkill にも接続しうる）。デッドラインは「新しいリクエストを始めない」だけで実現している。Write専用サーバには `gkill_get_kyous` が無いため、プラグイン本文を読むには ReadWrite サーバを使う。
>
> （2026-08-06以前の gkill は30秒デッドラインを排他ロックの**前**に張っていたため、同時発行するとロック待ちで期限を食い潰し、期限切れ時の `Process.Kill()` でプラグインプロセスが回収されていた。現在は期限をスロット取得後に張るのでこの誤射はない。）

**削除と過去版は `gkill_get_kyous` からは絶対に見えない。** gkill は追記型で、update は新しい版の追加、delete は `is_deleted=true` の版の追加。検索は最新版かつ未削除しか返さず、**`query.only_latest_data:false` は受理されるが無視される**（旧版を落としているゲートは `handle_get_kyous_mcp.go` の2箇所ではなく最低6箇所あり、うち `find_filter.go` の `replaceLatestKyouInfos` はフラグで分岐しないので、既知の2箇所を外しても件数は1件も変わらない。`dao/reps/repositories.go` がバケットキーを `ID+UpdateTime` に変える一方 `filterTagsKyous` は素のIDで引くため、開けるとタグ絞り込みが全部0件になる）。**代わりに使うもの**: 1件の全版は `gkill_get_kyou_history`（`data_type` 必須。型非依存の `/api/get_kyou` は `UnWrap()` でキャッシュを丸ごとバイパスするので使わないこと）、削除の取り消しは `gkill_restore_kyou`、削除済みの列挙は `query.include_deleted_data:true`（かつて併存した `query.is_deleted` は一度も実装されず、2026-08-24 に語彙ごと削除した。送ると未知キーとしてエラーになる）。**`update_time` は1秒解像度**なので、同じ秒の中で delete→restore すると新しい版が最新と見なされず黙って何も起きない。`write-handlers.mjs` の `nextUpdateTime` が必ず1秒以上進める。 却下案と6箇所のゲートの棚卸しは [ADR-0054](../../../documents/adr/0054-mcp-version-history-is-a-dedicated-tool.md)、削除の旗の意味論は [ADR-0071](../../../documents/adr/0071-deleted-data-opens-only-with-include-deleted-data.md)。

**ディスパッチと要約の正本は `lib/read-handlers.mjs` と `lib/write-handlers.mjs` の2本だけ。** サーバファイル（`gkill-*-server.mjs`）に case を書き足さないこと。以前は read が read/readwrite へ、write が write/readwrite へ逐語コピーされており、readwrite 側だけが廃止済み `callRead` を呼び続けて、IDF ファイルパス取得のツールが静かに TypeError で落ち続けた（そのツールは 2026-08-24 に削除済み）。`buildToolResult` と3段の要約（plugin→read→write）も `lib/mcp-server-base.mjs` に1本化してある（write サーバの版だけ file-link 注入と IDF 画像ブロックを欠いた劣化コピーだった）。**ただし `const TOOLS = [...]` の配列は各サーバファイルに残すこと** ―― `src/tools/verify_docs.mjs` がサーバファイル中のこのリテラルを走査してツール数を数えており、lib へ移すとツール数が黙って0になる。削除・復活・履歴が共有する型別エンドポイント表は `lib/constants.mjs` の `ENTITY_TARGETS` 1箇所（以前は enum・`DELETE_DATA_TYPES`・対応表2つの4箇所に散っていた）。

**Environment variables:** `GKILL_BASE_URL`, `GKILL_USER`, `GKILL_PASSWORD_SHA256`, `MCP_LOG` (default `info`), `GKILL_MCP_MAX_FILE_BYTES` (default 8MB — cap for `get_idf_file` base64), `GKILL_MCP_FILE_LINK_TTL_MS` (default 1h — HTTP-mode file URL token TTL), and for HTTP mode: `MCP_TRANSPORT`, `MCP_PORT`, `MCP_OAUTH_ISSUER`. Also read: `GKILL_PASSWORD` (plaintext alternative to the SHA256), `GKILL_SESSION_ID`, `GKILL_INSECURE`, `GKILL_LOCALE`, `GKILL_FETCH_TIMEOUT_MS`, `GKILL_HOME`.

**ツールスキーマはクライアントのセッション寿命で固定される。** クライアントは接続時に `tools/list` を1回取るだけで、サーバを再起動しても**生きているセッションには新しいスキーマが届かない**（2026-08-24 の再監査で、削除済みの `gkill_get_idf_file_path` を配り続ける live コネクタから実測）。そのため後から足したトップレベル引数は**正規JSON文字列**として届く（`"true"` / `'["nlog"]'`）。救済は `lib/normalization.mjs` の `reviveStaleSchemaArgs` で、`KYOUS_` / `GPS_` / `APP_CONFIG_` / `IDF_STALE_SCHEMA_ARG_KINDS` の4表に載っている引数だけが型を復元される。**引数をスキーマへ足したら必ず同じ表へも足すこと。** 忘れると boolean / number / 配列の新引数は既存の全クライアントから型エラーになり、「新しいセッションでだけ動く」状態で出荷される（`is_video` が実際にそうなった）。string 型（`group_by` / `cursor` / `thumb`）は素通しするので表に載せなくてよい。**同じ理由で、ツールを消したら `lib/constants.mjs` の `REMOVED_TOOL_HINTS` へ1行足す** —— 旧セッションはそのツールを一覧に持ったまま呼び続けるので、`Unknown tool` だけ返すと代替が分からない。

**IDF file access**: 経路は「優先順位」ではなく**用途で分かれる**（2026-08-24 実測、[ADR-0055](../../../documents/adr/0055-idf-file-reaches-ai-through-payload.md)）。stdio クライアントには `get_kyous` が `file_path`（絶対パス）を載せるので、それを直接読むのが最良（base64 もサイズ上限も無い）。HTTP クライアントには代わりに `file_url` / `file_url_full`（`FileLinkStore` が発行する1ファイル束縛・期限付き・推測不能な公開URL。`GET /files/{token}` が Bearer 無しで配信し、画像は `file_url` がサムネ・`file_url_full` が原寸）が載るが、**MCP はそのURLを代わりに取りに行かない**し、クライアントが自前の機能で取れても会話の外側のバイトになるだけなので、これは人間へ渡すリンクと考える。**HTTP 接続の AI が画像を「見る」唯一の経路は `gkill_get_idf_file`**（`buildToolResult` が `type:"image"` の content ブロックを作る唯一の入力）で、画像生成ツールへ参照画像として渡す場合もこれ。上限（`GKILL_MCP_MAX_FILE_BYTES`、既定8MB）に当たる画像・動画は `thumb="WxH"`（一辺最大1024 = Go 側 `thumbFileServer.maxSize`。動画は `is_video:true` 併用）で縮小取得する。上限超えの `thumb` を渡すと Go は**黙って原本を返す**ので MCP 側で先に弾いている。**「URL を優先しろ」と書き戻さないこと** —— 実使用の ChatGPT はその誤誘導を無視して正しく base64 を呼んでいた。Local paths never go to remote clients; the transport type (stdio vs HTTP) is the gate. gkill itself stays private.

**HTTPモードの1リクエスト文脈は `server.current*` 共有フィールドに書かず、不変の `requestContext={sessionId,userId,remoteAddr}` を `handlePayload→handleMessage→handleToolCall/buildToolResult` へ引数で流す**（2026-08-21、監査 C-02）。以前は `_lastTokenUserId` と `server.currentSessionId/currentUserId/currentRemoteAddr` に書いて await をまたいで読んでいたため、並行リクエストで別要求の user/session が混線し、他人のセッションに紐づく file-link URL 発行や書き込みレコードの作成者誤記が起きた。`mcp-server-base.mjs` の入口で `requestContext ?? Object.freeze({...this.current*})` にフォールバックするので stdio と既存の直接呼び出しテストは無改修。**http-transport 側から `server.current*` への書き込みを復活させないこと**（渡し忘れは sessionId=null→gkill側401で顕在化し、静かな混線には戻らない）。OAuth は S256 必須・未登録 client_id は認可拒否（`oauth-server.mjs` の `_validateAuthorizeParams`）、公開ファイル配信は nosniff + CSP sandbox（Go 側 `withUserContentSecurityHeaders` のミラー）、`oauth-store.mjs` の保存は temp+rename の 0600。守るテストは `src/mcp/__tests__/http-transport.test.mjs`（Bearer 401 = C-01 回帰・並行分離 = C-02 回帰・M-06）。 却下案（AsyncLocalStorage 等）と渡し忘れが顕在化する理由は [ADR-0050](../../../documents/adr/0050-mcp-request-context-immutable.md)。

## 関連スキル

- [gkill-plugin](../gkill-plugin/SKILL.md) — プラグインの stdio 直列化（並列に投げても速くならない理由）
- [gkill-find-query](../gkill-find-query/SKILL.md) — 旧形式JSONキーの移行（`LEGACY_USE_FLAG_KEYS`）
- [gkill-go-backend](../gkill-go-backend/SKILL.md) — サーバ側の認可・セキュリティヘッダ

## 詳しい設計と却下案（ADR）

- [ADR-0050 リクエスト文脈は不変で引数渡し](../../../documents/adr/0050-mcp-request-context-immutable.md)
- [ADR-0051 プラグイン本文はインライン埋め込み](../../../documents/adr/0051-mcp-inline-plugin-content.md)
- [ADR-0052 カーソルは期間の端を進める](../../../documents/adr/0052-mcp-cursor-pushes-period-end.md)（Superseded）
- [ADR-0053 複合カーソルと厳密上限・count_only/group_by](../../../documents/adr/0053-mcp-composite-cursor-strict-limits.md)
- [ADR-0054 版履歴と削除の取り消しは専用ツール](../../../documents/adr/0054-mcp-version-history-is-a-dedicated-tool.md)
- [ADR-0071 削除済みは include_deleted_data だけが開ける](../../../documents/adr/0071-deleted-data-opens-only-with-include-deleted-data.md)
