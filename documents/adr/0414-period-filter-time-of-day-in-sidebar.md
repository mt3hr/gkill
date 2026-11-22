# ADR-0414: サイドバーの「期間」で開始時刻・終了時刻を指定する（ワイヤ形式は変えず、丸めを UI の中だけで解く）

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-22 |
| Sources | `src/client/classes/use-calendar-query.ts` の `to_start_moment` / `to_end_moment` の doc コメント / [ADR-0413](0413-rename-calendar-date-to-time.md)（`Date` → `Time` の改名） / [ADR-0108](0108-period-of-time-second-of-day.md)（時間帯フィルタの秒） / [ADR-0106](0106-find-query-null-semantics.md)（null 判定セマンティクス） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/use-calendar-query.ts` の `to_start_moment` / `to_end_moment` / `start_time_string_from` / `end_time_string_from` の doc コメント、`src/client/classes/use-gps-log-map.ts` の `update_time_slider_max_value` の doc コメント |

## Context

GPSLog を共有するとき「この時間帯は渡したくない」が表現できなかった。サイドバーの期間フィルタが
日単位でしか切れず、`use-calendar-query.ts` が選んだ日を
`startOf('day')` と `+1日 -1ミリ秒` で日の境界へ丸めていたためである。

丸めているのは UI だけで、ワイヤもストレージも最初から時刻を運んでいた
（`CalendarStartTime` / `CalendarEndTime` は `*time.Time`。SQL は `unixepoch()` で秒精度、
Go 側の再判定は `Before` / `After` でナノ秒精度、どちらも両端 INCLUSIVE。ADR-0413）。
つまり足すべきものは UI の入力欄と、丸めの差し替えだけだった。

丸めの式は**3箇所に手書きされていた**。`clicked_date()` の emit、`get_start_date()`、`get_end_date()`。
片方だけ直すと「日付をクリックした直後は時刻が効くが、再検索すると消える」という形で静かに壊れる。

地図側（`use-gps-log-map.ts`）には別の罠があった。GPS の再取得の watch が
**日付文字列（`YYYY-MM-DD`）の computed に張られていた**ため、同じ日の中で時刻だけ動かしても
再取得が走らない。記録一覧は時刻で絞られているのに地図だけ丸一日ぶん出る、という食い違いになる。
スライダーの刻みも `(日数差 + 1) * 86400 - 1` で日数から出していたので、09:00〜18:00 を指定しても
1日ぶんを刻み、右半分が無効域になる。無効域の判定は無いので「動かしても最後の点に貼り付く」形で
静かに壊れる。

## Decision

サイドバーの「期間」に開始時刻・終了時刻の入力欄を2つ足す。**ワイヤ形式は変えない**
（`calendar_start_time` / `calendar_end_time` に時刻が載るだけ）。

- 丸めは `to_start_moment()` / `to_end_moment()` の2つの private helper に集約し、
  `clicked_date()` と getter の両方がそれを通る。
- **時刻を指定しなければ従来どおり 00:00:00.000 / 23:59:59.999。** 空欄が「未指定」を表す唯一の
  手段なので、入力欄には `clearable` を付ける（時間帯フィルタの先例には無い）。
- 終了時刻は「指定された分の末尾まで」と読む（`HH:mm` → `HH:mm:59.999`）。`VTimePicker` に秒の
  UI が無いため。
- props から着信した日時が日の境界ちょうどなら時刻欄は空のままにする。埋めると、既定期間で
  開いただけの列に 00:00 / 23:59 が表示される。
- メニュー名は i18n キー `CALENDAR_QUERY_TITLE` の値を7言語とも「期間」へ差し替える
  （キーは足さない）。時刻欄のラベルは時間帯フィルタの
  `PERIOD_OF_TIME_QUERY_START_TIME_TITLE` / `..._END_TIME_TITLE` を再利用する。
- 地図は再取得の watch を `props.start_time` / `props.end_time` の時刻そのものへ張り替え、
  スライダーの刻みを「期間の秒数」で出す。GPS の取得には props の時刻をそのまま渡す。
- `calendar-query.vue` は rykv と mi の共有なので、**両方で同時に効く**。サイドバー2本は
  getter 経由でしか触っていないので無改修で通る（メソッド名だけ `get_start_time` /
  `get_end_time` へ改名した）。

## Rejected alternatives

- **`calendar_start_time_second` のような別フィールドを足す** — 時間帯フィルタ（`period_of_time_*`）が1日の中の秒で持っている先例があるので一見もっともらしい。却下理由は、`*time.Time` が既に秒以下を保持しており、別フィールドにすると SQL 側・`find_filter.go` 側・移行3実装・MCP スキーマの全部に「日付と時刻を合成する」分岐が1つずつ増えること。合成の規則が5箇所に散ると、どこか1つがずれたときに rep 種別で結果が食い違う。
- **時刻を UI に出さず、時間帯フィルタ（`period_of_time_*`）で代用する** — 時間帯は「毎日 09:00〜18:00」の意味で、期間の端だけを切る用途とは違う。「初日は 09:00 から、最終日は 18:00 まで」を表せない。
- **丸めの式を3箇所のまま、それぞれに時刻を足す** — いまの不具合の作り方そのもの。どれか1つを直し忘れると、クリック直後と再検索で結果が変わる。
- **終了時刻 `HH:mm` を `HH:mm:00.000` と読む** — ちょうどその分の記録1件しか入らない。利用者が「18:30 まで」と書いたときの意図から外れる。秒の入力欄を足す案は、`VTimePicker` が秒を持たないので UI を自作することになり、得が小さい。
- **時刻欄に `clearable` を付けず、チェックボックスで「時刻を使う」を切り替える** — 期間フィルタ自身のチェックボックスと二重になり、どちらを外せば何が消えるのか読めない。空欄＝未指定のほうが段数が少ない。
- **着信した日時から時刻欄を常に埋める（00:00 / 23:59 も表示する）** — 既定期間で開いた列に時刻が入っているように見え、「自分で指定した」と区別できない。クリアしても既定へ戻るだけなので害は小さいが、未指定と区別できないほうが問題。
- **新しい i18n キー `CALENDAR_QUERY_PERIOD_TITLE` を足す** — 同じ概念の呼び名変更なので、キーを増やす理由が無い。`CALENDAR_QUERY_TITLE` は `calendar-query.vue` の1箇所でしか参照されていない。キーを増やすと7言語 × `verify_docs` の件数4箇所も動く。
- **地図のスライダーを日数ベースのまま残す** — 時刻範囲を指定しても右半分が無効域になる。無効域の判定が無いので、利用者には「スライダーが効かない」としか見えない。

## Consequences

- **時刻未指定の既定（00:00:00.000 / 23:59:59.999）が後方互換の正本になる。** ここが変わると、保存済みの全列の検索結果が黙ってずれる。`query-composables.test.ts` の「時刻を指定しなければ従来どおり」と `rykv-calendar-period.spec.ts` の「時刻未指定の開始が日の始まり」が固定する。
- **`clearable` を外すと未指定へ戻せなくなる。** E2E の「クリアで空に戻らない（未指定を表せなくなる）」が見張る。Vuetify の clear は `.v-field__clearable` の内側の `.v-icon` が持っているので、ラッパ div を押しても何も起きない（テストを書くときに踏む）。
- **時刻欄のラベルは時間帯フィルタと共有**（「開始」「終了」）。サイドバー全体からラベルで引くと、DOM 順で先に来る時間帯側を拾う。期間節だけに付けたクラス（`calendar_query_start_time_field` / `calendar_query_end_time_field`）で引くこと。
- **props からの逆同期中は emit しない。** 止めないと「着信 → 時刻欄を書き換え → emit → 再検索」で往復が終わらない。`is_syncing_from_props` が1 tick だけ立つ。
- 日付が1つも選ばれていない状態で時刻だけ動かしても何も emit しない（時刻に意味が無いため）。
- 地図の props は `start_date` / `end_date` → `start_time` / `end_time` へ改名した（呼び出しは rykv とダッシュボードの2箇所）。`/api/get_gps_log` のリクエスト欄名（`StartDate` / `EndDate`）は**この変更では触っていない**。秒精度の時刻範囲で効くので動作は正しいが、名前は実態と食い違ったまま残っている。
- `calendar-query.vue` は rykv と mi の共有なので、**片方だけ試して出荷してはいけない**。E2E は両方を見る。

## Evidence

実測なし — 丸めが `use-calendar-query.ts` の3箇所だけに手書きされていたこと、SQL と `find_filter.go` が
秒〜ナノ秒精度で比較していること、地図の再取得 watch が日付文字列の computed に張られていたことからの判断。
既定の保存と、同じ日の中での再取得はユニットテストと E2E で固定した。

## Related tests

- `src/client/__tests__/unit/composables/query-composables.test.ts` — 「useCalendarQuery: 期間の開始時刻・終了時刻」（時刻未指定で 00:00:00.000 / 23:59:59.999・入れた時刻ちょうど・終了は分の末尾まで・着信した時刻で欄が埋まる・時刻だけ動かしても emit する・日付未選択なら emit しない・逆同期では emit しない・クリアで欄も落ちる）
- `src/client/__tests__/unit/composables/gps-log-map.test.ts` — 「useGpsLogMap の time_slider_max」（日単位の指定は従来と同じ刻み・時刻範囲はその秒数）と「useGpsLogMap の再取得」（同じ日の中で開始/終了の時刻だけ動かしても引き直す・刻みが追従する）
- `src/client/__tests__/e2e/rykv-calendar-period.spec.ts` — 「期間」のメニュー名、時刻未指定が日の境界、時刻ピッカーで入れた時刻が条件に乗る、クリアで未指定へ戻る、mi 側にも同じ欄が出る
