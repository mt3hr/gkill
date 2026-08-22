# ADR-0033: 利用者がその場で作ったタグだけを、開いている列の検索条件へ足す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-19 |
| Sources | `a510fbc2` / `CLAUDE.md`「利用者がその場で作ったタグは、開いている列の検索条件へ足す」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/use-registered-tag-column-filter.ts` |

## Context

直している不具合は **「タグを付けて追加した記録が、追加した直後に一覧から消える」**。

既定クエリは「絞らない」を `tags = null` ではなく、**そのときの `check_when_inited` タグ名の列挙**として物質化する（`find-kyou-query.ts` の `generate_default_query_for_rykv`）。それが localStorage の列状態へ丸ごと落ちる一方、タグ宇宙（`tag_struct`）は毎回サーバから引き直して育つので、**列の条件だけが保存時点で凍る**。

タグが1つも無い時期に作られた列は `tags = ["no tags"]` の1件だけになり、`tags` は非nullなのでフィルタは有効 ＝ **タグの付いた記録は1件も通らない**。サーバ検索（`find_filter.go`）と局所挿入（`kyou-local-insert.ts`）の両方で落ち、**エラーも警告も出ない**。

## Decision

**「そのタグがタグツリーに無かった」という決定可能な事実だけ**を根拠に、そのタグを開いている列の条件へ足す。未知だった ＝ 利用者がついさっき作った ＝「意図的にチェックを外した」ことは原理的にありえない。

実装は `use-registered-tag-column-filter.ts` の1つで、rykv / mi が同じものを使う。

## Rejected alternatives

- **既知のタグも足す** — 「保存後に増えたタグ」と「利用者が意図的に外したタグ」は、現状の保存データでは**区別できない**。足すと利用者が外したはずのタグが復活する。

- **`tags_and === true` の列にも足す** — **列を丸ごと消す。** AND は `query.tags.every(...)` なので `["no tags", "新タグ"]` の積は必ず空になる（新タグ付きの記録は `has_no_tags === false` で落ち、タグ無しの記録は `has_tag_name(新タグ)` で落ちる）。目当ての記録も救えない。`tags === null` の列も触らない（元から通る）。

- **判定を `registered_tag` の emit 後に行う** — emit 先の `check_tag_update` がタグツリーへ足したあとでは、「ついさっき作った」ことを**二度と知れない**。判定は emit より前に、同期で行う。

- **localStorage へ自分で書く** — 「条件だけ変わって引き直さない」経路が生まれ、**次回起動時だけ列が変わる**という最悪の非対称になる。`search()` が必ず `set_saved_*` を通るので、引き直しを通せば揃う。

- **タグ1件ごとに `search()` を走らせる** — `add_tags_to_target` はタグを1件ずつ登録して1件ずつ emit するので、新タグ3つで列あたり3本 `search()` が走り2本が abort されるだけになる。1tick ぶん溜めてから1回だけ流す。

- **本命（既定クエリの物質化をやめて `tags = null` にする）を今やる** — 強制非表示タグの意味論が `query.tags` の中身に依存している（`kyou-local-insert.ts` の `!checked_tag_names.includes(...)` ＝「チェックされていなければ隠す」）ため、Go とクライアントの対称な再定義が要る**別件**。

## Consequences

ポート（rudbeckia）へも配る。タグが検索条件に効くようになったので `kyou-change-bus.ts` の「タグは配る必要が無い」という前提が崩れた。publish するのは**未知と判定した発生元だけ**で、受け手は既知判定を**やり直さない**（届く頃にはツリーに載っているので必ず取りこぼす）。

`querys` と `querys_backup` を同じ tick で揃え、`run_with_sidebar_search_suppressed` で包む。控えがずれるとサイドバーの機械的な残響が `search()` の `deep_equals` 早期returnで落ちなくなる。

**この修正では直らないもの**（区別できる情報が保存データに無い）: 他端末で作られたタグ、過去に作ったタグ、プラグインKyouのタグ。

## Evidence

実測なし — 決定可能性からの判断（「ツリーに無かった」だけが観測可能な事実）。

症状は実データで再現する（タグが1つも無い時期に作られた列は `tags = ["no tags"]` で凍っている）。

## Related tests

- `src/client/__tests__/unit/composables/registered-tag-column-filter.test.ts`
- `src/client/__tests__/unit/composables/new-tag-column-search.test.ts`（rykv・mi 両方）
- `src/client/__tests__/unit/classes/column-view-init-source-scan.test.ts`（emit より前に判定していることをソース走査で固定）
- `src/client/__tests__/unit/classes/kyou-change-bus.test.ts`
- `src/client/__tests__/e2e/regression-fixes.spec.ts`（**画面遷移しないことが本質** —— 遷移すると既定クエリを作り直すのでこの不具合をすり抜ける）
