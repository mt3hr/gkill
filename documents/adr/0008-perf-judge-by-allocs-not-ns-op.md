# ADR-0008: 性能判断は ns/op ではなく allocs/op・B/op・EXPLAIN QUERY PLAN で行う

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-18 |
| Sources | `43aed843` / `6665591a` / `5edfa431` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/find_filter_pipeline_bench_test.go` |

## Context

検索が重い理由を測ったところ、**CPUの約4割がGC**だった。SQLが遅いのではなく「1回の検索で作って捨てるオブジェクトが多すぎる」のが実体。

そして**このマシンは同じコードでも `ns/op` が倍近くぶれる**。速くなったつもりの変更が実際には遅くなっていた、という誤りが起こりうる状態だった。

## Decision

性能の判断基準を固定する。

- **`allocs/op` と `B/op`** で判断する（GCが支配的なので、確保を減らすことが速さに直結する）
- SQLは **`EXPLAIN QUERY PLAN`** で `SEARCH` になっていることを見る
- **`ns/op` は判断に使わない**

そして**否決した最適化を、否決した理由とともに記録に残す**。もっともらしい高速化案は繰り返し提案されるため。

## Rejected alternatives

実測して**否決した**手を残す。どれも「速くなりそう」に見えるが、測ると遅いか、意味論を壊す。

- **multi-row INSERT（`INSERT INTO ... VALUES (?,?),(?,?),...`）** — 素朴なループより **17倍遅い**。バインド変数が増えるほど Prepare のコストが跳ねる。

- **手書きの `MarshalJSON`** — 標準の `encoding/json` より **1.6倍遅い**。`json.Compact` 相当の再走査が入るため。

- **Mi の UNION を UNION ALL にする** — **不可。** TimeIs 4箇所・MiReKyou 2箇所は腕ごとに `DATA_TYPE` リテラルが違うので腕をまたぐ重複が原理的に出ず `UNION ALL` にできるが、**Mi の5射影は実仕事をしている**（同一行が複数の時刻で射影されうる）。畳むと結果が変わる。

- **取得列の絞り込み** — 効果が出なかった。行数ぶんの構造体確保が支配的で、列を減らしても確保の回数が変わらない。

- **`onlyLatestData` の相関サブクエリを JOIN へ書き換える** — 実測で**遅くなった**。

- **`ORDER BY` を残したまま後段で並べ直す** — 捨てられる `ORDER BY` は SQLite 側に一時ソートを作らせるだけなので外した（`6665591a`）。これは採用したほう。

## Consequences

効いた手は「確保を減らす」系に集中した。後段パイプライン（`api/find_filter.go`）は20万IDで **329MB / 503,202確保 → 156MB / 1,062確保**。

- `sortAndTrimKyousMap` に `len==1` の高速路（121MB/401,045確保 → 12.6MB/514確保）
- `replaceLatestKyouInfos` は入力スライスを再利用（73MB/201,045確保 → 12.6MB/515確保）
- `AllTags`（`Tag` 構造体240B×全件）を48バイトの中間表現へ
- 結果ソートを32バイトのキー配列＋その場でのサイクル置換へ

**索引の追加はトレードオフを明記して採る。** TIMEIS キャッシュの `START_TIME_UNIX` / `END_TIME_UNIX` 索引は期間絞り込みを `SCAN` → `SEARCH` にする一方、plaing 判定は逆に悪化する。期間絞り込みは検索のたび、plaing は `show_attached_timeis` が真の面だけなので採用した。

新しい最適化を提案するときは、この一覧に対して「なぜ今回は違うのか」を示すこと。

## Evidence

- 検索CPUの約4割がGC
- 後段パイプライン: 20万IDで 329MB/503,202確保 → 156MB/1,062確保
- `BeginLatestDataRepositoryAddressRead()` 新設で並列ベンチ 3.44ms → 1.84ms
- multi-row INSERT は素朴なループの **17倍遅い**、手書き `MarshalJSON` は標準の **1.6倍遅い**
- `ns/op` はこのマシンで倍近くぶれる

## Related tests

- `src/server/gkill/api/find_filter_pipeline_bench_test.go`
- `src/server/gkill/dao/sqlite3impl/bulk_insert_bench_test.go`
- `src/server/gkill/dao/reps/mi_find_kyous_parity_test.go`（Mi の UNION を畳めないことを固定）
- `src/server/gkill/dao/reps/kyou_json_bench_test.go`
- `src/server/gkill/main/common/gkill_log/no_eager_sql_format_test.go`
- `src/server/gkill/api/sort_result_kyous_test.go`
