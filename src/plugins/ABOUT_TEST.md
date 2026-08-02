# テスト仕様 — src/plugins/

## 概要

`src/plugins/` 配下のプラグインのうち、examples/gkill_example には独自のテストファイルは存在しない。

プラグイン機能のテストは以下でカバーされている：

| テスト場所 | テスト内容 |
|---|---|
| `src/server/gkill/api/gkill_server_api/` | プラグインのロード・subprocess起動・stdio通信・クラッシュ後自動再起動の統合テスト |
| `src/server/gkill/dao/reps/` | `plugin_repository_impl.go` のリポジトリ層テスト |
| `src/plugins/gkill_plugin_claudecode/loader_test.go` | ターン分割・ファイル種別判定・サブエージェント紐付け・ツール要約・HTML生成のユニットテスト（`testdata/` の合成トランスクリプトを使用） |
| `src/plugins/gkill_plugin_{claudecode,chatgpt,claudeai}/cache_path_test.go` | キャッシュDBの置き場所の解決。`GKILL_HOME` あり／なし（pluginDirから推定）／想定外の構成（プラグインフォルダにフォールバック）／pluginDirが空、の4パターン |

`clear_cache plugin` でキャッシュを消せることは
`src/server/gkill/main/common/common_test.go` でテストしている。

## 実行方法

プラグインは独立した Go モジュールなので、`npm test`（`src/server` のみ対象）には含まれない。
テストを持つプラグインは個別に実行する。

```bash
cd src/plugins/gkill_plugin_claudecode && go test ./...
cd src/plugins/gkill_plugin_chatgpt    && go test ./...
cd src/plugins/gkill_plugin_claudeai   && go test ./...
```

## プラグイン SDK

`src/server/gkill/plugin/sdk/config_test.go` で `EnsureConfig`（config.json の自動生成）をテストしている。
生成される／既存ファイルを上書きしない／`DefaultConfig` が nil なら作らない／`pluginDir` が空なら
カレントディレクトリを汚さない、の4点。`src/server` のテストなので `npm run test_server` で走る。

stdin/stdout ループ本体にはテストが無く、実際のプラグインバイナリを通じた統合テストで品質を担保する。

## 新しいプラグインのテスト方針

新しいプラグインを作成した場合は以下を推奨：

1. `main.go` の `FindKyous` ロジックをユニットテスト可能な関数に分離する
2. ローカルに `conversations.json`（または相当データファイル）を用意し、手動動作確認を行う
3. プラグインを gkill に組み込み、E2E で Kyou がタイムラインに表示されることを確認する
