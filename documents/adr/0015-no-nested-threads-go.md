# ADR-0015: threads.Go の入れ子は禁止。集約リポジトリには逐次版を用意する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-05 |
| Sources | `530b7ec4` / `7f0d1710` / `c91a3456` / `d577bba7` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/main/common/threads/threads.go` |

## Context

`threads.Go` は `runtime.NumCPU()` 個の**有界セマフォを同期取得**してから goroutine を起こす。スロットを保持したまま子のスロットを待つと、全スロットが「子を待つ親」で埋まった瞬間に**誰も進めなくなる**。

これが実際に起きた。cached 実装のキャッシュビルド中フォールバックが `threads.Go` のスロットを保持したまま、集約の並列 `FindKyous` をネストしていた。実機で**約254件の検索が38秒間1件も完了しなかった**。

デッドロックなので待っても解けない。しかも `-race` では出ず、CPU も食わないので「サーバが黙った」ようにしか見えない。

## Decision

`threads.Go` の入れ子を禁止する。集約リポジトリには**逐次版**（`FindKyousSequential` など）を用意し、既に `threads.Go` の中にいる呼び出し元はそちらを使う。

## Rejected alternatives

- **セマフォの上限を増やす（`NumCPU()` × 2 など）** — ネストの深さは構造で決まるので、上限をいくら上げても「その深さぶん同時に走れば」また埋まる。確率を下げるだけで、しかも下げた結果**再現しにくくなる**ぶん悪い。

- **スロットを一時的に手放してから子を呼ぶ** — 手放しと取り直しの間に別の仕事が入るので、プールの意味（同時実行数の上限）が保証できなくなる。取り直しで待つ点も変わらない。

- **セマフォを取らずに素の goroutine で起こす** — プールを置いた理由（repのファンアウトで数百のgoroutineがディスクを叩き合うのを防ぐ）が消える。

- **デッドロックを検出してタイムアウトで抜ける** — 保険として `inline` フォールバック（閾値30秒）は入れてあるが、これは**主たる対策にしない**。30秒待ってから進むのでは利用者から見て壊れているのと変わらず、しかも「たまに遅い」に化けて原因が追いにくくなる。

## Consequences

集約リポジトリの `Find` 系には並列版と逐次版が並ぶ。**キャッシュ実装は逐次版経由に固定する。**

同種の罠がもう1つある: 共有 `RWMutex` の**再帰 RLock**。cached の複合メソッドが外側でロックを取ったまま内側の単体メソッドを呼ぶと、間に Writer が並んだ瞬間に恒久ハングする（`c91a3456`）。「cached 複合メソッドは外側ロックを取らない」「集約 ReKyou は cached `FindKyous` を呼ばない」も同じ理由で守る。

プラグイン rep は `Repositories` のファンアウトでスロットを取らない（`goForRep`）。プラグインのロック待ちで検索全体が止まらないようにするため。

## Evidence

- 実機で約254件の検索が**38秒間1件も完了しない**状態を再現・観測（`d577bba7` の調査。dlv の非対話ダンプでスタックを取り、全スロットが子待ちで埋まっていることを確認した）
- `-race` では検出されない（データ競合ではなく資源の循環待ちなので）

## Related tests

- `src/server/gkill/main/common/threads/threads_test.go`
- `src/server/gkill/dao/reps/mi_re_kyou_cached_nested_pool_test.go`
- `src/server/gkill/dao/reps/git_commit_log_cached_nested_pool_test.go`
- `src/server/gkill/dao/reps/re_kyou_cached_deadlock_test.go`
