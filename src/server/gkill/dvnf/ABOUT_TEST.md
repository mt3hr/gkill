# dvnf テスト仕様

## 概要

DVNF (Data Versioning and Naming Framework) パッケージのテスト。ファイルのバージョン管理と命名規則に基づくファイル管理機能を検証する。

## テストフレームワーク

Go `testing` パッケージ

## テストファイル一覧

| ファイル | テスト内容 |
|---------|-----------|
| `dvnf_test.go` | DVNF コアロジック（ファイル管理・バージョニング） |
| `cmd/dvnf_cmd_test.go` | DVNF CLI コマンドハンドリング |

## テスト内容

- **ファイル管理**: DVNF 規則に基づくファイルの作成、移動、名前変更
- **バージョニング**: ファイルバージョンの追跡と管理
- **CLI コマンド**: `dvnf` サブコマンドの引数解析と実行

## 実行方法

```bash
cd src/server && go test ./gkill/dvnf/...
```

または:

```bash
npm run test_server
```
