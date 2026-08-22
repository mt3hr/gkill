---
name: gkill-find-query
description: "検索条件 FindQuery の null 判定セマンティクス。null=フィルタ未使用・非nullの空配列=0件指定、use_* フラグ全廃、ゲートヘルパ経由の判定、TypeScript 側の undefined 禁止、旧形式JSONの移行3実装（Go の find_query_legacy_json.go / client の normalize-legacy-find-kyou-query-json.ts / MCP の constants.mjs）を扱う。src/server/gkill/api/find/・find_filter.go・src/client/classes/api/find_query/・src/mcp/lib/constants.mjs を触るとき、「条件を足したら0件になった」「フィルタが効かない」を調べるとき必読。"
---

# FindQuery の null 判定セマンティクス（Go / TypeScript / MCP 共通）

対象: `src/server/gkill/api/find/**` / `src/server/gkill/api/find_filter.go` / `src/client/classes/api/find_query/**` / `src/mcp/lib/constants.mjs`

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

**検索条件（FindQuery）の null 判定セマンティクス:** かつて存在した `use_*` 有効化フラグ（14個）は**全廃**され、いまは **値フィールドが非null（Go では非nil）ならそのフィルタが有効**。`FindQuery` は 55→41 フィールド。間違えると例外もエラーも出ずに静かに 0 件になるので、以下は規約として守ること。
- `null` / `nil` = フィルタ未使用、**非nullの空配列 `[]` = フィルタ有効かつ0件指定**。唯一の例外は `timeis_words: []` で「任意の TimeIs に覆われた Kyou」を意味する
- 3値そろって初めて有効になるグループがある（地図の `map_latitude` / `map_longitude` / `map_radius`）。Go 側は `HasWordFilter()` / `HasTimeIsFilter()` / `HasCalendarFilter()` / `HasMapFilter()` / `HasPeriodOfTimeFilter()` の**ゲートヘルパ経由で判定する**（生の nil 比較を書き散らさない）
- `PeriodOfTimeWeekOfDays` は **nil を先行ガードで弾く**こと。`len==0` / `len!=7` の分岐へ落とすと全件が消える（`find_filter.go` の `sortAndTrimKyousMap` と `sqlite3impl_util.go` の両方に同じ罠がある）
- TypeScript 側で `undefined` は禁止。`JSON.stringify` でキーが落ち、localStorage 往復でコンストラクタ既定値が復活し、`deep_equals` のキー数比較が壊れてサイドバーの機械的 re-emit ガードが死ぬ。未使用は必ず `null` で表現する。**禁止の対象は「永続化・比較される値」**（`FindQuery` のフィールド、localStorage へ入るオブジェクト、`deep_equals` に掛ける値）。関数の省略可能引数（`show(query?: FindKyouQuery)`）や Vuetify の `:color="… ? 'error' : undefined"`（prop 既定値を効かせる用法）は対象外
- ただし `FindKyouQuery` のコンストラクタ既定は `tags` / `reps` だけ **`null` ではなく `[]`**（旧 `use_tags=true` + 空配列と厳密等価にするため）
- Mi の板名は `mi_board_name: null` が「すべて」。番兵は `classes/mi-board-names.ts` の **`MI_ALL_BOARD_KEY`（= ハードコードの `"すべて"`。ロケール非依存）** でサイドバー専用、null への変換は `use-mi-query-editor-sidebar.ts` の1点に集約されている。**i18n の訳語（`MI_ALL_BOARD_NAME_TITLE`）と比較してはいけない** ―― ツリーが emit するのはノードの `key` で、それは `append_all_mi_board()` が入れた `"すべて"` 固定なので、訳語と比べると日本語以外のロケールで「すべて」が全件に戻らず 0 件になる（表示名だけが `ALL_MI_BOARD_NAME` / `MI_ALL_TITLE`）
- 旧形式JSONの移行は3実装が**同じ16キー**を扱う: Go `api/find/find_query_legacy_json.go`、client `classes/api/find_query/normalize-legacy-find-kyou-query-json.ts`、MCP `mcp/lib/constants.mjs` の `LEGACY_USE_FLAG_KEYS`。どれかが欠けると、そのフラグを送る古いクライアントの保存クエリが移行されない（MCP では未知キー扱いで throw する）。共有URL用の `share_kyou_info.db` は起動時にスキーマ 1.0.0→1.1.0 で**保存済みJSONそのものを書き換える**（共有URLは配布済みで再発行できないため） 却下案（フラグを残す／値が空ならフラグを無視する）は [ADR-0006](../../../documents/adr/0006-find-query-null-semantics.md)。

## 関連スキル

- [gkill-go-backend](../gkill-go-backend/SKILL.md) — 検索フィルタの実装側（rep名・タグ・IDsチャンク）
- [gkill-client-columns](../gkill-client-columns/SKILL.md) — 局所挿入の判定は `find_filter.go` の意味論の写し
- [gkill-mcp](../gkill-mcp/SKILL.md) — MCP の旧形式キー移行（`LEGACY_USE_FLAG_KEYS`）

## 詳しい設計と却下案（ADR）

- [ADR-0006 FindQuery の null 意味論](../../../documents/adr/0006-find-query-null-semantics.md)
