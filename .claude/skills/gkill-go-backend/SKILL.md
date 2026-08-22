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
- `gkill/api/gkill_server_api/` — HTTP API handlers (101 files incl. tests, 1 handler per file). `GkillServerAPI` struct with `serve.go`, `close.go`, route definitions in `gkill_server_api_address.go`. Auth middleware (`auth.go`, `auth_context.go`, `auth_middleware.go`) extracts session→account→device→repositories via `AuthContext`, `authMiddleware`, `authWithReposMiddleware`. Handler registration uses wrapper functions: `wrapNoAuth` (no session), `wrapAuth` (session + account), `wrapAuthRepos` (session + account + device + repositories). Utility files: `filter_local_only.go`, `utils.go`, `web_push.go`. ZIP browsing: `handle_browse_zip_contents.go` (path traversal prevention, Shift_JIS→UTF-8, singleflight dedup).
- `gkill/api/req_res/` — Request/response structs for every endpoint (186 files)
- `gkill/usecase/` — HTTP-independent business logic (17 files). Extracted from handlers to enable reuse without HTTP context. Functions operate on DAO/repository types directly.
- `gkill/dao/` — Data access layer with `GkillDAOManager` managing SQLite3 databases
- `gkill/dao/reps/` — Repository interfaces and implementations for each data type.

**Key dependencies:** gorilla/mux, modernc.org/sqlite (pure Go), spf13/cobra+viper, asticode/go-astilectron, SherClockHolmes/webpush-go, go-git/go-git, twpayne/go-gpx, nicksnyder/go-i18n/v2, google/uuid

**Doc comments:** `dao/reps/*_repository.go` のインターフェースメソッドは **457/457 documented**、`gkill_server_api` の HandleXxx は **92/92 handlers documented**。どちらも verify_docs が網羅率を機械検査するので、メソッドやハンドラを追加したら doc コメントも書くこと。共通契約の完全文は基底 `dao/reps/repository.go` に1回だけ書き、型別ファイルは「契約は Repository.Xxx を参照。」の1行参照＋差分のみにする（457箇所へ複製するとドリフト源になるため）。ハンドラは「1行説明 / 空行 / パス・HTTPメソッド・認証区分 / req_res 型」の形式。

**Repository pattern:** SQLite-backed data types have 4 implementation layers: `*_repository.go` (interface) → `*_repository_sqlite3_impl.go` → `*_repository_cached_sqlite3_impl.go` → `*_repository_temp_sqlite3_impl.go`. Exceptions: `git_commit_log` (`_local_dir_impl` + `_cached_sqlite3_impl`), `gps_log` (`_gpx_dir_impl`), and `plugin` (`plugin_repository_impl`) read external sources directly and do not have all 4 layers.

**`FindQuery.IDs` は分割して渡す（SQLのバインド変数には上限がある）:** **新しくIDリストを渡す検索を書くときは `findChunkedByIDs`（`repositories.go`）を通すこと**。Mi / MiReKyou は5射影のUNIONでバインド変数が `5N+5` になり、SQLite の上限を **N=6553 で超える**。今かかっているのは `Repositories.findKyous` / `MiRepositories.FindMi` / `MiReKyouRepositories.FindMiReKyou` と、`GkillRepositories.FindTags` / `FindTexts` の中で最新版アドレスからIDリストを作っている箇所。上限は `maxIDsPerFindQuery`（4000）。**この壊れ方は静かなので気付きにくい** ―― 失敗しても `GkillError` が立たず HTTP 200 + `errors: null` + 0件で返る。受け皿として `message.EnsureNotEmpty` で `ERR000410` を必ず立てるようにしてある。**「失敗したのに GkillError が空」で return する分岐を新しく作らないこと。** 回帰は `repositories_id_chunk_test.go` / `handle_get_kyous_mcp_test.go` の `TestHandleGetKyousMCP_ManyIDs` / `message/gkill_error_test.go` が守る。経緯と却下案は [ADR-0005](../../../documents/adr/0005-chunk-find-query-ids.md)。

**rep名の絞り込みは「検索するrep」ではなく「検索結果」でやる:** `find_filter.go` の `selectMatchRepsFromQuery` で **`UnWrap()` の戻り値を `MatchReps` に入れてはいけない**。`UnWrap()` はインメモリキャッシュrep（と `--cache_reps_local` のローカルコピー層）を剥がして生のディスクrepに戻すので、**キャッシュを丸ごとバイパス**して端末別重複repぶんディスクを舐めることになる。`UnWrap()` を使ってよいのは「そのラッパに選ばれた実repが1つでもあるか」の**枝刈り判定だけ**（これは省けない）。実際の絞り込みは `findKyous` の `filterKyousByRepName` が `Kyou.RepName` で行う。落とし穴が5つある: **(1)** 本文ヒット由来の2本目の検索（`matchTextFindByIDQuery`）にも同じ絞り込みが要る、**(2)** 全部落ちたIDは**キーごと消す**（空スライスを残すと `kyous[0]` を見る `filterLocationKyous` / `filterMiForMi` / `overrideKyous` が panic する）、**(3)** `Reps == nil` は「未指定」で `len()` で判定すると全件消える、**(4)** **`RepName` が空の行は残す** ―― 追加直後の行は `REP_NAME` が空なので、落とすと**いま追加した記録が最大1分間一覧から消える**。**(5)** 書き込み側に**実在しないrep名を入れさせない** ―― 非空のrep名は「実在するが選ばれていないrep」として落とされるので、合成した名前を渡すと記録が黙って消える。フィルタ側に `*Temp` の例外を足してはいけない（直すのは常に書き込み側）。`dao/reps` 側にも置いてはいけない（ReKyou/MiReKyou のワード委譲が利用者のクエリをそのまま `FindKyousSequential` へ渡すので、チェックしていないrepに参照先があるリポストが黙って語句検索に当たらなくなる）。**SQL の WHERE へ降ろすのも不可**（相関サブクエリに足すと最新版判定が壊れる → [ADR-0002](../../../documents/adr/0002-no-rep-name-in-sql.md)）。守るテストは `select_match_reps_cache_test.go` / `find_kyou_rep_name_filter_test.go` / `get_kyous_rep_filter_test.go` / `get_kyous_tx_rep_filter_test.go`（**すべてキャッシュON/OFFの両方**）。実測（11rep→約940rep・20.7秒）と事件の経緯は [ADR-0001](../../../documents/adr/0001-filter-rep-after-cache.md)。

**タグ絞り込みの取得は2経路。切り替えは「タグ名の個数」で決める:** `find_filter.go` の `collectTagsForFilter` が `MatchTags`（クエリのタグ名に一致するタグ）・`RelatedTagIDs`（タグが1つでも付いているIDの集合）・`AllHideTagsWhenUnchecked`（強制非表示タグ）の3つを作る。切り替えの閾値は `maxTagNamesForSQLFilter`（32。**クエリのタグ名と非表示タグ名の合計**で見る）。**「タグ無し」仮想タグ（`NoTags`）を使う検索では、名前の個数によらずGo側で照合する**（`RelatedTagIDs` のために結局は全タグを取るので、そこから名前を拾うぶんはタダになる）。照合は**完全一致・大小無視**で、`filterTagsKyous` のAND分岐と同じ意味論（SQLは TAG 列だけでなく ID 列とも突き合わせていたので、そこも写してある）。**2経路が同じ結果を出すことをテストで固定してある**（`find_filter_test.go`）—— ずれるとタグの個数によって検索結果が変わるという静かな壊れ方になる。`RelatedTagIDs` の読み手は `NoTags` 分岐しか無いので、`NoTags` を使わない検索では作らない（走らせ忘れると**全件が「タグなし」扱い**になりタグの付いた記録まで返る。回帰は `get_kyous_tag_filter_test.go` と `get_kyous_regressions_test.go` の `TestHandleGetKyous_TimeIsTagsFilterWorksWithoutKyouTagFilter`）。強制非表示タグも2経路の一致を `TestHandleGetKyous_HideTagsBothPaths` が固定する。実測表（2万タグでの交差点）と却下案は [ADR-0003](../../../documents/adr/0003-tag-filter-threshold-32.md)、実測表そのものは `dao/reps/tag_find_bench_test.go`。

**キャッシュのフルリビルドは「実DBが変わったときだけ」:** cached rep の `UpdateCache` は `DELETE FROM` + 全行再INSERT なので、変わっていない rep まで作り直すと共有の書き込みロックを握ったまま全種類の検索が止まる。抑止は `dbFileChangeDetector`（mtime + サイズ）で、基準を進めるのは**再構築成功後の `CommitCacheRebuild` だけ**（失敗した回を取りこぼさないため）。`--cache_reps_local=true` のときに挟まる `*_local_cached.go` も同じ約束を守ること ―― **「コピーが要るか」の判定は必ず `os.Remove` より前に置く**。消してから `os.Stat` すると常に「要コピー」になり、`LastUpdateCacheChanged()` が常に true を返して抑止が丸ごと無効化される。共通ヘルパは `local_rep_cache_copy.go`、回帰検出は `local_rep_cache_granular_test.go`。**ReKyou / MiReKyou は変更検知に載せてはいけない**（コピー省略のみ可）: この2つはターゲット解決のためアドレス確定後にもう一度更新される仕様で、mtime判定を入れると2回目が飛んでターゲット未解決の中身が残る。実測（phase1 が 0.2秒→1〜2分）と却下案は [ADR-0011](../../../documents/adr/0011-rebuild-cache-only-on-db-change.md)。

**`len(XxxReps) == 1` でキャッシュrepを判定してはいけない。** アダプタの append は「キャッシュrepで `XxxReps` を1個に差し替える」処理より後なので、`provides` を持つプラグインが1つ入るだけで長さが2になる。書き込み後のキャッシュ反映は構築時に控えた `GkillRepositories.CachedReps` を見る `repositories.WriteThroughXxxCache(ctx, ...)` を使うこと（54箇所）。読み取りはキャッシュrepしか見ず下層repへフォールバックしないので、反映を飛ばすと追加したタグが最大1分見えず、その間にPWAが古い応答をキャッシュし直すと**恒久的に古いまま焼き付く**。再発は `usecase/write_through_cache_test.go` の `TestNoRepsCountCacheGuard` がソース走査で落とす。経緯と却下案は [ADR-0012](../../../documents/adr/0012-write-through-cache-not-reps-count.md)。

### HTTP セキュリティ（2026-08 外部監査由来）

**共有ページのファイル配信は共有クエリの結果に含まれるファイルだけ**（2026-08-21、監査 C-03）。`handle_file_serve.go` の `sharedID != ""` 経路は、rep名一致だけで IDF rep へ委譲せず、`shared_file_authz.go` の `collectSharedIDFFilePaths` で共有 `FindQuery` を再評価して許可パス集合を作り、要求パスが集合に無ければ403にする（`handle_get_shared_kyous.go` と同一手順を共有＝一覧を2箇所で維持しない）。パス正規化は `idf_file_url.go` の `cleanRelativeURLPath` と同一。セッション経路はフルアクセスのまま（`sharedKyouInfo != nil` のときだけゲート）。守るテストは `shared_file_authz_test.go`。 却下案（許可パスを共有作成時に固定する等）は [ADR-0042](../../../documents/adr/0042-shared-file-authz-by-query.md)。

**利用者入力URL・そのページが指す og:image / #landingImage の取得は必ず `api/safefetch` を通す**（2026-08-21、監査 H-04）。`safefetch.GetCapped`（scheme検査・`Dialer.Control` での接続先IP検証・サイズ上限）と `safefetch.CheckImageDimensions`（`image.DecodeConfig` で復号前に総ピクセル検査）で SSRF・無制限read・画像爆弾を防ぐ。`dao/reps/ur_log.go` の getBody/getFavicon/getImageOG/getAmazonImage と `gkill_server_api` の `httpGetBase64Data`（ブックマークレット）が利用。既定は private 拒否（loopback/RFC1918/link-local(メタデータ)/multicast/unspecified）。`http.Get` を新しく直に書かないこと。守るテストは `api/safefetch/safefetch_test.go`。 文字列検査では守れない理由（DNSリバインディング）は [ADR-0043](../../../documents/adr/0043-safefetch-for-user-urls.md)。

**型別 `GetXxx(id, nil)` は最新版を返す**（2026-08-21、監査 H-07）。`dao/reps` の各 `*_repository_sqlite3_impl.go` / `*_repository_cached_sqlite3_impl.go` の単体取得は `onlyLatestData := query.OnlyLatestData`（`false` 固定にしない。`GenerateFindSQLCommon` はこの引数しか見ない）と `slices.MaxFunc(UpdateTime)`（`&xxx[0]` を返さない）で最新版を選ぶ。`GkillRepositories.GetKyou` はアドレス表 nil でも panic しない（プラグインKyou・追加直後の記録）。守るテストは `get_typed_latest_version_test.go` / `gkill_repositories_get_kyou_test.go` / `rows_err_check_test.go`。

**外部URLを取る新規コードや子goroutine内の index 参照に注意**: `strings.SplitN(x, ",", 2)` は末尾要素 `parts[len(parts)-1]` を取る（`[1]` を長さ未確認で参照しない。子goroutine内 panic は `recoverMiddleware` で回収できずプロセスが落ちる＝監査 H-01）。ファイル書き込みは tmp+rename で原子的に（Override で原本を壊さない＝M-03）。ZIP展開は件数・総展開量・圧縮比の上限つき（`handle_browse_zip_contents.go` の `maxZip*`＝M-04）。`http.Server` は `ReadHeaderTimeout`/`IdleTimeout`/`MaxHeaderBytes` を張り、認証前ボディは `maxAuthBodyBytes` でキャップ（H-03）。全レスポンスに `securityHeadersMiddleware`（nosniff / X-Frame-Options / Referrer-Policy）。CLIサブコマンドは `InitGkillServerAPI` 失敗で `return`（nil panic 回避＝M-8）、CLIの自己発行セッションは `IsLocalAppUser=false`（最小権限）。ログインは非存在ユーザとパスワード誤りを同じ error_code + 文言に統一し、非存在時もダミー Argon2id を実行（ユーザ列挙対策＝S3-login）。

**IDF走査の一時停止は参照カウント**（2026-08-21、監査 M-02）。`SetSkipIDF(true/false)` は共有 `*bool` ではなく `*atomic.Int64` を増減し「カウント>0 で skip」。重なるアップロード（と UpdateCache）が互いのフラグを倒し合って watcher が走る/変更を取りこぼすのを防ぐ。カウントが0へ戻ったときだけ catch-up 走査を1回キックする（`UpdateCache` の Add(-1) では catch-up しない＝1分周期のリビルド無限ループになるため。`SetSkipIDF` はアップロードハンドラからしか呼ばれず watcher から再入しない）。

## 関連スキル

- [gkill-find-query](../gkill-find-query/SKILL.md) — `FindQuery` の null 意味論（Go/TS/MCP の3実装共通）
- [gkill-plugin](../gkill-plugin/SKILL.md) — プラグイン rep のアダプタ・プロセス寿命（`dao/reps/plugin_*.go`）
- [gkill-cli-ops](../gkill-cli-ops/SKILL.md) — 派生キャッシュのユーザー別ディレクトリと CLI の自己発行セッション
- [gkill-mobile](../gkill-mobile/SKILL.md) — KFTL 送信の冪等キー（`handle_submit_kftl_text.go`）

## 詳しい設計と却下案（ADR）

- [ADR-0001 rep名の絞り込みはキャッシュの後で](../../../documents/adr/0001-filter-rep-after-cache.md)
- [ADR-0002 rep名を SQL の WHERE へ降ろさない](../../../documents/adr/0002-no-rep-name-in-sql.md)
- [ADR-0003 タグ絞り込みの閾値32](../../../documents/adr/0003-tag-filter-threshold-32.md)
- [ADR-0004 RelatedTagIDs は NoTags のときだけ](../../../documents/adr/0004-related-tag-ids-only-for-no-tags.md)
- [ADR-0005 FindQuery.IDs のチャンク分割](../../../documents/adr/0005-chunk-find-query-ids.md)
- [ADR-0007 ReKyou ターゲット解決のメモ化](../../../documents/adr/0007-memoize-rekyou-target-resolution.md)
- [ADR-0008 性能判定は allocs で（ns/op ではなく）](../../../documents/adr/0008-perf-judge-by-allocs-not-ns-op.md)
- [ADR-0010 追記専用 DAO](../../../documents/adr/0010-append-only-dao.md)
- [ADR-0011 キャッシュ再構築は実DBが変わったときだけ](../../../documents/adr/0011-rebuild-cache-only-on-db-change.md)
- [ADR-0012 ライトスルーは reps 数判定ではなく](../../../documents/adr/0012-write-through-cache-not-reps-count.md)
- [ADR-0013 journal_mode は DELETE のまま](../../../documents/adr/0013-keep-journal-mode-delete.md)
- [ADR-0014 unixepoch 式インデックス](../../../documents/adr/0014-unixepoch-expression-index.md)
- [ADR-0015 threads.Go を入れ子にしない](../../../documents/adr/0015-no-nested-threads-go.md)
- [ADR-0016 URLog サムネイルをキャッシュから除外](../../../documents/adr/0016-exclude-urlog-thumbnail-from-cache.md)
- [ADR-0017 Git リポジトリ判定は os.Stat で](../../../documents/adr/0017-git-repo-detect-by-os-stat.md)
- [ADR-0040 パスワードは Argon2id](../../../documents/adr/0040-argon2id-password-storage.md)
- [ADR-0041 共有の所有者はセッションから](../../../documents/adr/0041-share-owner-from-session.md)
- [ADR-0042 共有ファイル認可はクエリ再評価で](../../../documents/adr/0042-shared-file-authz-by-query.md)
- [ADR-0043 利用者URLは safefetch 経由](../../../documents/adr/0043-safefetch-for-user-urls.md)
- [ADR-0044 派生キャッシュはユーザー別ディレクトリ](../../../documents/adr/0044-per-user-derived-cache-dir.md)
