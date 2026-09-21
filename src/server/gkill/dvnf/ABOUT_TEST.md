# dvnf テスト仕様

## 概要

DVNF（DeVice Name Folder Naming Framework）パッケージのテスト。ファイルのバージョン管理と命名規則に基づくファイル管理機能を検証する。

## テストフレームワーク

Go `testing` パッケージ

## テストファイル一覧

| ファイル | テスト内容 |
|---------|-----------|
| `dvnf_test.go` | DVNF コアロジック（ファイル管理・バージョニング） |
| `cmd/dvnf_cmd_test.go` | DVNF CLI コマンドの引数解析（設定・パス分解・無視パターン・サブコマンド登録） |
| `cmd/copy_file_test.go` | copy / move 共用 `copyFile` の実ファイル操作。コピー後の内容一致と、`copyLastMod` 有効時に mtime が保存されること（Close の**後に** Chtimes する順序がこの関数の存在理由。二重クローズ修正 6b49723e の対象） |

## テスト内容

- **ファイル管理**: DVNF 規則に基づくファイルの作成、移動、名前変更
- **バージョニング**: ファイルバージョンの追跡と管理
- **CLI コマンド**: `dvnf` サブコマンドの引数解析（copy / move / get の実ファイル操作のうちテストがあるのは `copyFile` のみ）

## 実行方法

```bash
cd src/server && go test ./gkill/dvnf/...
```

または:

```bash
npm run test_server
```
