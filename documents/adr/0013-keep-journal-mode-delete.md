# ADR-0013: 実データDBの journal_mode は DELETE のまま変えない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-04 |
| Sources | `ca5ef610` / `src/server/gkill/dao/sqlite3impl/` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/sqlite3impl/sqlite_connection_test.go` |

## Context

SQLite の性能改善を検討すると **WAL 化がほぼ必ず候補に挙がる**。実際 gkill でも `cache_size` / `temp_store` / `mmap_size` が未設定で、接続プールも実データDB側は全部既定のままだったので、まとめて見直す機会があった。

## Decision

実データDBの `journal_mode` は **DELETE のまま変えない**。代わりに PRAGMA と接続プールだけ調整する。

- `cache_size` … 既定 -2000（2MB）。接続ごとの上限で実体ファイル数ぶん積み上がりうるので、控えめに 8MB
- `temp_store` … 既定 FILE。`ORDER BY` の一時ソートがディスクに落ちていたので MEMORY へ
- `mmap_size` … 256MB
- 接続プール … 既定の `MaxIdleConns` は2しかなく、超えた接続は使い終わると即閉じられて SQLite のページキャッシュが毎回捨てられていた。`MaxOpenConns` と同数を保持させる

**キャッシュ側は従来どおり WAL のまま。**

## Rejected alternatives

- **実データDBを WAL にする** — **`-wal` / `-shm` のサイドカーができる。** gkill は rep を**端末ごとの `.db` ファイルとして持ち回る**作りで、バックアップも単純なファイルコピーで済ませたい（→ ADR-0010 の前提そのもの）。`.db` 単体をコピーすると未チェックポイントの内容が落ちるので、**利用者が気付かないうちにデータを失う**。速さと引き換えにしてよい性質ではない。

- **WAL にしたうえで、コピー前に必ずチェックポイントさせる** — 「持ち回り」は gkill の外（同期ツール・USBメモリ・手動コピー）で起きるので、gkill からは強制できない。

- **`mmap_size` を増やさない** — 大きいDBで効果が大きい。90MBの `URLog.db` の全走査が実測 **156ms → 5.6ms**。実行順を入れ替えても同じなのでOSキャッシュの温まりではない。小さいDBでは差が出ないので害もない。

## Consequences

**この方針が将来ひっくり返らないよう、PRAGMA の実値を検査する回帰テストを置いた**（とくに `journal_mode = delete`）。文書だけだと「速くなるから」で戻される。

キャッシュ側と実データ側で `journal_mode` が違う、という非対称が残る。キャッシュは持ち回らない派生物なので WAL でよい。

## Evidence

- `mmap_size` 256MB: 90MBの `URLog.db` の全走査が **156ms → 5.6ms**（実行順を入れ替えても同じ）
- `MaxIdleConns` 既定2: 超過分の接続が返却のたびに閉じられ、SQLite のページキャッシュが毎回捨てられていた

## Related tests

- `src/server/gkill/dao/sqlite3impl/sqlite_connection_test.go`（PRAGMA の実値を検査。とくに `journal_mode = delete`）
