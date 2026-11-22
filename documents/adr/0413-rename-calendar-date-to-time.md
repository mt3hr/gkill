# ADR-0413: 期間フィルタのフィールドは CalendarStartTime / CalendarEndTime へ改名する（`Date` と名乗っていたが実体は時刻）

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-22 |
| Sources | `src/server/gkill/api/find/find_query.go` の `FindQuery` / [ADR-0806](0806-fix-spellings-instead-of-freezing.md)（綴りは凍結せず直す） / [ADR-0106](0106-find-query-null-semantics.md)（null 判定セマンティクス） / [ADR-0111](0111-drop-never-implemented-query-fields.md)（未知キーは弾く） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/find/find_query_legacy_json.go` の `MigrateLegacyFindQueryJSON` と `legacyCalendarKeyRenames` の doc コメント |

## Context

期間フィルタのフィールドは `CalendarStartDate` / `CalendarEndDate`（JSON では `calendar_start_date` /
`calendar_end_date`）という名前だったが、型は `*time.Time` で、**時刻成分を秒以下まで保持していた**。
SQL 側（`sqlite3impl_util.go` の `GenerateFindSQLCommon`）は `unixepoch()` を通して秒精度で比較し、
Go 側の再判定（`find_filter.go` の `newKyouTimeFilter`）は `Before` / `After` でナノ秒精度・両端 INCLUSIVE に比較する。
日単位へ丸めていたのは UI（`use-calendar-query.ts` が `startOf('day')` と `+1日 -1ミリ秒` を手で組んでいた）と、
MCP の入力正規化（`NormalizeDateOnlyToRfc3339` が `YYYY-MM-DD` だけの指定を展開する）の2か所だけで、
**ワイヤもストレージも最初から時刻だった**。

サイドバーの「日付」を「期間」へ改名し、開始時刻・終了時刻を指定できるようにすると、
この名前は実態と食い違うだけでなく積極的に誤解を招く。「`Date` だから日単位だろう」と読んだ人が、
時刻を載せる実装を別フィールドとして足しにかかる。

名前の食い違いは、隣にある別のフィルタとの見分けも悪くしていた。`PeriodOfTimeStartTimeSecond` /
`PeriodOfTimeEndTimeSecond`（1日の中の時間帯を指定するフィルタ。ADR-0108）は名前に `Time` を持ち、
期間フィルタは `Date` を持つ。どちらが壁時計の時刻を運ぶのかが名前から逆に読める。

## Decision

`CalendarStartDate` / `CalendarEndDate` を `CalendarStartTime` / `CalendarEndTime` へ、
JSON キー `calendar_start_date` / `calendar_end_date` を `calendar_start_time` / `calendar_end_time` へ改名する。
Go 本体・TypeScript の `FindKyouQuery`・MCP のツールスキーマ・プラグインとの契約（`plugin_protocol.go`）の
すべてで同じ名前にし、**互換は一切残さない**（ADR-0806 の方針）。

`Calendar` の部分は残す。`Period` へ寄せると `PeriodOfTimeStartTimeSecond` と見分けがつかなくなる。

永続化済みの JSON（localStorage の保存クエリ、設定 DB の `saved_find_query_json_data`、
共有情報の `FindQueryJSON`）は、parse 境界で一度きり書き換える。入口は既存の移行実装をそのまま使う
（Go は `MigrateLegacyFindQueryJSON`、クライアントは `normalize_legacy_find_kyou_query_json`）。

**改名は `use_*` フラグの変換より先に走らせる。** `use_*` の変換は新キー名で値を探すため、
順序を入れ替えると `use_calendar: false` が旧キー名の値を見つけられない。

旧キーの再出現は `npm run verify_docs` の `RETIRED_SPELLINGS` 検査が止める。

## Rejected alternatives

- **名前を変えず、コメントで「実体は時刻」と書く** — [ADR-0802](0802-freeze-plaing-spelling.md) が綴りを凍結したときと同じ形で、ADR-0802 自身が「文書だけが防御線で、一括置換で壊せる状態にある」と書いていた。[ADR-0806](0806-fix-spellings-instead-of-freezing.md) でその方針を反転したので、今回も直す側に倒す。
- **`PeriodStartTime` / `PeriodEndTime` へ改名する** — 「期間」というメニュー名には素直に揃うが、`PeriodOfTimeStartTimeSecond`（時間帯フィルタ）と先頭6文字が同じになり、補完候補でも grep でも混ざる。見分けを良くするための改名で見分けを悪くしては本末転倒なので `Calendar` を残した。
- **新旧どちらのキーも受け付ける移行期間を置く** — ワイヤに名前が2つ乗ると、移行3実装（Go・クライアント・MCP）とスキーマの全部で両方を同期させ続けることになる。[ADR-0111](0111-drop-never-implemented-query-fields.md) は「受理して無視するより未知キーとして弾く」を選んでおり、その逆を行くことになる。MCP には `schema_revision` という「クライアントのツール定義が古い」を伝える仕組みが既にあるので、受け付けずに弾くほうが筋が通る。
- **新フィールドを足して旧フィールドを deprecated にする** — 上と同じ。加えて、両方が非 null のときにどちらを使うかという判定が全実装に1つずつ増える。
- **改名を `use_*` 変換の後に走らせる** — `use_calendar: false` の変換が新キー名で値を探すため、旧キー名のまま残っている値を null にできない。結果として「フィルタ未使用だったはずの期間が、有効な期間として復活する」。エラーも警告も出ないので、利用者には「保存しておいた検索条件の結果が変わった」としか見えない。
- **永続データを変換せず、読めなくなった旧キーを捨てる** — 保存済みの検索条件から期間だけが黙って消える。ADR-0806 は「永続データに乗っていた場合は旧綴りのデータを一度きりの変換で直すこと」を求めている。

## Consequences

- **プラグインのバイナリを再ビルドして配り直す必要がある。** `plugin_protocol.go` は別プロセスのプラグインとの契約で、フィールドは `omitempty` 付きなので、**旧バイナリは新しいキーを読めず、キーが欠けてもエラーを立てない**。期間フィルタだけが黙って効かなくなり、プラグイン由来の記録が期間外まで出る。`src/plugins/` の各モジュールは `src/server` を `replace` で参照しているので、本体と一緒にビルドし直すこと。
- **MCP のツール定義を古いまま持っているクライアントは、旧キーを未知キーとして弾かれる**（ADR-0111）。`schema_revision` が変わるので、`gkill_status` の応答と `tools/list` の採り直しがその合図になる。
- **`add_tag` のルール JSON は変換しない。** CLI の引数として利用者が書いたファイルで、gkill は保存場所を知らない。旧キーのまま渡すと未知キーとして弾かれるので、利用者が書き直す。
- サンプル設定 DB（`resources/gkill_sample_data/configs/user_config.db`）には旧キーが入ったままだが、ほかの永続データと同じく起動時の移行で変換される。バイナリなので `RETIRED_SPELLINGS` 検査の対象外（拡張子で除外済み）。
- 移行の実装とその試験は、旧キー名を本文に書く必要があるため `RETIRED_SPELLINGS` の除外に入れた。除外を増やすときは、ここに理由を1行足すこと（allowlist が育つと防御線がまた文書へ戻る）。
- 改名そのものは挙動を変えない。**時刻を指定できるようにする UI の変更は別の決定**で、このフィールドの型も意味も変わらない。

## Evidence

実測なし — 型が `*time.Time` であること、SQL が `unixepoch()` で秒精度・両端 INCLUSIVE に比較すること、
`newKyouTimeFilter` が `Before` / `After` で比較することからの判断。
改名と変換の順序はユニットテストで固定した。

## Related tests

- `src/server/gkill/api/find/find_query_legacy_json_test.go` — 「use_* を持たない JSON でも改名される」「改名は use_* 変換より先に走る」「ネストの奥でも改名が効く」「改名は冪等・新キーが既にあれば旧キーを捨てる」
- `src/client/__tests__/unit/api/normalize-legacy-find-kyou-query-json.test.ts` — 「旧キー名の改名」（use_* 無しでの改名・use_* より先に走ること・冪等・新キーが勝つこと）
- `src/server/gkill/dao/share_kyou_info/share_kyou_info_schema_migration_test.go` — 共有情報の `FindQueryJSON` が起動時に移行されること
- `src/server/gkill/mcp/golden_test.go` — ツールスキーマと応答のゴールデン（`schema_revision` を含む）
