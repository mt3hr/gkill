# ADR-0618: urlog の外向き取得は既定で行い、引数で項目別に抑止する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-30 |
| Sources | 2026-08-30 の MCP レビュー（AI からのブックマーク登録が無断の外向き通信になる指摘）。[gkill-mcp](../../.claude/skills/gkill-mcp/SKILL.md) の節「書き込みの後付けフラグはリクエストの修飾子であって、エンティティの列ではない。」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/mcp/write_tools.go`（`gkill_add_urlog` の `fetch_metadata` / `fetch_favicon`）/ `src/server/gkill/mcp/write_handlers.go`（`skip_fetch_*` への反転）/ `src/server/gkill/api/gkill_server_api/handle_add_urlog.go` / `src/server/gkill/api/req_res/add_ur_log_request.go`（`skip_fetch_metadata` / `skip_fetch_favicon`）/ `src/server/gkill/dao/reps/ur_log.go`（`FillURLogFieldSkipping`） |

## Context

ブックマーク（urlog）の登録時、サーバは対象URLのページ本文（title / description /
サムネイル）と、第三者の favicon サービスからアイコンを取得して空欄を補完する。
Web UI・ブックマークレットにとっては便利な既定だが、MCP 経由で AI が登録する場合、
**「保存して」と頼んだだけで対象サイトと第三者サービスへ通信が飛ぶ**ことになる。
URL が社内システムやワンタイムリンクだと、取得自体が副作用になる。

2026-08-30 のレビューでこの点が指摘され、抑止手段が必要になった。
ただし既定を変えると Web / ブックマークレット / 既存 MCP クライアントの
挙動が一斉に変わるため、「どちら向きに倒すか」「フラグの粒度」「Go 側の表現」に
判断が要った。

## Decision

**外向き取得は既定で行う（従来どおり）。抑止は MCP 引数の項目別 opt-out** とする。

- MCP の `gkill_add_urlog` に `fetch_metadata` / `fetch_favicon`（いずれも既定 true）を追加。
  false でページ取得 / favicon 取得を**項目別に**抑止し、両方 false なら
  このブックマークについて外向き通信は1回も起きない
- Go 側のワイヤ表現は **skip の負論理**（`skip_fetch_metadata` / `skip_fetch_favicon`、
  ゼロ値 false = 従来どおり取得）。旧クライアントがフィールド無しで送っても挙動が変わらない
- ID・RelatedTime の補完は抑止と無関係に必ず行う（ID が無いと登録自体が成立しない）
- `gkill_update_urlog` にフラグは**足さない** — Go は `re_get_urlog_content:true` を
  明示されたときだけ再取得し、MCP はそれを送らないので、update 経路には抑止すべき取得が無い
  （抑止して登録したブックマークは update しても未取得のまま）

## Rejected alternatives

- **既定を「取得しない」へ倒す** — Web UI・ブックマークレット・既存の MCP クライアント
  すべてで「タイトルが自動で埋まらなくなる」互換破壊になる。取得はこの機能の本来の価値で、
  例外側（センシティブな URL）に opt-out を与えるのが釣り合う
- **2つを1フラグに束ねる** — 「favicon だけ止めたい（ページ本文は要る）」が表せない。
  通信先も対象サイトと第三者サービスで別物なので、同意の単位として分けておく
- **Go 側を正論理 `fetch_*` で持つ** — Go の bool ゼロ値は false。フィールドを知らない
  旧クライアント（Web UI 含む）からの送信が全部「取得しない」に化け、後方互換が壊れる。
  MCP 層で反転して負論理へ写すほうが安い
- **エンティティ（URLog 構造体）へフラグを載せる** — 取得抑止は「この1回のリクエストの
  指示」であって記録の属性ではない。実体へ載せると保存・応答・履歴に意味の無い列が残る

## Consequences

- AI クライアントはセンシティブな URL を `fetch_metadata:false` + `fetch_favicon:false` で
  外向き通信ゼロのまま保存できる（ツール説明にも明記し、title 説明は
  「省略かつ fetch_metadata:true のときだけサーバが埋める」の条件付きへ改めた）
- 後付け boolean 引数なので、古いスキーマを掴んだセッションからは正規JSON文字列で届く。
  `revivesStaleBoolean` + `STALE_SCHEMA_ARG_KINDS_BY_TOOL` の両登録で復元し、古さを警告する
  （[ADR-0609](0609-stale-tool-schema-is-warned-only-when-proven.md)）
- ハンドラの引数順（SkipFetchMetadata, SkipFetchFavicon）は両方 bool のため
  入れ替えてもコンパイルが通る。ソース走査テストで順序を固定した
- `/api/urlog_bookmarklet` は従来どおり常に取得（URL しか送られてこない）。
  KFTL 経由の URL 記録は従来どおり一切取得しない。経路ごとの差は
  error-handling-and-security.md §2.10 に一覧がある

## Evidence

- 2026-08-30 レビュー: 「ブックマーク保存 = 対象サイトへの通信」が AI クライアントから
  制御できず、説明文にも書かれていなかった
- 実装コミット 65c58bce（Go 側 `FillURLogFieldSkipping` + MCP フラグ + reps 層テスト）

## Related tests

- `src/server/gkill/dao/reps/ur_log_fill_skip_test.go`
  - 両方 skip で外向き取得ゼロ + ID / RelatedTime のみ補完、favicon だけ skip、互換ラッパは両方取得
- `src/server/gkill/api/gkill_server_api/handle_add_urlog_skip_wiring_test.go`
  - ハンドラが `request.SkipFetchMetadata, request.SkipFetchFavicon` の順で渡すこと（ソース走査）
- `src/server/gkill/api/req_res/req_res_test.go`
  - `AddURLogRequest` のワイヤ契約（`skip_fetch_metadata` / `skip_fetch_favicon` のタグ名固定）
- `src/server/gkill/mcp/write_handlers_test.go`
  - `skip_fetch_*` への反転・フラグ独立・実体へ漏れないこと・update が `re_get_urlog_content` を送らないこと
- `src/server/gkill/mcp/write_normalization_test.go`
  - 既定値 true・addOnly・古スキーマ文字列の復元（trim 込み）
