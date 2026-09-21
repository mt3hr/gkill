# ADR-0629: 応答の定常オーバーヘッドを削る（付随 ID はオプトイン・is_deleted は true のときだけ・重複欄を落とす・設定ツリーを圧縮）

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-19 |
| Sources | 2026-09-18 の MCP 実利用報告「設計課題（コンテキスト経済）」（`rep_struct` だけで 25,000 トークン超、`is_deleted:false` 常時、`tags[]` と `tag_entities[]` の二重持ちでタグ4つ約500字、plugin payload の `data_type` / `rep_name` / `kyou_id` と git の `commit_hash==id` の重複）。利用者の判断: 設計課題のうちこの束だけを今回やり、`is_deleted` は true のときだけにする |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/req_res/kyou_mcp_dto.go` / `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp.go`（`include_attached_ids`）/ `src/server/gkill/mcp/read_handlers.go`（`compactAppConfigStructs` / `filterAppConfigStructs` / `capAppConfigSize`）/ `src/server/gkill/mcp/plugin_tools.go`（`collectPluginPayloads`） |

## Context

2026-08 の指摘（C1 / C2 / C4 / C5）で、AI の追撃クエリに要る欄を「要求フラグ無しで常時載せる」方向に
揃えた。実利用が進むと逆側が見えた —— 読むだけの呼び出しが大半で、毎件の `is_deleted:false`、`tags[]` と同じ
内容の `tag_entities[]`、Kyou 側と同値の plugin payload 3欄、`id` と同値の `commit_hash`、
そして設定ツリーの葉が毎回持つ `name==rep_name` / `children:null` / `is_dir:false` / `ignore_check_rep_rykv:false`
が、LLM のコンテキストを定常的に食っていた。

## Decision

- `tag_entities[]` / `text_entities[]` は `include_attached_ids:true`（既定 false）のときだけ組む（Go）。
- `is_deleted` は `omitempty`。削除済みの記録にだけ現れ、無ければ生きている（`is_zip` / `addition` / `deletion` は
  据え置き —— あちらは false / 0 に「値が取れなかった」と紛れる意味がある）。
- `PluginPayloadMCPDTO` は `kind` / `plugin_name` だけ。本文取得の鍵は Kyou 側の `rep_name` / `id`
  （MCP の `collectPluginPayloads` がそちらを読む）。`GitPayloadMCPDTO.commit_hash` は落とす（`id` がハッシュ）。
- `gkill_get_application_config` に `compact`（既定 true。`children` の null / 空、`is_dir:false`、
  `ignore_check_rep_rykv:false`、識別欄と同じ `name` を落とす。`check_when_inited` / `is_force_hide` は触らない）、
  `contains`（葉の名前で刈る）、`max_size_mb`（超える struct 欄を大きい順に `{omitted_bytes}` へ置き換えて必ず収め、
  `warnings[]` で案内）を足す。

## Rejected alternatives

- **`tags[]` を落として `tag_entities[]` だけにする** — Web の列と Wear OS が `[]string` を前提にしている
  （ワイヤ互換。DTO のコメント）。
- **`is_deleted` を据え置く（指摘 C4 のまま）** — 利用者が「true のときだけ」を選んだ。`include_deleted_data` を
  使うときの判別は true が載ることで足りる。
- **plugin payload に `rep_name` / `kyou_id` を残して Node の読み取り先を変えない** — 毎件3欄の重複がそのまま残る。
  Node の読み取り先を Kyou 側へ寄せるのは1箇所（`collectPluginPayloads`）で済む。
- **設定ツリーの `check_when_inited:true` も落とす** — 可視タグの判定（`is_force_hide=false AND check_when_inited=true`）
  に要る値で、「無ければ true」を AI に覚えさせる価値より取り違えの害が大きい。

## Consequences

- `include_attached_ids` を渡さない読み取りでは注釈 ID が取れない。タグ・テキストを直す・消すときは true にする。
- `is_deleted` の有無で生死を読む。`false` は現れない。
- plugin payload / git payload の重複欄を読んでいたクライアントは Kyou 側の欄へ移る。
- `compact:true` の設定ツリーは「無い欄 = 既定値」。説明文に既定値の意味を書いてある。上限に当たった struct は
  `{omitted_bytes}` になるので、`fields` / `contains` で絞り直す。

## Evidence

- 報告値: `rep_struct` 25,000 トークン超、タグ4つで約500字の二重持ち。
- 修正後: 葉ノードは `{rep_name, check_when_inited}` の2欄（既定値の4欄が消える）。`tag_entities` は
  `include_attached_ids:true` のときだけ。`is_deleted` はゼロ値で JSON に出ない。plugin payload は2欄。

## Related tests

- `src/server/gkill/api/req_res/req_res_test.go`（omitempty の固定と欄名）
- `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp_v2_test.go`（`TestHandleGetKyousMCP_AttachedIDsAreOptIn` / `TestHandleGetKyousMCP_GitPayloadZeroDiffAndHash`）
- `src/server/gkill/mcp/read_handlers_test.go`（「gkill_get_application_config compact / contains / max_size_mb」）
- `src/server/gkill/mcp/plugin_tools_test.go`（`collectPluginPayloads`）
