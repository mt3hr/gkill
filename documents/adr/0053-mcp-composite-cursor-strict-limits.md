# ADR-0053: MCPのページングは複合カーソル（時刻+ID）にし、Limit/MaxSizeMBを厳密な上限へ戻す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-23 |
| Sources | `23043385` |
| Supersedes | [0052](0052-mcp-cursor-pushes-period-end.md) |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp.go` / `src/server/gkill/api/gkill_server_api/get_kyous_mcp_helpers.go` |

## Context

ADR-0052 は時刻のみのカーソル（RFC3339Nano）を採り、その帰結として
「同一 RelatedTime のかたまりをページ境界で割らない＝Limit と MaxSizeMB は目安」
「TotalCount はカーソル以降の残り件数」という契約になっていた。

1年分124,056件の外部監査で、この契約が実際のAIクライアントの使い方と衝突することが確定した。

- `limit:1` で件数だけ読むつもりが同一時刻の束（Fitbit日次指標は全て12:00:00固定で
  19件、バルク取り込みのIDFで12件）が丸ごと返り、**最安のはずの呼び出しが最安にならない**
- `total_count` がカーソルの有無で「全件数/残件数」と意味を変え、
  **進捗率の分母に使うとページをまたいだ瞬間に静かに壊れる**（実測: 2,074→72、445→…→46）
- 件数・分布の取得手段が無く、監査の244回のMCP呼び出しの過半が
  「total_count 1個を読むためだけの limit:1」に消えた

## Decision

- カーソルを **`{RFC3339Nano}::{ID}` の複合形式**にし、並び順を
  **(RelatedTime降順, ID昇順) の全順序**にする。これで同一時刻のかたまりの途中からでも
  正確に再開できるため、**Limit と MaxSizeMB を厳密な上限に戻す**
  （唯一の例外はページ先頭の1件が単独で MaxSizeMB を超えるとき: 返さないと
  0件+has_more=true の永久ループになるため、その1件だけ返して警告する）
- 区切りは左端の `::`。RFC3339Nano にコロン2連は現れないので、
  プラグインの任意文字列IDに `::` が含まれても安全
- **TotalCount は cursor 無しの応答にのみ**入れ（`*int`）、全応答に **RemainingCount** を入れる。
  カーソルの期間上限への押し下げ（ADR-0052 の性能上の核心）は維持するため、
  2ページ目以降のハンドラは全件数を知らない
- 旧形式カーソル（RFC3339単独・日付のみ）は受理を続ける（移行中の in-flight カーソルと
  APIを直接叩くクライアントを壊さない）。解釈できないカーソルはエラー（0052と同じ）
- あわせて **count_only / group_by / data_types / num_min / num_max / idf_kinds** を
  リクエストレベルに追加し、「件数のためのフルDTO往復」と「窓分割による境界二重計上」を根絶する。
  未知のフィルタ値は Warnings で指摘する（正当な0件の経路はエラーにしない）
- id / rep_name は常時付与し、include_id / include_rep_name フラグは廃止する
  （追撃クエリの前提であり、フラグの立て忘れが往復を1回増やしていた。
  サイズ増は厳密化した MaxSizeMB 自体が吸収する）

## Rejected alternatives

- **時刻のみカーソル + グループ非分割の維持（0052）** — 上限が守れず、件数取得の
  コストが読めない。監査で実害が確定した。
- **offsetページング** — 深いページほど全走査になり、0052が潰した
  「1ページのために毎回全期間を検索する」事故（ピーク18GB/1ページ1時間53分）へ逆戻りする。
- **カーソルページでも全件数を返す（毎ページ全期間COUNT）** — 押し下げの意味が消え、
  ページ数ぶん全期間検索が走る。0052の事故の再来。
- **base64等の完全不透明トークン** — デバッグ性・手打ち検証性を失う割に得るものが無い。
  複合形式は目視で読め、Node側は素通しするだけなので不透明契約としても機能する。
- **group_by を Node(MCPサーバ)側で実装する** — 全件をHTTPで運んでから数えることになり、
  count_only が最安にならない。集計は検索結果を持っているGo側でやるのが唯一安い。

## Consequences

- `limit` / `max_size_mb` は文字どおりの上限になった。`returned_count > limit` は起きない
- 呼び出し側は残量を `remaining_count`、全件数を1ページ目の `total_count` で読む。
  cursor と count_only / group_by の併用はエラー
- 旧v1クライアント（include_id 等を送る）は無害に動くが、`total_count` を
  カーソルページで読んでいたコードは `remaining_count` へ移行が必要（破壊的変更）
- NextCursor の形式が変わったため、デプロイはGoを先行させる（旧形式受理がフェイルセーフ）

## Evidence

- 監査実測: limit:1 に対し returned_count 10/12/19。total_count 2,074→72（最終ページを
  97%完了と誤読しうる）。244呼び出し中、過半が件数取得のみの limit:1
- 改修後: `TestHandleGetKyousMCP_CompositeCursor` 系が limit=1 で同一時刻3件の束を
  1件ずつ3ページで取りこぼし・重複ゼロで回収することを固定

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp_test.go`
- `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp_v2_test.go`
- `src/mcp/__tests__/read-handlers.test.mjs`
- `src/mcp/__tests__/normalization.test.mjs`
