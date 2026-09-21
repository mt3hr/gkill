# ADR-0215: 実データDB・設定DBの synchronous は FULL にする

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-30 |
| Sources | `0f25b19a` / `src/server/gkill/dao/sqlite3impl/` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/sqlite3impl/sqlite_connection_test.go` |

## Context

SQLite の耐久性は `journal_mode` 単独では決まらず、**`synchronous` との組み合わせ**で決まる。

| 組み合わせ | 電源断・I/O断で起きること |
|---|---|
| WAL + NORMAL | 最後のトランザクションが失われる。DBは壊れない |
| DELETE + FULL | 最後のトランザクションが失われる。DBは壊れない |
| **DELETE + NORMAL** | **DBそのものが壊れうる** |

[ADR-0204](0204-keep-journal-mode-delete.md) は、rep を端末ごとの `.db` ファイルとして持ち回る都合から
`journal_mode` を DELETE のまま据え置くことを決めた。この判断自体は正しいが、**`synchronous` には触れていない**。
結果として gkill は上の表で唯一DBが壊れる組み合わせを選んでいた。
`sqlite3impl_util.go` のコメントも「synchronous も NORMAL のまま変えない。耐久性の意味が変わるため」と
書いてあるだけで、DELETE との相性は検討されていなかった。

2026-08-30、実際に壊れた。USB接続のディスク上にある directory rep の索引DB
（`〈外部ストレージ〉/〈記録保管場所〉/.gkill/gkill_id.db`）が SQLITE_CORRUPT になり、**rep 1本の破損でそのユーザの全APIが500**になった
（`GetRepositories` が丸ごと失敗し、auth middleware が全リクエストで落ちるため。こちらは ADR-0216 で扱う）。

破損の形は `Tree 3 page 3 right child: invalid page number 66`。
**ヘッダのページ数（61）と実ファイルサイズ（249,856 = 4096×61）は整合していた**ので、
同期コピーの中断による尻切れではなく、ロールバックジャーナルが不完全なまま適用された形だった。
その8分前（01:53:25）に Windows のイベントログへ `disk` の
「ドライバーは \Device\HarddiskN\DRn でコントローラー エラーを検出しました」が記録されている。

設定DB（`configs/` 配下の `account.db` / `user_config.db` など）も同じ DSN 文字列を8ファイルが直書きしており、
同じ組み合わせだった。**設定DBが壊れると rep 定義そのものが読めなくなる**ので、影響は実データDBより広い。

## Decision

実データDB（`sqliteDataDSNParams`）と設定DB（`ConfigDBDSNParams`）の `synchronous` は **FULL** にする。
NORMAL へ戻さない。

設定DBの8ファイルは同じ文字列のコピーだったので `ConfigDBDSNParams` へ集約した。
実データ用とは別定数にしてある（設定DBは小さく起動時しか全走査しないので、
`cache_size` / `temp_store` / `mmap_size` を持たせる理由がない）。

派生キャッシュ（`git_commit_log` のキャッシュ、プラグインの `cache.db` 6本）は**対象外**。
あちらは WAL + NORMAL で、破損しない組み合わせだから。

## Rejected alternatives

- **NORMAL のまま据え置き、速さを取る** — 実測で +18%（下記 Evidence）。
  この 18% で買っているのは「電源断・I/O断でDBが壊れない」ことで、壊れ方は
  「最後の1トランザクションが消える」ではなく「**DBが開けなくなる**」。
  gkill の rep は追記専用（[ADR-0201](0201-append-only-dao.md)）なので、壊れると
  そのrepの全履歴が読めなくなる。18% と引き換えにしてよい性質ではない。

- **`journal_mode` を WAL にして NORMAL のままにする** — [ADR-0204](0204-keep-journal-mode-delete.md) が却下済み。
  `-wal` / `-shm` のサイドカーができ、`.db` 単体をコピーしても未チェックポイントの内容が落ちる。
  gkill は同期ツール・USBメモリ・手動コピーで rep を持ち回るので、**利用者が気付かないうちにデータを失う**。

- **`EXTRA` にする** — FULL に加えてジャーナル削除時のディレクトリ同期まで行う。
  守れるのは「commit 直後に電源断が起きるとトランザクションがロールバックされうる」だけで、
  **DBの破損に対しては FULL と同じ**。今回の事象は破損なので、追加のコストに見合わない。

- **設定DBは対象外にする（実データDBだけ FULL）** — 設定DBが壊れると `user_config.db` の rep 定義が読めず、
  そのユーザの全APIが失敗する。実データDB1本の破損より広く効くので、片方だけ直す理由がない。
  実際、8ファイルが同じ文字列を直書きしていたために「実データ側と設定側で耐久性が非対称」という状態に
  誰も気付いていなかった。定数へ集約したのはその再発防止でもある。

- **USBディスクを使わないという運用で回避する** — 壊れる下地を減らす効果はあるが、
  gkill は端末ごとに `.db` を持ち回る作りで、どの記憶装置に置かれるかを gkill 側から強制できない。
  Android 端末の内部ストレージでも電源断は起きる。運用で消せる前提に立たない。

## Consequences

書き込みが **+18%** 遅くなる。1件ずつの裸INSERT（記録の追加）でもトランザクション（キャッシュ再構築）でも
同じ増加率だった。確保量・確保回数は変わらない（fsync を待つ時間が増えるだけ）。

`synchronous` と `journal_mode` は**対で**見ること。片方だけを見て「WALでもNORMALで平気なのだから」と
戻すと、壊れ方が「1トランザクション消える」から「DBが開けなくなる」へ変わる。
`sqlite_connection_test.go` が PRAGMA の実値を検査しているので、戻すとテストが落ちる。

この決定はすでに壊れたDBを直さない。破損したDBは別途復旧が要る
（今回は `sqlite3 .recover` で救出できる範囲を確認したうえで、健全なコピーで置換した）。

## Evidence

`synchronous_bench_test.go`（100行, `-benchtime 20x -count 3`, 3回の中央値, 2026-08-30,
Intel Core i7-10510U / 内蔵NVMe SSD）:

| 書き方 | NORMAL(1) | FULL(2) | 差 |
|---|---|---|---|
| 1件ずつの裸INSERT (100行) | 442.5 ms | 522.1 ms | **+18%** |
| トランザクション1回 (100行) | 4.68 ms | 5.52 ms | **+18%** |

裸INSERTは文ごとに暗黙のトランザクションになるので fsync が100回、トランザクション側は1回。
所要時間の絶対値は95倍違うが、増加率はどちらも +18% だった。

既存の `bulk_insert_bench_test.go` は `mode=memory` + `synchronous(OFF)` で fsync のコストを
一切測れないため、この比較には流用できない。実ファイルを開くベンチを別に置いてある。

実障害（2026-08-30）: USB接続ディスク上の `gkill_id.db` が SQLITE_CORRUPT
（`Tree 3 page 3 right child: invalid page number 66`）。ヘッダのページ数61と実サイズ249,856は整合。
8分前に `disk` イベントID 11「コントローラー エラー」。同種の警告は直近30日で22件。

## Related tests

- `src/server/gkill/dao/sqlite3impl/sqlite_connection_test.go`（PRAGMA の実値を検査。`synchronous = 2`）
- `src/server/gkill/dao/sqlite3impl/synchronous_bench_test.go`（NORMAL/FULL の書き込みコスト比較）
