# ADR-0308: プラグイン1本が複数の rep 名を名乗れる（get_rep_name の rep_names）

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-14 |
| Sources | `.claude/skills/gkill-plugin/SKILL.md`「1本のプラグインが複数の rep 名を名乗れる」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/plugin_repository_impl.go` / `src/server/gkill/dao/reps/repository.go` |

## Context

zip に固めて保管した Git リポジトリのコミットログを読むプラグイン（`gkill_plugin_archived_git_commit_log`）を作るにあたり、
**rep 名は native の git rep と同じくリポジトリごと**（ディレクトリ名）にしたかった。実データでは 54 個の zip に 63 個の `.git`（コミットのあるもので 48 個の名前）が入っている。

ところが gkill の rep 名は「プラグイン1本 = manifest の `rep_name` 1つ」を前提に列挙されていた。
`convertPluginKyouToKyou` は Kyou ごとの別の `RepName` を残す（警告は組み合わせごとに1回）ものの、**名前を列挙する側**が manifest しか見ない:

- `GkillRepositories.GetAllRepNames` は各 rep の `GetRepName` → サイドバーの rep 一覧に載らず、利用者が選べない
- `find_filter.go` の Step4（rep 名指定のとき検索対象の rep を選ぶ）は `UnWrap()` した leaf の `GetRepName` を `query.reps` と突き合わせる → GUI は常に `reps` を送るので、**プラグインが検索対象から外れて静かに0件になる**
- `PluginManager.GetPluginByRepName` は manifest の `rep_name` だけ → クライアントが `Kyou.rep_name` で本文（`get_plugin_content_html`）を取りに来ると「プラグインが見つかりません」

## Decision

- プロトコルの `get_rep_name` 応答に **`rep_names []string`** を足す。**null（欄なし）と `[]` は別**: null は「この応答に対応していない」（古いプラグイン・SDK の `Handler.RepNames` 未設定）で manifest の `rep_name` 1つ、`[]` は「いまは名乗る名前が無い」で 0 個。FindQuery の null / 空配列と同じ規約
- SDK は `Handler.RepNames func(ctx, cfg) ([]string, error)` を任意で受け、実装済みなら nil を返されても `[]` を出す（欄を落とすと未対応と読まれる）。`pluginResponse.RepNames` は `*[]string` で、nil ポインタなら欄ごと省く
- 本体は任意インターフェース **`reps.RepNamesProvider { GetRepNames(ctx) }`** と **`reps.RepNamesOf(ctx, rep)`**（未実装なら `[GetRepName()]`）を置き、名前を**列挙する側**はこれを通す: `GetAllRepNames`・Step4 の照合・`get_rep_infos.plugins[]`（申告名1つにつき1行）・`get_kyous_mcp` の rep 名→manifest 対応表・`get_plugin_list.rep_names`。`GetRepName` は引き続き代表名（`MatchReps` のキー・ログ用）
- `pluginRepositoryImpl.GetRepNames` は `get_rep_name` を送って **TTL 60 秒**でキャッシュし、失敗しても**エラーにせず**前回値か manifest 名にフォールバックする（失敗もキャッシュする）。`UpdateCache` で古くする（捨てない）
- `PluginManager.GetPluginByRepName(ctx, name)` は manifest 名で引けなければ各プラグインの申告名でも引く（2段にして既存プラグインの引き当てで stdio へ行かない）
- 申告済みの名前は `warnPluginRepNameMismatchOnce` の対象外
- 型別アダプタ（`pluginAdapterBase`）もプラグイン本体へ委譲して `RepNamesProvider` を満たす（`rep_types` 指定の検索でも Kyou ごとの rep 名で選ばれる）

## Rejected alternatives

- **型別索引（PluginTypedIndex）のスナップショットから rep 名を DISTINCT する** — `provides` を宣言しないプラグイン（chatgpt / claudeai / claudecode / codex）には索引が無く、仕組みが型別プラグイン限定になる。索引の構築は `find_kyous` 1回ぶん（fitbit なら2万件）を運ぶので、名前を知るためだけに走らせるのも重い。プロトコルに欄を足すほうが小さく、全プラグインで同じ経路になる。

- **manifest.json に rep 名を列挙する** — 名前は取り込んだ zip で決まり、config.json を変えれば増減する。manifest はバイナリと一体（`--gkill-print-manifest` で置き直す）なので、動的な値を持たせられない。

- **プラグイン1本 = リポジトリ1つにする** — 54 個の zip に 48 個のリポジトリ名。プラグインディレクトリを 48 個作り、配布スクリプトの明示列挙（`Git/scripts` の3ファイル5箇所）に 48 行足すことになる。zip を足すたびに配布が要る。

- **`GetRepNames` の失敗を検索のエラーにする** — Step4 は fan-out（`goForRep`）の外で逐次に走る。1本のプラグインが応答しないだけで全検索が落ち、毎回取りに行くと全検索が期限（`pluginCallTimeout` 30 秒）ぶん止まる。`FindKyous` の失敗が警告止まりなのと揃え、失敗もキャッシュして1分に1回に抑える。

- **`GetRepName` の戻り値を `[]string` に変える** — `Repository` の全実装（4層 × 13 データ型 + 集約 + キャッシュ）に波及する。列挙する側は3箇所しか無いので、任意インターフェース + ヘルパで足りる。

## Consequences

- `get_rep_name` は今まで本体から一度も送られていなかったが、**rep 名の列挙のたび（TTL 60 秒）に送られる**。既存プラグインは `rep_name` だけ返せばよく、挙動は変わらない（`gkill_plugin_uguisu` のように SDK を使わずプロトコルを写している実装も含む）
- 取り込みが進んで名前が増えても、一覧に現れるまで最大 1 分（「追加した記録が最大1分見えない」と同じ約束）
- 申告した名前が native の rep 名と同じ（`ocha` など）だと、サイドバーでは1つのチェックボックスになり、両方が検索される。同じコミットが両方にあるときの畳み込みは [ADR-0309](0309-plugin-provides-git-commit-log.md)

## Evidence

- 実データ: 54 zip → 48 rep 名 / 721 コミット。初回構築は 10 秒弱（`get_rep_name` の応答は取り込みが進むにつれ `[]` → 46 個 → 48 個と増える）
- 稼働中の `$HOME/Git` と同名の rep: `ocha` / `urlog` / `Scripts` / `rykvh`（244 コミットが両方にある）

## Related tests

- `src/server/gkill/plugin/sdk/sdk_test.go`（`TestRunLoop_GetRepName_*`）
- `src/server/gkill/dao/reps/plugin_repository_impl_test.go`（`TestPluginRepository_GetRepNames_*` / `TestPluginRepository_FindKyousKeepsDeclaredRepNames`）
- `src/server/gkill/dao/reps/plugin_kyou_rep_name_test.go`（`TestConvertPluginKyouToKyouWith_DeclaredRepNameDoesNotWarn`）
- `src/server/gkill/dao/reps/gkill_repositories_test.go`（`TestGkillRepositoriesGetAllRepNames_IncludesDeclaredRepNames`）
- `src/server/gkill/api/select_match_reps_cache_test.go`（`TestSelectMatchRepsFromQuery_RepsMatchDeclaredRepNames`）
- `src/server/gkill/dao/plugin_manager_rep_names_test.go`
