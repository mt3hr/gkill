# ADR-0007: ReKyou / MiReKyou のターゲット解決はリクエスト単位でメモ化する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-04 |
| Sources | `ce9578b6` / `f80ff665` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/target_resolution_memo.go` |

## Context

ReKyou も MiReKyou もタイトルを持たないので、ワード検索を**ターゲット Kyou を持つ実データrepへ委譲する**。この委譲が入れ子になっていて、既定構成の1検索で

```
ReKyou   → 実データrep + MiReKyou  （MiReKyou はさらに実データrepへ）
MiReKyou → 実データrep
```

と、**まったく同じ query で実データrepを3回舐めていた**。委譲先には query がそのまま渡っているので3回とも結果は同一。

## Decision

リクエスト単位のメモを `context` に載せ、同じ query・同じrep集合の解決を1回に畳む。ReKyou の委譲先は「実データrep + MiReKyou」なので、実データrepぶんは MiReKyou 側と共有し、MiReKyou ぶんだけ追加で引く。

メモが無い経路（repの直叩き・単体テスト）は今までどおり自前で解決する。

## Rejected alternatives

- **委譲クエリを1本に正規化する** — **結果が変わる。** 委譲先の絞り込み（カレンダー・IDs・期間）が効かなくなる。例: 今日作った、2020年の Kmemo を指す ReKyou が、カレンダー範囲＝今日の検索で落ちるようになる。速さのために意味論を変えることになるので採らない。

- **委譲そのものをやめて、ReKyou 側にターゲットのタイトルを非正規化して持つ** — 追記専用DAO（→ ADR-0010）では、ターゲットが更新されても ReKyou の版は増えないので、非正規化した値が古いまま残る。ターゲットの本文を編集したら参照側の検索結果も変わってほしい、が要件。

- **メモをプロセス全体のキャッシュにする** — 追記専用なのでデータは増え続け、無効化の条件が「どのrepのどのIDが更新されたか」に依存する。1リクエストの中でだけ同一性が保証されればこの問題は解けるので、リクエスト単位に閉じる。

## Consequences

最新版アドレスも共有キャッシュから引くようにした。`find_filter` が検索の頭で `RefreshLatestDataRepositoryAddresses` を呼んで `GkillRepositories` に載せているのに、ReKyou / MiReKyou はそれを使わず `GetAllLatestDataRepositoryAddresses` で全件を map に写し取り直していた。検索経路以外から呼ばれたときのために `EnsureLatestDataRepositoryAddresses`（未取得なら読み込む）を通す。

メモは**リクエスト単位に閉じる**。`context` から取れなければ自前で解決する経路を残しておくこと（repの直叩きと単体テストがそこを通る）。

## Evidence

- 既定構成の1検索で、同一 query による実データrepの走査が **3回 → 1回**

## Related tests

- `src/server/gkill/dao/reps/re_kyou_repository_sqlite3_impl_test.go`
- `src/server/gkill/dao/reps/mi_re_kyou_repository_sqlite3_impl_test.go`
- `src/server/gkill/dao/reps/gkill_repositories_test.go`
