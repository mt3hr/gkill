# ADR-0621: カーソル頁では Mi の代表射影を元の期間で再検証する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-18 |
| Sources | 2026-09-18 の MCP 実利用報告（150件を limit 5 → 3 → 3 で読むと `remaining_count` が 145 → 145 → 143）。`.claude/skills/gkill-mcp/SKILL.md`「cursor 頁では Mi / MiReKyou の entry を元の窓で選び直す。」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp.go`（`batch` の直後の再検証呼び出し）/ `src/server/gkill/api/gkill_server_api/get_kyous_mcp_helpers.go`（`revalidateMiEntriesAgainstOriginalWindow`） |

## Context

ADR-0604 の複合カーソルは「カーソル時刻を `CalendarEndDate` へ押し下げる」ことで
2頁目以降の検索対象を縮め、1頁のために全期間を検索し直す事故（ADR-0603）を避けている。
その押し下げが **Mi の代表射影を変える**ことが、実利用報告の非単調な残件数で表に出た。

- Mi rep の `FindKyous` は Include フラグに関係なく5射影を UNION し、各腕が**自分の時刻列**で期間フィルタされる
  （`mi_create` = CREATE_TIME、`mi_check` = UPDATE_TIME、`mi_limit` / `mi_start` / `mi_end`）
- `for_mi` 無しの検索は `replaceLatestKyouInfos` が `newestKyouEntry` で代表1件へ潰す。優先は `_start` 射影 →
  DataType 辞書順なので `mi_check` < `mi_create`
- 1頁目（窓は全期間）では、すべての Mi が `mi_check`（更新時刻）で出る
- 2頁目（窓の上限 T < 更新時刻）では `mi_check` の腕が窓外へ落ち、`mi_create`（作成時刻 ≤ T）が代表になって
  **カーソルより後ろに再出現**する。`len(batch)` が膨らんで `remaining_count` が減らず、返却にも同じ記録が
  別の射影名で重複する。カーソルが遡るほど該当 Mi が増えるので、報告の +3 → +4 と膨らむ

残件数は副作用で、実害は**同じ記録がページをまたいで重複する**こと（ADR-0604 の「重複ゼロ」の契約違反）。
TimeIs は start / end が別 entry として両方残るので窓非依存。`for_mi` 時は RelatedTime が射影時刻へ上書きされ
再判定されるので同じく安定だが、対象には含めた。

## Decision

cursor 頁では、`batch`（カーソル走査後）の Mi / MiReKyou の entry を、**押し下げ前の検索条件 + その ID 群**で
`FindFilter.FindKyous` を再実行して選び直す。再実行結果の `(ID, DataType)` に無い entry は「元の窓での代表は
カーソルより上にあった＝返却済み」なので落とす。残件数・次カーソル・候補選定は落とした後の `batch` から計算する。
再実行に失敗したときは頁を失敗させず、警告を1行足して `batch` をそのまま返す。

## Rejected alternatives

- **潰し込みを窓非依存にする（代表を全射影から選んでから窓を当てる）** — Web の rykv で「その日に作ったタスク」が
  消える。作成日を検索しても、代表が窓外の更新時刻（`mi_check`）になって出てこなくなる。利用者に見える退行。
- **押し下げをやめる** — ADR-0603 / 0604 が潰した「1頁のために毎回全期間を検索する」事故（ピーク十数GB、1頁 2時間近く）へ
  逆戻りする。
- **Node（MCP サーバ）側で ID の重複を除く** — 前頁の ID を知らない（セッションを跨いで状態を持たない設計）。
  仮に持てても `remaining_count` は Go が数えるので直らない。
- **全射影を別 entry として出す（潰し込みを外す）** — ADR-0610 で否決済み。素の検索で1つの Mi が最大5件に増え、
  全クライアントの件数とページングが変わる。
- **`newestKyouEntry` の規則をハンドラへ複製して元の窓での代表を計算する** — 同じ規則が2形態になる（ADR-0611）。
  `FindKyous` に IDs を渡せば「1頁目と同じ規則で選んだ代表」がそのまま返る。

## Consequences

- cursor 頁で Mi / MiReKyou が `batch` にあるときだけ、ID 群を絞った `FindKyous` が1回増える（Mi が無ければ増えない）。
  ID 群は他条件と AND されるので、元の窓の全件を引き直すことにはならない。
- 押し下げの意味は変わらない。再検証は Mi の ID 群に閉じている。
- 再検証に失敗すると**警告つきで**旧挙動（重複しうる）に戻る。黙って直らない。
- 守れていないこと: TimeIs に `playing_time` を組み合わせた場合は `sortAndTrimKyousMap` が代表1件へ潰すが、
  `_start` 優先で start 射影が窓内なら代表は同じ、窓外なら end も窓外なので再出現しない。理屈で安定と判断しており、
  テストでは固定していない。

## Evidence

- 報告値: page1 5件 → remaining 145、page2 3件 → 145、page3 3件 → 143（全期間、1頁目の total 150）
- 再現テスト（修正前）: kmemo 146 + 完了済み Mi 4（CREATE_TIME を古い側、UPDATE_TIME を全メモより新しい側）で
  page2 の remaining が 146（期待 142）、Mi 4件の ID がページをまたいで2回返り、Σreturned = 154（期待 150）
- 修正後: 145 → 142 → 139、Σreturned = 150、重複ゼロ。`for_mi` の頁送りでも重複ゼロ

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp_v2_test.go`（`TestHandleGetKyousMCP_RemainingCountMonotonicAcrossPages` / `TestHandleGetKyousMCP_ForMiPagingDoesNotRepeatTasks` / `TestHandleGetKyousMCP_RemainingCountSemantics`）
- `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp_test.go`（`TestHandleGetKyousMCP_PagingDoesNotDropSameRelatedTime`）
