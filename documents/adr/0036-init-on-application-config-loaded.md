# ADR-0036: 列ビューの初期化トリガはサイドバーの @inited ではなく ApplicationConfig.is_loaded

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-17 |
| Sources | `a07f4a5a` / `f4c2b7c3` / `.claude/skills/gkill-client-columns/SKILL.md`「init() の起動条件は props.application_config.is_loaded の watch」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/use-rykv-view.ts` |

## Context

rykv / mi の `init()` はサイドバーの `@inited` で起動していた。一見すると「サイドバーの準備ができたら初期化する」で筋が通っている。

**実際には偶然だった。** `@inited` は子クエリビューの「その節が描けた」の集約でしかなく、「設定が来た」を表してはいない。それが機能していたのは「`immediate` の付いていない `application_config` watch から emit する子がいる」という偶然による。

mi では実質 CalendarQuery 1つが律速で、しかもその節は `application_config` のフィールドを1つも読まない。**節を1つ画面から外すだけで画面ごとスピナーで固まる。**

## Decision

`init()` の起動条件を **`props.application_config.is_loaded` の watch** にする。

## Rejected alternatives

- **サイドバーの `@inited` を使い続ける（改修前）** — 表しているものが違う。子ビューの構成を変えるだけで壊れ、しかも壊れ方は「画面がスピナーで固まる」なので原因が見えない。

- **サイドバーの `inited` 集約そのものを消す** — 集約は消してよいが、**`inited_*_for_query_sidebar` の各フラグは残す**。子へ `:inited` prop として降り、子が「初回同期か再同期か」を判定している。消すと props 同期のたびにチェックが列をまたいで累積する。

- **`onMounted` で無条件に初期化する** — `ApplicationConfig` が未ロードのまま既定クエリを作ると、`generate_default_query_for_rykv` が読む `device_struct` / `rep_type_struct` / `tag_struct` / `rykv_default_period` / `hide_tags` が空になり、**既定期間と強制非表示タグが黙って落ちる**（→ ADR-0035）。

- **設定取得が失敗したら黙って既定値で進む** — 失敗の事実が利用者に伝わらない。`application_config_load_failed` を立ててエラー表示＋再試行ボタンにする（文言は既存の `FAILED_GET_APPLICATION_CONFIG_MESSAGE` / `RELOAD_TITLE`）。

## Consequences

「イベント名が表しているもの」と「実際に待ちたいもの」がずれていた、という形の不具合なので、**同種のものが他にもありうる**。初期化の起動条件を書くときは「このイベントは本当に待ちたい事実を表しているか」を確認すること。

起動条件がソース走査で固定されている（`column-view-init-source-scan.test.ts`）。

## Evidence

実測なし — 構造からの判断（`@inited` の発火が `application_config` と因果でつながっていないことをソースで確認した）。

症状は実機で観測（mi でサイドバーの節を1つ外すと画面ごとスピナーで固まる）。

## Related tests

- `src/client/__tests__/unit/composables/mi-sidebar-inited.test.ts`
- `src/client/__tests__/unit/composables/rykv-view-initial-load.test.ts`
- `src/client/__tests__/unit/composables/mi-view-initial-load.test.ts`
- `src/client/__tests__/unit/classes/column-view-init-source-scan.test.ts`
