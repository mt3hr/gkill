# ADR-0611: 同じ対応表を「表」と「直書き」の2形態で持たない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-25 |
| Sources | 2026-08-25 の「同名APIを完成度の高いほうへ一本化してほしい」という依頼と、その前提となった実利用レビュー3本 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/mcp/write_handlers.go`（`UPDATE_TARGETS` / `runUpdate`）/ `src/server/gkill/mcp/payload.go`（`entityNotFoundMessage`）/ `src/server/gkill/mcp/server_base.go`（`makeOAuthAuthenticateUser`）/ `src/server/gkill/api/gkill_server_api/utils.go`（`resolveSelfAuthContext`） |

## Context

「同じ名前のAPIは完成度の高いほうへ一本化してほしい」という依頼で全体を洗ったところ、
**MCP サーバ3種の間は既に一本化済み**だった（ツール名の重複0件、クライアントは1クラス、
ディスパッチも `buildToolResult` も1本）。にもかかわらず、次の4箇所で
**同じ処理が2形態で併存**していた。

1. **`gkill_update_*` 9本** — 取得→patch→更新の手順は9本とも同じで、
   型ごとに違うのは正規化関数と patch する欄だけ。ところが24行のブロックが9本並んでおり、
   取得先・更新先・応答キーの対応表（`ENTITY_TARGETS`）は**既に存在して**
   `softDeleteOne` と `gkill_get_kyou_history` はそちらを使っていた
2. **「見つからない」の文言が3種類** — 直書きだった9本が `Kmemo not found: {id}` を、
   read 側が `Entity not found: {id}` を、write の削除経路だけが
   「型を取り違えると ID 違いと見分けが付かない」と説明する完成版を返していた
3. **OAuth の `authenticateUser` が3サーバへ逐語コピー**（md5 一致の18行）
4. **`wrapNoAuth` の自前認証3段**（アカウント→端末→リポジトリ）が、
   `authWithReposMiddleware` と同じ処理の3度目の複製として各ハンドラに散っていた

いずれも「片方だけ直すと静かにずれる」形で、実際に (2) は
**同じ状況で受け取る説明が呼んだツールによって違う**という形で表に出ていた。

## Decision

**手順が同じで型ごとに違うのが「データ」だけなら、そのデータを表にする。**

- 9本の update は `UPDATE_TARGETS`（正規化関数と patch 欄だけの表）＋ `runUpdate` へ集約し、
  取得先・更新先・応答キーは既存の `ENTITY_TARGETS` から引く
- 「見つからない」は `entityNotFoundMessage`（`payload.go`）1本。read / write / update が同じものを使う。
  **型を取り違えたのか ID が無いのかは型別エンドポイントの応答から区別できない**ので、
  区別できないことを言う（「ID が存在しない」と断定しない）
- `authenticateUser` は `makeOAuthAuthenticateUser`（`server_base.go`）1本
- `wrapNoAuth` の自前認証は `resolveSelfAuthContext`（`utils.go`）1本

**方向は「READ 側へ」ではなく「完成度の高いほうへ」で決める。**
今回は (1) が表側（read も write も使っていた側）、(2) は write 側が最も完成度が高かった。

## Rejected alternatives

- **9本を残したまま文言だけ揃える** — 表と直書きの併存が残る。
  次に欄を1つ足すとき9箇所を触ることになり、1つ落としても**テストは緑のまま**
  （各ツールのテストは自分の欄しか見ない）
- **`ENTITY_TARGETS` に正規化関数と patch 欄も持たせて表を1つにする** —
  `constants.go` は「定数」の置き場で、関数参照を入れると
  write 専用の関心事が read からも見える。表は用途ごとに分け、
  **エンドポイントの対応だけを共有**する形にした
- **`/api/get_kyous` を廃して `get_kyous_mcp` へ一本化** — 依頼の文字どおりだが、
  Web クライアントの列（rykv / mi / dashboard）は**件数無制限の生 Kyou 行**を前提に
  仮想スクロールしており、Wear OS companion も同じ形を使う。
  `get_kyous_mcp` は `limit` 1..1000 の厳密上限が契約（ADR-0604）なので、寄せると両方壊れる。
  代わりに**`/api/get_kyous` にしか無かった機能（プラグイン検索失敗の警告）を MCP 側へ移した**
- **`handle_get_shared_kyous.go` の11型ファンアウトを共有** — `IsShareWith*` の出し分けと
  `ViewType=="mi"` 分岐があり、認証も SessionID ではなく SharedID。共有側の設計が別
- **`GkillAPIForSharedKyou` を統合** — セッション無しで動くことが存在理由。統合不可
- **`GetAllTagNames` と `…IncludingDeletedTargets` を統合** — 意図的な対（ADR-0112）。
  後者を使わないと `include_deleted_data` の検索に誤警告が出る

## Consequences

- `write_handlers.go` が 297行 → 194行。欄を足すときに触るのは表1箇所
- 「見つからない」が1文になり、**どのアカウントに繋がっているか確認せよ**という案内も同時に入った
  （別アカウントに書いた記録を探して詰まる、が実際に起きていた）
- `wrapNoAuth` の MCP 2本のボディ上限は、最終的に `wrapNoAuthCapped`（2026-08-30、
  経路別上限＋読み取り期限）として実装された（アップロード系のための免除が MCP にも及んでいた）
- **9型すべてが表の口を使うことをテストで固定した**。1本だけ直書きへ戻しても落ちる

## Evidence

- 実利用レビューで `data_type` を `nlog` に取り違えて履歴を引いたところ「Entity not found」が返り、
  ID不在・型違い・アカウント違いの3つが同じ文言だった
- `authenticateUser` は3ファイルとも md5 `05e59085d70d92481cd2f95d3695d0ad`

## Related tests

- `src/server/gkill/mcp/write_handlers_test.go`
  - `update tools are table-driven`（9型すべてが `ENTITY_TARGETS` の口を使う）
  - `not-found says the lookup is per-type instead of blaming the id`
  - `handleWriteToolCall — stale tool schema warning`
