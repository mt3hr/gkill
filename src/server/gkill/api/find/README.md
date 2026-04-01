# find - 検索クエリ型定義

## 概要

Kyou 検索に使用するクエリ構造体と列挙型を定義する。
`FindQuery` は API のデータ取得系エンドポイントで検索条件を指定するために使用される。

## ファイル一覧（5ファイル）

| ファイル | 説明 |
|---------|------|
| `find_query.go` | `FindQuery` 構造体 — 62 フィールドの検索条件 |
| `mi_check_state.go` | `MiCheckState` 列挙型 — Mi タスクのチェック状態フィルタ |
| `mi_sort_type.go` | `MiSortType` 列挙型 — Mi タスクのソート順 |
| `week_of_days.go` | `WeekOfDays` 列挙型 — 曜日フィルタ |
| `find_query_test.go` | JSON シリアライズ・デシリアライズテスト（100+ ケース） |

## FindQuery 構造体

62 フィールドを持つ検索条件構造体。フィルタグループは `UseXxx: true` フラグで有効化する設計。

### フィルタグループ

| グループ | 有効化フラグ | 関連フィールド |
|---------|-------------|---------------|
| キーワード検索 | `UseWords` | `Words`, `NotWords`, `WordsAnd` |
| タグフィルタ | `UseTags` | `Tags`, `HideTags`, `TagsAnd` |
| リポジトリフィルタ | `UseReps` | `Reps` |
| リポジトリ種別フィルタ | `UseRepTypes` | `RepTypes` |
| ID フィルタ | `UseIDs` / `UseIncludeID` | `IDs` |
| カレンダー | `UseCalendar` | `CalendarStartDate`, `CalendarEndDate` |
| 地図 | `UseMap` | `MapLatitude`, `MapLongitude`, `MapRadius` |
| TimeIs 検索 | `UseTimeIs` | `TimeIsWords`, `TimeIsNotWords`, `TimeIsWordsAnd` |
| TimeIs タグ | `UseTimeIsTags` | `TimeIsTags`, `HideTimeIsTags`, `TimeIsTagsAnd` |
| 時間帯 | `UsePeriodOfTime` | `PeriodOfTimeStartTimeSecond`, `PeriodOfTimeEndTimeSecond`, `PeriodOfTimeWeekOfDays` |
| 更新時刻 | `UseUpdateTime` | `UpdateTime` |
| Mi タスク | `ForMi` / `UseMiBoardName` | `MiBoardName`, `MiCheckState`, `MiSortType`, `IncludeCreateMi` 等 |
| Plaing | `UsePlaing` | `PlaingTime` |

### 共通フラグ

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `OnlyLatestData` | `bool` | 最新データのみ取得（履歴を含めない） |
| `IncludeDeletedData` | `bool` | 論理削除済みデータを含める |
| `IsImageOnly` | `bool` | 画像ファイルのみ取得 |
| `UpdateCache` | `bool` | キャッシュを更新してから検索 |

## 列挙型

### MiCheckState

| 値 | 定数名 | 説明 |
|----|--------|------|
| `"all"` | `All` | 全状態 |
| `"checked"` | `Checked` | チェック済み |
| `"uncheck"` | `UncCheck` | 未チェック |

### MiSortType

| 値 | 定数名 | 説明 |
|----|--------|------|
| `"create_time"` | `CreateTime` | 作成日時順 |
| `"estimate_start_time"` | `EstimateStartTime` | 見積開始日時順 |
| `"estimate_end_time"` | `EstimateEndTime` | 見積終了日時順 |
| `"limit_time"` | `LimitTime` | 期限日時順 |

### WeekOfDays

| 値 | 定数名 |
|----|--------|
| `0` | `SunDay` |
| `1` | `MonDay` |
| `2` | `TuesDay` |
| `3` | `WednesDay` |
| `4` | `ThrusDay` |
| `5` | `FriDay` |
| `6` | `SaturDay` |

## 関連ドキュメント

- [api/README.md](../README.md) — API 共通基盤（FindFilter がこの型を使用）
- [usecase/README.md](../../usecase/README.md) — ユースケース層（`GetKyous` で使用）
