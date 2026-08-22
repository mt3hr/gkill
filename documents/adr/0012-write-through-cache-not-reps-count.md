# ADR-0012: 書き込み後のキャッシュ反映は要素数ではなく構築時に控えた CachedReps で判定する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-12 |
| Sources | `5afbd408` / `.claude/skills/gkill-go-backend/SKILL.md`「`len(XxxReps) == 1` でキャッシュrepを判定してはいけない」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/gkill_dao_manager.go` |

## Context

書き込み後にメモリキャッシュrepへ反映する処理が `len(repositories.TagReps) == 1 && *gkill_options.CacheTagReps` という**要素数による代用判定**で守られていた。

これは「キャッシュONなら `XxxReps` はキャッシュrep1個に差し替えられている」という観察に基づく。ところが `GkillDAOManager` は、キャッシュrepで `XxxReps` を1個に差し替えた**あとに**プラグインの型別アダプタを append する（→ ADR-0021）。`provides` を持つプラグインが1つ入るだけで長さが2になり、判定が常に false になる。

読み取りはキャッシュrepしか見ず下層repへフォールバックしないため、**付けたタグが次の `UpdateCache` まで最大1分間見えなくなっていた**。

さらに悪いことに、その1分の間に PWA が古い応答をキャッシュし直すと、最新版アドレス表の `UpdatedTime` は前進しないぶん検索直前の `get_updated_datas_by_time` から**二度と通知されず、古い一覧が恒久的に焼き付く**。「別のブラウザで付けたタグが、検索し直しても反映されない」の正体はこれで、PWA 側のキャッシュ削除機構そのものは壊れていなかった。

## Decision

個数判定を全廃し、**構築時に控えた `GkillRepositories.CachedReps`** を見る `repositories.WriteThroughXxxCache(ctx, ...)` に集約する（54箇所）。

## Rejected alternatives

- **アダプタの append を差し替えより前に移す** — 先に append すると型別アダプタが差し替えで消えるか、キャッシュrepと二重に登録されて**二重検索**になる。登録順は「`KCReps`→`Reps` コピーループより後」でなければならない別の理由がある（→ ADR-0021）。順序で辻褄を合わせると、両方の制約を同時に満たす順序が無くなる。

- **個数判定を `len(XxxReps) <= 2` のように緩める** — `provides` を持つプラグインの数だけ増えるので、次にプラグインが増えたときにまた壊れる。しかも壊れ方は同じく静か。

- **読み取り側で下層repへフォールバックする** — キャッシュの意味が無くなる（キャッシュに無い＝下層を見る、では毎回下層を舐める）。

- **書き込み後に `UpdateCache` を即座に回す** — `UpdateCache` は `DELETE` ＋ 全行再INSERT で共有ロックを握る（→ ADR-0011）。タグを1つ付けるたびに全検索が止まる。

## Consequences

**`len(XxxReps) == 1` でキャッシュrepを判定してはいけない**、が規約として残った。再発は `usecase/write_through_cache_test.go` の `TestNoRepsCountCacheGuard` がソース走査で落とす。

反映を飛ばしたときの症状は二段構えで、**1段目（最大1分見えない）は目立たないが、2段目（PWAへの恒久的な焼き付き）は利用者が自力で回復できない**。すでに焼き付いた古い一覧は更新IDの再通知が来ないので個別削除では回収できず、ServiceWorker の `activate` で `gkill-post-kyou-cache` を丸ごと破棄する必要があった。

## Evidence

実測なし — 構造からの判断（`provides` を持つプラグインが1つでも入れば必ず破れる）。

症状は実機で再現している（「別のブラウザで付けたタグが、検索し直しても反映されない」）。Goテスト4ケースと Vitest 6ケースは、いずれも修正前の実装に戻すと落ちることを確認済み。

## Related tests

- `src/server/gkill/usecase/write_through_cache_test.go`（`TestNoRepsCountCacheGuard` を含む）
- `src/server/gkill/usecase/cached_rep_insert_alignment_test.go`
- `src/client/__tests__/unit/classes/delete-gkill-cache.test.ts`
