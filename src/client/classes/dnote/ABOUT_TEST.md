# dnote テスト仕様

## 概要

Dynamic Note (D-note) モジュールのテスト。Kyou データに対する集計・フィルタリング・シリアライゼーション機能を検証する。

## テストフレームワーク

Vitest

## テストファイル一覧

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/unit/dnote/predicates.test.ts` | フィルタ述語関数（実装は38種。うち27種をテスト） |
| `src/client/__tests__/unit/dnote/key-getters.test.ts` | 9種のキー取得関数 |
| `src/client/__tests__/unit/dnote/aggregate-targets.test.ts` | 集計ターゲット（実装は19種） |
| `src/client/__tests__/unit/dnote/aggregators.test.ts` | DnoteAgregator / DnoteListAggregator |
| `src/client/__tests__/unit/dnote/serialization.test.ts` | 辞書データのシリアライゼーション |
| `src/client/__tests__/unit/dnote/trend-aggregator.test.ts` | DnoteTrendAggregator（トレンドグラフの時系列バケット集計: 日/週/月粒度、ゼロ埋め、バケット上限） |
| `src/client/__tests__/unit/dnote/kyou-loader.test.ts` | Dnote 用 Kyou ローダ（対象 Kyou の読み込み） |

## テスト内容

- **Predicates**: データ型判定、日付範囲フィルタ、タグマッチ、テキスト検索など。`dnote-predicate/` に33種 + `target-kyou-predicate/` に5種＝38種あり、そのうち27種を `predicates.test.ts` がカバーする。未カバーは `git-commit-log-code-*` 系6種、`related-time-week`、`text-content-contains` / `text-content-equal`、`equal-rep-data-type-target-kyou` / `equal-title-target-kyou` の計11種
- **Key Getters**: 日付、タグ名、リポジトリ名など9種のグルーピングキー取得
- **Aggregate Targets**: カウント、合計、平均、最小、最大など。`dnote-aggregate-target/` の21ファイルのうち19種が集計対象（残り2つは `average-info.ts` / `time-of-day-average-info.ts` のヘルパ）
- **Aggregators**: `DnoteAgregator`（単一集計）と `DnoteListAggregator`（リスト集計）の動作検証
- **Serialization**: D-note 設定辞書の JSON シリアライゼーション / デシリアライゼーション

## 対象ディレクトリと対応表

`src/client/classes/dnote/` のサブディレクトリと、それを検証しているテストの対応。

| ディレクトリ | ファイル数 | 検証しているテスト |
|-------------|-----------|------------------|
| （ルート） | 18 | `aggregators.test.ts` / `trend-aggregator.test.ts` |
| `dnote-predicate/` | 33 | `predicates.test.ts` |
| `dnote-predicate/target-kyou-predicate/` | 5 | `predicates.test.ts` |
| `dnote-aggregate-target/` | 21 | `aggregate-targets.test.ts` |
| `dnote-key-getter/` | 9 | `key-getters.test.ts` |
| `dnote-trend/` | 3 | `trend-aggregator.test.ts` |
| `serialize/` | 5 | `serialization.test.ts` |
| `dnote-filter/` | 2 | 専用テストなし（`aggregators.test.ts` から間接的に通る） |
| `pulldown-menu/` | 6 | 専用テストなし（UI プルダウンの選択肢定義） |

## 実行方法

```bash
npm run test_client_unit
```
