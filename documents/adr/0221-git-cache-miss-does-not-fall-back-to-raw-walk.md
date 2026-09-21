# ADR-0221: Git コミットログのキャッシュ包装は、構築済みなら外れた ID で生リポジトリへ落ちない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-16 |
| Sources | `.claude/skills/gkill-go-backend/SKILL.md`「Git コミットログのキャッシュ包装は」/ [ADR-0218](0218-get-kyou-histories-via-cached-reps.md) / [ADR-0309](0309-plugin-provides-git-commit-log.md) / [ADR-0101](0101-filter-rep-after-cache.md) |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/git_commit_log_repository_cached_sqlite3_impl.go`（`shouldFallbackOnMiss` / `cacheBuilt`）・`src/server/gkill/dao/reps/git_commit_log_repository_local_dir_impl.go`（`hasCommit`） |

## Context

「`generate_plugin_cache all <user>` は本当にキャッシュを作れているのか。archived_git が怪しい。画面からの読み込みがめっちゃ遅い」
という報告から調べた。プラグインのキャッシュは正常だった（`result = built`、cache.db は repo 数十 / commit 数千 /
`build_state idle`、サーバの `typed_index` も ok / 同じ件数）。遅さの正体は native 側にあった。

rykv を「記録分類 = Git」だけで開くと数千行（過半が archived プラグイン由来）で、実測は
`get_kyous` 十数秒、行ごとの `get_git_commit_log` が平均数秒・最大 20 秒程度、`get_tags_by_id` すら数秒。
サーバは複数コアを数十分使い続け（数秒周期の鋸波）、接続が 0 本になると同時に止まった。
再起動直後の事前ロードは数分で終わるので、起動時の内部処理ではない。

待機中のサーバで 1 要求ずつ測ると、`/api/get_git_commit_log` は native のハッシュで十数 ms、
**archived プラグインのハッシュで約 1 秒、存在しないハッシュでも約 1 秒**。Git 限定の `get_kyous` は
数千件で 100 ms 弱、`get_tags_by_id` は十数 ms 以下。つまり「キャッシュに無い ID」のときだけ約 1 秒の実処理が走る。

原因は2段。

1. `gitCommitLogRepositoryCachedSQLite3Impl` の `GetGitCommitLog` / `GetKyou` / `GetKyouHistories` は、
   SQL の結果が 0 件だと「キャッシュ未構築など」のつもりで下層の集約 `GitCommitLogRepositories` へ逐次
   フォールバックしていた。`isCacheBuilding` とは別に、**構築済みでも**無条件に落ちる。
2. 下層の `gitCommitLogRepositoryLocalImpl` は `Log(From: hash)` が失敗（そのリポジトリにそのハッシュが無い）すると
   `Log(All: true)` で**全履歴を走査**して切り分けていた。外れた ID 1 件につき十数リポジトリ×数千コミットを復号する。

`GitCommitLogReps` は [キャッシュ包装, プラグインアダプタ] の集約で（ADR-0309）、プラグインのコミットは
アダプタが即答する一方、キャッシュ包装側は必ず外れて毎回この走査を踏む。`provides: ["git_commit_log"]` を持つ
プラグインが入って初めて「キャッシュに無い ID が大量に引かれる」状況が生まれた。ADR-0218 が `/api/get_kyou` の
版履歴について直した「git 84本は ID がコミットハッシュに当たらず `Log(All:true)` で全走査」が、
`/api/get_git_commit_log` と、アドレス表に無い ID の `GkillRepositories.GetKyou`（全 leaf を `UnWrap()`）に残っていた。

## Decision

1. **キャッシュ包装は `UpdateCache` を1回でも完走したら（`cacheBuilt`）、SQL の 0 件をそのまま「無い」で返す。**
   下層へのフォールバックが許されるのは `isCacheBuilding` 中と、まだ構築していない（バックグラウンド構築の失敗後を含む）
   あいだだけ（`shouldFallbackOnMiss`）。`cacheBuilt` が立つのは ref 一致・差分なし・差分適用・バックグラウンド構築の完了のどれか。
   失敗した回は立てない（部分的なキャッシュなので、読み取りは下層へ落ちる従来の形を保つ）。
2. **生実装は `CommitObject(hash)` で存在を引いてから `Log` する（`hasCommit`）。** packfile の索引を引くだけでコミット数に比例しない。
   `ErrObjectNotFound` のときだけ nil を即返し、それ以外の失敗は従来の `Log(From)` → `Log(All: true)` に任せる
   （Android 環境などで `Log(From)` が別の理由で失敗する想定を残す）。これで `GkillRepositories.GetKyou` の全 leaf 配布でも
   git の leaf は数 ms で外れる。

## Rejected alternatives

- **フォールバックを丸ごと消す** — 初回起動でバックグラウンド構築が走る前、および構築が失敗した後は、
  キャッシュ表が空か部分的なので、外れ＝無いとは言えない。`TestGitCommitLogCachedBuildingFallbackReturnsUnderlyingData`
  が「構築中に存在しない扱いにしない」を固定しているのはそのため。構築済みかどうかで分けるのが最小の変更。
- **キャッシュ表の行数で「構築済み」を判定する** — 永続 DB は前回の run の内容で行数があり、`UpdateCache` 前は古い可能性がある。
  行数を見るとその間に外れた ID を「無い」と答える。「この process で `UpdateCache` が完走した」を立てるほうが意味が正確で、
  `GetRepositories` の末尾で `UpdateCache` が走るので実運用では最初の要求より前に立つ。
- **`GkillRepositories.GetKyou` の `UnWrap()` 先から git の leaf を外す** — 意味論の変更で、アドレス表に載らない ID
  （プラグイン Kyou・追加直後の記録）の解決順に触る。2026-08-24 の ERR000092 事故（ADR-0210）と同じ場所なので、
  生実装側を安くするだけで足りる今回は触らない。
- **集約 `getGitCommitLog` でプラグインアダプタを先に聞き、当たればキャッシュ包装を飛ばす** — 順序依存を持ち込むうえ、
  存在しないハッシュ（削除済み・別ユーザーの ID）では結局キャッシュ包装まで来る。外れを安くするほうが根本。
- **`Log(From)` の失敗をエラー文字列で「無い」と判定する** — go-git のエラー型（`plumbing.ErrObjectNotFound`）で
  `errors.Is` する。文字列照合は ADR-0208 と同じ罠。

## Consequences

- 待機中サーバの `/api/get_git_commit_log`（archived プラグインのハッシュ）は約 1 秒 → SQL 1 回＋索引 1 回。
  存在しないハッシュも同じ。native のハッシュは変わらず十数 ms。
- 構築済みのキャッシュにまだ載っていない新しいコミット（次の `UpdateCache` まで）は `GetGitCommitLog` で nil になる。
  検索（`FindKyous`）も同じキャッシュから返すので、一覧に出ない行を引かれることはない。
- 守るテスト: `git_commit_log_cached_miss_test.go`（構築前は落ちる・構築済みは落ちない・構築中は落ちる、を下層の呼び出し回数で固定）、
  `git_commit_log_repository_local_dir_impl_test.go` の `TestGitCommitLogLocalDirMissingHashReturnsNil`。

## Evidence

- 実環境（2026-09-16、修正前のバイナリ `2dd8b307`）: native の Git rep は内蔵と外付けの2か所で
  十数リポジトリ（ユニーク数千コミット、走査対象は複製込みでその倍）、archived プラグインは数十 zip /
  数千コミット / 数十 rep 名。`threads` のプールは `NumCPU()`
- 画面（同一プロファイルの別タブで `performance.getEntriesByType('resource')` を集計。rykv「記録分類 = Git」数千行）:
  `get_kyous` 十数秒、`get_git_commit_log` 十数回で平均数秒・最大 20 秒程度、`get_tags_by_id` 十数回で平均数秒
- サーバの CPU（`Win32_Process` の Kernel+User 時間の 1 秒差分）: 起動から複数コアが数十分続き、数秒周期の鋸波。
  同時記録した確立済み接続が 0 本になると同時に 0 へ。再起動後の事前ロードは数分で終わり、
  以後は接続があっても 0.0/秒
- 待機中サーバで `document.cookie` の `gkill_session_id` を付けて `fetch` を直接計測（`/api/get_git_commit_log`）:
  native のハッシュは十数 ms、archived プラグインのハッシュは約 1 秒、存在しないハッシュも約 1 秒（HTTP 500・ERR000118）。
  同じ場で `get_tags_by_id` 十数 ms 以下、Git 限定の `get_kyous`（数千件・数 MB）100 ms 弱
- 混雑中の同じ計測: native は数秒 → 2 回目は数十 ms、プラグインは数秒のまま、存在しないハッシュも数秒。
  ばらつきの大きさは「実処理 + プール待ち」の形で、MCP の `count_only`（7 日間・数千件）も十数秒だった
- プラグイン側は無実: `generate_plugin_cache gkill_plugin_archived_git_commit_log <user>` は `result = built`・1 秒未満、
  cache.db は repo 数十 / commit 数千 / `build_state idle`、サーバの `typed_index` は ok / 同じ件数、プラグインプロセスの CPU は数十分で十秒足らず
- 修正後の所要時間は本番へ配布してから同じ操作で再測定する（この ADR を書いた時点では未測定）

## Related tests

- `src/server/gkill/dao/reps/git_commit_log_cached_miss_test.go`
- `src/server/gkill/dao/reps/git_commit_log_repository_local_dir_impl_test.go`（`TestGitCommitLogLocalDirMissingHashReturnsNil`）
- `src/server/gkill/dao/reps/git_commit_log_cached_nested_pool_test.go`（構築中のフォールバックが変わっていないこと）
