# ADR-0035: rykv/mi/dashboard は初期取得の完了を待たずに画面を可視化する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-17 |
| Sources | `5ab34d74` / `13818a0d` / `a5c913bf` / `.claude/skills/gkill-client-columns/SKILL.md`「rykv/mi/dashboard の初期化順序」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/use-dashboard-page.ts` |

## Context

初期取得の完了まで全画面をオーバーレイで隠していた。そのため**検索が1本でも解決しないだけで画面全体が固まる**。プラグインが詰まった、rep が1つ重い、といった局所的な事情が全画面の停止になっていた。

## Decision

順序を **列の骨組みを確定 → 可視化 → 検索** にする。`inited` / `is_loading` は初期検索の完了に依存させない。全画面オーバーレイは `ApplicationConfig` の待ちだけに縮める。

## Rejected alternatives

- **初期取得の完了まで隠す（改修前）** — 検索1本の遅延が全画面の停止になる。

- **オーバーレイを完全に無くす** — `ApplicationConfig` の待ちは残す。`generate_default_query_for_rykv` が `device_struct` / `rep_type_struct` / `tag_struct` / `rykv_default_period` / `hide_tags` を読むので、未ロードで既定クエリを作ると**既定期間と強制非表示タグが黙って落ちる**。保存済みクエリがある通常ケースは localStorage 由来なので影響しないぶん、**初回起動のユーザだけが踏む**。

- **列を1本ずつ足しながら検索する** — 「列が確定した瞬間」が定義できない。復元中にユーザが列を足すと `search(i, ...)` の固定 index と衝突する。`init()` は hot reload の ON/OFF で分岐せず、`querys` / `querys_backup` / `match_kyous_list` を検索より前に確定させる。

- **`skip_search_this_tick` を初期化全体の門番に流用する** — あれは「1tick分の残響を捨てる」短命フラグ。流用すると機械的な emit が1つ届いただけで `onSidebarUpdatedQuery` が消費し、複数列のとき1列目の完了で**抑止が途中で解ける**。抑止は `run_with_sidebar_search_suppressed` だけを使う。

- **`onSidebarUpdatedQuery` に「初期化が終わるまで捨てる」早期returnを置く** — 初期検索の飛行中でも**ユーザの編集は通す**。同じ `query_id` を共有するので `abort_controllers` が復元を中断し、`search_seqs` の世代照合が遅れて届いた復元結果を捨てる（＝ユーザが勝つ。→ ADR-0034）。

- **E2E の準備完了信号に全画面オーバーレイのスピナーを使い続ける** — 全画面オーバーレイが初期検索を待たなくなったので、`.v-overlay .v-progress-circular` の `.first()` は列スピナーを掴み「出る前に `toBeHidden` が通る」窓ができる。ルート要素の `data-gkill-view-ready` を待つ。**真偽値をそのまま bind してはいけない**（Vue は false のとき属性ごと消すので「属性の有無」で判定するセレクタが壊れる）。

## Consequences

復元の検索は **`preserve_scroll=true`** で呼ぶ。`inited` が早期に立つので、落とすと `search()` が `scroll_to(0)` を撃って保存済みの復元先を潰す。

`onColumnScrollList` は**検索中の列の通知を捨てる**。リストを空にした副作用のスクロール通知を取り込むと `preserve_scroll` の復元先が0で潰れ、保存位置にも焼き付く。

**設定取得の失敗は永久スピナーにしない。** `load_application_config()` は `res.errors` で早期returnし ref を差し替えないうえ `.catch()` も無かったので、失敗すると `is_loaded` が永久に false で画面が固まっていた。`application_config_load_failed` を立ててオーバーレイの中身をエラー＋再試行ボタンへ差し替える。

サイドバーの `inited` 集約は無くしたが、`inited_*_for_query_sidebar` の各フラグは**残す**。子へ `:inited` prop として降り、子が「初回同期か再同期か」を判定している（消すと props 同期のたびにチェックが列をまたいで累積する）。

## Evidence

実測なし — 構造からの判断（`query_id` キーの abort と世代照合・機械的emitの抑止が揃ったので、可視化を前倒ししても誤配送が起きない）。

## Related tests

- `src/client/__tests__/unit/composables/rykv-view-initial-load.test.ts`
- `src/client/__tests__/unit/composables/mi-view-initial-load.test.ts`（rykv と対）
- `src/client/__tests__/unit/classes/column-view-init-source-scan.test.ts`（ソース走査）
- `src/client/__tests__/e2e/column-view-initial-load.spec.ts`
