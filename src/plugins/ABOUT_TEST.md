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
| `src/plugins/gkill_plugin_{claudecode,chatgpt,claudeai,codex}/cache_path_test.go` | キャッシュDBの置き場所の解決。`GKILL_HOME` あり／なし（pluginDirから推定）／想定外の構成（プラグインフォルダにフォールバック）／pluginDirが空、の4パターン |

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
- `sdk_test.go`（14テスト）— `Run()` の stdin/stdout ループ本体（`TestRunLoop_*`）。コマンド分岐、
  未実装時のフォールバック、壊れた JSON でも止まらないこと、`close` / stdin クローズでの終了などを固定している。

## 新しいプラグインのテスト方針

新しいプラグインを作成した場合は以下を推奨：

1. `main.go` の `FindKyous` ロジックをユニットテスト可能な関数に分離する
2. ローカルに `conversations.json`（または相当データファイル）を用意し、手動動作確認を行う
3. プラグインを gkill に組み込み、E2E で Kyou がタイムラインに表示されることを確認する
