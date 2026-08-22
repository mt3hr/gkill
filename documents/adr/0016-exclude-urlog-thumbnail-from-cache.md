# ADR-0016: URLog のサムネイルはインメモリキャッシュに載せない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-04 |
| Sources | `688bf0e0` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/ur_log_repository_cached_sqlite3_impl.go` |

## Context

URLog の `THUMBNAIL_IMAGE` は base64 で **1行あたり平均406KB・最大10MB**あり、実データ227行の合計が **90MB** ある。これがキャッシュ表に入っていたため、`--cache_in_memory`（既定 true）でそのままメモリに常駐していた。

## Decision

キャッシュ表から `THUMBNAIL_IMAGE` 列を外す。`UpdateCache` が下層repを読む段階でも除外する（除外しないと全件を `[]URLog` に載せる時点で Go のヒープに90MBが乗る）。

サムネイルが要る経路は扱いを分ける。

- `GetURLog` / `GetURLogHistories`（URLogカード表示ごとに呼ばれる）… キャッシュから引いたうえで、**`REP_NAME` からその版を持つrepを特定してそこだけ読み直す**
- `FindURLog`（共有ページのみ。頻度が低い）… サムネイル要求時は下層repへ委譲する
- `FindKyous` / `GetKyou` / `GetKyouHistories` … 無変更（Kyou はメタデータのみでサムネイルを触らない）

## Rejected alternatives

- **集約（`u.urlogRep`）へ丸ごと委譲する** — 配下rep数ぶんのクエリが飛ぶ。実データ17repで**カード1枚あたり約13.8ms**。持ち主repだけなら約0.6〜1.0ms。

- **丸ごと委譲する（もう1つの理由）** — cached の `AddURLogInfo` は**キャッシュにしか書かない**（本番では `handle_commit_tx` がディスクとキャッシュの両方に書いて辻褄を合わせている）。委譲すると**キャッシュにしか無い行が引けなくなる**。

- **`FAVICON_IMAGE` も一緒に外す** — 対象外にした。合計0.10MB・平均0.5KBしかなく、外す意味がないうえ経路が増える。

- **サムネイルを別テーブルへ切り出す** — 実データDB側のスキーマ変更になる（持ち回る `.db`）。キャッシュ表から外すだけで目的を達する。

## Consequences

フラグ名を `ExcludeBinaryData` から **`ExcludeURLogThumbnailImage`** に変えた。意味をサムネイル限定にして、「バイナリ全般を外す」と読み違えて favicon まで外されるのを防ぐ。

「キャッシュから引いて、必要な列だけ持ち主repから読み直す」という形が1つ増えた。持ち主の特定に `REP_NAME` を使うので、`REP_NAME` の意味論（→ ADR-0001）に依存している。

## Evidence

- 本番相当（227行×406KB）で実測: キャッシュDB **90MB相当 → 0.17MB**
- 集約への丸投げ: 実データ17repでカード1枚あたり約13.8ms / 持ち主だけなら約0.6〜1.0ms
- `FAVICON_IMAGE` は合計0.10MB・平均0.5KB

## Related tests

- `src/server/gkill/dao/reps/cache_find_bench_test.go`
