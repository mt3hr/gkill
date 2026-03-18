# dnote - Dynamic Note システム

## 概要

DNote（Dynamic Note）は、Kyou データの動的集計・フィルタリング・グルーピング・可視化を行うシステム。
Predicate パターン（AND/OR/NOT 論理演算）でデータを絞り込み、
KeyGetter でグルーピングし、AggregateTarget で集計値を算出する。

## ディレクトリ構造

```
dnote/
├── (ルートファイル 17個)         # コア型定義
├── dnote-agregate-target/      # 集計対象（22ファイル）
├── dnote-filter/               # フィルタ（2ファイル）
├── dnote-key-getter/           # グルーピングキー（9ファイル）
├── dnote-predicate/            # 検索条件述語（39ファイル）
│   └── target-kyou-predicate/  # 対象 Kyou 述語（3ファイル）
├── pulldown-menu/              # UI プルダウンメニュー（4ファイル）
└── serialize/                  # シリアライズ辞書（5ファイル）
```

## 設計思想

### 3つの柱

```
1. Predicate（フィルタリング）
   → どのデータを対象とするか
   → AND / OR / NOT の論理演算で組み合わせ可能

2. KeyGetter（グルーピング）
   → データをどの基準でグループ分けするか
   → 日付、曜日、週、月、タグ、データ型、タイトル等

3. AggregateTarget（集計）
   → グループごとにどの値を集計するか
   → 平均、合計、最大、最小、カウント
```

### データフロー

```
全 Kyou データ
  ↓  Predicate (フィルタリング)
対象データ
  ↓  KeyGetter (グルーピング)
グループ別データ
  ↓  AggregateTarget (集計)
集計結果
  ↓  Filter (Top/Bottom)
最終結果
```

## ルートファイル（17ファイル）

| ファイル | 役割 |
|---------|------|
| `dnote-aggregator.ts` | DNote 集計メインロジック |
| `dnote-agregate-target.ts` | 集計対象の基底インタフェース |
| `dnote-item.ts` | DNote アイテム（集計結果の1行） |
| `dnote-key-getter.ts` | キー取得の基底インタフェース |
| `dnote-kyou-filter.ts` | Kyou フィルタの基底インタフェース |
| `dnote-list-aggregator.ts` | リスト集計ロジック |
| `dnote-matcher.ts` | データマッチングロジック |
| `dnote-predicate.ts` | 述語の基底インタフェース |
| `dnote-select-item.ts` | 選択アイテム |
| `kyou-loader.ts` | Kyou データローダー |
| `predicate-card.ts` | 述語カード（UI 表示用） |
| `predicate-group-type.ts` | 述語グループ型（AND/OR） |
| `predicate-group.ts` | 述語グループ |
| `predicate.ts` | 述語の基底型 |
| `related-kyou-query.ts` | 関連 Kyou クエリ |
| `related-time-match-type.ts` | 関連時刻マッチ型 |
| `aggregate-grouping-list-result-record.ts` | グルーピング集計結果レコード |

## `dnote-agregate-target/`（22ファイル）— 集計対象

グループごとの集計値を算出するクラス群。

### 平均（Average）

| ファイル | 集計対象 |
|---------|---------|
| `agregate-average-kc-num-value.ts` | KC 数値の平均 |
| `agregate-average-lantana-mood.ts` | Lantana 気分値の平均 |
| `agregate-average-nlog-amount.ts` | Nlog 金額の平均 |
| `agregate-average-timeis-time.ts` | TimeIs 経過時間の平均 |
| `agregate-average-timeis-start-time.ts` | TimeIs 開始時刻の平均 |
| `agregate-average-timeis-end-time.ts` | TimeIs 終了時刻の平均 |
| `agregate-average-git-commit-log-code-add.ts` | Git コミット追加行数の平均 |
| `agregate-average-git-commit-log-code-delete.ts` | Git コミット削除行数の平均 |
| `agregate-average-git-commit-log-code-diff.ts` | Git コミット差分行数の平均 |

### 合計（Sum）

| ファイル | 集計対象 |
|---------|---------|
| `agregate-sum-kc-num-value.ts` | KC 数値の合計 |
| `agregate-sum-lantana-mood.ts` | Lantana 気分値の合計 |
| `agregate-sum-nlog-amount.ts` | Nlog 金額の合計 |
| `agregate-sum-timeis-time.ts` | TimeIs 経過時間の合計 |
| `agregate-sum-git-commit-log-code-add.ts` | Git コミット追加行数の合計 |
| `agregate-sum-git-commit-log-code-delete.ts` | Git コミット削除行数の合計 |
| `agregate-sum-git-commit-log-code-diff.ts` | Git コミット差分行数の合計 |

### 最大/最小/カウント

| ファイル | 集計対象 |
|---------|---------|
| `agregate-max-kc-num-value.ts` | KC 数値の最大 |
| `agregate-min-kc-num-value.ts` | KC 数値の最小 |
| `agregate-count-kyou.ts` | Kyou の件数カウント |

### ユーティリティ

| ファイル | 説明 |
|---------|------|
| `average-info.ts` | 平均値計算の補助情報 |

## `dnote-filter/`（2ファイル）— データフィルタ

集計結果の上位/下位を絞り込むフィルタ。

| ファイル | 説明 |
|---------|------|
| `filter-top-kyous.ts` | 上位 N 件を取得 |
| `filter-bottom-kyous.ts` | 下位 N 件を取得 |

## `dnote-key-getter/`（9ファイル）— グルーピングキー

データをグループ分けする基準を定義。

| ファイル | グルーピング基準 |
|---------|----------------|
| `rerated-date-getter.ts` | 関連日付 |
| `related-week-getter.ts` | 関連週 |
| `related-week-day-getter.ts` | 関連曜日 |
| `related-month-getter.ts` | 関連月 |
| `tag-getter.ts` | タグ名 |
| `data-type-getter.ts` | データ型 |
| `title-getter.ts` | タイトル |
| `lantana-mood-getter.ts` | Lantana 気分値 |
| `nlog-shop-name-getter.ts` | Nlog 店名 |

## `dnote-predicate/`（39ファイル）— 検索条件述語

AND / OR / NOT の論理演算で組み合わせ可能な述語群。

### 論理演算子

| ファイル | 説明 |
|---------|------|
| `and-predicate.ts` | AND 結合 |
| `or-predicate.ts` | OR 結合 |
| `not-predicate.ts` | NOT 否定 |

### データ型フィルタ

| ファイル | 条件 |
|---------|------|
| `data-type-prefix-predicate.ts` | データ型プレフィックス一致 |

### Kmemo 述語

| ファイル | 条件 |
|---------|------|
| `kmemo-content-contain-predicate.ts` | 内容に文字列を含む |
| `kmemo-content-not-contain-predicate.ts` | 内容に文字列を含まない |

### KC 述語

| ファイル | 条件 |
|---------|------|
| `kc-title-contain-predicate.ts` | タイトルに文字列を含む |
| `kc-title-not-contain-predicate.ts` | タイトルに文字列を含まない |

### Lantana 述語

| ファイル | 条件 |
|---------|------|
| `lantana-mood-contain-predicate.ts` | 気分値が範囲内 |
| `lantana-mood-equal-predicate.ts` | 気分値が一致 |
| `lantana-mood-not-contain-predicate.ts` | 気分値が範囲外 |

### Mi 述語

| ファイル | 条件 |
|---------|------|
| `mi-title-contain-predicate.ts` | タイトルに文字列を含む |
| `mi-title-not-contain-predicate.ts` | タイトルに文字列を含まない |

### Nlog 述語

| ファイル | 条件 |
|---------|------|
| `nlog-amount-contain-predicate.ts` | 金額が範囲内 |
| `nlog-amount-not-contain-predicate.ts` | 金額が範囲外 |
| `nlog-shop-name-contain-predicate.ts` | 店名に文字列を含む |
| `nlog-shop-name-not-contain-predicate.ts` | 店名に文字列を含まない |
| `nlog-title-contain-predicate.ts` | タイトルに文字列を含む |

### TimeIs 述語

| ファイル | 条件 |
|---------|------|
| `timeis-title-contain-predicate.ts` | タイトルに文字列を含む |
| `timeis-title-not-contain-predicate.ts` | タイトルに文字列を含まない |

### Text 述語

| ファイル | 条件 |
|---------|------|
| `text-content-contain-predicate.ts` | 内容に文字列を含む |
| `text-content-not-contain-predicate.ts` | 内容に文字列を含まない |

### Git Commit Log 述語

| ファイル | 条件 |
|---------|------|
| `git-commit-log-code-add-contain-predicate.ts` | 追加行数が範囲内 |
| `git-commit-log-code-add-not-contain-predicate.ts` | 追加行数が範囲外 |
| `git-commit-log-code-delete-contain-predicate.ts` | 削除行数が範囲内 |
| `git-commit-log-code-delete-not-contain-predicate.ts` | 削除行数が範囲外 |
| `git-commit-log-code-diff-contain-predicate.ts` | 差分行数が範囲内 |
| `git-commit-log-code-diff-not-contain-predicate.ts` | 差分行数が範囲外 |

### 関連時刻述語

| ファイル | 条件 |
|---------|------|
| `related-time-between-predicate.ts` | 関連時刻が範囲内 |
| `related-time-not-between-predicate.ts` | 関連時刻が範囲外 |
| `related-time-in-today-predicate.ts` | 今日のデータ |

### タグ述語

| ファイル | 条件 |
|---------|------|
| `tag-equal-predicate.ts` | タグが一致 |

### 対象 Kyou 述語（`target-kyou-predicate/` 3ファイル）

| ファイル | 条件 |
|---------|------|
| `equal-rep-data-type-target-kyou-predicate.ts` | リポジトリデータ型一致 |
| `equal-tags-target-kyou-predicate.ts` | タグ一致 |
| `equal-title-target-kyou-predicate.ts` | タイトル一致 |

## `pulldown-menu/`（4ファイル）— UI プルダウンメニュー

DNote 設定 UI 用のプルダウンメニューアイテム定義。

| ファイル | 説明 |
|---------|------|
| `aggregate-target-menu-items.ts` | 集計対象の選択肢 |
| `kyou-getter-menu-items.ts` | Kyou 取得方法の選択肢 |
| `predicate-menu-items.ts` | 述語の選択肢 |
| `rep-type-menu-items.ts` | リポジトリ型の選択肢 |

## `serialize/`（5ファイル）— シリアライズ辞書

DNote 設定の JSON シリアライズ/デシリアライズ用辞書。

| ファイル | 説明 |
|---------|------|
| `dnote-aggregate-target-dictionary.ts` | 集計対象の型辞書 |
| `dnote-key-getter-dictionary.ts` | キー取得の型辞書 |
| `dnote-kyou-filter-dictionary.ts` | フィルタの型辞書 |
| `dnote-predicate-dictionary.ts` | 述語の型辞書 |
| `regist-dictionary.ts` | 辞書登録ユーティリティ |

## 開発ガイドライン

### 新しい集計対象を追加する場合

1. `dnote-agregate-target/` に `agregate-xxx-yyy.ts` を作成
2. `dnote-agregate-target.ts` の基底インタフェースを実装
3. `pulldown-menu/aggregate-target-menu-items.ts` に選択肢を追加
4. `serialize/dnote-aggregate-target-dictionary.ts` に型を登録

### 新しい述語を追加する場合

1. `dnote-predicate/` に `xxx-predicate.ts` を作成
2. `dnote-predicate.ts` の基底インタフェースを実装
3. `pulldown-menu/predicate-menu-items.ts` に選択肢を追加
4. `serialize/dnote-predicate-dictionary.ts` に型を登録
