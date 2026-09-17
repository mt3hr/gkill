# テスト仕様 — src/plugins/

## 概要

`src/plugins/` 配下のプラグインのうち、examples/gkill_example には独自のテストファイルは存在しない。

プラグイン機能のテストは以下でカバーされている：

| テスト場所 | テスト内容 |
|---|---|
| `src/server/gkill/api/gkill_server_api/` | プラグインのロード・subprocess起動・stdio通信・クラッシュ後自動再起動の統合テスト |
| `src/server/gkill/dao/reps/` | `plugin_repository_impl.go` のリポジトリ層テスト |
| `src/plugins/gkill_plugin_claudecode/loader_test.go` | ターン分割・ファイル種別判定・サブエージェント紐付け・ツール要約・HTML生成のユニットテスト（`testdata/` の合成トランスクリプトを使用） |
| `src/plugins/gkill_plugin_codex/{reader,loader,fold,cache,render,config}_test.go` | 巨大行リーダ・`session_meta` の identity/environment 分離・IDE前置きの剥がし・Kyou ID の安定性・差分再構築・構築中の並行読み取りのユニットテスト（`testdata/` の合成ロールアウトを使用） |
| `src/plugins/gkill_plugin_archived_git_commit_log/{cache,scan,find_kyous,config,render}_test.go` | zip の中の Git リポジトリの取り込み。testdata に `.git` は置けないので、テストのたびに go-git でリポジトリを作って `archive/zip` で固める（`testutil_test.go`）。native の git rep と同じ列（ID=ハッシュ・rep 名=ディレクトリ名・コミッタ日時とゾーン・author・行数）で入ること、同じリポジトリを2つの zip に入れても1件で zip を外すと他に無いコミットだけ消えること、コミット0件の `git init` 直後は0件で正常、`.git` の無い zip は素通り、1 zip に複数（入れ子・ルート直下）、packfile、指紋による増分、`.git` の上限超過は理由を残して飛ばす、構築中の並行読み取り |
| `src/plugins/gkill_plugin_{chatgpt,claudeai}/cache_test.go` | エクスポート ZIP の直読み。テストのたびに `archive/zip` で ZIP を組む（バイナリは commit しない）。ZIP を作り直しても中身が同じなら作り直さない（署名は Path:CRC32:Size で更新時刻を見ない。Claude.ai の実物は 1980 年固定）、展開済みの `conversations.json` を直置き・直接指定しても読まず既存キャッシュを残して `source_problems` に出る、会話ファイルを含まない ZIP（`projects-000.zip` 等）は素通り、同じ ZIP に分割形式と旧形式があれば分割形式だけ（優先はアーカイブ単位）、同じ会話 ID が2つの ZIP にあれば update_time が新しい版だけ、プラグインフォルダ自身の `manifest.json` / `config.json` は「展開済みフォルダ」と警告しない、バッチ commit の永続性、読み取りが `buildMu` を取らないこと、gen 掃除 |
| `src/plugins/gkill_plugin_{chatgpt,claudeai,claudecode,codex,fitbit,archived_git_commit_log}/find_kyous_test.go` | FindKyous のワード判定。SDK の `Query.MatchText`（gkill 本体と同じ規則）で肯定語・除外語・AND/OR・ID 前方一致が効くこと、chatgpt / claudeai は会話タイトル、codex はスレッド名、archived_git_commit_log はリポジトリ名と author 名にも当たること、fitbit は数値でも当たり空文字の語で全件が消えないこと、LIMIT が絞った後に掛かること。gkill 本体はプラグインが返した Kyou のワードを再判定しないので、ここが唯一の判定 |
| `src/server/gkill/plugin/sdk/match_words_test.go` | `sdk.Query.MatchText` / `Matcher` の判定規則と、元の Query を書き換えないこと |
| `src/server/gkill/plugin/sdk/cache_path_test.go` | キャッシュDBの置き場所の解決（`sdk.CacheDBPath`）。`GKILL_HOME` あり／なし（pluginDirから推定）／想定外の構成（プラグインフォルダにフォールバック）／pluginDirが空、の4パターンとパス要素の検証。6プラグインが1文字違わず同じものを持っていたのでSDKへ移した |

`clear_cache plugin` でキャッシュを消せることは
`src/server/gkill/main/common/common_test.go` でテストしている。

## 実行方法

プラグインは独立した Go モジュールなので `cd src/server && go test ./...`（`npm run test_server`）では実行されない。
モジュールを1つずつ回す `npm run test_plugins`（`src/tools/test_plugins.mjs`）を用意しており、`npm test` からも呼ばれる。

```bash
# 全プラグインまとめて（go.mod を持つディレクトリを自動で探して回る）
npm run test_plugins

# 個別に実行する場合
cd src/plugins/gkill_plugin_claudecode && go test ./...
cd src/plugins/gkill_plugin_codex      && go test ./...
cd src/plugins/gkill_plugin_chatgpt    && go test ./...
cd src/plugins/gkill_plugin_claudeai   && go test ./...
cd src/plugins/gkill_plugin_archived_git_commit_log && go test ./...
```

`gkill_plugin_codex` でとくに落としてはいけないテスト:

- `TestParseRolloutTakesIdentityFromFirstMetaOnly` — サブエージェントのファイルには2つ目として
  「親の」`session_meta` が入っている。identity をマージすると自分が親にすり替わり、
  親子の Kyou ID が衝突する
- `TestKyouIDIsStableAcrossAppend` — 追記しても既存の Kyou ID が動かないこと
  （動くとユーザが付けたタグやテキストが迷子になる）
- `TestClassifyHeadUnknownPayloadIsKept` — 判定できない行を捨てないこと
  （捨てるとログの形式が変わった日に会話が静かに消える）
- `TestConcurrentReadDuringBuild` — 構築中に検索が詰まらないこと（WAL + ロック分割の回帰）

## プラグイン SDK

SDK 自体のテストは `src/server/gkill/plugin/sdk/` にあり、`src/server` のテストなので
`npm run test_server` で走る。詳細は [server/gkill/plugin/sdk/ABOUT_TEST.md](../server/gkill/plugin/sdk/ABOUT_TEST.md) を参照。

- `config_test.go`（4テスト）— `EnsureConfig`（config.json の自動生成）。生成される／既存ファイルを
  上書きしない／`DefaultConfig` が nil なら作らない／`pluginDir` が空ならカレントディレクトリを汚さない。
- `sdk_test.go`（18テスト）— `Run()` の stdin/stdout ループ本体（`TestRunLoop_*`）。コマンド分岐、
  未実装時のフォールバック、壊れた JSON でも止まらないこと、`close` / stdin クローズでの終了、
  `get_rep_name` の `rep_names`（未実装なら欄なし・実装済みで0個なら `[]`・エラーは `errors`）を固定している。

## 新しいプラグインのテスト方針

新しいプラグインを作成した場合は以下を推奨：

1. `main.go` の `FindKyous` ロジックをユニットテスト可能な関数に分離する
2. ローカルにエクスポート ZIP（または相当データファイル）を用意し、手動動作確認を行う
3. プラグインを gkill に組み込み、E2E で Kyou がタイムラインに表示されることを確認する
