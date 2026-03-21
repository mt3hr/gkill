# kftl テスト仕様

## 概要

KFTL (gkill 独自テキストフォーマット) の TypeScript パーサをテストする。テキスト入力からの構文解析、ステートメント分類、リクエスト生成までの全工程をカバーしている。

## テストフレームワーク

Vitest

## テストファイル一覧

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/unit/kftl/kftl-request-map.test.ts` | リクエストマップの構築 |
| `src/client/__tests__/unit/kftl/kftl-statement.test.ts` | ステートメントの解析ロジック |
| `src/client/__tests__/unit/kftl/kftl-type-detection.test.ts` | ステートメント型の判定 |
| `src/client/__tests__/unit/kftl/kftl-request-generation.test.ts` | 解析結果からの API リクエスト生成 |
| `src/client/__tests__/unit/kftl/kftl-individual-types.test.ts` | 個別ステートメント型のテスト（42種） |

## テスト内容

- **Request Map**: KFTL テキストから `RequestMap` への変換処理
- **Statement Parsing**: 行単位のステートメント解析（プレフィックス、内容、メタ情報）
- **Type Detection**: `kmemo:`, `mi:`, `timeis:` 等のステートメント型判定
- **Request Generation**: 解析済みステートメントから GkillAPI リクエストオブジェクトへの変換
- **Individual Types**: 全42種のステートメント型（Kmemo, Mi, TimeIs, URLog, Nlog, Lantana, KC, Tag, Text 等）の個別動作検証

## 実行方法

```bash
npm run test_client_unit
```
