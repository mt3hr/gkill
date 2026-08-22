# ADR-0034: 列の同一性は query_id、検索は世代番号で最後の1回だけ書き戻す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-09 |
| Sources | `CLAUDE.md`「rykv/mi の『列×検索』不変条件」節 / `src/client/classes/use-rykv-view.ts` / `src/client/classes/use-mi-view.ts` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/use-mi-view.ts` |

## Context

rykv / mi は複数の列を並べ、それぞれが独立に検索を走らせる。ここで**「検索結果が別の列に出る」誤配送**が繰り返し起きていた。

原因は「列をどう識別するか」が場所ごとにばらばらだったこと。配列の添字で識別すると列の増減でずれ、オブジェクト参照で識別すると再生成で切れる。

## Decision

列の同一性は **`query_id`**（**列の誕生時のみ採番、以後不変**）。`:key`・DOM id・テンプレート ref 逆引き（`get_kyou_list_view`）・`abort_controllers` / `search_seqs`（Map）のキーは全部これ。

検索は列ごとの**世代番号 `search_seqs`** で「最後の検索だけが書き戻せる」。

## Rejected alternatives

- **列リロード・画像トグル・サイドバーの clear 系で `query_id` を再採番する** — 列が remount され、**検索結果の帰属も切れる**。誤配送が再発する。

- **配列の添字（`column_index`）で識別する** — 列を挿入・削除すると全部ずれる。とくにサイドバー編集の宛先は `focused_column_index` ではなく **`querys.findIndex(query_id)`** で解決しなければならない。

- **飛行中の検索を abort せず、全部書き戻す** — 遅れて着地した古い結果が新しい結果を上書きする。世代番号で最後の1回だけ通す。

- **`focused_query` を無条件で更新する** — サイドバーが別列の条件に乗っ取られ、`query_id` の重複 → 誤配送になる。更新してよいのは**フォーカス列の検索だけ**。

- **サイドバーの `emits_current_query` を常に emit する** — フォーカス切替で子ビューのprops同期の残響が機械的に届き、それが検索になる。「検索中の列をクリック→飛行中の検索が abort され最初からやり直し」が再発する。**再生成結果が同期済みクエリと同値なら emit しない**（値比較ガード）。

- **フォーカス切替時の検索抑止をフラグの直接操作で書く** — 順序が本質。`skip=true` → `fn()` でリアクティブ書き込み → `nextTick(解除)` の順でなければならない。書き込みより先に `nextTick` を登録すると、Vue の `nextTick` が resolvedPromise へ直結して解除がウォッチャ flush より先に走り、**抑止が一度も効かない**（2026-08-10 のタブフリーズ回帰の正体）。コールバック式 `run_with_sidebar_search_suppressed(fn)` だけを使う。

## Consequences

検索ボタンはサイドバーの `generate_query(列のquery_id)` で「今見えている条件」から検索する（`rykv_hot_reload` OFF 時は編集が列に保存されないため）。`generate_query` は同期済みクエリに対して**恒等**であること —— とくに `include_*_mi` を true 固定でドリフトさせない。

サイドバーの子クエリビュー（Rep/Tag/TimeIs/Map/Calendar）は **props 同期では emit しない**。TimeIs は同期経路に `disable_emits=true` + `pre_uncheck_all=true`、Map は同期時 emit なし + radius ウォッチャの値ガード、Calendar は `clicked_date` の同値エコーガード。

**`use-rykv-view.ts` と `use-mi-view.ts` はコピー由来の対称実装なので、修正は必ず両方へ。**

## Evidence

実測なし — 構造からの判断（識別子が場所ごとに違うと帰属が切れる）。

症状は実機で繰り返し観測されていた（「検索結果が別の列に出る」「検索中の列をクリックするとやり直しになる」）。

## Related tests

- `src/client/__tests__/unit/composables/rykv-view-search-routing.test.ts`
- `src/client/__tests__/unit/composables/mi-view-search-routing.test.ts`（rykv と対）
- `src/client/__tests__/unit/composables/rykv-sidebar-mechanical-emission.test.ts`
- `src/client/__tests__/unit/composables/sidebar-child-query-sync-emission.test.ts`
