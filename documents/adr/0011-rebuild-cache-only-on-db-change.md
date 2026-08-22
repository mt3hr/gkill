# ADR-0011: キャッシュのフルリビルドは実DBが変わったときだけ

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-12 |
| Sources | `44b2be68` / `69e14f48` / `94bdb13b` / `CLAUDE.md`「キャッシュのフルリビルドは『実DBが変わったときだけ』」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/local_rep_cache_copy.go` |

## Context

cached rep の `UpdateCache` は `DELETE FROM` ＋ 全行再INSERT である。変わっていない rep まで作り直すと、**共有の書き込みロックを握ったまま全種類の検索が止まる**。`UpdateCache` は既定1分周期で回るので、これは定常的に効く。

## Decision

再構築の抑止を `dbFileChangeDetector`（mtime ＋ サイズ）で行い、**基準を進めるのは再構築成功後の `CommitCacheRebuild` だけ**にする。

`--cache_reps_local=true` のときに挟まる `*_local_cached.go` も同じ約束を守る。**「コピーが要るか」の判定は必ず `os.Remove` より前に置く。**

## Rejected alternatives

- **毎回フルリビルドする（改修前）** — 共有ロックを握るので全検索が止まる。実データで phase1 が1〜2分。

- **判定の基準を「再構築を試みた時点」で進める** — 失敗した回を取りこぼす。失敗したのに基準だけ進むと、その変更は**次に別の変更が来るまで永久に反映されない**。

- **ローカルコピー層で `os.Remove` してから `os.Stat` で判定する（実際にそうなっていた）** — 消してから見るので**常に「要コピー」**になる。毎回1.3GBコピーし直したうえ、`LastUpdateCacheChanged()` が常に true を返して**抑止が丸ごと無効化**される。実データで phase1 が 0.2秒 → 1〜2分（2026-08-12 修正）。順序を1つ間違えるだけで機能全体が無効になる形なので、共通ヘルパ `local_rep_cache_copy.go` に寄せた。

- **ReKyou / MiReKyou も変更検知に載せる** — **載せてはいけない。** この2つはターゲット解決のため**アドレス確定後にもう一度更新される**仕様で、mtime判定を入れると2回目が飛んで**ターゲット未解決の中身が残る**。コピーの省略だけなら可。

- **内容ハッシュで判定する** — repは実データで1.3GB規模。毎周期ハッシュを取ると抑止の意味がない。mtime ＋ サイズで実用上足りている（Takeout の ZIP のように mtime が動かないケースは別問題で、そちらは CRC32 を使う → ADR-0022）。

## Consequences

`LastUpdateCacheChanged()` の値は「本当に再構築したか」を表す。ここが常に true を返す実装を作ると、上位の抑止が全部無効になる（実際にそうなっていた）。

判定の順序（`os.Stat` → `os.Remove`）は**コードを読んでも「なぜこの順序なのか」が分からない**種類の制約なので、`local_rep_cache_copy.go` にコメントを置いてある。

## Evidence

- `--cache_reps_local=true` の本番構成で phase1 が **0.2秒 → 1〜2分**（毎回1.3GBのコピー ＋ 抑止の無効化）。2026-08-12 に修正して 0.2秒へ戻した

## Related tests

- `src/server/gkill/dao/reps/db_file_change_detector_test.go`
- `src/server/gkill/dao/reps/local_rep_cache_granular_test.go`
- `src/server/gkill/dao/reps/idf_granular_cache_test.go`
- `src/server/gkill/dao/reps/re_kyou_granular_cache_test.go`
