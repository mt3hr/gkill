# ADR-0417: 出力 HTML のサイドバーは絞り込み専用にし、ワード照合の4実装目を作らない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-22 |
| Sources | `src/client/classes/kyou-local-insert.ts` の `does_kyou_match_query` / `can_decide_query_locally` / [ADR-0402](0402-insert-registered-kyou-locally.md)（判定できない条件を判定できると宣言しない） / [ADR-0113](0113-word-filter-columns-and-id-prefix.md)（ワード照合の実装を増やさない） / [ADR-0415](0415-exported-view-is-a-separate-flag.md)（出力モードと通信の1点止め） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/api/gkill-api.ts` の `narrow_column_kyous` の doc コメント、`src/client/classes/exported-kyou-payload.ts` の注入部、`src/client/pages/views/rykv-query-editor-side-bar.vue` の `v-show` のコメント |

## Context

「HTML で出力」した画面にもサイドバーを出せる（出力時に選ぶ）。出力先には gkill が居ないので、
絞り込みはクライアントでやるしかない。

ここには2つの危険がある。

1. **ワード照合の実装がもう3つある**（SQL / Go の `find_word` / プラグイン SDK）。
   TS で4つ目を書くと、ずれたときに rep 種別ごとに結果が黙って食い違う。
2. **絞り込みの起点を間違えると、焼いていない記録が出うる。** 共有の境界線はここ1本しかない。

## Decision

**出力先のサイドバーは、焼いた記録を絞ることしかできない。**
どんな条件を入れても、焼かれていない記録が現れることはない。

実装は「`get_kyous` が必ずその列の焼いた配列を起点にする」ことで担保する。
`GkillAPIForExportedKyou` は絞り込みの**口だけ**を持ち（`narrow_column_kyous`）、
`build_exported_api` が注入する。

判定は `kyou-local-insert.ts` の `does_kyou_match_query` を**そのまま使う**。
これは `find_filter.go` の意味論の写しとして既にあるもので、出力のためにもう1つ書くと3実装目になる。
ワード・状況・場所・playing は見ない（Kyou 単体では判定できない条件。
`can_decide_query_locally` が同じ線引きを持っている）ので、それらを理由に記録が落ちることはない。

サイドバーでは、判定できない条件の節を**隠す**（`application_config.for_export_kyou`）。
保存済み検索条件の FAB も隠す（出力先では開いても意味が無い）。

**出力した時点の条件そのままなら、絞り込みを通さず焼いた配列をそのまま返す。**
サーバが本文一致で入れた記録を「この条件では一致しない」と判定して、出力直後の画面が欠けるのを防ぐ。

**サイドバーのチェックツリーは焼いたデータから組み直す**（`use-exported-kyou-page.ts` の
`build_sidebar_structs`）。利用者の設定した木をそのまま焼くと、出力に含めなかった
タグ名・rep 名まで共有先に見える。

## Rejected alternatives

- **出力専用の絞り込みモジュールを新規に書く**（当初案） — `does_kyou_match_query` と同じ判定の2実装目になる。`find_filter.go` を直したとき、片方だけ直す形で静かにずれる。
- **ワード検索も TS で実装する** — ワード照合の4実装目。ずれたときは「rep 種別ごとに結果が食い違う」形で出て、どれが正しいのか分からなくなる（ADR-0113 で同じ事故を踏んでいる）。
- **`gkill-api.ts` から `kyou-local-insert.ts` を import する** — データクラスと req_res がこのファイルと循環 import になっており、単一チャンクへ畳む出力ビルドでは import を1本足すだけで評価順が変わって `class extends value undefined` になる（ADR-0903）。口だけ持たせ、注入する側から import する。
- **判定できない節を `v-if` で落とす** — `generate_query()` が各節の template ref から値を集めるので、null 参照になる。`v-show` で隠し、節は焼いた条件のまま残す。
- **利用者の設定した木をそのまま焼く** — 階層は保てるが、出力に含めなかったタグ名・rep 名が共有先に残る。階層名そのものが個人情報になりうる。
- **木を組む前に画面を描き、あとから木を差し込む** — サイドバーの各節が「チェック0個」を生成して emit し、`tags = []` / `reps = []`（非 null の空配列 = 0件指定）の条件で**列が全部空になる**。

## Consequences

- **`tags` / `reps` の既定は `[]`（0件指定）であって `null` ではない。** サーバが焼く列の条件には、サイドバーが生成するのと同じ形（チェック済みの全タグ＋`no tags`、全 rep）を載せること。欠けた条件を焼くと、サイドバーが初回に生成した条件と食い違った瞬間に全部消える。
- **タグは焼き込みから先に埋める**（`attached_tags` と `is_attached_tags_loaded`）。遅延ロード待ちのまま絞ると「まだ読んでいないので持っていない」と判定してタグ条件が全部落ちる。
- 木には「タグ無し」の仮想タグを入れる。無いと、タグの付いていない記録が「チェックされていないタグ」扱いで一覧から消える。
- 木の階層は落ちてフラットになる。
- サイドバーを含めずに出力したときは注入が無く、`get_kyous` は焼いた配列をそのまま返す。

## Evidence

実測なし —— `does_kyou_match_query` が `find_filter.go` の写しとして既にあること、
ワード照合が3実装あることからの判断。
「木が空のまま画面を描くと列が空になる」「木のルートが葉のままだと子が描かれない」は、
ブラウザで出力 HTML を開いて分かった。

## Related tests

- `src/client/__tests__/unit/classes/exported-kyou-payload.test.ts` — 「`build_exported_api` が注入する絞り込み」（タグ・記録保管場所・期間で絞れる／ワード条件は見ない／**どんな条件でも焼いた配列の部分集合しか返さない**／出力時の条件そのままなら焼いた配列をそのまま返す／タグを先に埋める）
- `src/client/__tests__/unit/api/exported-kyou-api.test.ts` — 「絞り込みの注入には、その列の焼いた配列が渡る」「渡るのは画面の今の条件と焼いた条件の2つ」「注入が無ければ焼いたまま返す」
- `src/client/__tests__/unit/composables/exported-kyou-page.test.ts` — 「サイドバーのチェックツリーは焼いたデータから組む」（木が組み終わるまで `is_ready` が立たない／焼いたタグと「タグ無し」が出る／焼いていないタグは出ない／ルートが入れ物になっている）
