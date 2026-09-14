# ADR-0309: zip の Git リポジトリは provides に git_commit_log を書いて native と同じ経路に載せる

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-14 |
| Sources | `.claude/skills/gkill-plugin/SKILL.md`「プラグインの型別/付随データ」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/plugin_typed_adapters.go` / `src/plugins/gkill_plugin_archived_git_commit_log/gitread.go` |

## Context

過去のリポジトリは作業を終えるたびに zip に固めて IDF rep（`Box_*`）へ入れ、タグ `rep` を付けて保管してきた（114 件。`.git` 入りは 88 件）。
稼働中の Git rep（`$HOME/Git/*`）は native の `git_commit_log` rep が読むが、zip の中は誰も読まないので、そのコミット履歴はタイムラインに出なかった。

プラグインとして読む設計で、data_type をどうするかが分かれ目だった。zip の中身を調べると:

- **3,691 コミット、ハッシュの重複除去で 3,447 件**。同じリポジトリを別の時期に固めた zip が複数組あり（`ocha` は 2018 / 2020 / 2026 の3つ）、改名前後（`weplog` → `urlog`、`ytf` → `ytflmf`）で履歴を共有する組もある
- **244 コミットは稼働中の `$HOME/Git`（`ocha` / `urlog` / `Scripts` / `rykvh`）にも同じハッシュがある**
- 検索結果の重複除去は `find_filter.go` の `dedupAdjacentResultKyous` で、キーは **`(ID, data_type, related_time)`**（TimeIs の start / end を残すため素の ID 重複除去はできない）

つまり ID をコミットハッシュにしても、data_type が native と違えば **244 件が2件ずつ並ぶ**。

## Decision

- `provides` に **`git_commit_log`** を足す（`PluginProvidesGitCommitLog`）。`PluginTypedData.GitCommitLog{commit_message, addition, deletion}`・SDK の `TypedData.GitCommitLog`・型別索引の `pluginTypedRecord.GitCommitLog`・アダプタ `pluginGitCommitLogRepositoryImpl`（`GitCommitLogRepository` を満たす）・`gkill_dao_manager.go` での `GitCommitLogReps` への登録、の一式
- プラグインは **data_type を native と同じ `git_commit_log`** にし、ID = コミットハッシュ・rep 名 = リポジトリのディレクトリ名・RelatedTime / CreateTime / UpdateTime = コミッタ日時（コミット固有のゾーン）・CreateUser / UpdateUser = author 名・CreateApp / UpdateApp = `git`、と **native の `git_commit_log_repository_local_dir_impl.go` と同じ列**で返す
- アダプタのワード検索の対象列は **コミットメッセージだけ**（native の `findWordTextOfGitCommit` と同じ）。プラグイン側の `find_kyous` はメッセージ + リポジトリ名 + author 名で照合する（rep 名で引けるのはプラグイン経路だけ、という差は KC の fitbit と同じ扱い）
- zip は展開しない。`sdk.OpenSources` で中央ディレクトリを列挙し、`.git/**` のうち go-git が読むエントリ（HEAD / config / packed-refs / shallow / refs / objects）だけを **go-billy の memfs** へ流し込み、`filesystem.NewStorage` + `git.Open(storer, nil)` で開く。行数は `commit.StatsContext` の合計

## Rejected alternatives

- **別 data_type（`archived_git_commit_log`）のまま iframe HTML で native 風に描く** — 本体の変更は「複数 rep 名」だけで済むが、稼働中 rep と重なる 244 件が2件ずつ並ぶ。Dnote のコミット集計・`rep_types=git_commit_log` の絞り込み・MCP の `git_commit_log` payload にも載らない。「同じコミットは1件」を満たせないので不採用。

- **Kyou ID を UUIDv5(ハッシュ) にして native と衝突させない** — 衝突しない代わりに、重なる 244 件が確実に2件並ぶ。ハッシュをそのまま ID にすれば、同じ data_type との組で畳まれる。

- **`data_type` を `git` で始まる別名にする** — クライアントは data_type の接頭辞 `git` で `/api/get_git_commit_log` を叩く（`kyou.ts` の `load_typed_git_commit_log`）ので、型別アダプタが無ければ `not_found_git_commit_log` になって何も描かれない。接頭辞だけ合わせても解決しない。

- **zip を一時ディレクトリへ展開して `git.PlainOpen` する** — 本体の `handle_browse_zip_contents.go` と同じ問題（派生キャッシュが zip と二重にディスクを食い、キーがパスなので中身を差し替えても古い展開が残る）。実データの `.git` は大きくても数 MB なので memfs で足りる。`.git` の合計に上限（`max_git_dir_mb`、既定 256）を置く。

- **`git` コマンドで読む** — 本番の gkill は LocalSystem のサービスで動き、PATH 上の `ffmpeg` が見えない前例がある。go-git は純 Go なので Android/arm64 のクロスビルドもそのまま通る。

- **共通のコミットの rep 名を「最初に見た zip」で決める** — zip のパス順（=設定の並びと Box rep 名）で決まり、改名前後の組で古い名前が勝ったり（`ytf` / `ytflmf`）新しい名前が勝ったり（`urlog` / `weplog`）とばらつく。アーカイブの mtime は同期で動くので使えず、リポジトリの最新コミット日時なら中身だけで決まり、後から固めた zip に古いコミットしか無いことは無いので「新しい方」と一致する。

- **行数集計の失敗で構築全体を止める（native と同じ）** — native は検索のたびに読むので失敗を返すしかないが、こちらは常駐ビルダ。1コミットの失敗で他の数千件を出さないのは損なので、そのコミットだけ行数 0・理由を `stats_error` に残して続ける。

## Consequences

- 稼働中 rep と重なるコミットは検索結果で1件になるが、**`get_kyou_histories` は native とアダプタの両方から1件ずつ返す**（同じ ID・同じ data_type・同じ時刻の2版）。履歴の表示が2行になるが、内容は同じ
- 同じハッシュが複数の zip にあれば1行で、rep 名は**そのコミットを含むリポジトリのうち最新のコミットを持つもの**（同着なら zip のパス順。構築のたびに `assignRepNames` が決め直す）。改名前後の組（`weplog` → `urlog`）の共通の履歴は新しい名前に付く。zip を config から外すと、他の zip にも無いコミットだけが消え、名前も残った zip から決め直される
- native の rep 名と同じ名前を名乗るので、サイドバーでは1つのチェックボックスに両方がぶら下がる（[ADR-0308](0308-plugin-multiple-rep-names.md)）
- 型別アダプタは読み取り専用（`AddXxxInfo` はエラー、`WriteXxxRep` に入らない）。GitCommitLog は native にも編集経路が無いので、画面の変更は要らない

## Evidence

- `rep` タグ 114 件のうち `.git/HEAD` 入り 88 件（`.git` は 97 個）。`.git` の位置は `name/.git/`（大半）・ルート直下 `.git/`・1 zip に複数（最大 5）・入れ子（`kokko/wiki/.git`）
- loose のみ / pack のみ / 両方が混在。`git init` 直後（コミット 0 件）が 11 個
- 3,691 コミット → 3,447 件（同じリポジトリの zip と改名前後の組で重複）。244 件は 2026-09 の zip を足す前の時点で稼働中 rep にもあった
- 初回構築 約 60 秒（88 zip・3,447 コミット・go-git の行数集計込み。最大の zip は 88MB）。2回目以降は指紋（sha256 of (エントリ名, CRC32, サイズ)）の比較だけ

## Related tests

- `src/server/gkill/dao/reps/plugin_git_commit_log_adapter_test.go`
- `src/plugins/gkill_plugin_archived_git_commit_log/cache_test.go`（`TestBuildFromScratch` / `TestSameRepositoryInTwoArchivesIsDeduplicated` / `TestPackedRepositoryIsReadable`）
- `src/plugins/gkill_plugin_archived_git_commit_log/config_test.go`（`TestManifestMatchesConstants`）
