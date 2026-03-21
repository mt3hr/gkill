# kftl テスト仕様

## 概要

Go バックエンドの KFTL パーサパッケージのテスト。KFTL テキストの解析、ステートメント処理、リクエストマップ構築をカバーする。

## テストフレームワーク

Go `testing` パッケージ

## テストファイル一覧

| ファイル | テスト内容 |
|---------|-----------|
| `kftl_factory_test.go` | KftlFactory のインスタンス生成とKFTLテキスト全体の解析・リクエスト生成 |
| `kftl_statement_test.go` | ステートメント単位の解析ロジック |
| `kftl_request_map_test.go` | リクエストマップの構築と各データ型へのマッピング |

## テスト内容

- **KftlFactory**: `GenerateAndExecuteRequests` による KFTL テキスト全体の処理フロー、コンテキスト管理、時刻計算（`nowFromCtx`）
- **Statement**: 行ごとのステートメント解析、型プレフィックス判定、メタ情報抽出
- **Request Map**: 解析結果からの API リクエストマップ構築、各データ型（Kmemo, Mi, TimeIs 等）へのマッピング

## 実行方法

```bash
cd src/server && go test ./gkill/api/kftl/...
```

または:

```bash
npm run test_server
```
