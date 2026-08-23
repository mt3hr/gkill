# ADR-0055: IDF ファイルはペイロードの file_url / file_path で渡し、到達できないツールは一覧に載せない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | `2e478be5` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/mcp/lib/payload.mjs`（`applyFileLinks`）/ `src/mcp/lib/http-transport.mjs`（`GET /files/{token}`） |

## Context

MCP から IDF（ファイル・画像・動画）の実体へ届く経路は3つあった。

| 経路 | 効く条件 |
|---|---|
| `file_url` / `file_url_full` | HTTP 接続。`gkill_get_kyous` の IDF ペイロードへ自動注入 |
| `file_path` | stdio 接続 **かつ** gkill への接続元が loopback |
| `gkill_get_idf_file`（base64） | どこでも。`GKILL_MCP_MAX_FILE_BYTES`（既定8MB）で打ち切り |

これとは別に `gkill_get_idf_file_path` というツールがあり、rep 名とファイル名から絶対パスを引いていた。
これが**固有の到達性を1つも持っていなかった**。

このツールが動く条件は `isLocalTransport`（MCP 側）と `isLocalRequest`（Go 側）の AND だが、
`gkill_get_kyous` のペイロードに `file_path` が載る条件も同じ2つの AND
（`handle_get_kyous_mcp.go` が同じ `isLocalRequest` を見る）。
片方だけ真になる組み合わせが存在しないので、**このツールは `file_path` の真部分集合**でしかない。

さらに悪いことに、2本の IDF ツールは互いを指していた。
`get_idf_file` はサイズ超過時に「`get_idf_file_path` を使え」と案内するが、
HTTP クライアントではそのツールが必ず失敗する。行き止まりになる。
正解は `file_url_full`（`/files/` 配信・サイズ上限なし）だが、どちらのメッセージも触れていなかった。

## Decision

`gkill_get_idf_file_path`（MCP ツール）と `/api/get_idf_file_path`（Go エンドポイント）を削除する。
IDF の実体へは**ペイロードに載る `file_url` / `file_url_full` / `file_path` で届く**ことを唯一の説明にし、
`gkill_get_idf_file` は「どちらも無いときだけのフォールバック」として残す。
サイズ超過時の案内も `file_url_full` へ向け直す。

`isLocalRequest`（`filter_local_only.go`）は残す。`filterLocalOnly` と
`handle_get_kyous_mcp.go` の `file_path` 可否がこれで決まるため。
判定を固定する8ケースのテストは `filter_local_only_test.go` へ移した。

## Rejected alternatives

- **`gkill_get_idf_file_path` を残して説明だけ直す** — 説明を直しても HTTP クライアントからは
  依然として1回も成功しない。ツール一覧は AI が毎回読む固定コストで、
  到達できないものが載っていること自体が選択を誤らせる
- **`gkill_get_idf_file`（base64）も同時に消す** — 消してよいかはログで決着しなかった。
  `/files/` へのアクセスは0件だったが、`http_request` ログ自体が `a510fbc2`（2026-08-19）で
  入ったばかりで、記録のある2日ぶんはほぼ監査トラフィック。
  「使われていない」ではなく「この2日は誰も画像を取っていない」しか言えない。
  実際に URL 経路が使われていることを実測してから消す
- **`/api/get_idf_file_path`（Go）だけ残す** — 呼び出し元は削除した MCP ツールだけだった。
  Vue クライアントも Wear OS も使っていない（`gkill-api.test.ts` の
  `NON_WEB_CLIENT_ADDRESSES` に「MCPサーバ専用」として明示的に載っていた）
- **`/files/{repName}/{filePath}`（Go）も整理する** — こちらは Vue 本体の画像・動画・Markdown 表示の
  中核で、共有ページの認可もここ。MCP の `file_url` 配信も内部でこれを叩く。触ってはいけない

## Consequences

- **`file_url` のトークンは1時間で失効する**（`GKILL_MCP_FILE_LINK_TTL_MS`）。
  切れたら同じ検索をやり直せば新しい URL が発行される。この回復手順を知らないと
  「画像が取れなくなった」で止まるので、スキーマの説明に書く必要がある
- `file_url` は Bearer 不要で配信する（画像取得は認証ヘッダを付けられないため、
  トークンが唯一の防御線）。この設計は変えていない
- **エラーコードが3つ消える**（`ERR000387` / `ERR000388` / `ERR000389`）。
  採番に穴が空くが連番性を要求する検査は無く、`errCodeMax` は変わらない。
  消すときは `http_status.go` の表・`http_status_test.go` の分布期待値
  （400/403/500 の3つ）・`verify_docs` の件数を同時に直さないと落ちる
- ツール数は Read 11→10 / ReadWrite 32→31。Write は元から非搭載で 26 のまま。
  `resources/manual_src/` の7言語のうち `verify_docs` が見るのは ja だけなので、
  残り6言語のハードコード値は手で直す（`checkManualParity` は h2/h3/table の個数しか比べない）
- **削除しても `lib/payload.mjs` の関数は1つも減らない。** `file_url` 経路が使い続けるため

## Evidence

2026-08-24 の実測:

- `lib/http-transport.mjs` は `isLocalTransport = false` を**無条件**で立てる。
  つまり claude.ai コネクタや ChatGPT のような HTTP クライアントからは、
  `gkill_get_idf_file_path` は gkill へ問い合わせる前に必ず throw していた
- `$GKILL_HOME/logs/gkill_mcp_read_access.log`（812行、2026-08-22〜24）で
  read サーバの `server_start` は4回とも `transport: http`。
  `gkill_get_kyous` は307回呼ばれているのに `gkill_get_idf_file_path` は1回だけ
- Go 側の全参照を走査して、削除後に残る `GetIDFFilePath` / `get_idf_file_path` は0件
- `b68d6bb7` の跡地が示すとおり、このツールは readwrite サーバ（stdio）で
  廃止済み `client.callRead` を呼び続けて**長期間 TypeError で静かに壊れていたのに誰も気付かなかった**

## Related tests

- `src/mcp/__tests__/tool-handlers.test.mjs`
- `src/mcp/__tests__/server.test.mjs`
- `src/mcp/__tests__/file-link.test.mjs`
- `src/server/gkill/api/gkill_server_api/filter_local_only_test.go`
- `src/client/__tests__/unit/api/gkill-api.test.ts`
