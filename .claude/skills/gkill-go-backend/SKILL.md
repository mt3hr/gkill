---
name: gkill-go-backend
description: "gkill の Go バックエンド（src/server/gkill/）の禁止事項と契約。検索フィルタ（rep名の絞り込み・タグ絞り込みの2経路・FindQuery.IDs のチャンク分割）、キャッシュrepの判定とライトスルー、キャッシュのフルリビルド抑止、リポジトリ4層と doc コメント網羅、safefetch・共有ファイル認可・型別 GetXxx の最新版取得などの HTTP セキュリティを扱う。src/server/gkill/api/・dao/・usecase/・req_res/ を編集・追加するとき必読。「検索が黙って0件になる」「追加したタグが最大1分見えない」「全種類の検索が数分止まる」「タグの個数で検索結果が変わる」を調べるときも必読。"
---

# Go バックエンドの不変条件

対象: `src/server/gkill/api/**` / `src/server/gkill/dao/**` / `src/server/gkill/usecase/**`

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

### Backend (Go) — `src/server/`

Module: `github.com/mt3hr/gkill/src/server` (Go 1.26.6)

Key packages:

- `gkill/api/` — Shared infrastructure: `embed.go` (`//go:embed` serves Vue SPA at `/`), `version.go`, `gkill_version_data.go`, `find_filter.go`, `find_filter_helpers.go`, `find_kyou_context.go`
- `gkill/api/gkill_server_api/` — HTTP API handlers (112 files incl. tests, 1 handler per file). `GkillServerAPI` struct with `serve.go`, `close.go`, route definitions in `gkill_server_api_address.go`. Auth middleware (`auth.go`, `auth_context.go`, `auth_middleware.go`) extracts session→account→device→repositories via `AuthContext`, `authMiddleware`, `authWithReposMiddleware`. Handler registration uses wrapper functions: `wrapNoAuth` (no session), `wrapAuth` (session + account), `wrapAuthRepos` (session + account + device + repositories). Utility files: `filter_local_only.go`, `utils.go`, `web_push.go`. ZIP browsing: `handle_browse_zip_contents.go` (path traversal prevention, Shift_JIS→UTF-8, singleflight dedup).
- `gkill/api/req_res/` — Request/response structs for every endpoint (189 files)
- `gkill/usecase/` — HTTP-independent business logic (17 files). Extracted from handlers to enable reuse without HTTP context. Functions operate on DAO/repository types directly.
- `gkill/dao/` — Data access layer with `GkillDAOManager` managing SQLite3 databases
- `gkill/dao/reps/` — Repository interfaces and implementations for each data type.

**Key dependencies:** gorilla/mux, modernc.org/sqlite (pure Go), spf13/cobra+viper, asticode/go-astilectron, SherClockHolmes/webpush-go, go-git/go-git, twpayne/go-gpx, nicksnyder/go-i18n/v2, google/uuid

**Doc comments:** `dao/reps/*_repository.go` のインターフェースメソッドは **461/461 documented**、`gkill_server_api` の HandleXxx は **93/93 handlers documented**。どちらも verify_docs が網羅率を機械検査するので、メソッドやハンドラを追加したら doc コメントも書くこと。共通契約の完全文は基底 `dao/reps/repository.go` に1回だけ書き、型別ファイルは「契約は Repository.Xxx を参照。」の1行参照＋差分のみにする（457箇所へ複製するとドリフト源になるため）。ハンドラは「1行説明 / 空行 / パス・HTTPメソッド・認証区分 / req_res 型」の形式。

**Repository pattern:** SQLite-backed data types have 4 implementation layers: `*_repository.go` (interface) → `*_repository_sqlite3_impl.go` → `*_repository_cached_sqlite3_impl.go` → `*_repository_temp_sqlite3_impl.go`. Exceptions: `git_commit_log` (`_local_dir_impl` + `_cached_sqlite3_impl`), `gps_log` (`_gpx_dir_impl`), and `plugin` (`plugin_repository_impl`) read external sources directly and do not have all 4 layers.

**`FindQuery.IDs` は分割して渡す（SQLのバインド変数には上限がある）:** **新しくIDリストを渡す検索を書くときは `findChunkedByIDs`（`repositories.go`）を通すこと**。Mi / MiReKyou は5射影のUNIONでバインド変数が `5N+5` になり、SQLite の上限を **N=6553 で超える**。今かかっているのは `Repositories.findKyous` / `MiRepositories.FindMi` / `MiReKyouRepositories.FindMiReKyou` と、`GkillRepositories.FindTags` / `FindTexts` の中で最新版アドレスからIDリストを作っている箇所。上限は `maxIDsPerFindQuery`（4000）。**この壊れ方は静かなので気付きにくい** ―― 失敗しても `GkillError` が立たず HTTP 200 + `errors: null` + 0件で返る。受け皿として `message.EnsureNotEmpty` で `ERR000410` を必ず立てるようにしてある。**「失敗したのに GkillError が空」で return する分岐を新しく作らないこと。** 回帰は `repositories_id_chunk_test.go` / `handle_get_kyous_mcp_test.go` の `TestHandleGetKyousMCP_ManyIDs` / `message/gkill_error_test.go` が守る。経緯と却下案は [ADR-0105](../../../documents/adr/0105-chunk-find-query-ids.md)。

**rep名の絞り込みは「検索するrep」ではなく「検索結果」でやる:** `find_filter.go` の `selectMatchRepsFromQuery` で **`UnWrap()` の戻り値を `MatchReps` に入れてはいけない**。`UnWrap()` はインメモリキャッシュrep（と `--cache_reps_local` のローカルコピー層）を剥がして生のディスクrepに戻すので、**キャッシュを丸ごとバイパス**して端末別重複repぶんディスクを舐めることになる。`UnWrap()` を使ってよいのは「そのラッパに選ばれた実repが1つでもあるか」の**枝刈り判定だけ**（これは省けない）。実際の絞り込みは `findKyous` の `filterKyousByRepName` が `Kyou.RepName` で行う。落とし穴が5つある: **(1)** 本文ヒット由来の2本目の検索（`matchTextFindByIDQuery`）にも同じ絞り込みが要る、**(2)** 全部落ちたIDは**キーごと消す**（空スライスを残すと `kyous[0]` を見る `filterLocationKyous` / `filterMiForMi` / `overrideKyous` が panic する）、**(3)** `Reps == nil` は「未指定」で `len()` で判定すると全件消える、**(4)** **`RepName` が空の行は残す** ―― 追加直後の行は `REP_NAME` が空なので、落とすと**いま追加した記録が最大1分間一覧から消える**。**(5)** 書き込み側に**実在しないrep名を入れさせない** ―― 非空のrep名は「実在するが選ばれていないrep」として落とされるので、合成した名前を渡すと記録が黙って消える。フィルタ側に `*Temp` の例外を足してはいけない（直すのは常に書き込み側）。**(6)** 同じ約束は `/api/get_kyou` の版履歴（`Repositories.GetKyouHistoriesByRepName`）にも効く ―― **rep名の指定が無いときは `UnWrap()` しない**。剥がすと約850本の leaf（USB上の元DB・git の全コミット走査・プラグインIPC）を引き直しのたびに舐め、プラグインのキュー待ち超過で `/api/get_kyou` 全体が ERR000101 になる。剥がしてよいのは rep名で leaf を照合するときだけ（キャッシュrepの `GetRepName()` は集約名を返す）。前段の `GkillRepositories.GetKyouHistoriesByRepName` は最新版アドレス表に載っているIDでプラグインrepを外す（プラグインKyouのIDは表に載らない）。守るテストは `repositories_get_kyou_histories_cache_test.go`、経緯は [ADR-0218](../../../documents/adr/0218-get-kyou-histories-via-cached-reps.md)。`dao/reps` 側にも置いてはいけない（ReKyou/MiReKyou のワード委譲が利用者のクエリをそのまま `FindKyousSequential` へ渡すので、チェックしていないrepに参照先があるリポストが黙って語句検索に当たらなくなる）。**SQL の WHERE へ降ろすのも不可**（相関サブクエリに足すと最新版判定が壊れる → [ADR-0102](../../../documents/adr/0102-no-rep-name-in-sql.md)）。守るテストは `select_match_reps_cache_test.go` / `find_kyou_rep_name_filter_test.go` / `get_kyous_rep_filter_test.go` / `get_kyous_tx_rep_filter_test.go`（**すべてキャッシュON/OFFの両方**）。実測（11rep→約940rep・20.7秒）と事件の経緯は [ADR-0101](../../../documents/adr/0101-filter-rep-after-cache.md)。

**タグ絞り込みの取得は2経路。切り替えは「タグ名の個数」で決める:** `find_filter.go` の `collectTagsForFilter` が `MatchTags`（クエリのタグ名に一致するタグ）・`RelatedTagIDs`（タグが1つでも付いているIDの集合）・`AllHideTagsWhenUnchecked`（強制非表示タグ）の3つを作る。切り替えの閾値は `maxTagNamesForSQLFilter`（32。**クエリのタグ名と非表示タグ名の合計**で見る）。**「タグ無し」仮想タグ（`NoTags`）を使う検索では、名前の個数によらずGo側で照合する**（`RelatedTagIDs` のために結局は全タグを取るので、そこから名前を拾うぶんはタダになる）。照合は**完全一致・大小無視**で、`filterTagsKyous` のAND分岐と同じ意味論（SQLは TAG 列だけでなく ID 列とも突き合わせていたので、そこも写してある）。**2経路が同じ結果を出すことをテストで固定してある**（`find_filter_test.go`）—— ずれるとタグの個数によって検索結果が変わるという静かな壊れ方になる。`RelatedTagIDs` の読み手は `NoTags` 分岐しか無いので、`NoTags` を使わない検索では作らない（走らせ忘れると**全件が「タグなし」扱い**になりタグの付いた記録まで返る。回帰は `get_kyous_tag_filter_test.go` と `get_kyous_regressions_test.go` の `TestHandleGetKyous_TimeIsTagsFilterWorksWithoutKyouTagFilter`）。強制非表示タグも2経路の一致を `TestHandleGetKyous_HideTagsBothPaths` が固定する。実測表（2万タグでの交差点）と却下案は [ADR-0103](../../../documents/adr/0103-tag-filter-threshold-32.md)、実測表そのものは `dao/reps/tag_find_bench_test.go`。

**キャッシュのフルリビルドは「実DBが変わったときだけ」:** cached rep の `UpdateCache` は `DELETE FROM` + 全行再INSERT なので、変わっていない rep まで作り直すと共有の書き込みロックを握ったまま全種類の検索が止まる。抑止は `dbFileChangeDetector`（mtime + サイズ）で、基準を進めるのは**再構築成功後の `CommitCacheRebuild` だけ**（失敗した回を取りこぼさないため）。`--cache_reps_local=true` のときに挟まる `*_local_cached.go` も同じ約束を守ること ―― **「コピーが要るか」の判定は必ず `os.Remove` より前に置く**。消してから `os.Stat` すると常に「要コピー」になり、`LastUpdateCacheChanged()` が常に true を返して抑止が丸ごと無効化される。共通ヘルパは `local_rep_cache_copy.go`、回帰検出は `local_rep_cache_granular_test.go`。**ReKyou / MiReKyou は変更検知に載せてはいけない**（コピー省略のみ可）: この2つはターゲット解決のためアドレス確定後にもう一度更新される仕様で、mtime判定を入れると2回目が飛んでターゲット未解決の中身が残る。実測（phase1 が 0.2秒→1〜2分）と却下案は [ADR-0202](../../../documents/adr/0202-rebuild-cache-only-on-db-change.md)。

**Git コミットログのキャッシュ包装は、構築済みなら外れた ID で生リポジトリへ落ちない:** `git_commit_log_repository_cached_sqlite3_impl.go` の `GetGitCommitLog` / `GetKyou` / `GetKyouHistories` は、SQL で 0 件のとき `shouldFallbackOnMiss`（`UpdateCache` が1回も完走していない、またはバックグラウンド構築が失敗した後）が真のときだけ下層の集約へ逐次フォールバックする。**構築済みの 0 件は「native の rep に無い」の答えそのもの**で、集約 `GitCommitLogReps` の隣のプラグインアダプタが持っていれば集約側で拾う。2026-09-16 まで無条件に落ちていて、`provides: ["git_commit_log"]` の archived プラグインが載せた 3,447 件のコミット行を rykv が行ごとに `/api/get_git_commit_log` で引くたびに、下層の `gitCommitLogRepositoryLocalImpl` が `Log(From: hash)` の失敗を `Log(All: true)` の**全履歴走査**で切り分け（18 リポジトリ×約 4,260 コミット）、1 行約 1 秒×多コアでサーバ全体が飽和した（`get_tags_by_id` すら 3〜6 秒）。生実装側は `hasCommit`（`CommitObject` で packfile の索引を引く。`ErrObjectNotFound` のときだけ nil 即返し、他の失敗は従来の `Log` 経路）で、アドレス表に無い ID を全 leaf へ配る `GkillRepositories.GetKyou` でも git の leaf は数 ms で外れる。**フォールバックを丸ごと消してはいけない**（初回のバックグラウンド構築中と失敗後は表が空か部分的。`TestGitCommitLogCachedBuildingFallbackReturnsUnderlyingData` が固定）し、**行数で「構築済み」を判定してもいけない**（永続 DB は前回 run の内容で行数がある）。守るテストは `git_commit_log_cached_miss_test.go` と `git_commit_log_repository_local_dir_impl_test.go` の `TestGitCommitLogLocalDirMissingHashReturnsNil`。実測と却下案は [ADR-0221](../../../documents/adr/0221-git-cache-miss-does-not-fall-back-to-raw-walk.md)。

**最新版アドレス表へ焼く rep 名は「行ごとの実rep名」。集約名を焼いてはいけない:** キャッシュrepの `GetLatestDataRepositoryAddress` は `LATEST_DATA_REPOSITORY_NAME` に **`REP_NAME` 列を射影する**（`? AS ...` に `GetRepName()` をバインドしない）。キャッシュrepが包んでいるのは集約なので `GetRepName()` は `"KmemoReps"` / `"IDFKyouReps"` のような**どのrepにも一致しない固定文字列**を返す。`GkillRepositories.GetKyou` はこの名前を `Reps.UnWrap()` が返す leaf rep の実名と突き合わせて問い合わせ先を1repへ絞るので、集約名を入れると比較が永遠に外れ、**エラーも立たず `(nil, nil)`** が返る。2026-08-24 に `usecase/tag.go` / `usecase/text.go` の実在検査がこれを踏み、**実在する記録へのタグ/テキスト追加が軒並み `ERR000092` で失敗**した。**`?` を1つ減らしたら `QueryContext` の引数も対で落とす**（残すとバインドがずれて静かに0件。2026-08-02 の `notification_repository_cached_sqlite3_impl.go` と同型）。**MiReKyou だけは例外** —— `UnWrap()` が自分自身を返すので両辺が同じ `GetRepName()` で自己一貫しており、`REP_NAME` へ変えるとむしろ壊れる。壊れ方は条件つきで、APIの書き込み経路はアドレス表へ leaf 名を直接書くため**同じプロセスで追加した直後の記録には付けられる** —— 失敗するのは UpdateCache が索引した後（スキャン由来の全記録と再起動後の全記録）。キャッシュOFFでは再現しない。守るテストは `dao/reps/gkill_repositories_get_kyou_test.go` と `dao/reps/latest_data_address_rep_name_scan_test.go`（ソース走査）。経緯と実測は [ADR-0210](../../../documents/adr/0210-latest-data-address-uses-row-rep-name.md)。

**`len(XxxReps) == 1` でキャッシュrepを判定してはいけない。** アダプタの append は「キャッシュrepで `XxxReps` を1個に差し替える」処理より後なので、`provides` を持つプラグインが1つ入るだけで長さが2になる。書き込み後のキャッシュ反映は構築時に控えた `GkillRepositories.CachedReps` を見る `repositories.WriteThroughXxxCache(ctx, ...)` を使うこと（28箇所）。読み取りはキャッシュrepしか見ず下層repへフォールバックしないので、反映を飛ばすと追加したタグが最大1分見えず、その間にPWAが古い応答をキャッシュし直すと**恒久的に古いまま焼き付く**。再発は `usecase/write_through_cache_test.go` の `TestNoRepsCountCacheGuard` がソース走査で落とす。経緯と却下案は [ADR-0203](../../../documents/adr/0203-write-through-cache-not-reps-count.md)。

**読めない rep は切り離すが、黙って落とさない。`IsEnable` も書き換えない。** `GetRepositories` は rep を1本ずつ組み立てるが、どこか1本の失敗で `return nil, err` すると auth middleware が全リクエストで落ち、**そのユーザの全APIが500**になる（設定画面も開けず `/api/reload_repositories` 自身も `GetRepositories` を呼ぶので、自力復旧の手段が残らない。2026-08-30にUSB接続ディスク上の `gkill_id.db` の破損で実際にそうなった）。読み取り専用の rep は `canDetachRepOnLoadFailure` で1本だけ切り離し、`AppendLoadFailure` へ記録して検索の応答に `MSG000090` の警告として返す。**書き込み先 rep は切り離さない** —— `WriteXxxRep` が nil のまま書き込み経路へ入ると nil ポインタ参照で落ちる（`commit_tx` は書く前に `CommitTxWriteRepMissingError` で返すが、tx を使わない `add_*` 経路は落ちる）。**失敗の理由で分岐しない**（SQLite のエラーコードを見て「壊れているときだけ」に絞ると、ドライバや OS の実装詳細が変わった回にだけ静かに全滅へ戻る。[ADR-0208](../../../documents/adr/0208-git-repo-detect-by-os-stat.md) と同じ罠）。**続行/中止の判定は `GetRepositories` の1箇所に集約する** —— 組み立て本体の `loadRepIntoRepositories` へ書き戻さないこと。警告に載せる名前は `repNameFromDefine` で作り、各 leaf の `GetRepName` と同じ規則を保つ（ずれると `query.reps` へ渡せない名前を案内することになる）。**`GkillMessage` には `GkillError` のような伏せ処理が無いので、パスを載せない**。構築に失敗したら部分構築済みの rep を必ず `Close` すること（失敗はキャッシュされないのでリクエストのたびに `*sql.DB` が積み上がり、Windows では壊れた `.db` の差し替えすらできなくなる）。守るテストは `dao/gkill_dao_manager_broken_rep_test.go` と `api/gkill_server_api/broken_rep_warning_test.go`。却下案は [ADR-0216](../../../documents/adr/0216-detach-a-broken-rep-but-never-silently.md)。

**Mi の1件取得は、どの射影を名乗るかを SQLite の UNION 出力順に決めさせない。** `MI` テーブルに `DATA_TYPE` 列は無く、5射影（`mi_create` / `mi_check` / `mi_limit` / `mi_start` / `mi_end`）は**同じ1行から SQL が合成するラベル**。5つとも `UPDATE_TIME` が同着なので、素の `slices.MaxFunc` は「UNION が返した先頭」を返す。SELECT の列の並びが違うだけで勝つ射影が変わり、実際 `GetMi` は `mi_check`・`GetKyou` は `mi_create` を返して**1つの応答の中で種別名が食い違っていた**（2026-08-24 の再監査）。同着は `compareMiProjectionPreference` で正準（`mi_create`＝検索の既定 `mi_sort_type=create_time` と同じ）に割る。**単体取得は `&kyous[0]` を返さない**という既存の規則（外部監査 H-07）から外れていた2箇所（`mi_repository_cached_sqlite3_impl.go` / `mi_re_kyou_repository_sqlite3_impl.go`）も揃えてある。

**索引を持つ rep は鮮度を出す。** rep ディレクトリへ置いただけのファイルは `UpdateCache` が `IDF()` を走らせるまで検索に出ないが、**定期実行も監視も無く、警告も出ない**ので「0件」が「まだ取り込んでいない」なのか「本当に無い」なのか区別できなかった。`get_rep_infos` の `indexed_at`（任意インタフェース `IndexUpdatedAt` を実装した rep だけ）で判断させる。**検索のたびにディレクトリを全走査して未採番を数えてはいけない**（実データは56万行規模）。

**派生キャッシュの「失敗の印」は、そのファイル固有の失敗にだけ焼く。** サムネイル（`markThumbFailed`）と互換動画（`markCompatFailed`）は、生成に失敗したファイルの脇へ `.failed` を置いて二度と挑まない —— デコードできないファイルが一括生成のたびに全件やり直しになるため。**焼いてよいのは「何度やっても同じ結果になる失敗」だけ。** `ffmpeg` / `ffprobe` が PATH に無いのは「このファイルが変換できない」ではなく「今この環境ではどの動画も変換できない」なので、印にすると ffmpeg が使えるようになっても二度と生成されず、`clear_cache thumb` でキャッシュを丸ごと捨てるまで直らない（消したぶんの作り直しに数十分〜数時間かかる）。2026-09-06 にそうなった —— 本番サービスは LocalSystem 起動でシステムのPATHしか見えず、ffmpeg は利用者のPATHにしか入っていなかったので、**ブラウザで一覧を開いた1回**で動画1repぶんの印が焼き付き、ffmpeg の見える CLI から `generate_thumb_cache` を何度回しても作られなくなった。判定はセンチネル `errFFToolsNotAvailable` を `errors.Is` で見る（メッセージ文字列で照合しない）。`ctx` が切れているときに焼かないのと同じ判断で、互換動画側は ffmpeg 不在なら手前で原本へフォールバックして印を残さない。守るテストは `idf_thumb_batch_test.go` の `TestMarkThumbFailedSkipsMissingFFTools` / `TestGenerateThumbCacheDoesNotMarkFailedWhenFFToolsMissing`。

**commit_tx は1つの SQLite トランザクション**（2026-09-15、[ADR-0219](../../../documents/adr/0219-commit-tx-is-one-sqlite-transaction.md)）。`dao/reps/commit_tx.go` の `GkillRepositories.CommitTx` が、temp rep に積まれた全種別を「行がある種別の書き込み rep のファイルを1接続に ATTACH した1つのトランザクション」で追記する。途中で失敗したら ROLLBACK で**何も書かれない**（ERR000419。rep ファイルは journal_mode=DELETE + synchronous=FULL なので super-journal が効き、プロセス断でも半端は残らない）。守ること: **(1)** 各 leaf の INSERT は `insertXxxRow(ctx, db sqlite3impl.Preparer, x)` にだけ置き、`AddXxxInfo` は接続を選んでそれを呼ぶだけにする。`AddXxxInfo` 側へ SQL や検査を複製すると tx 経由の追記だけが静かにずれる。IDF は自 rep 名を引数で受ける（`TARGET_REP_NAME` を空にする判定に使う）。**(2)** 13 型のテーブル名（KMEMO / TAG / ... / IDF）を重複させない —— ATTACH した接続では非修飾名が main → ATTACH 順に解決されるので、重複すると別のファイルへ落ちる（`commit_tx_test.go` の `TestCommitTx_LeafTableNamesAreUnique` がソース走査で固定）。**(3)** usecase の `Add*` / `Update*` は `txID != nil` のとき最新版アドレス表を進めない（commit が書く）。2026-09-15 まで進めていて、失敗 → `discard_tx` のあと表だけが未来を指し、`find_filter.go` の「表より古い版は除外」で**既存の TimeIs が検索から消えていた**（`handle_commit_tx_atomic_test.go` の `TestHandleUpdateTimeIsInTx_DiscardKeepsExistingTimeIsVisible`）。**(4)** commit は成功したら temp の行を消す（tx を消費する。残すと同じ txID の再 commit で丸ごと二重登録）。失敗時は残す（再 commit / discard できる）。**(5)** Go 側 KFTL（`api/kftl/`）は実 rep へ直書きせず `TempReps.XxxTempRep.AddXxxInfo(ctx, x, TXID, UserID, Device)` に積み、`GenerateAndExecuteRequests` の末尾で `CommitTx`、失敗したら `DiscardTx` して `created` は空で返す。**(6)** キャッシュへのライトスルーと最新版アドレス表の更新はトランザクションの**外**（commit 後、`commitTxAfterRows`）で、書き込み rep 名を入れ直してから渡す（temp の合成名 "KmemoTemp" 等を持ち込むと `filterKyousRepName` で一覧から消える。`source_conventions_scan_test.go` が順序を固定）。ATTACH できる数は SQLite のコンパイル時上限（modernc は 10、main と合わせて 11 ファイル）に従い、超えれば ATTACH 自体が失敗して何も書かずに返る（既存の操作は KFTL 最大 10 種別）。

**SQLite の 'localtime' は Go の time.Local と独立に決まる**（2026-09-16、[ADR-0220](../../../documents/adr/0220-sqlite-localtime-follows-libc-zone.md)）。`'localtime'` 修飾子は libc の `localtime_r` の値で、gkill の SQLite（modernc、musl 転写）は **`TZ` 環境変数 → `/etc/localtime` → 無ければ UTC** で決める。Go の `time.Local` はこれと別に決まる（Android では `main/common/fix_timezone.go` が getprop から直す）ので、揃っていない環境では時間帯フィルタの SQL 段（`GenerateFindSQLCommon` の `strftime(..., 'localtime')`）と Go 段（`find_filter.go` の `newKyouTimeFilter`）が別の壁時計で判定し、**9時間より狭い窓の検索がエラーも警告も出ないまま常に0件**になる（Android で実際にそうなった。PC では再現しない）。`'localtime'` を使う SQL を新しく書くときはこの前提を知っておくこと（TimeIs の playing 判定は両辺に掛かるので相殺される）。**SQL 段の時間帯・曜日条件を外して Go 段だけにしてはいけない**（全履歴走査で全行が Go へ実体化される）、**クエリ時点のオフセット1つをバインドする形にもしない**（DST 地域で窓の端1時間が黙って落ちる）。起動時の `checkSQLiteLocaltime` が両者を同じ瞬間で突き合わせ、食い違えば `gkill_error.log` に Error で1行残す。守るテストは `dao/sqlite3impl/localtime_check_test.go`（Linux で `TZ=:/nonexistent` の子プロセスに Android の食い違いを再現）。

**タグ語彙の列挙は2つある。検証には「対象の生死を問わない」ほうを使う。** `GkillRepositories.GetAllTagNames` は**対象が削除済みのタグを落とす** —— 記録を消してもタグは消えない（消すと `gkill_restore_kyou` で復活したときにタグが失われる）ので、落とさないと「選んでも0件」の候補が溜まり続ける。生存判定は最新版アドレス表を引くだけで**追加のI/Oは無い**（`GetAllTags` が既に `TargetID` を持って返る）。**アドレス表に載っていない対象は落とさないこと** —— プラグインや git の記録は表に載らないので、落とすと語彙が黙って痩せる。一方「そのタグ名は実在するか」の検証（`collectMCPUnknownValueWarnings`）は `GetAllTagNamesIncludingDeletedTargets` を使う。フィルタ済みの一覧で検証すると、`include_deleted_data:true` で削除済みを開いたタグ検索に**未知のタグという誤警告**が出る。却下案（カスケード削除・SQLへの降ろし）は [ADR-0112](../../../documents/adr/0112-tag-vocabulary-drops-dead-targets.md)。

### HTTP ステータス（2026-08 導入）

**エラーコード → HTTP ステータスの表が正本。** `api/message/http_status.go` の
`errorCodeHTTPStatus`（377件、400/401/403/404/409/413/429/500）と `HTTPStatusForErrors`。
2026-08 まで `/api/*` の JSON ハンドラは**全部が暗黙の200**で、セッション切れも権限不足も
内部エラーも「成功」に見えていた。ステータスを見る層（監視・プロキシ・アクセスログ・
素朴なHTTPクライアント）から障害が完全に隠れていた。

守ること3つ。どれも破っても目の前ではエラーにならない。

1. **エラーコードを足したら `http_status.go` の表にも1行足す。** 迷ったら 500。
   落とすのは `message/http_status_test.go`（`error_codes.go` をソース走査して未分類を検出。
   ステータスごとの件数も固定してあるので「全部500にしておく」も落ちる）。
2. **ハンドラを足したら `writeErrorStatus(w, response.Errors)` を
   `json.NewEncoder(w).Encode(response)` の直前に置く。** 順序が命で、
   本文を1バイト書くと net/http が 200 を確定させ、あとから `WriteHeader` を呼んでも
   `superfluous response.WriteHeader` がログに出るだけで**ステータスは200のまま返る**。
   errors 配列は正しいので画面は普段どおり動き、気付けない。
   落とすのは `gkill_server_api/response_status_guard_test.go`（全 `handle_*.go` を走査。
   免除は `handle_file_serve.go` / `handle_urlog_bookmarklet_*.go` の3本だけ）。
3. **ハンドラより手前で打ち切る経路（`auth_middleware.go` / `filter_local_only.go`）は
   `writeGkillErrorResponse` を通す。** 直に `json.NewEncoder(w).Encode(...)` を書くと
   ステータスを書き忘れる。**本文を必ず JSON で返すのも要件** ——
   クライアント（`gkill-api.ts`）はステータスを見ずに `res.json()` するので、
   本文が空だとそこで例外になり、ログイン画面に「証明書が必要です」という
   無関係な文言が出る。

**名前から機械的に導けないので推論に置き換えないこと。** `error_codes.go` の語彙には
`Forbidden` / `Unauthorized` / `Denied` / `Permission` が1件も無く、`Invalid*` が 400 と 500 に、
`NotFound*` が 401 と 404 に跨る。取り違えると実害が出る割り当ては
`http_status_test.go` の `TestHTTPStatusOf_KnownAssignments` が名指しで固定してある
（`AccountSessionNotFoundError` は404ではなく**401**、`NotFoundTLSCertFileError` は
サーバの設定不備なので404ではなく**500**、など）。

**`AccountNotFoundError`(ERR000002) は認証経路（`auth.go`）専用。** 操作対象のアカウントが
無いときは `TargetAccountNotFoundError`(ERR000413)。混ぜると、クライアントの `check_auth` が
ERR000002 でログアウトさせるので、**存在しないユーザIDにパスワードリセットを実行した
管理者がその場で締め出される**（2026-08 まで実際にそうなっていた）。

**`recoverMiddleware` は `serve.go` で最外層と最内層の両方に登録してある。** 内側が要るのは、
`gzipMiddleware` の `defer gzipWriter.Close()` が panic の巻き戻しで先に走り、
空の gzip ストリームを書いて暗黙200を確定させるため。外側の recover が書く500は捨てられ、
**200 ＋ 復号すると空の本文**が返っていた（再現テストで確認）。順序を戻さないこと。
守るのは `response_status_test.go` の `TestResponseStatus_PanicReturns500WithGzip`。

**外向き（gkill がHTTPクライアントのとき）も 2xx を見る。** `api/safefetch` の `GetCapped` は
2xx 以外をエラーにする。見ていなかったので404ページのHTMLが favicon として base64 で
保存されていた。favicon は加えて `LooksLikeSupportedImage`（gif/jpeg/png/webp。判定形式は
`use-ur-log-view.ts` の `base64_to_data_uri` と揃える）と `CheckImageDimensions` を通す。
スキーム無しURLは `u.Hostname()` が空になり Google が汎用アイコンを**200**で返すので、
リクエスト前に弾く（`dao/reps/ur_log.go` の `getFavicon`）。

### エラーの種類と理由（2026-09 導入）

**`GkillError` は `error_code` + `error_message` に加えて、marshal 時に `error_kind`（誰の問題か）と `reason`（何が起きたか）を
機械語のトークンで載せる。** 文面（633箇所の `ErrorMessage`）は操作単位のまま変えない —— 「メモ追加に失敗しました」に
「書き込み先が未設定」「USB が外れた」「DB がロック中」を足すのは、文面ではなく別フィールドの仕事。
正本は `api/message/error_kind.go`（`KindOf`。既定は HTTP ステータス、500 のうち設定不備だけ `errorCodeKindOverride` で `config`）と
`error_reason.go`（`ReasonOf`。`errors.Is` / `errors.As` だけで分類、**文字列照合はしない**）。
ヒント文はワイヤに載せず、消費者側（Web の `error-hints.ts` + i18n）が引く。経緯と却下案は [ADR-0710](../../../documents/adr/0710-error-kind-and-reason-on-the-wire.md)。

守ること4つ。

1. **`if err != nil` の中で `GkillError` を組み立てるときは `Cause: err` を付ける。** reason の分類と、
   `writeErrorStatus` が出す `request failed` の1行の `causes` の源。付け忘れても応答は返るので目の前では気付かない。
   `gkill_error_cause_scan_test.go` がソース走査で落とす。`EnsureNotEmpty` の第4引数も同じ（手元の `err` を渡す）。
   `err` 以外の名前（`decodeErr` 等）で受けているならその識別子を渡す。
2. **`errors` / `messages` は成功時も `[]`。** レスポンス構造体の型は `message.GkillErrors` / `message.GkillMessages`
   （`[]*message.GkillError` に戻すと成功時が `null` に戻る）。ハンドラより手前で書く経路（`writeGkillErrorResponse` /
   `recoverMiddleware`）も同じ形（`messages` も配列、`error_kind` 付き）。
3. **dao の番兵・型付きエラーは `message.Reasoner`（`ErrorReason() string`）を実装する**（`reps.ErrPluginBusy` /
   `ErrPluginReturnedErrors` / `CommitTxWriteRepMissingError`）。`errors.Is` の同一性は保つ。新しい理由を足すときは
   Go の `reasonTokens`、Web の `error-hints.ts`、i18n 7言語の `ERROR_HINT_REASON_*` を揃える（`error-hints.test.ts` が突き合わせる）。
4. **書き込み先 rep が nil なら `writeRepMissingError`（usecase/write_rep_missing.go）で返す。** 2026-09-15 まで tx を使わない
   `add_*` は nil ポインタ参照で panic し「内部エラーが発生しました」だけが出ていた。`WriteRepMissingError`（ERR000422）は
   500 だが kind `config` / reason `write_rep_missing` で、Web は「設定 → 保存先で書き込み先を選ぶ」と案内する。

`GkillMessage` には `Level`（`info` 既定 / `warning`）がある。成功はしたが対処が要る知らせ（rep 読み込み失敗 MSG000090・
プラグイン警告 MSG000088）は `warning` で返す —— Web は閉じるまで残す（info は 2.5 秒で消える）。

### HTTP API のルート表（2026-09 導入）

**ルートの正本は `gkill_server_api_address.go` の `apiRoutes()` の表1つ。`serve.go` にもテストハーネス
（`gkill_server_api_test.go` の `setupTestRouter`）にも `router.HandleFunc` を直に書かない。** 両方が
`registerAPIRoutes` で同じ表を登録する。表の1行は `{Path: "/api/xxx", Method: "POST", Auth: authXxx, Body: bodyXxx, Handler: g.HandleXxx},`
を**1行で**書く（`verify_docs` と `gkill-api.test.ts` が正規表現で読む。`Method` は `http.MethodPost` ではなく文字列リテラル）。
2026-09-14 まで同じ表が Go 定義・`serve.go`・テストハーネスの部分コピー・`gkill-api.ts` の4箇所に手書きされ、
「アドレス定義はあるがハンドラ未登録で実行時404」の残骸が2件、テストハーネスには本番と違うラッパーで登録された経路が
4本あった（アップロード2本が上限なし、MCP 2本が `wrapAuthRepos`）。

守ること3つ。

1. **エンドポイントを足したら、ハンドラ・表の1行・`api_routes_test.go` の golden の1行・Web から叩くなら
   `gkill-api.ts` の `xxx_address` / `xxx_method` の対、を揃える。** 表へ足し忘れるとハンドラがあっても 404
   だが、`TestAPIRoutes_EveryHandlerIsRouted` が `HandleXxx` の反射列挙と表を双方向に突き合わせて落とす。
2. **認証区分（`Auth`）は表と golden の2箇所を意図して直す。** `TestAPIRoutes_AuthKindGolden` が名指しで
   固定しているので、`wrapAuthRepos → wrapNoAuth` のような退行はテストを落とさずには通らない。
   無認証で POST を受ける経路は `Body` に `bodyAuth` / `bodyUpload` を付ける（`validateAPIRoutes` が起動時に拒否、
   `TestAPIRoutesNoAuthBodyRoutesAreCapped` が名指しで固定）。
3. **doc コメントの `// POST /api/xxx（wrapXxx）` 行は表と一致させる**（`TestAPIRoutes_DocCommentMatchesTable`）。
   `（wrapNoAuth）` は capped 版も含む区分名。

Web クライアント側は生成せず、`gkill-api.test.ts`「endpoint address parity with Go」が表とパス・メソッド・
`_address`↔`_method` の対・`gkill_fetch` の配線を突き合わせる。OpenAPI 正本化＋生成を採らなかった理由と却下案は
[ADR-0709](../../../documents/adr/0709-api-route-table-single-source.md)。

### HTTP セキュリティ（2026-08 外部監査由来）

**共有ページのファイル配信は共有クエリの結果に含まれるファイルだけ**（2026-08-21、監査 C-03）。`handle_file_serve.go` の `sharedID != ""` 経路は、rep名一致だけで IDF rep へ委譲せず、`shared_file_authz.go` の `collectSharedIDFFilePaths` で共有 `FindQuery` を再評価して許可パス集合を作り、要求パスが集合に無ければ403にする（`handle_get_shared_kyous.go` と同一手順を共有＝一覧を2箇所で維持しない）。パス正規化は `idf_file_url.go` の `cleanRelativeURLPath` と同一。セッション経路はフルアクセスのまま（`sharedKyouInfo != nil` のときだけゲート）。守るテストは `shared_file_authz_test.go`。 却下案（許可パスを共有作成時に固定する等）は [ADR-0703](../../../documents/adr/0703-shared-file-authz-by-query.md)。

**利用者入力URL・そのページが指す og:image / #landingImage の取得は必ず `api/safefetch` を通す**（2026-08-21、監査 H-04）。`safefetch.GetCapped`（scheme検査・`Dialer.Control` での接続先IP検証・サイズ上限）と `safefetch.CheckImageDimensions`（`image.DecodeConfig` で復号前に総ピクセル検査）で SSRF・無制限read・画像爆弾を防ぐ。`dao/reps/ur_log.go` の getBody/getFavicon/getImageOG/getAmazonImage と `gkill_server_api` の `httpGetBase64Data`（ブックマークレット）が利用。既定は private 拒否（loopback/RFC1918/link-local(メタデータ)/multicast/unspecified）。`http.Get` を新しく直に書かないこと。守るテストは `api/safefetch/safefetch_test.go`。 文字列検査では守れない理由（DNSリバインディング）は [ADR-0704](../../../documents/adr/0704-safefetch-for-user-urls.md)。

**型別 `GetXxx(id, nil)` は最新版を返す**（2026-08-21、監査 H-07）。`dao/reps` の各 `*_repository_sqlite3_impl.go` / `*_repository_cached_sqlite3_impl.go` の単体取得は `onlyLatestData := query.OnlyLatestData`（`false` 固定にしない。`GenerateFindSQLCommon` はこの引数しか見ない）と `slices.MaxFunc(UpdateTime)`（`&xxx[0]` を返さない）で最新版を選ぶ。`GkillRepositories.GetKyou` はアドレス表 nil でも panic しない（プラグインKyou・追加直後の記録）。守るテストは `get_typed_latest_version_test.go` / `gkill_repositories_get_kyou_test.go` / `rows_err_check_test.go`。

**外部URLを取る新規コードや子goroutine内の index 参照に注意**: `strings.SplitN(x, ",", 2)` は末尾要素 `parts[len(parts)-1]` を取る（`[1]` を長さ未確認で参照しない。子goroutine内 panic は `recoverMiddleware` で回収できずプロセスが落ちる＝監査 H-01）。ファイル書き込みは tmp+rename で原子的に（Override で原本を壊さない＝M-03）。ZIP展開は件数・総展開量・圧縮比の上限つき（`handle_browse_zip_contents.go` の `maxZip*`＝M-04）。`http.Server` は `ReadHeaderTimeout`/`IdleTimeout`/`MaxHeaderBytes` を張り、認証前ボディは `maxAuthBodyBytes` でキャップ（H-03）。全レスポンスに `securityHeadersMiddleware`（nosniff / X-Frame-Options / Referrer-Policy）。CLIサブコマンドは `InitGkillServerAPI` 失敗で `return`（nil panic 回避＝M-8）、CLIの自己発行セッションは `IsLocalAppUser=false`（最小権限）。ログインは非存在ユーザとパスワード誤りを同じ error_code + 文言に統一し、非存在時もダミー Argon2id を実行（ユーザ列挙対策＝S3-login）。

**レスポンスへ載る自由文は端末固有の情報を伏せてから出す**（2026-08-24）。`GkillError.MarshalJSON` が `ErrorMessage` を `message.RedactEnvironmentSpecific` へ通し、ホームディレクトリのユーザー名（`C:\Users\〈ユーザー名〉` / `/home/〈ユーザー名〉/`）とメールアドレスをプレースホルダにする。**伏せるのは生成側ではなく marshal 側**（`ErrorMessage:` の代入は684箇所あり、そのうち `err.Error()` を埋めているのは4箇所。生成側で1つずつ包む方式は必ず足し忘れるし、新しいハンドラが同じ書き方をしても自動で載るのはこちらだけ）。`GkillError` でない自由文——プラグイン診断の `last_error` と `typed_index.last_build_error`——は `handle_get_plugin_list.go` で明示的に通す。パスの**形は残す**ので `C:\Windows\System32\config\systemprofile` は素通しになり、LocalSystem 起動でホームが化ける事故の診断は従来どおり成立する。**サーバのコンソールログには適用しない**（端末に閉じた人間の診断チャネルで、リポジトリへ入る経路が無い）。守るテストは `api/message/redact_test.go` / `handle_get_plugin_list_test.go`。 なぜ書き手側の約束ではなく出口で伏せるのかは [ADR-0707](../../../documents/adr/0707-redact-environment-specific-strings.md)。

**待受の既定はループバック限定。既定値は `server_config` の定数1組から引く**（2026-09-14）。初回起動で自動生成されるサーバ設定は `Address = server_config.DefaultListenAddress`（`127.0.0.1:9999`）・`IsLocalOnlyAccess = server_config.DefaultIsLocalOnlyAccess`（true）で、初回起動ブロック（`gkill_server_api.go`）と DAO の既定マップ（`serverConfigDefaultValue`）の両方がこの定数を引く。**既定値をどちらかにリテラルで書かないこと** —— 2026-09-14 まで初回起動ブロックは `":9999"` + 許可なし、DAO 側は `IsLocalOnlyAccess=true` と食い違っており、実効既定（初回起動ブロック）が全インターフェース待受だった。人生記録を保存するアプリなので、起動しただけで LAN の第三者から届く状態にしない。LAN の他端末から使うのはサーバ設定画面で「アドレス」と「ローカルアクセスのみ許可」を開く明示操作。**既存の `server_config.db` は移行も拒否もしない**（旧既定のまま LAN で使っている環境を締め出す。設定を直す CLI が無いので自力復旧できない）。`--address` は設定 DB を書き換えない実行時上書きなので、Android（`127.0.0.1:9999`）・E2E（空きポート）は既定に依存しない。TLS 無しで非ループバック待受のときの起動時警告（`close.go` の `printInsecureBindWarning`）はそのまま。守るテストは `close_bind_address_test.go` / `gkill_server_api_test.go` の `TestNewGkillServerAPI_FirstRunDefaultsAreLocalOnly` / `dao/server_config/server_config_dao_sqlite3_impl_test.go` の `TestGetDefaultServerConfigUsesListenDefaults`。却下案は [ADR-0708](../../../documents/adr/0708-local-only-listen-by-default.md)。

**IDF走査の一時停止は参照カウント**（2026-08-21、監査 M-02）。`SetSkipIDF(true/false)` は共有 `*bool` ではなく `*atomic.Int64` を増減し「カウント>0 で skip」。重なるアップロード（と UpdateCache）が互いのフラグを倒し合って watcher が走る/変更を取りこぼすのを防ぐ。カウントが0へ戻ったときだけ catch-up 走査を1回キックする（`UpdateCache` の Add(-1) では catch-up しない＝1分周期のリビルド無限ループになるため。`SetSkipIDF` はアップロードハンドラからしか呼ばれず watcher から再入しない）。

### ログレベル（2026-08-30）

**既定のログレベルは `error`。** `--log` を付けなくても `gkill_error.log` に出る。
以前の既定は `none` で、本番サービスのログファイルは全部0バイトだった。
「`gkill_error.log` に出ていなければ起きていない」と言えることが前提になっている。

**レベルは事象の重さで決める。判断は次の2つの軸をこの順で見る**（正本は [ADR-1001](../../../documents/adr/1001-log-level-by-severity.md)）:

1. **そのエラーは呼び出し元へ返るか。** 返る（`return err` / `gkillErrors` に積んで `return`）なら
   応答の `errors` に載り、`writeErrorStatus` が境界で1行出すので深部は **Debug でよい**
   （同じ失敗を2回 Error で書かない）。返らない（ログして継続・`defer` 内・goroutine 内）なら
   **そのログが唯一の記録なので Debug 禁止**。結果が痩せるだけなら Warn、データが壊れる・機能が止まるなら Error。
2. **原因が利用者側か、サーバ側か。** 入力・認証・認可・レート制限は Warn 以下（`gkill_error.log` を
   未ログインのアクセスで埋めない）。ディスク・DB・プロセス・設定は Error。

**失敗したリクエストの1行は `writeErrorStatus(ctx, w, response.Errors)` が出す。**
レベルはステータスから機械的に決まる（5xx=Error / 401・403・429=Warn / その他4xx=Debug。呼び出し側の中断
`reason=canceled` だけの 5xx は Debug）。行には `error_codes` に加えて `reasons` と `causes`（`GkillError.Cause` の文面）が載る
ので、深部のログが Debug でも原因はここで追える。
新しいハンドラは `writeErrorStatus(r.Context(), w, response.Errors)` の形で書くこと
（`response_status_guard_test.go` がエンコード行の直前にあることを機械検査する）。
アクセスログのミドルウェアは `Access` のまま触らない（レベルを可変にすると `gkill_access.log` の網羅性が崩れる）。

**メッセージに操作名を入れる。** `slog.Log(ctx, gkill_log.Debug, "error", "error", ...)` の形は禁止
（メッセージでの集計もアラートも作れない）。直前の `fmt.Errorf("error at XXX ...")` の操作名を
メッセージへ移す。値は `fmt.Sprintf("%q", ...)` で包む（CodeQL の log-injection バリアがこの形）。

**`log.Fatal` を使わない。** 標準ロガーの stderr へ書くだけで `gkill_error.log` に残らない。
`gkill_log.Fatal(msg, err)` を通すこと（例外は `gkill_log.Init` 自身のレベル不正だけ。ルータがまだ無い）。

守るテスト: `main/common/gkill_log/log_level_source_scan_test.go`（無情報メッセージ・defer Close の
対象別レベル・握り潰しの検出）/ `gkill_server_api/response_status_log_test.go`（ステータス→レベル）。

## 関連スキル

- [gkill-find-query](../gkill-find-query/SKILL.md) — `FindQuery` の null 意味論（Go/TS/MCP の3実装共通）
- [gkill-plugin](../gkill-plugin/SKILL.md) — プラグイン rep のアダプタ・プロセス寿命（`dao/reps/plugin_*.go`）
- [gkill-cli-ops](../gkill-cli-ops/SKILL.md) — 派生キャッシュのユーザー別ディレクトリと CLI の自己発行セッション
- [gkill-mobile](../gkill-mobile/SKILL.md) — KFTL 送信の冪等キー（`handle_submit_kftl_text.go`）

## 詳しい設計と却下案（ADR）

- [ADR-0101 rep名の絞り込みはキャッシュの後で](../../../documents/adr/0101-filter-rep-after-cache.md)
- [ADR-0102 rep名を SQL の WHERE へ降ろさない](../../../documents/adr/0102-no-rep-name-in-sql.md)
- [ADR-0103 タグ絞り込みの閾値32](../../../documents/adr/0103-tag-filter-threshold-32.md)
- [ADR-0104 RelatedTagIDs は NoTags のときだけ](../../../documents/adr/0104-related-tag-ids-only-for-no-tags.md)
- [ADR-0105 FindQuery.IDs のチャンク分割](../../../documents/adr/0105-chunk-find-query-ids.md)
- [ADR-0107 ReKyou ターゲット解決のメモ化](../../../documents/adr/0107-memoize-rekyou-target-resolution.md)
- [ADR-0801 性能判定は allocs で（ns/op ではなく）](../../../documents/adr/0801-perf-judge-by-allocs-not-ns-op.md)
- [ADR-1001 ログレベルは事象の重さで決める](../../../documents/adr/1001-log-level-by-severity.md)
- [ADR-0201 追記専用 DAO](../../../documents/adr/0201-append-only-dao.md)
- [ADR-0202 キャッシュ再構築は実DBが変わったときだけ](../../../documents/adr/0202-rebuild-cache-only-on-db-change.md)
- [ADR-0203 ライトスルーは reps 数判定ではなく](../../../documents/adr/0203-write-through-cache-not-reps-count.md)
- [ADR-0204 journal_mode は DELETE のまま](../../../documents/adr/0204-keep-journal-mode-delete.md)
- [ADR-0205 unixepoch 式インデックス](../../../documents/adr/0205-unixepoch-expression-index.md)
- [ADR-0206 threads.Go を入れ子にしない](../../../documents/adr/0206-no-nested-threads-go.md)
- [ADR-0207 URLog サムネイルをキャッシュから除外](../../../documents/adr/0207-exclude-urlog-thumbnail-from-cache.md)
- [ADR-0208 Git リポジトリ判定は os.Stat で](../../../documents/adr/0208-git-repo-detect-by-os-stat.md)
- [ADR-0210 最新版アドレス表の rep 名は行ごとの実rep名](../../../documents/adr/0210-latest-data-address-uses-row-rep-name.md)
- [ADR-0218 `/api/get_kyou` の版履歴はキャッシュ rep を回す](../../../documents/adr/0218-get-kyou-histories-via-cached-reps.md)
- [ADR-0219 commit_tx は1つの SQLite トランザクション](../../../documents/adr/0219-commit-tx-is-one-sqlite-transaction.md)
- [ADR-0220 SQLite の 'localtime' は libc のゾーンで決まる](../../../documents/adr/0220-sqlite-localtime-follows-libc-zone.md)
- [ADR-0221 Git コミットログのキャッシュ包装は構築済みなら外れた ID で生リポジトリへ落ちない](../../../documents/adr/0221-git-cache-miss-does-not-fall-back-to-raw-walk.md)
- [ADR-0701 パスワードは Argon2id](../../../documents/adr/0701-argon2id-password-storage.md)
- [ADR-0702 共有の所有者はセッションから](../../../documents/adr/0702-share-owner-from-session.md)
- [ADR-0703 共有ファイル認可はクエリ再評価で](../../../documents/adr/0703-shared-file-authz-by-query.md)
- [ADR-0704 利用者URLは safefetch 経由](../../../documents/adr/0704-safefetch-for-user-urls.md)
- [ADR-0705 派生キャッシュはユーザー別ディレクトリ](../../../documents/adr/0705-per-user-derived-cache-dir.md)
- [ADR-0707 端末固有の文字列は出口で伏せる](../../../documents/adr/0707-redact-environment-specific-strings.md)
- [ADR-0708 待受の既定はループバック限定](../../../documents/adr/0708-local-only-listen-by-default.md)
- [ADR-0710 `errors` / `messages` は成功時も `[]`、`error_kind` と `reason` を載せる](../../../documents/adr/0710-error-kind-and-reason-on-the-wire.md)
