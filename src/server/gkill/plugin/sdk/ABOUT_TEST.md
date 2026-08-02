# plugin/sdk テスト仕様

## 概要

プラグイン作者向け Go SDK のテスト。**18テスト（2ファイル）**。

`sdk.Run()` の stdin/stdout ループと、`sdk.EnsureConfig()` の `config.json` 自動生成を検証する。
SDK は gkill 本体と別プロセスで動くプラグイン側のライブラリなので、ここでの回帰は
全プラグイン（chatgpt / claudeai / claudecode / example）に同時に波及する。

## テストファイル

| ファイル | テスト数 | 対象 |
|---------|---------|------|
| `sdk_test.go` | 14 | `Run()` のメッセージループ（`TestRunLoop_*`） |
| `config_test.go` | 4 | `EnsureConfig()`（`TestEnsureConfig_*`） |

## sdk_test.go — stdio ループ

`Run()` は「1行 = 1 JSON メッセージ」のループ本体。プラグインが gkill 本体と話す唯一の経路なので、
コマンド分岐・未実装時のフォールバック・異常入力での継続を個別に固定している。

| テスト | 検証内容 |
|-------|---------|
| `TestRunLoop_Ping` | `ping` に応答する |
| `TestRunLoop_GetRepName` | `get_rep_name` がマニフェストの `rep_name` を返す |
| `TestRunLoop_FindKyous` | `find_kyous` が Handler の結果を返す |
| `TestRunLoop_FindKyousNotImplemented` | `FindKyous` 未設定時にエラー応答になる |
| `TestRunLoop_FindKyousError` | Handler がエラーを返したとき、エラーとして中継される |
| `TestRunLoop_GetKyouFallsBackToFindKyous` | `GetKyou` 未実装なら `FindKyous` で代替する |
| `TestRunLoop_GetContentHTMLDefault` | `GetContentHTML` 未実装時の既定 HTML |
| `TestRunLoop_PostConfigDefaultSavesForm` | `PostConfig` 未実装時、送られたフォームを既定処理で保存する |
| `TestRunLoop_UnknownCommand` | 未知のコマンドでエラー応答を返す |
| `TestRunLoop_InvalidJSONContinues` | 壊れた JSON を受けてもループを止めない |
| `TestRunLoop_CloseStopsLoop` | `close` でループを終了する |
| `TestRunLoop_StdinCloseReturnsFalse` | stdin が閉じたらループを抜ける |
| `TestRunLoop_PassesUserIDToHandler` | `user_id` が Handler へ渡る |
| `TestRunLoop_PassesConfigToHandler` | `config.json` の内容が Handler へ渡る |

## config_test.go — EnsureConfig

初回起動時に `manifest.json` と同じフォルダへ `config.json` を既定値で自動生成する処理。
**既存ファイルを上書きしないこと**がこの機能の要件なので、そこを固定している。

| テスト | 検証内容 |
|-------|---------|
| `TestEnsureConfig_CreatesWhenMissing` | 無ければ既定値で作る |
| `TestEnsureConfig_KeepsExisting` | 既にあれば触らない（ユーザの設定を壊さない） |
| `TestEnsureConfig_NoDefaults` | `DefaultConfig` を持たない Handler では何も作らない |
| `TestEnsureConfig_EmptyPluginDir` | プラグインディレクトリが空文字なら何もしない |

## 実行方法

```bash
cd src/server && go test ./gkill/plugin/sdk/...
```

`npm run test_server`（`cd src/server && go test ./...`）にも含まれる。

## 関連ドキュメント

- [../README.md](../README.md) — SDK の使い方
- [../../../../plugins/ABOUT_TEST.md](../../../../plugins/ABOUT_TEST.md) — SDK を使う各プラグインのテスト（独立モジュールのため `npm test` の対象外）
- [../../../../../documents/reverse/plugin-system.md](../../../../../documents/reverse/plugin-system.md) — プラグインシステム仕様
