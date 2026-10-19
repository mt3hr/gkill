# ADR-0636: 連鎖削除をサーバの1実装にし、MCP は既定で連鎖させる。MCP にリポスト・タスク化・通知・ファイルの記録の作成と編集を足す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-10-18 |
| Sources | 利用者の要求「MCP 側に、連鎖して消すオプションを用意してほしい。挙動は画面と同じで。IDF についても対応してほしい。ReKyou と MiReKyou、通知の MCP 対応もお願いしたい」。壁打ちで決めたこと: 連鎖削除はサーバに1本置き、画面と MCP の両方がそれを使う。`gkill_delete_kyou` は既定で連鎖する。復元は1件ずつのまま |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/usecase/cascade_delete.go` / `src/server/gkill/api/gkill_server_api/handle_delete_kyou_cascade.go` / `src/client/classes/cascade-delete-kyou.ts` / `src/server/gkill/mcp/write_handlers.go` / `src/server/gkill/mcp/write_tools.go` / `src/server/gkill/usecase/idf_kyou.go` / `src/server/gkill/usecase/notification.go` |

## Context

画面の削除は連鎖削除だった。記録を消すと、付いているタグ・テキスト・通知と、その記録を参照するリポスト（ReKyou）・タスク化（MiReKyou）を
再帰的に集め、1つの tx で論理削除する（ADR-0410）。この探索と tx の組み立ては画面の TypeScript（`cascade-delete-kyou.ts`）だけが持っていた。
MCP の `gkill_delete_kyou` は対象1件だけを消すので、付属データと参照が、消えた記録を指したまま残った。

あわせて、MCP からはファイルの記録（idf）を消す・戻す・関連時刻を直すことができず（`EntityTargets` に idf が無い）、
リポスト・タスク化・通知は削除・復元・履歴しかできなかった（作成と編集が無い）。ADR-0635 のアップロードで override を使うと
同じファイルを指す古い記録が残るが、それを MCP から始末できなかった。

## Decision

- 連鎖削除をサーバへ移す。`usecase.DeleteKyouCascade`（`/api/delete_kyou_cascade`）が画面の TypeScript と同じ手順を持つ:
  幅優先・訪問済み集合で循環を止める・深さ上限 32・同じ id は最新版だけ・削除済みの参照元は辿らない・探索が1つでも失敗したら書かない・
  付随データ → 参照元（深い方から）→ 起点の順に1つの txID で積み、`CommitTx` で確定する（全部消えるか何も消えないか）。
  応答は消した全件の `[{id, data_type}]`（エンティティ名）。画面の `cascade_delete_kyou()` はこれを1回呼び、キャッシュを落として行を取り除くだけになる。
- 全件に刻む `update_time` は同じ値で、どれかの現在の版と同じ秒なら1秒進める（1秒解像度で新しい版が最新と見なされない穴。MCP の `nextUpdateTime` と同じ）。
  `update_user` / `update_device` はセッションから決め、`update_app` だけを要求から受ける（画面は `gkill`、MCP は自分のアプリ名）。
- MCP の `gkill_delete_kyou` に `cascade`（既定 true）を足す。記録は `/api/delete_kyou_cascade` を呼び、`deleted[]` を返す。
  タグ・テキスト・通知は連鎖の意味が無いので従来の1件削除。`cascade:false` は従来どおり対象1件だけ。一括では、先の連鎖が消した id を消し直さない。
- 復元は連鎖させない。`deleted[]` を `gkill_restore_kyou` の `targets[]` へそのまま渡せば戻せる形にする。
- `EntityTargets` に idf を足し（`/api/get_idf_kyou` / `/api/update_idf_kyou`）、MCP の書き込みツールを7本足す:
  `gkill_add_rekyou` / `gkill_update_rekyou` / `gkill_add_mirekyou` / `gkill_update_mirekyou` / `gkill_add_notification` /
  `gkill_update_notification` / `gkill_update_idf`。引数は画面の入力欄と同じで、検証と更新は既存の表（`entityFieldSpecs` / `updateTargets`）に載せる。
  作成の3本は `target_id` の実在を `/api/get_kyous_mcp`（`query.ids` + `count_only`）で確かめる（gkill の add は確かめない）。
- 見つかった本体の穴を2つ直す（どちらも画面にも効いていた）:
  - `UpdateIDFKyou` が `ContentPath`（`json:"-"`）の空の記録をキャッシュへ書き戻し、キャッシュ ON では関連時刻を直した・復元したファイルの記録が
    次の UpdateCache まで検索から消えていた。`existIDFKyou.ContentPath` を引き継ぐ（アップロードで直した ADR-0635 の穴と同じ）。
  - 通知の履歴（`/api/get_gkill_notification_histories_by_notification_id`）だけが削除済みの版を落としていて（`GkillRepositories.GetNotificationHistories`）、
    「消したのに履歴の先頭は消えていない版」になり、MCP の削除・復元の判定が狂っていた。タグ・テキストと同じく集約 rep（`NotificationReps`）を通す。

## Rejected alternatives

- **MCP に画面の手順を Go で写す（サーバに API を足さない）** — 画面を触らずに済むが、同じ探索と tx の組み立てが TypeScript と Go の2実装になり、
  片方だけ直すと消える範囲が食い違う（ADR-0611「同じ対応表を2形態で持たない」と同じ理由。KFTL をサーバの1実装へ寄せたのも同じ判断）。
- **既定は連鎖しない（`cascade:true` のときだけ）** — 今の MCP の挙動を変えずに済むが、AI が付け忘れると参照先の無いリポストや孤立したタグが残る。
  画面の削除は常に連鎖するので、既定もそれに揃える。
- **復元も連鎖させる** — 「同じ連鎖削除で消えたもの」を探すには、削除済みのタグなどを引く問い合わせが新たに要る。画面に復元が無く、
  `deleted[]` を targets[] に渡せば同じことができるので、今回は持たない。
- **通知の履歴を `GkillRepositories` の側で直す（削除済みを落とすのをやめる）** — 呼び出し元はこの usecase だけだが、集約の実装を
  タグ・テキストの経路（型別の集約 rep）に揃えるほうが、3つの付随データで同じ規則になる。

## Consequences

- 画面の連鎖削除は API 1回になり、Service Worker のキャッシュ（以前の `force_reget`）を気にする必要が無くなった。逆引きはサーバの中で行う。
- 画面側のエラーコード `ERR900093`（深さ超過）は欠番になり、サーバの `ERR000451` が返る。サーバの新しいコードは `ERR000445`〜`ERR000451`。
- 既定が連鎖に変わったので、MCP の `gkill_delete_kyou` で記録を消すと付属データと参照元も消える。単体で消したいときは `cascade:false`。
- ツールが7本増え、`gkill_delete_kyou` の説明と引数が変わったので、`schema_revision` が3サーバで変わる（read も、履歴の data_type の列挙に idf が入るので変わる）。
  ChatGPT と Claude.ai のコネクタは接続し直しが要る。

## Evidence

- tools/list の予算（`tool_schema_budget.json` の値。コード由来）: read 42,469 → 42,493、write 65,728 → 73,585、readwrite 90,283 → 98,140。
- golden を再生成した差分は、既存ケースでは initialize・tools/list・status・help の目次 / `idf` / `deleted` / 未知の topic と、
  既定が連鎖に変わった削除のケース、idf が型の列挙に入ったケースだけで、ほかは追加した 19 ケース（ゴールデンの要求コーパス由来の件数）。
- `ContentPath` と通知の履歴の2つは、テストを書いて修正を外すと落ちることを確かめた（キャッシュ ON だけで落ちる / 通知の削除済みの版が履歴に出ない）。

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_delete_kyou_cascade_test.go`（連鎖の範囲・多段・循環・深さ上限・断る条件・射影名・通知の履歴。キャッシュ ON / OFF）
- `src/server/gkill/api/gkill_server_api/gkill_server_api_test.go` の `TestHandleUpdateIDFKyou_StaysSearchableAfterEditAndRestore`
- `src/client/__tests__/unit/classes/cascade-delete-kyou.test.ts`（API を1回呼ぶ・行の id・キャッシュ・失敗時・共有画面・`errors: null`）
- `src/server/gkill/mcp/write_handlers_test.go` の `TestHandleWriteToolCallCascadeDelete` / `TestHandleWriteToolCallReferenceTypes`
- `src/server/gkill/mcp/golden_test.go`（`delete: *` / `add rekyou: *` / `add mirekyou: *` / `add notification: *` / `update *` / `help: reference_types`）

## 追記（2026-10-19）

削除の版の `update_device` は、要求の `update_device` があればその値になった（空なら従来どおりセッションの端末名）。
MCP は "mcp" を送り、MCP の書き込みの端末欄が経路によらず "mcp" に揃った（[ADR-0637](0637-mcp-writes-always-record-device-mcp.md)）。
temp rep のキーには従来どおりセッションの端末名を使う。`update_user` はセッションから決めるまま。

あわせて、ファイルの記録の temp rep が INSERT で作成者と作成端末を入れ替えていて、ファイルの記録を連鎖削除するたびに
削除の版の作成欄が入れ替わって確定していたのを直した（`idf_kyou_temp_repository_sqlite3_impl.go` の列の並び）。
