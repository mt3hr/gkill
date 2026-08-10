# dnote - Dynamic Note システム

## 概要

DNote（Dynamic Note）は、Kyou データの動的集計・フィルタリング・グルーピング・可視化を行うシステム。
Predicate パターン（AND/OR/NOT 論理演算）でデータを絞り込み、
KeyGetter でグルーピングし、AggregateTarget で集計値を算出する。

## ディレクトリ構造

```
dnote/
├── (ルートファイル 18個)         # コア型定義 + トレンドグラフ集計（dnote-trend-aggregator.ts）
├── dnote-aggregate-target/      # 集計対象（21ファイル）
├── dnote-filter/               # フィルタ（2ファイル）
├── dnote-key-getter/           # グルーピングキー（9ファイル）
├── dnote-predicate/            # 検索条件述語（33ファイル）
│   └── target-kyou-predicate/  # 対象 Kyou 述語（5ファイル）
├── dnote-trend/                # トレンドグラフ型定義（3ファイル）
├── pulldown-menu/              # UI プルダウンメニュー（6ファイル）
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

## ルートファイル（18ファイル）

| ファイル | 役割 |
|---------|------|
| `dnote-aggregator.ts` | DNote 集計メインロジック |
| `dnote-aggregate-target.ts` | 集計対象の基底インタフェース |
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
| `dnote-trend-aggregator.ts` | トレンドグラフの時系列バケット集計（詳細は「トレンドグラフ」節） |

## `dnote-aggregate-target/`（21ファイル）— 集計対象

グループごとの集計値を算出するクラス群。

### 平均（Average）

| ファイル | 集計対象 |
|---------|---------|
| `aggregate-average-kc-num-value.ts` | KC 数値の平均 |
| `aggregate-average-lantana-mood.ts` | Lantana 気分値の平均 |
| `aggregate-average-nlog-amount.ts` | Nlog 金額の平均 |
| `aggregate-average-timeis-time.ts` | TimeIs 経過時間の平均 |
| `aggregate-average-timeis-start-time.ts` | TimeIs 開始時刻の平均 |
| `aggregate-average-timeis-end-time.ts` | TimeIs 終了時刻の平均 |
| `aggregate-average-git-commit-log-code-addition-count.ts` | Git コミット追加行数の平均 |
| `aggregate-average-git-commit-log-code-deletion-count.ts` | Git コミット削除行数の平均 |
| `aggregate-average-git-commit-log-code-count.ts` | Git コミット差分行数の平均 |

### 合計（Sum）

| ファイル | 集計対象 |
|---------|---------|
| `aggregate-sum-kc-num-value.ts` | KC 数値の合計 |
| `aggregate-sum-lantana-mood.ts` | Lantana 気分値の合計 |
| `aggregate-sum-nlog-amount.ts` | Nlog 金額の合計 |
| `aggregate-sum-timeis-time.ts` | TimeIs 経過時間の合計 |
| `aggregate-sum-git-commit-log-code-addition-count.ts` | Git コミット追加行数の合計 |
| `aggregate-sum-git-commit-log-code-deletion-count.ts` | Git コミット削除行数の合計 |
| `aggregate-sum-git-commit-log-code-count.ts` | Git コミット差分行数の合計 |

### 最大/最小/カウント

| ファイル | 集計対象 |
|---------|---------|
| `aggregate-max-kc-num-value.ts` | KC 数値の最大 |
| `aggregate-min-kc-num-value.ts` | KC 数値の最小 |
| `aggregate-count-kyou.ts` | Kyou の件数カウント |

### ユーティリティ

| ファイル | 説明 |
|---------|------|
| `average-info.ts` | 平均値計算の補助情報 |
| `time-of-day-average-info.ts` | 時刻（TimeIs 開始/終了時刻）の平均計算の補助情報 |

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
| `related-date-getter.ts` | 関連日付 |
| `related-week-getter.ts` | 関連週 |
| `related-week-day-getter.ts` | 関連曜日 |
| `related-month-getter.ts` | 関連月 |
| `tag-getter.ts` | タグ名 |
| `data-type-getter.ts` | データ型 |
| `title-getter.ts` | タイトル |
| `lantana-mood-getter.ts` | Lantana 気分値 |
| `nlog-shop-name-getter.ts` | Nlog 店名 |

## `dnote-predicate/`（33ファイル）— 検索条件述語

AND / OR / NOT の論理演算で組み合わせ可能な述語群（33 .ts ファイル + 1 サブディレクトリ）。

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
| `kmemo-content-contains-predicate.ts` | 内容に文字列を含む |
| `kmemo-content-equal-predicate.ts` | 内容が一致 |

### KC 述語

| ファイル | 条件 |
|---------|------|
| `kc-title-contains-predicate.ts` | タイトルに文字列を含む |
| `kc-title-equal-predicate.ts` | タイトルが一致 |

### Lantana 述語

| ファイル | 条件 |
|---------|------|
| `lantana-mood-greater-than-predicate.ts` | 気分値が指定値より大きい |
| `lantana-mood-less-than-predicate.ts` | 気分値が指定値より小さい |
| `lantana-mood-equal-predicate.ts` | 気分値が一致 |

### Mi 述語

| ファイル | 条件 |
|---------|------|
| `mi-title-contains-predicate.ts` | タイトルに文字列を含む |
| `mi-title-equal-predicate.ts` | タイトルが一致 |

### Nlog 述語

| ファイル | 条件 |
|---------|------|
| `nlog-amount-greater-than-predicate.ts` | 金額が指定値より大きい |
| `nlog-amount-less-than-predicate.ts` | 金額が指定値より小さい |
| `nlog-shop-contains-predicate.ts` | 店名に文字列を含む |
| `nlog-shop-equal-predicate.ts` | 店名が一致 |
| `nlog-title-contains-predicate.ts` | タイトルに文字列を含む |
| `nlog-title-equal-predicate.ts` | タイトルが一致 |

### TimeIs 述語

| ファイル | 条件 |
|---------|------|
| `timeis-title-contains-predicate.ts` | タイトルに文字列を含む |
| `timeis-title-equal-predicate.ts` | タイトルが一致 |

### Text 述語

| ファイル | 条件 |
|---------|------|
| `text-content-contains-predicate.ts` | 内容に文字列を含む |
| `text-content-equal-predicate.ts` | 内容が一致 |

### Git Commit Log 述語

| ファイル | 条件 |
|---------|------|
| `git-commit-log-code-addition-greater-than-predicate.ts` | 追加行数が指定値より大きい |
| `git-commit-log-code-addition-less-than-predicate.ts` | 追加行数が指定値より小さい |
| `git-commit-log-code-deletion-greater-than-predicate.ts` | 削除行数が指定値より大きい |
| `git-commit-log-code-deletion-less-than-predicate.ts` | 削除行数が指定値より小さい |
| `git-commit-log-code-greater-than-predicate.ts` | 差分行数が指定値より大きい |
| `git-commit-log-code-less-than-predicate.ts` | 差分行数が指定値より小さい |

### 関連時刻述語

| ファイル | 条件 |
|---------|------|
| `related-time-after-predicate.ts` | 関連時刻が指定時刻より後 |
| `related-time-before-predicate.ts` | 関連時刻が指定時刻より前 |
| `related-time-week-predicate.ts` | 関連時刻の曜日一致 |

### タグ述語

| ファイル | 条件 |
|---------|------|
| `tag-equal-predicate.ts` | タグが一致 |

### 対象 Kyou 述語（`target-kyou-predicate/` 5ファイル）

| ファイル | 条件 |
|---------|------|
| `equal-id-target-kyou-predicate.ts` | ID一致 |
| `equal-rep-data-type-target-kyou-predicate.ts` | リポジトリデータ型一致 |
| `equal-tags-and-target-kyou-predicate.ts` | タグがすべて対象と等しい |
| `equal-tags-or-target-kyou-predicate.ts` | タグが一部対象と等しい |
| `equal-title-target-kyou-predicate.ts` | タイトル一致 |

## `pulldown-menu/`（6ファイル）— UI プルダウンメニュー

DNote 設定 UI 用のプルダウンメニューアイテム定義。

| ファイル | 説明 |
|---------|------|
| `aggregate-target-menu-items.ts` | 集計対象の選択肢 |
| `kyou-getter-menu-items.ts` | Kyou 取得方法の選択肢 |
| `predicate-menu-items.ts` | 述語の選択肢 |
| `rep-type-menu-items.ts` | リポジトリ型の選択肢 |
| `trend-granularity-menu-items.ts` | トレンドグラフ集計粒度（日/週/月）の選択肢 |
| `trend-chart-type-menu-items.ts` | トレンドグラフ種別（折れ線/棒）の選択肢 |

## `serialize/`（5ファイル）— シリアライズ辞書

DNote 設定の JSON シリアライズ/デシリアライズ用辞書。

| ファイル | 説明 |
|---------|------|
| `dnote-aggregate-target-dictionary.ts` | 集計対象の型辞書 |
| `dnote-key-getter-dictionary.ts` | キー取得の型辞書 |
| `dnote-kyou-filter-dictionary.ts` | フィルタの型辞書 |
| `dnote-predicate-dictionary.ts` | 述語の型辞書 |
| `register-dictionary.ts` | 辞書登録ユーティリティ |

### 型判別文字列は永続データ

集計対象の型判別文字列（`{ "type": "AggregateCountKyou" }` など19種）は
**ユーザの集計定義として永続化される**（`user_config` の `APPLICATION_CONFIG` /
`KEY='DNOTE_JSON_DATA'` に JSON で入り、`/api/get_application_config` と
`/api/update_application_config` で往復する）。

辞書に無い `type` を読むと `build_dnote_aggregate_target_from_json` が
`Unknown aggregate type` を投げ、`use-dnote-view.ts` の `from_json()` は
これを捕捉しないため **Dnote 画面全体が描画不能になる**。
したがって型判別文字列のリネームは、辞書のキーを差し替えるだけでは済まず、
保存済みデータ（サンプルデータ `resources/gkill_sample_data/configs/user_config.db` を含む）の
移行が必須。かつての `Agregate*` → `Aggregate*` の改名は移行済み。

なお `TextContentContainsPredicate` → `KmemoContentContainsPredicate` は
述語辞書に旧名エイリアスを登録する方式で互換を取っている。

## トレンドグラフ（`dnote-trend/` + `dnote-trend-aggregator.ts`）

集計項目・集計リストに続く第3の DNote エンティティ。既存 DNote と同じく**親から渡された kyous**を対象に、
Predicate と AggregateTarget を流用して `related_time` を日/週/月単位のバケットに区切って時系列集計する。
結果は VSparkline で描画される（ビュー幅いっぱいで縦積み）。
バケット期間はクエリの calendar 範囲から、範囲がない場合は kyous の related_time の min/max から導出する。

| ファイル | 説明 |
|---------|------|
| `dnote-trend-aggregator.ts` | 時系列バケット集計ロジック（ゼロ埋め・昇順保証。400バケット上限、超過時は新しい側優先） |
| `dnote-trend/dnote-trend-types.ts` | 粒度（day/week/month）とグラフ種別（line/bar）の型 |
| `dnote-trend/dnote-trend-point.ts` | バケット1点分の集計結果 |
| `dnote-trend/aggregated-value-to-number.ts` | 累積値（number / AverageInfo）の数値化 |

定義は `dnote_json_data` の各定義オブジェクト内 `dnote_trend_graph_view_data` に保存される（キー欠落時は空扱い＝後方互換）。

**既知の制約**: 複数バケットをまたぐ TimeIs は `related_time`（開始時刻）のバケットにのみ計上される。
バケット内の経過時間はバケット境界でトリムされるため、またいだ先のバケットには時間が計上されない。

## 開発ガイドライン

### 新しい集計対象を追加する場合

1. `dnote-aggregate-target/` に `aggregate-xxx-yyy.ts` を作成
2. `dnote-aggregate-target.ts` の基底インタフェースを実装
3. `pulldown-menu/aggregate-target-menu-items.ts` に選択肢を追加
4. `serialize/dnote-aggregate-target-dictionary.ts` に型を登録

### 新しい述語を追加する場合

1. `dnote-predicate/` に `xxx-predicate.ts` を作成
2. `dnote-predicate.ts` の基底インタフェースを実装
3. `pulldown-menu/predicate-menu-items.ts` に選択肢を追加
4. `serialize/dnote-predicate-dictionary.ts` に型を登録
