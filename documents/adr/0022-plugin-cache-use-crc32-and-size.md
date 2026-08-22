# ADR-0022: Google Takeout は ZIP のまま読み、差分判定は (CRC32, Size)、世代は「フォルダ + 書き出し時刻」

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-11 |
| Sources | `3c3bfade` / `documents/reverse/plugin-system.md`「取り込み元の ZIP — sdk.OpenSources」 / `CLAUDE.md` の `src/plugins` 構造コメント |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/plugin/sdk/source.go` |

## Context

Google Takeout（Fitbit / Google Health / ロケーション履歴）を取り込むプラグイン2本を作るにあたって、「書き出しをどう置かせるか」「何をもって差分とするか」「同じフォルダに複数回の書き出しがあったらどうするか」を決める必要があった。

いずれも素直な選択が実データで壊れる。

## Decision

- **ZIP のまま置かせる。展開させない。** 走査は `plugin/sdk/source.go`（`sdk.OpenSources`）に置き、2つのプラグインで共通にする
- **差分判定は `(CRC32, Size)`**。mtime は使わない
- **取り込み世代は「ZIP を含むフォルダ ＋ ZIP名の書き出し時刻」**。分割された `-1-001` / `-1-002` は1つの世代にまとまり、同じフォルダに置いた別の書き出しは分かれる

## Rejected alternatives

- **差分判定に mtime を使う** — **Takeout は書き出し時刻を全エントリに同じ値で入れる。** 実データで全11,813エントリが同一値だった。中身が変わってもエントリの更新時刻は動かないので、**差分がまったく検出されない**（エラーは出ず、ただ古いまま）。

- **取り込み世代の単位をフォルダだけにする** — 翌月の書き出しを同じフォルダに足したときに、**歩数などの合算する指標が2倍になる**。エラーも警告も出ず、グラフの値だけが倍になる。

- **世代の単位をZIPファイル1本にする** — Takeout は大きい書き出しを `-1-001` / `-1-002` に分割する。ファイル単位にすると1回の書き出しが複数世代に割れ、日をまたぐ集計が分断される。

- **利用者に展開させて、展開後のフォルダを読む** — 実データで271MB／11,813エントリ。展開すると容量が倍要るうえ、展開の失敗や部分展開を検出する手立てがない。ZIP のまま読めば、エントリ単位で CRC32 が取れるので差分判定もそこに乗る。

## Consequences

「1つの書き出し」の定義がプラグイン間で共有されるので、`sdk.OpenSources` は2本の共通コードになった。ここを変えると両方に効く。

世代が複数ありうるので、**重なった日の扱いをプラグインごとに決める必要がある**。

- fitbit は `export` 表に順位を持ち、日が重なったら **rank が最小の世代だけを合算**する
- ロケーション履歴は読み出し時の `SELECT DISTINCT (time, lat, lng)` が別の書き出しの同じ点を畳むので、**順位は要らない**

新しい Takeout 系プラグインを足すときは、この「重なりをどう畳むか」を必ず決めること。決めないと合算する指標が黙って倍になる。

## Evidence

- 実データの Takeout ZIP: **全11,813エントリの更新時刻が同一値**（書き出し時刻）
- 271MB の ZIP に対し初回構築 約155秒 / 増分 1秒未満

## Related tests

- `src/server/gkill/plugin/sdk/source_test.go`
- `src/plugins/gkill_plugin_fitbit/cache_test.go`
- `src/plugins/gkill_plugin_google_locationhistory/cache_test.go`
