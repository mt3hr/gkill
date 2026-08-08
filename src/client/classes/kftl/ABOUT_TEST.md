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
| `src/client/__tests__/unit/kftl/kftl-type-detection.test.ts` | ステートメント型の判定（日本語プレフィックス + ASCIIプレフィックス、否定ケース含む） |
| `src/client/__tests__/unit/kftl/kftl-request-generation.test.ts` | 解析結果からの API リクエスト生成（ASCIIプレフィックス入力: `#tag1,tag2`、`?日時`、`,`/`,,` 区切り、`--` ブロック、保存文字 `!` を含む） |
| `src/client/__tests__/unit/kftl/kftl-individual-types.test.ts` | ステートメント型ごとの補足テスト（Split と SplitAndNextSecond の排他、Kmemo の catch-all、プレフィックスの一意性、startsWith 型と exact 型の差） |

## テスト内容

- **Request Map**: KFTL テキストから `RequestMap` への変換処理
- **Statement Parsing**: 行単位のステートメント解析（プレフィックス、内容、メタ情報）
- **Type Detection**: `kmemo:`, `mi:`, `timeis:` 等のステートメント型判定
- **Request Generation**: 解析済みステートメントから GkillAPI リクエストオブジェクトへの変換
- **Individual Types**: ステートメント型は全41種（基底 `KFTLStatementLine` を除く）。このファイルはその全数を個別に回すものではなく、型判定で取り違えが起きやすい箇所（Split / SplitAndNextSecond、Kmemo の catch-all、exact-match プレフィックスの重複、startsWith と exact の違い）を補足的に検証する

## 実行方法

```bash
npm run test_client_unit
```
