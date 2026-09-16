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
という報告から調べた。プラグインのキャッシュは正常だった（`result = built`、cache.db は repo 97 / commit 3,447 /
`build_state idle`、サーバの `typed_index` も ok / 3,447 件）。遅さの正体は native 側にあった。

rykv を「記録分類 = Git」だけで開くと 5,547 行（うち archived プラグイン由来 3,447 行）で、実測は
`get_kyous` 19 秒、行ごとの `get_git_commit_log` が平均 6.7 秒・最大 20 秒、`get_tags_by_id` すら 3〜6 秒。
サーバは 2〜5 コアを 45 分間使い続け（約 7 秒周期の鋸波）、接続が 0 本になると同時に止まった。
再起動直後の事前ロードは CPU 累計 149 秒・約 2.5 分で終わるので、起動時の内部処理ではない。

待機中のサーバで 1 要求ずつ測ると、`/api/get_git_commit_log` は native のハッシュで 9〜16 ms、
**archived プラグインのハッシュで 1,055〜1,127 ms、存在しないハッシュで 1,018 ms**。Git 限定の `get_kyous` は
5,547 件で 87〜99 ms、`get_tags_by_id` は 10 ms。つまり「キャッシュに無い ID」のときだけ約 1 秒の実処理が走る。

原因は2段。

1. `gitCommitLogRepositoryCachedSQLite3Impl` の `GetGitCommitLog` / `GetKyou` / `GetKyouHistories` は、
   SQL の結果が 0 件だと「キャッシュ未構築など」のつもりで下層の集約 `GitCommitLogRepositories` へ逐次
   フォールバックしていた。`isCacheBuilding` とは別に、**構築済みでも**無条件に落ちる。
2. 下層の `gitCommitLogRepositoryLocalImpl` は `Log(From: hash)` が失敗（そのリポジトリにそのハッシュが無い）すると
   `Log(All: true)` で**全履歴を走査**して切り分けていた。外れた ID 1 件につき 18 リポジトリ×約 4,260 コミットを復号する。

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

- 待機中サーバの `/api/get_git_commit_log`（archived プラグインのハッシュ）は約 1.1 秒 → SQL 1 回＋索引 1 回。
  存在しないハッシュも同じ。native のハッシュは変わらず 9〜16 ms。
- 構築済みのキャッシュにまだ載っていない新しいコミット（次の `UpdateCache` まで）は `GetGitCommitLog` で nil になる。
  検索（`FindKyous`）も同じキャッシュから返すので、一覧に出ない行を引かれることはない。
- 守るテスト: `git_commit_log_cached_miss_test.go`（構築前は落ちる・構築済みは落ちない・構築中は落ちる、を下層の呼び出し回数で固定）、
  `git_commit_log_repository_local_dir_impl_test.go` の `TestGitCommitLogLocalDirMissingHashReturnsNil`。

## Evidence

- 実環境（2026-09-16、修正前のバイナリ `7a324e95`）: native の Git rep は `$HOME/Git/*` と外付け SSD の `Git/*` で
  18 リポジトリ（ユニーク 2,130 コミット、走査対象は複製込みで約 4,260）、archived プラグインは 88 zip / 97 `.git` /
  3,447 コミット / 78 rep 名。`threads` のプールは `NumCPU()` = 8
- 画面（同一プロファイルの別タブで `performance.getEntriesByType('resource')` を集計。rykv「記録分類 = Git」5,547 行）:
  `get_kyous` 19,113 ms、`get_git_commit_log` 16 回で平均 6,732 ms・最大 20,303 ms、`get_tags_by_id` 12 回で平均 3,195 ms・最大 6,427 ms
- サーバの CPU（`Win32_Process` の Kernel+User 時間の 1 秒差分）: 22:11 の起動から 2〜5 コアが 45 分続き、約 7 秒周期の鋸波。
  同時記録した `:9999` の確立済み接続が 0 本になると同時に 0 へ。再起動後の事前ロードは CPU 累計 149 秒・約 2.5 分で終わり、
  以後は接続があっても 0.0/秒
- 待機中サーバで `document.cookie` の `gkill_session_id` を付けて `fetch` を直接計測（`/api/get_git_commit_log`）:
  native のハッシュ 16 ms / 9 ms、archived プラグインのハッシュ 1,055 ms / 1,127 ms、存在しないハッシュ 1,018 ms（HTTP 500・ERR000118）。
  同じ場で `get_tags_by_id` 10 ms、Git 限定の `get_kyous`（5,547 件・2.27 MB）87 ms / 99 ms
- 混雑中の同じ計測: native 9,090 ms → 2 回目 22 ms、プラグイン 4,165 ms → 2,765 ms、存在しない 9,555 ms。
  ばらつきの大きさは「実処理 + プール待ち」の形で、MCP の `count_only`（7 日間 4,624 件）も 15,031 ms だった
- プラグイン側は無実: `generate_plugin_cache gkill_plugin_archived_git_commit_log <user>` は `result = built`・200 ms、
  cache.db は repo 97 / commit 3,447 / `build_state idle`、サーバの `typed_index` は ok / 3,447 件、プラグインプロセスの CPU は 33 分で 9 秒
- 修正後の所要時間は本番へ配布してから同じ操作で再測定する（この ADR を書いた時点では未測定）

## Related tests

- `src/server/gkill/dao/reps/git_commit_log_cached_miss_test.go`
- `src/server/gkill/dao/reps/git_commit_log_repository_local_dir_impl_test.go`（`TestGitCommitLogLocalDirMissingHashReturnsNil`）
- `src/server/gkill/dao/reps/git_commit_log_cached_nested_pool_test.go`（構築中のフォールバックが変わっていないこと）
