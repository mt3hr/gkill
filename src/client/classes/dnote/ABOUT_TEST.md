# dnote テスト仕様

## 概要

Dynamic Note (D-note) モジュールのテスト。Kyou データに対する集計・フィルタリング・シリアライゼーション機能を検証する。

## テストフレームワーク

Vitest

## テストファイル一覧

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/unit/dnote/predicates.test.ts` | 28種のフィルタ述語関数 |
| `src/client/__tests__/unit/dnote/key-getters.test.ts` | 9種のキー取得関数 |
| `src/client/__tests__/unit/dnote/aggregate-targets.test.ts` | 7種の集計ターゲット |
| `src/client/__tests__/unit/dnote/aggregators.test.ts` | DnoteAgregator / DnoteListAggregator |
| `src/client/__tests__/unit/dnote/serialization.test.ts` | 辞書データのシリアライゼーション |

## テスト内容

- **Predicates**: データ型判定、日付範囲フィルタ、タグマッチ、テキスト検索など28種の述語関数
- **Key Getters**: 日付、タグ名、リポジトリ名など9種のグルーピングキー取得
- **Aggregate Targets**: カウント、合計、平均、最小、最大など7種の集計対象
- **Aggregators**: `DnoteAgregator`（単一集計）と `DnoteListAggregator`（リスト集計）の動作検証
- **Serialization**: D-note 設定辞書の JSON シリアライゼーション / デシリアライゼーション

## 実行方法

```bash
npm run test_client_unit
```
