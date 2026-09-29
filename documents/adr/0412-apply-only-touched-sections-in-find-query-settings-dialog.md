# ADR-0412: 設定の「検索条件」ダイアログは、適用で触ったセクションだけを渡す（旧ダッシュボードダイアログが未設定を空の条件で書き潰していた）

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-23 |
| Sources | `756ad897` / `.claude/skills/gkill-client-foundation/SKILL.md`「設定は「適用」を押すまでサーバへ送らない」節 / [ADR-0106](0106-find-query-null-semantics.md)（空配列＝0件指定） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/use-edit-saved-find-query-dialog.ts` の `useEditSavedFindQueryDialog` の doc コメント |

## Context

設定画面には検索条件にかかわるダイアログが3つあった。「検索条件」（保存済みの検索条件＝検索ショートカット）、
「実行中」（実行中ビューの条件）、「ダッシュボード」（集計とタスクの条件）で、それぞれ別のボタンと別の適用・キャンセルを持ち、
設定画面のあちこちに散っていた。設定の子ダイアログの適用は「設定画面の clone へ組み立てるだけ」で、サーバへ送るのは
設定画面の「適用」1か所（gkill-client-foundation スキル）なので、子ダイアログが適用で何を渡すかがそのまま保存内容になる。

ダッシュボードの条件は「未設定（null）」を第一級の状態として持つ。null のときは画面側の既定の条件
（`use-dashboard-view.ts` が rep とタグのフィルタを未使用にした、その日の全記録）で集計する。
ところが旧「ダッシュボード」ダイアログは、エディタの v-model が非 null を要るために、開くときに未設定を `new FindKyouQuery()`
（空の条件）で埋め、適用ではその値を無条件に2欄とも渡していた。つまり **開いて適用しただけで、未設定だった条件が空の条件で
書き潰される**。空の `FindKyouQuery` は `tags` と `reps` が `[]`（フィルタ有効・チェック0個＝0件指定。ADR-0106）なので、
書き潰された後のダッシュボードは既定の条件ではなく「0件指定」で集計し、**エラーも警告も出ずに空になる**。
旧「実行中」ダイアログは null を第一級で持っていた（チェックボックスの OFF）ので、こちらは書き潰さなかった。

3つを1つの「検索条件」ダイアログ（検索ショートカット・実行中・ダッシュボードの3セクション、適用・キャンセルは1組）に
まとめるにあたり、旧ダッシュボードダイアログと同じ「現在値を全部渡す」作りにすると、この事故が「検索ショートカットを
1つ足しただけ」の利用者にも起きるようになる。

```
756ad897 で削除した旧ダイアログ（ロジックは use-edit-saved-find-query-dialog.ts へ移した）:
  src/client/classes/use-edit-dashboard-dialog.ts
  src/client/classes/use-edit-playing-time-is-dialog.ts
  src/client/pages/dialogs/edit-dashboard-dialog.vue
  src/client/pages/dialogs/edit-dashboard-dialog-props.ts
  src/client/pages/dialogs/edit-dashboard-dialog-emits.ts
  src/client/pages/dialogs/edit-playing-time-is-dialog.vue
  src/client/pages/dialogs/edit-playing-time-is-dialog-props.ts
  src/client/pages/dialogs/edit-playing-time-is-dialog-emits.ts
  src/client/__tests__/unit/composables/edit-playing-time-is-dialog.test.ts（edit-saved-find-query-dialog.test.ts へ移した）
```

## Decision

「検索条件」ダイアログの適用で親（設定画面の clone）へ渡すのは、そのダイアログで触ったセクションだけにする。
触ったかどうかはセクションごとの印で決める（一覧の適用・エディタの適用・実行中のチェック操作で立て、開き直しで戻す）。
ダッシュボードは片方の条件しか触っていなくても `DashboardConfig` 丸ごとを渡すので、触っていない側には開いた時点の値
（null を含む）をそのまま入れる。

## Rejected alternatives

- **3つのダイアログを別々のまま置く** — 検索条件の設定が設定画面の3か所に散り、どのボタンで何が変わるかを利用者が探すことになる。旧ダッシュボードダイアログの書き潰しもそのまま残る。
- **1つにまとめて、適用では3セクション全部の現在値を渡す** — 旧ダッシュボードダイアログの作りをそのまま広げた形。未設定の条件はエディタに載せるために空の条件で開いてあるので、触っていないセクションが空の条件として保存される。検索ショートカットを1つ足しただけで、ダッシュボードの集計が0件になる。
- **触った印ではなく、開いた時点の値との比較で「変わったか」を判定する** — 未設定（null）はエディタに載せる時点で空の条件に置き換わっている。比較の相手を null のままにすると「触っていないのに違う」と判定され、比較の相手も空の条件にすると「利用者が空の条件を意図して適用した」場合と区別できない。何を触ったかは値ではなく操作で分かるので、操作の側に印を置く。
- **ダッシュボードの2欄を別々のイベントで渡し、片方だけ更新できるようにする** — 設定画面側は受け取った `DashboardConfig` の JSON を `dashboard_json_data` へ丸ごと入れる（`use-application-config-view.test.ts`「この画面の適用で、子ダイアログの組み立て結果をまとめて送る」）。欄ごとの部分更新を親へ持ち込むより、ダイアログが開いた時点の値を控えて埋めるほうが、親の作りを変えずに済む。
- **一覧やエディタで適用した時点で親へ渡す** — このダイアログのキャンセルで戻らなくなる。一覧・エディタの適用はこのダイアログのローカル反映だけにし、親へ渡すのはこのダイアログの適用だけ（旧ダイアログから引き継いだ方針）。
- **セクションの見出しを h タグで書く** — `useFloatingDialog` は本文の最初の見出しをダイアログ名に使うので、セクション見出しを h タグにするとダイアログ名が最初のセクション名になる。見出しは div で書き、文言は新しい i18n キー `SAVED_FIND_QUERY_SHORTCUT_TITLE`（7言語）。

## Consequences

- 触った印を立てるのは各セクションの操作ハンドラ（一覧の適用・エディタの適用・実行中のチェック）の側で、**ハンドラを足して印を立て忘れると、そのセクションの編集は適用で黙って捨てられる**（エラーも警告も出ない）。エディタの適用だけで印が立つことは `edit-saved-find-query-dialog.test.ts` の「実行中のエディタで適用した条件は、チェックを触らなくても実行中の設定として渡る」が固定する。
- 逆に印を常に立てる（または印を見ずに全部渡す）と旧事故が戻る。「何も触らずに適用しても何も渡さない」が落ちる。
- ダッシュボードは開いた時点の2欄の値を控えておく必要がある。控えを落とすと、片方だけ触ったときにもう片方が空の条件で保存される（「ダッシュボードの片方だけ触ったら、もう片方は元の値（未設定なら null）のまま渡す」）。
- 設定画面側は `saved_find_query_json_data` / `playing_timeis_json_data` / `dashboard_json_data` の3つを解いて渡し、未設定は null のまま渡す（`use-application-config-view.test.ts`「clone の3つの設定を解いて渡す」「どれも未設定なら null で開く」）。undefined や空の条件で埋めると、「触っていない側は開いた時点の値のまま」が空の条件を返す。
- ダッシュボードのセクションには実行中のような「カスタマイズする」チェックが無いので、一度条件を保存すると、このダイアログから未設定（null）へは戻せない。旧ダイアログで既に空の条件へ書き潰された設定も自動では直らず、エディタで条件を入れ直すことになる。
- 「触った」印は開き直しで戻るので、キャンセルした編集を次に開いたときに引きずらない（「開き直すと「触った」印も戻る」）。

## Evidence

実測なし — 旧ダッシュボードダイアログの適用が2欄を無条件に渡し、開くときに未設定を `new FindKyouQuery()` で埋めていたコード
（756ad897 で削除）と、空の `FindKyouQuery` の `tags` / `reps` が `[]`＝0件指定であることからの判断。
「触ったセクションだけ渡す」はユニットテストで固定した。

## Related tests

- `src/client/__tests__/unit/composables/edit-saved-find-query-dialog.test.ts` — 「適用で渡すのは触ったセクションだけ」（何も触らなければ何も渡さない・ショートカットだけ・実行中のチェック OFF で null・エディタの適用だけで印が立つ・ダッシュボードの片方だけ触ったらもう片方は開いた時点の値・キャンセルでは何も渡さない・開き直すと印が戻る）
- `src/client/__tests__/unit/classes/use-application-config-view.test.ts` — 「「検索条件」ダイアログを開く」（clone の3つの設定を解いて渡す・どれも未設定なら null で開く）
