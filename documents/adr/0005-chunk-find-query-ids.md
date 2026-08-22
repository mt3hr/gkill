# ADR-0005: FindQuery.IDs は分割して渡し、検索が失敗したのに GkillError が空のまま return しない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-18 |
| Sources | `7c5f07f2` / `CLAUDE.md`「FindQuery.IDs は分割して渡す」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/repositories.go` |

## Context

IDリストは各repのSQLへ `ID IN (?, ?, ...)` として展開される。ところが **Mi / MiReKyou は5射影のUNION**で、5本それぞれに同じリストを丸ごと展開する。バインド変数は `5N+5` になり、SQLite の上限（`SQLITE_MAX_VARIABLE_NUMBER` = 32766）を **N=6553** で超えて Prepare が落ちる（実測: 6552は成功、6553で `too many SQL variables`）。

**この壊れ方は静かで気付けなかった。** 失敗が `GkillError` にならず、`/api/get_kyous` も `/api/get_kyous_mcp` も **HTTP 200 ＋ `errors: null` ＋ 0件**で返っていた（内部の `err` は Debug ログにしか出ない）。呼び出し側からは「成功・該当0件」と区別が付かない。

実データでは確認待ちの記録7,122件のIDを一度に渡して踏んだ。

## Decision

2つを同時に決めた。片方だけでは足りない。

1. `reps` 側の入口に `findChunkedByIDs`（`repositories.go`）を置き、**4000件ずつに割って和を取る**。新しくIDリストを渡す検索を書くときは必ずこれを通す
2. 検索が失敗したのに `GkillError` が空のときは `message.EnsureNotEmpty` で **`ERR000410` を必ず立てる**

## Rejected alternatives

- **分割せず、上限に近づいたらエラーを返す** — 利用者から見れば「検索できない」になる。IDリストはORの羅列なので分割は意味論を変えない（塊どうしでIDが重ならないので重複も出ない）。できることをエラーにする理由がない。

- **`Mi` / `MiReKyou` の5射影UNIONを1本にまとめてバインド変数を減らす** — UNION の5本は実仕事をしている（作成/確認/期限/開始/終了の5つの時刻でそれぞれ射影する）。まとめると意味論が変わる。→ ADR-0008 でも「Mi の UNION は実仕事をしている」として同じ結論に達している。

- **呼び出し側（ハンドラ）で分割する** — 入口が複数あり、新しい呼び出しを書く人が毎回気付く必要がある。`reps` 側の入口に置けば、通らない経路のほうが例外になる。

- **エラーを立てる件だけ直す** — 上限を超える呼び出しは実在するので、エラーになるだけでは利用者は救われない。逆に分割だけ直すと、**次に同じ「静かに0件」を作ったときにまた気付けない**。この2つは対で意味を持つ。

## Consequences

かかっているのは `Repositories.findKyous` / `MiRepositories.FindMi` / `MiReKyouRepositories.FindMiReKyou` と、`GkillRepositories.FindTags` / `FindTexts` の中で最新版アドレスからIDリストを作っている箇所。上限は `maxIDsPerFindQuery`（4000）。

**「失敗したのに `GkillError` が空」で return する分岐を新しく作らないこと。** この不具合の本体は上限そのものではなく、失敗が呼び出し側に届かなかったことのほうだった。

## Evidence

- 実測: 6552件は成功、6553件で `too many SQL variables`（Mi の 5N+5 = 32771 > 32766）
- 実データで確認待ち7,122件のIDを渡して再現

## Related tests

- `src/server/gkill/dao/reps/repositories_id_chunk_test.go`
- `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp_test.go`（`TestHandleGetKyousMCP_ManyIDs`）
- `src/server/gkill/api/message/gkill_error_test.go`
