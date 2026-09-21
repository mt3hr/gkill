# ADR-0619: ツール一覧の世代は gkill_status の schema_revision で見せ、未知の引数名では再接続を案内し、tools/list のバイト量を予算で固定する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-14 |
| Sources | MCP レビュー（ChatGPT からの実呼び出しで「tools/list どおりに呼ぶと未知の引数で拒否される」を再現。P0 の2件）。[gkill-mcp](../../.claude/skills/gkill-mcp/SKILL.md) の節「ツール一覧の世代は `gkill_status` の `schema_revision` で見せる。」「未知の引数名には「古い一覧の可能性」を必ず添える。」「tools/list のバイト量は予算ファイルで固定する。」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/mcp/status_tool.go`（`computeSchemaRevision` / `stampSchemaRevision`）/ `src/server/gkill/mcp/server_base.go`（コンストラクタの焼き込み・`describeServer`・`initialize` の version）/ `src/server/gkill/mcp/read_tools.go`（`gkill_status` の定義）/ `src/server/gkill/mcp/read_handlers.go`（`buildStatusPayload`）/ `src/server/gkill/mcp/validation.go`（`unknownKeyMessage`）/ `src/server/gkill/mcp/bootstrap.go`（`server_start` ログ）/ `src/server/gkill/mcp/tool_schema_budget.go` と `src/server/gkill/mcp/tool_schema_budget.json` / `src/server/gkill/main/common/mcp.go` |

## Context

2026-09-14 のレビューは ChatGPT から実際に gkill MCP を叩き、公開されている
`gkill_get_kyous` のスキーマに載っている検索条件名（2026-09-08 の全面改名で消えた旧綴り）を
そのまま送ると `Invalid argument 'query.<旧名>': is not supported.` になることを再現した。
正しいスキーマに従った AI が失敗する、というのは「AI が間違った使い方をしない」を
掲げる MCP にとって一番たちの悪い形で、しかもエラー文には再接続の案内が一切無かった。

原因を切り分けると、**サーバのプロセスは古くなかった**。常駐させていた MCP サービス 2 本
（read / readwrite。当時は node が旧 `src/mcp` の作業ツリーを直接実行していた。2026-09-20 からは `gkill_server.exe mcp` で本体と同じ exe を配る。ADR-0631）は
2026-09-10 03:41 起動で、改名コミット `51752fe7`（2026-09-08 18:57）より後。
古かったのは ChatGPT のコネクタが接続時に取った tools/list で、これはサーバを
何度再起動しても更新されない（[ADR-0609](0609-stale-tool-schema-is-warned-only-when-proven.md) が
claude.ai コネクタで実測した「セッション寿命で固定」の ChatGPT 版）。

既存の古スキーマ検出（`detectStaleSchemaSignals`）が拾える証拠は「後付け引数が JSON 文字列で
届く」「廃止済み引数が届く」の2つだけで、**改名で消えた名前は未知キーとして拒否されるだけ**。
つまり現行の対策は「引数の型が古い」には強く、「名前自体が消えた」には無力だった。
また、その改名前の名前を受理する互換は書けない —— 旧綴りは `verify_docs` の
`RETIRED_SPELLINGS` が追跡ファイル全体で禁止しており（[ADR-0806](0806-fix-spellings-instead-of-freezing.md)
「互換は一切残さない」）、書けたとしても次の改名で同じことが起きる。

同じレビューが指摘したもう1つの P0 は tools/list のサイズで、readwrite が約 94KB
（`gkill_get_kyous` 単体 27KB、うち inputSchema 22KB）。事故対策を説明文へ書き足すたびに
一覧は太り、個々の追記は正しいのに合計は誰にも見えていなかった。

## Decision

- **`gkill_status` ツールを3サーバ全部に足す**（引数なし）。返すのは `server_kind` /
  `server_name` / `server_version` / `schema_revision` / `tool_count` / `transport` /
  `started_at` / `uptime_seconds` と、gkill から取った `account.{user_id,device}` /
  `gkill.{version,commit_hash,build_time}`。gkill へ届かないときは失敗にせず
  `gkill_reachable:false` + `gkill_error:"HTTP <status>"`（本文は載せない。ADR-0707）
- **`schema_revision` はツール一覧から決定的に計算する**（`gkill_status` を除いた tools 配列の
  JSON の SHA-256 先頭12桁）。**`gkill_status` の description 末尾に焼き込み**、`initialize` の
  `serverInfo.version` に `+schema.<revision>` を添え、`server_start` ログにも載せる。
  AI は description の値と応答の値を比べれば、自分の一覧が古いことを自分で分かる
- **未知の引数名のエラー文は1本（`unknownKeyMessage`）にし、「書き間違いか古い一覧かは
  区別できない」と言って再接続と `gkill_status` の照合を案内する**。トップレベル
  （`assertKnownKeys`）と `query` の中（`normalizeKyouQuery`）の両方がこれを使う
- **「tools/list どおりに呼ぶと失敗しない」を機械検査する**（`schema_contract_test.go`）:
  スキーマのキー集合 = 受理集合 − 廃止済み、各ツールを全プロパティ指定で呼んで未知キーで
  落ちない、3サーバの同名ツールが同じ JSON（`gkill_status` の印だけ剥がして比較）
- **tools/list のバイト量を予算ファイル `src/server/gkill/mcp/tool_schema_budget.json` で固定する**。
  値は現状の実測。超えたら失敗、1024 バイト以上下回っても失敗（予算を追随させる）。
  更新は `gkill_server mcp schema-budget --update` で明示的に行う

## Rejected alternatives

- **改名前の名前を新名へ写して受理する（旧綴りの `*_time` → `playing_time`）** — ADR-0806 と
  正面から矛盾する（旧綴りは `verify_docs` が禁止し、書けない）。書けたとしても
  「消えた名前」は改名のたびに増え、受理表が育つほど公開スキーマと受理集合が乖離する。
  古さを見せる仕組みのほうが、次の改名にも効く
- **手書きの版番号（`SCHEMA_REVISION = "2026-09-14-01"`）** — 更新し忘れる。説明文を1文
  直しただけの変更は「版が変わった」と誰も思わないが、クライアントから見れば別の一覧。
  一覧そのものから計算すれば忘れようがない
- **tools/list 応答の `_meta` に revision を載せる** — MCP 仕様上は置けるが、AI クライアントは
  `_meta` を AI に見せない。AI が「自分の握っている一覧の世代」を知る経路は description
  しか無い。人間・クライアント UI 向けには `serverInfo.version` に添えた
- **すべての応答に「一覧が古いかもしれない」と推測で警告する** — ADR-0609 の否決案そのもの。
  証拠なしの常時警告はノイズになって読まれなくなる。ここで足した証拠は「未知の引数名が
  届いた」（区別できないことを言う）だけで、他は AI 自身の照合に委ねる
- **`gkill_status` を `gkill_get_application_config` の `fields` に畳む** — 接続先の確認は
  既にそこで出来ていたが、gkill へ届かないときにそのツールは失敗する。「MCP は生きているが
  gkill が落ちている」を返せる別のツールが要る。引数なしで安く、write 専用サーバにも載る
- **`gkill_status` に `locale_name` 等の引数を持たせる** — 後付け引数はすべて
  `STALE_SCHEMA_ARG_KINDS_BY_TOOL` の対象になる。古さを確かめるツール自身が古い一覧で
  壊れる形を作らない
- **tools/list の予算に閾値（例: 64KB）を決めて一気に削る** — Compact profile を含む
  スキーマ縮小は別の作業（2026-09-14 のレビュー P1/P2。今回は対象外）。予算は回帰検知の
  道具であって目標ではないので、現状値から始める。減ったら追随させる規則のほうが効く
- **予算を verify_docs の `--list` に載せる** — verify_docs は静的走査で、tools を評価するには
  サーバ3本を import する必要がある。テスト（`npm run test_mcp`）に置けば CI で回る

## Consequences

- ツール数は read 11 / write 28 / readwrite 32。資料・テストの固定値を更新した
- `gkill_status` の description は3サーバで末尾12桁だけ違う。「read と readwrite の同名
  ツールは同一定義」の唯一の例外で、`schema_contract_test.go` は印を剥がしてから比べる。
  静的な `READ_TOOLS` は書き換えない（`McpServerBase` が自分用の配列を持つ）
- 説明文を1文直すだけで `schema_revision` は変わる。それが正しい —— クライアントから見れば
  別の一覧で、古い一覧を握ったセッションはその1文を知らない
- **プロセスが新しくても古い一覧が見えることはある**。切り分けの順は、起動ログ /
  `gkill_status` 応答の revision → AI が引用する description の印 → 一致しなければ
  クライアント側の接続し直し（ChatGPT のコネクタはサーバ再起動では直らない）
- 予算ファイルは説明文を直すたびに `--update` が要る。手間だが、それが「合計を見る」こと
  そのもの。増やしたときは理由をコミットメッセージに書く
- MCP のサービス（当時は node が作業ツリーを直接実行）には、この変更はコミットしただけでは
  届かず、サービスの再起動と、各クライアントの接続し直しが要る

## Evidence

- 2026-09-14 実測: node の MCP サービス2プロセスは 2026-09-10 03:41:54 起動
  （`Win32_Process.CreationDate`）。改名コミット `51752fe7` は 2026-09-08 18:57:04 +0900。
  ChatGPT から `query.<旧名>` で呼ぶと `is not supported`、`playing_time` なら成功
- tools/list の実測（gkill_status 追加・廃止引数除去の後）: read 47,187 B / write 59,911 B /
  readwrite 93,980 B。追加前は read 43,237 B / readwrite 93,740 B で、`gkill_get_kyous`
  単体 27,061 B（inputSchema 21,874 B）、`gkill_submit_kftl` 8,472 B
- 変異テスト: `FIND_QUERY_SCHEMA.properties` に架空のキーを1つ足すと
  `schema_contract_test.go` の2テスト（キー集合一致・全プロパティ指定スモーク）が落ちる

## Related tests

- `src/server/gkill/mcp/status_tool_test.go`
  - 決定性・自己参照除外・焼き込みの冪等・3サーバで値が違うこと・`gkill_status` は引数なし
- `src/server/gkill/mcp/schema_contract_test.go`
  - キー集合の一致・全プロパティ指定スモーク（全ツール）・3サーバの同名ツール同一・世代の一致
- `src/server/gkill/mcp/tool_schema_budget_test.go`
  - 予算内・計測の決定性・over / under / missing の判定と文言
- `src/server/gkill/mcp/read_handlers_test.go`（`handleReadToolCall — gkill_status`）
  - 応答の形・gkill 不達時の `gkill_error` と本文非掲載・引数拒否・1行要約
- `src/server/gkill/mcp/server_test.go`
  - `initialize` の version・description の印・応答の `schema_revision` が一致
- `src/server/gkill/mcp/normalization_test.go`（`unknown argument names point at a possibly stale tool list`）
  - トップレベルと `query` の未知キーが同じ文言で再接続と `gkill_status` を案内
