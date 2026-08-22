# ADR-0014: 時刻列は unixepoch の式インデックスで引く

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-04 |
| Sources | `e5d294d4` / `306f4288` / `43aed843` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/sqlite3impl/sqlite3impl_util.go` |

## Context

`WHERE` と `ORDER BY` が `datetime(列, 'localtime')` で**列を関数に通しており**、どんな索引を足しても `SEARCH` にならず毎回全走査＋一時ソートになっていた。

## Decision

比較と整列を `unixepoch(列)` に統一し、各テーブルに `CREATE INDEX ... (unixepoch(列) DESC)` を張る。比較の両辺に同じ変換が掛かるので順序関係は変わらない。

索引が効いていることを **`EXPLAIN QUERY PLAN` に `SEARCH` が出ること**として回帰テストで固定する。

## Rejected alternatives

- **`datetime()` を外して素の文字列比較にする** — **誤り。** 実データの `RELATED_TIME` は**オフセットが混在している**（TAG は `+00:00` が6,194行 / `+09:00` が853行）。文字列順は時系列にならない。

- **`datetime(列, 'localtime')` のまま索引を工夫する** — `'localtime'` は**非決定的**（プロセスのタイムゾーンに依存する）なので、SQLite は式インデックスに使わせない。仕様上どうやっても効かない。

- **時刻を整数列として非正規化して持つ** — キャッシュ側では実際にそうしている（`_UNIX` 整数列）。実データDB側は持ち回る `.db` のスキーマを変えることになるので採らない。なおキャッシュ側にも、**先頭が時刻列の索引**を追加した（既存の `(ID, RELATED_TIME_UNIX, UPDATE_TIME_UNIX)` は ID が先頭で、範囲にも並び替えにも使えない）。

## Consequences

**生成SQLと索引の式は完全に一致していないと、エラーにならず黙って全走査に戻る。** これがこの決定のいちばん危険なところで、だから `EXPLAIN QUERY PLAN` の検査をテストにしてある。

時刻列まわりのSQLを触るときは、`SEARCH` が消えていないかを必ず見ること。

## Evidence

- 実測（TAG 7,047行、1か月範囲）: **2.44ms → 0.39ms**。`SCAN + TEMP B-TREE` から `SEARCH` のみへ
- オフセット混在の実データ: TAG の `RELATED_TIME` は `+00:00` が6,194行 / `+09:00` が853行

## Related tests

- `src/server/gkill/dao/sqlite3impl/unixepoch_index_test.go`
- `src/server/gkill/dao/sqlite3impl/index_usage_test.go`
- `src/server/gkill/dao/sqlite3impl/latest_data_index_test.go`
- `src/server/gkill/dao/sqlite3impl/timeis_range_index_test.go`
