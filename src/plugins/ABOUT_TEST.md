# テスト仕様 — src/plugins/

## 概要

`src/plugins/` 配下の各プラグイン（gkill_plugin_claudeai, gkill_plugin_chatgpt, examples/gkill_example）には独自のテストファイルは存在しない。

プラグイン機能のテストは以下でカバーされている：

| テスト場所 | テスト内容 |
|---|---|
| `src/server/gkill/api/gkill_server_api/` | プラグインのロード・subprocess起動・stdio通信・クラッシュ後自動再起動の統合テスト |
| `src/server/gkill/dao/reps/` | `plugin_repository_impl.go` のリポジトリ層テスト |

## プラグイン SDK

`src/server/gkill/plugin/sdk/` の SDK 自体にもテストは存在しない。SDK は単純な stdin/stdout ループのため、実際のプラグインバイナリを通じた統合テストで品質を担保する。

## 新しいプラグインのテスト方針

新しいプラグインを作成した場合は以下を推奨：

1. `main.go` の `FindKyous` ロジックをユニットテスト可能な関数に分離する
2. ローカルに `conversations.json`（または相当データファイル）を用意し、手動動作確認を行う
3. プラグインを gkill に組み込み、E2E で Kyou がタイムラインに表示されることを確認する
