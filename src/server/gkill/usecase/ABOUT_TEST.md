# usecase テスト仕様

## 概要

`gkill/usecase/` パッケージのテスト仕様。HTTP 非依存のビジネスロジック層（16ファイル）。

## テスト状態

現在、usecase パッケージには専用のテストファイルは存在しない。ビジネスロジックは `gkill/api/gkill_server_api/` の API ハンドラ統合テストを通じて間接的にテストされている。

## 実行方法

```bash
cd src/server && go test ./gkill/usecase/...
```

または:

```bash
npm run test_server
```
