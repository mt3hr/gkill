# ADR-0637: MCP の書き込みの端末欄は、経路によらず "mcp" にする

| | |
|---|---|
| Status | Accepted |
| Date | 2026-10-19 |
| Sources | 利用者の判断（作成・更新の欄の取り違えの調査で、MCP の書き込みの端末欄が経路によって "mcp" とサーバの端末名に割れていると分かり、"mcp" に揃えると決めた） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/mcp/write_handlers.go` / `src/server/gkill/mcp/upload_files.go` / `src/server/gkill/usecase/cascade_delete.go` / `src/server/gkill/api/gkill_server_api/handle_upload_files.go` / `src/server/gkill/api/gkill_server_api/handle_submit_kftl_text.go` / `src/server/gkill/api/kftl/kftl_statement.go` |

## Context

MCP の書き込みは、記録の作成・更新の端末欄（`create_device` / `update_device`）に定数 `WriteDevice = "mcp"` を入れていた。
ただしそれは MCP が版を組み立てて送る経路（`gkill_add_*` / `gkill_update_*` / 連鎖しない削除 / 復元）だけで、
gkill 側で版を作る3経路は端末欄を gkill が決めていた。

- 記録の連鎖削除（`/api/delete_kyou_cascade`、ADR-0636）: `update_device` はセッションの端末名（サーバの端末名）
- ファイルのアップロード（`/api/upload_files`、ADR-0635）: `create_device` / `update_device` はサーバの端末名
- KFTL の送信（`/api/submit_kftl_text`）: 同じくサーバの端末名

アプリ欄はどの経路も `gkill_mcp_*`（KFTL だけはメモ帳と同じ `gkill_kftl`）で、端末欄だけが同じ MCP の書き込みの中で割れていた。
同じ `gkill_delete_kyou` でも、記録本体（既定で連鎖）ならサーバの端末名、タグ・テキスト・通知なら "mcp" になり、
連鎖削除した記録を復元するとその版は "mcp" になる。

## Decision

- MCP の書き込みの端末欄は、経路によらず `WriteDevice`（"mcp"）にする。ADR-0636 の「`update_device` はセッションから決める」は、
  要求に端末名が無いときの既定として残る（ADR-0636 全体は置き換えない）。
- gkill 側で版を作る3経路の要求に、省略可の端末名の欄を足す。空なら従来どおりサーバの端末名（Web の画面・Wear は送らない）。
  - `DeleteKyouCascadeRequest.UpdateDevice`（`update_device`）
  - `UploadFilesRequest.CreateDevice`（`create_device`）
  - `SubmitKFTLTextRequest.CreateDevice`（`create_device`）
- gkill はこの値を**記録の作成・更新の端末欄にだけ**使う。temp rep のキー（`txID` / `userID` / `device`）、
  書き込み先の rep や設定の引き当て、KFTL の冪等キーの名前空間には、従来どおりセッションの端末名を使う。
  KFTL は `KFTLStatement.RecordDevice` → 行の文脈の `RecordDevice` を通し、作成・更新の端末欄を書く箇所だけがそれを読む。
- KFTL の冪等キーの指紋には、送られていれば `create_device` も入れる（本文・`create_app` と同じ扱い。空なら従来と同じ値）。
- KFTL のアプリ欄は今までどおり `gkill_kftl`。MCP 由来かは端末欄で分かる。

## Rejected alternatives

- **今のまま（経路によって割れたまま）にし、資料に書くだけ** — 変更は要らないが、同じツールの同じ操作で端末欄が変わり、
  連鎖削除と復元を往復すると版ごとに値が入れ替わる。端末欄で MCP 由来の記録を見分けることもできない。
- **端末欄はサーバの端末名に揃え、MCP 由来かはアプリ欄だけで表す（`WriteDevice` をやめる）** — 「端末欄 = 書いた機械」として
  筋は通るが、既存の MCP の記録（"mcp"）とこれからの記録で値が割れる。KFTL はアプリ欄がメモ帳と同じなので、MCP 由来を見分ける欄が無くなる。
- **gkill が端末欄を常にセッションから刻み直す（クライアントの値を受けない）** — Wear の打刻終了のような書き戻しの漏れも同時に塞げるが、
  "mcp" を残す手段が無くなる（MCP はサーバのセッションで書く）。既存の `add_*` / `update_*` もクライアントの値を受けているので、3経路だけ別の規則にもなる。
- **要求の端末名を `device` として全体に使う（temp rep のキーにも）** — 1か所の差し替えで済むが、連鎖削除の tx を積む側と確定する側で
  キーが食い違ったときに静かに何も確定しなくなり、KFTL の冪等キーの名前空間や設定の引き当てまで変わる。記録の欄だけに効かせる。

## Consequences

- MCP から書いた記録は、どの経路でも作成・更新の端末欄が "mcp" になる。この変更より前に3経路で書いた記録はサーバの端末名のまま残る
  （既存データは書き換えない）。
- MCP のゴールデンが変わる: 連鎖削除・アップロード・KFTL の上流要求に端末名の欄が増え、アップロードの応答の端末欄と、
  `gkill_get_mcp_help` の `kftl` の Provenance の段落が変わる。tools/list と `schema_revision` は変わらない。
- 3つの要求の欄は Web の画面からは送らない（TS の要求クラスにも持たせない。既存の `create_app` と同じ扱い）。

## Evidence

- golden を再生成した差分（テスト由来）: 10 ファイル・106 行。中身は連鎖削除の上流要求の `update_device`、アップロードの上流要求の
  `create_device` と応答の端末欄、KFTL の上流要求の `create_device`、`help: kftl` の本文だけ。プラグイン本文の並列取得で上流要求の並びが
  入れ替わっただけの行（比較は順序を問わない）は元の行に戻した。
- 実測なし（挙動の変更で、性能への影響は無い）。

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_submit_kftl_text_test.go` の `TestHandleSubmitKFTLText_CreateDevice`
- `src/server/gkill/api/gkill_server_api/gkill_server_api_test.go` の `TestHandleUploadFiles_CreateDevice`
- `src/server/gkill/api/gkill_server_api/handle_delete_kyou_cascade_test.go` の `TestHandleDeleteKyouCascade_UpdateDeviceStampsEveryDeletedVersion` / `TestHandleDeleteKyouCascade_KeepsCreateFieldsAndStampsUpdateFields`
- `src/server/gkill/api/gkill_server_api/kftl_idempotency_test.go` の `TestKFTLSubmissionFingerprint`
- `src/server/gkill/mcp/golden_test.go`（`delete: *` / `upload files: *` / `kftl: *` / `help: kftl`）
