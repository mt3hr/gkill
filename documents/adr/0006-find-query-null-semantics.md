# ADR-0006: FindQuery の use_* 有効化フラグを全廃し「値が非nullならフィルタ有効」に一本化する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-11 |
| Sources | `de0f330c` / `397d5100` / `.claude/skills/gkill-find-query/SKILL.md`「検索条件（FindQuery）の null 判定セマンティクス」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/find/find_query.go` |

## Context

`FindQuery` には値フィールドとは別に `use_*` 有効化フラグが14個あった。「フィルタを使うか」と「フィルタの値」を二重に持つ形で、どちらかを書き忘れると**例外もエラーも出ずに静かに0件になる**。

とくに危険だったのは「フラグは true なのに値が空」「値は入っているのにフラグが false」の2状態で、どちらも呼び出し側からは「該当0件」と見分けが付かない。

## Decision

`use_*` フラグを全廃し、**値フィールドが非null（Go では非nil）ならそのフィルタが有効**に一本化する。`FindQuery` は 55 → 41 フィールドになった。

- `null` / `nil` ＝ フィルタ未使用
- **非nullの空配列 `[]` ＝ フィルタ有効かつ0件指定**

判定は生の nil 比較を書き散らさず、`HasWordFilter()` / `HasTimeIsFilter()` / `HasCalendarFilter()` / `HasMapFilter()` / `HasPeriodOfTimeFilter()` の**ゲートヘルパ経由**で行う。

## Rejected alternatives

- **フラグを残して「値が空ならフラグを無視する」ルールにする** — 「フィルタ有効かつ0件指定」が表現できなくなる。これは実在する状態で、rykv がタグを1つもチェックしていない列でまさにこれを送る。

- **フラグ側を正として値を無視する** — 二重管理が残るので、書き忘れの静かな0件も残る。

- **TypeScript 側で `undefined` を「未使用」に使う** — **禁止。** `JSON.stringify` でキーが落ち、localStorage 往復でコンストラクタ既定値が復活し、`deep_equals` のキー数比較が壊れてサイドバーの機械的 re-emit ガードが死ぬ。未使用は必ず `null` で表現する。

- **旧形式JSONの移行を実装1つに任せる** — 移行は Go・client・MCP の**3実装が同じ16キー**を扱う必要がある（`find_query_legacy_json.go` / `normalize-legacy-find-kyou-query-json.ts` / `constants.mjs` の `LEGACY_USE_FLAG_KEYS`）。どれかが欠けると、そのフラグを送る古いクライアントの保存クエリが移行されない（MCP では未知キー扱いで throw する）。

## Consequences

意味論の例外が2つ残る。どちらも「揃えるほうが壊れる」ので意図的。

- `timeis_words: []` は「任意の TimeIs に覆われた Kyou」を意味する（空配列＝0件指定ではない）
- `FindKyouQuery` のコンストラクタ既定は `tags` / `reps` だけ **`null` ではなく `[]`**（旧 `use_tags=true` ＋ 空配列と厳密等価にするため）

3値そろって初めて有効になるグループがある（地図の `map_latitude` / `map_longitude` / `map_radius`）。

`PeriodOfTimeWeekOfDays` は **nil を先行ガードで弾く**こと。`len==0` / `len!=7` の分岐へ落とすと全件が消える（`find_filter.go` の `sortAndTrimKyousMap` と `sqlite3impl_util.go` の両方に同じ罠がある）。

共有URL用の `share_kyou_info.db` は起動時にスキーマ 1.0.0→1.1.0 で**保存済みJSONそのものを書き換える**（共有URLは配布済みで再発行できないため）。

## Evidence

実測なし — 構造からの判断（二重管理の不整合が原理的に表現できなくなる）。

## Related tests

- `src/server/gkill/api/find/find_query_test.go`
- `src/server/gkill/api/find/find_query_legacy_json_test.go`
- `src/server/gkill/api/find_filter_sort_trim_test.go`
- `src/client/__tests__/unit/api/normalize-legacy-find-kyou-query-json.test.ts`
- `src/client/__tests__/unit/api/find-kyou-query.test.ts`
