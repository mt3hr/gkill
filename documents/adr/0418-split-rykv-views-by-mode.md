# ADR-0418: rykv はモードごとにビューを分け、列の機構を共有の子へ置く

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-23 |
| Sources | `.claude/skills/gkill-client-columns/SKILL.md`「rykv/mi の「列×検索」不変条件」節 / `src/client/pages/views/rykv-columns-view-props.ts` の doc コメント / [ADR-0415](0415-exported-view-is-a-separate-flag.md)（出力モードのフラグ） / [ADR-0407](0407-init-on-application-config-loaded.md)（列の初期化トリガ） |
| Supersedes | [ADR-0415](0415-exported-view-is-a-separate-flag.md) のビュー分岐部分 |
| Superseded-by | なし |
| Anchors | `src/client/pages/views/rykv-columns-view-props.ts`、`src/client/classes/use-rykv-columns.ts` の doc コメント |

## Context

rykv の画面は3つの顔を1つのビューで出していた。

- 普段使う rykv（編集できる）
- 共有ページ（`/shared_page`。読み取り専用で、渡された検索結果を1列で見せるだけ）
- HTML へ出力したファイルの中（読み取り専用で、区画ごとに含めるかを選べる）

分け方はフラグだった。`is_shared_rykv_view` と `is_exported_rykv_view` を props で受け、
テンプレートが `is_readonly` ほか4つの computed を見て区画を出し分ける
（ADR-0415 の決定）。利用者から「分岐が多すぎて保守しづらい。画面ごとに Vue を分けて
分岐しないようにしてほしい」という要望が出た。

数えると、ビューのテンプレートに約26箇所、composable に早期 return が7箇所あった。
しかも**共有のぶんは一度も動かないコード**だった ——
`init()` を早期 return で止め、共有専用の初期化を別に持っていたので、
列の採番・世代番号・列状態の localStorage・サイドバー連携は共有では走らない。
読む人は「共有のときはここを通らない」を常に考えなければならなかった。

一方で、3つの顔が持つものは重なっている。**列の機構は3つとも同じではない** ——
共有は使わないが、通常と出力は**完全に同じ**（検索もサイドバーも、出力では
焼いた記録の中での絞り込みとして動く）。違うのは周りだけで、
アプリバーの中身・編集系のダイアログ群・FAB・ヘルプは通常にしか無い。

## Decision

**顔ごとにビューを分ける。共有は独立、通常と出力は列の子を共有する。**

| ファイル | 役割 | モードの分岐 |
|---|---|---|
| `rykv-view.vue` + `use-rykv-view.ts` | 普段使う rykv。アプリバーの中身・編集系ダイアログ群・FAB・ヘルプ | 無し |
| `exported-rykv-view.vue` | HTML 出力。題名だけ出し、編集系は1つも置かない | 無し |
| `shared-rykv-view.vue` + `use-shared-rykv-view.ts` | 共有ページ。1列固定の独立実装（列の機構を使わない） | 無し |
| `rykv-columns-view.vue` + `use-rykv-columns.ts` | 列のテーブル・検索・列状態の復元・サイドバー連携・詳細ペイン・Ryuu・集計・マップ・件数カレンダー | 無し |

ADR-0415 の決定のうち、置き換わるのは**テンプレートが見ていた4つの computed だけ**。通信を `gkill_fetch` の1点で止める決定と、出力用 API が共有用 API を継承する形はそのまま残る。

**モードのフラグは1つも残さない。** 何ができるかは、ホストする側が子へ
**リテラルで**渡す（`:enable_context_menu="false"`、`:show_column_footer="false"` など）。
先例は共有 mi（`shared-mi-view.vue`）で、そこは以前からこの作法だった。

ラッパが持つものは**スロット**で差し込む（`app_bar_title` / `app_bar_actions` /
`dialogs` / `fab`）。**`dialogs` スロットには中継束と列の状態をスロット props で渡す** ——
保存した記録は再検索せず列へ差し込む（ADR-0402）ので、ダイアログの emit が
列の中継束へ流れていなければ「追加した記録が一覧に出ない」になる。

出力で選ばれなかった区画のフラグ（`is_exported_with_dnote` ほか3つ）だけは残るが、
これは**モードではなく焼き込んだ内容そのもの**で、出力のラッパの中だけに居る
（含めていない区画のデータはファイルの中に無い）。

## Rejected alternatives

- **フラグのまま、分岐を computed へ寄せ続ける（ADR-0415 の形）** — 3つ目の顔（出力）を足した時点で、区画の軸が「読み取り専用か」「集計を出すか」「マップを出すか」「サイドバーを出すか」の4つに増えた。顔が増えるたびに軸も増え、テンプレートの各行が「どの顔で出るのか」を読み解く対象になる。利用者の「分岐が多すぎる」はこの状態を指している。
- **3つのビューへ全部複製する** — 列の機構（検索の世代番号・query_id の採番・localStorage の復元・サイドバー連携）が2箇所になる。この機構は「例外もエラーも出さずに静かに壊れる」不変条件の塊（ADR-0405 / ADR-0407）で、片方だけ直す事故が確実に起きる。実際 rykv と mi の対称実装でそれが何度も起きている。
- **列の子を、通常・出力・共有の3つで共有する** — 共有は列の機構を1つも使わない。使わせると、使われない機構を「共有のときは通らない」ために読み続けることになり、分割前と同じ状態が子の中に移るだけ。共有は独立実装にして、列の機構を**持たせない**。
- **ラッパが子のメソッドをテンプレート ref 経由で呼ぶ（スロットを使わない）** — 編集系ダイアログの保存を列へ届けるために、ラッパ → 子の呼び出しが10本前後必要になる。どれも「落としても型では落ちず、画面が静かに更新されなくなる」たぐい。スロット props なら中継束をそのまま渡せるので、配線の本数が増えない。
- **アプリバーごとラッパへ持たせる** — 詳細・集計・マップ・件数カレンダーのトグルは3つの顔で同じで、しかも状態は列の子が持つ。ラッパへ出すと「ラッパが持つ4つの真偽値を子へ props で降ろす」配線が増える。**同じものはラッパへ出さない**。アプリバーの骨格は子に置き、顔ごとに違う題名と右側のボタンだけをスロットにした。
- **出力のラッパも composable を持つ** — 出力のラッパは題名を出して子へリテラルを渡すだけで、状態も処理も無い。`.vue` 1枚（約40行）で足りる。

## Consequences

- **列の機構の正本は `use-rykv-columns.ts` になった。** `mi` の対になる実装は引き続き `use-mi-view.ts`（mi は共有も出力も無いので分割していない）。**対称実装の修正は必ず両方へ**という約束は変わらないが、見比べる相手のファイル名が変わった。ADR-0401 / ADR-0405 / ADR-0407 / ADR-0415 が `use-rykv-view.ts` を Anchors / Sources に挙げているのは、現在はこのファイルを指す。
- **E2E が見るセレクタは変えていない**（`.rykv_view_wrap` / `[data-gkill-view-ready]` / `.kyou_detail_view` / `.gkill-floating-dialog*`）。準備完了信号を出す要素は列の子へ移ったので、ソース走査テストの対象ファイル一覧も合わせた。
- **出力のバンドルから編集系のビューが落ちる。** 出力のラッパは編集系のダイアログを1つも import しないので、`exported-kyou-page` 側のチャンクグラフに入らない（`RykvDialogHost` 経由で入るものは残る）。
- サイドバーの入れ物は**出力でも mount したままにする**という ADR-0415 の帰結はそのまま残る。列の子の中で `v-if` を付けないこと（付けると列が1本も作られず読み込み中オーバーレイが消えない）。
- 常に `true` だった見せかけのフラグ（`enable_context_menu` / `enable_dialog` の `ref(true)`）を rykv と mi の両方から外し、リテラルで書くようにした。読む人が「どこで false になるのか」を探さなくて済む。

## Evidence

実測なし。分岐の数（テンプレート約26箇所・composable の早期 return 7箇所）はソースを数えたもの（コード由来）。
分割の前後で、lint・型検査・フロントエンドのユニットテスト・E2E 全件が通ることを確かめた。

## Related tests

- `src/client/__tests__/unit/classes/column-view-init-source-scan.test.ts` — 列の初期化の不変条件。対象は `use-rykv-columns.ts` と `use-mi-view.ts`（対称実装）、準備完了信号は `rykv-columns-view.vue`
- `src/client/__tests__/unit/composables/shared-rykv-view.test.ts` — 共有の独立実装（取得は1回だけ・スピナー・地図の時間窓）
- `src/client/__tests__/unit/composables/rykv-view-initial-load.test.ts` / `rykv-view-search-routing.test.ts` / `rykv-view-map-date-picker.test.ts` — 列の機構（`useRykvColumns` を直接呼ぶ）
- `src/client/__tests__/e2e/kyou-export-html.spec.ts` — 出力したファイルを `file://` で開く通し
