# find - 検索クエリ型定義

## 概要

Kyou 検索に使用するクエリ構造体と列挙型を定義する。
`FindQuery` は API のデータ取得系エンドポイントで検索条件を指定するために使用される。

## ファイル一覧（10ファイル）

| ファイル | 説明 |
|---------|------|
| `find_query.go` | `FindQuery` 構造体 — 39 フィールドの検索条件とゲート判定ヘルパー |
| `find_query_legacy_json.go` | 旧形式（`use_*` フラグ入り）JSON を新形式へ書き換える移行ウォーカー |
| `period_of_time.go` | 時間帯フィルタの秒値正規化（`NormalizeSecondOfDay` で 0..86399 の秒オブデイへ、`SecondOfDayToHHMMSS` 等） |
| `rep_types.go` | `KyouRepTypes` — `RepTypes` が受理する正準値の一覧（`selectMatchRepsFromQuery` との集合一致をテストで固定） |
| `mi_check_state.go` | `MiCheckState` 列挙型 — Mi タスクのチェック状態フィルタ |
| `mi_sort_type.go` | `MiSortType` 列挙型 — Mi タスクのソート順 |
| `week_of_days.go` | `WeekOfDays` 列挙型 — 曜日フィルタ |
| `find_query_test.go` | JSON シリアライズ・デシリアライズテスト |
| `find_query_legacy_json_test.go` | 旧形式 JSON 移行（`MigrateLegacyFindQueryJSON`）のテスト |
| `period_of_time_test.go` | 時間帯フィルタの秒値正規化のテスト |

## FindQuery 構造体

39 フィールドの検索条件構造体。フィルタグループの有効/無効は **値の null 判定** で表す:

- `nil`（JSON では `null` またはキー欠落）= フィルタ未使用
- 非nilの空スライス（JSON では `[]`）= フィルタ有効だが空指定
  （`Tags`/`Reps`/`IDs`/`RepTypes`/`TimeIsTags` は0件、`TimeIsWords` は「任意のTimeIsに覆われたKyouのみ」）

### フィルタグループとゲート判定

| グループ | ゲート判定 | 関連フィールド | `[]`（非nil空）の意味 |
|---------|-----------|---------------|---------------------|
| キーワード検索 | `HasWordFilter()` = `Words`≠nil ∨ `NotWords`≠nil | `Words`, `NotWords`, `WordsAnd` | 両方空→SQL条件なし=素通し |
| タグフィルタ | `Tags != nil` | `Tags`, `HideTags`, `TagsAnd` | 0件 |
| リポジトリフィルタ | `Reps != nil` | `Reps` | 0件 |
| リポジトリ種別フィルタ | `RepTypes != nil` | `RepTypes` | タイプ候補0件 |
| ID フィルタ | `IDs != nil` | `IDs` | 0件（SQL `0 = 1`） |
| カレンダー | `HasCalendarFilter()` = start≠nil ∨ end≠nil | `CalendarStartDate`, `CalendarEndDate`（`*time.Time`） | — |
| 地図 | `HasMapFilter()` = 3値すべて非nil | `MapLatitude`, `MapLongitude`, `MapRadius`（`*float64`） | — |
| TimeIs 検索 | `HasTimeIsFilter()` = `TimeIsWords`≠nil ∨ `TimeIsNotWords`≠nil | `TimeIsWords`, `TimeIsNotWords`, `TimeIsWordsAnd` | `TimeIsWords=[]`→「任意のTimeIsに覆われたKyou」 |
| TimeIs タグ | `HasTimeIsFilter() && TimeIsTags != nil` | `TimeIsTags`, `TimeIsTagsAnd` | 0件 |
| 時間帯 | `HasPeriodOfTimeFilter()` = 3値いずれか非nil | `PeriodOfTimeStartTimeSecond`, `PeriodOfTimeEndTimeSecond`（`*int64`）, `PeriodOfTimeWeekOfDays` | WeekOfDays: nil=曜日制限なし / `[]`=0件 / 全7曜日=制限なし |
| 更新時刻 | `UpdateTime != nil` | `UpdateTime`（`*time.Time`） | — |
| Mi タスク | `ForMi` / `MiBoardName != nil` | `MiBoardName`（`*string`）, `MiCheckState`, `MiSortType`, `IncludeCreateMi` 等 | — |
| Playing | `PlayingTime != nil` | `PlayingTime`（`*time.Time`） | — |

`MiCheckState` / `MiSortType` は常時有効（ゲートなし。サーバが値を無条件に読む）。

### 共通フラグ

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `OnlyLatestData` | `bool` | 最新データのみ取得（履歴を含めない） |
| `IncludeDeletedData` | `bool` | 論理削除済みデータを含める。Kyou検索（`FindFilter`）とTag/Text/Notificationの集約が参照する。既定 false で従来どおり除外 |
| `IsImageOnly` | `bool` | 画像ファイルのみ取得 |
| `UpdateCache` | `bool` | キャッシュを更新してから検索 |

### 削除済みデータ

削除済みを含めたいときは `IncludeDeletedData` を使う。
かつて `IsDeleted` / `HideTimeIsTags` という「定義だけあって参照されない」フィールドがあったが、
送っているクライアントが実在しなかったため 2026-08-24 に削除した。
MCP でも語彙から外してあるので、送ると未知キーとしてエラーになる（黙って無視されるより気付ける）。

### 旧形式からの移行

かつては `UseXxx: true` フラグでフィルタグループを有効化する設計だった。
`MigrateLegacyFindQueryJSON`（`find_query_legacy_json.go`）が旧形式 JSON を新形式へ書き換える
（`use_X=false`→値をnull化、`use_X=true`→値維持・null/欠落配列は `[]` を物質化、最後に `use_*` キー除去。冪等）。
`share_kyou_info.db` の共有クエリはDAOの起動時スキーマ移行（1.0.0→1.1.0）がこれを適用する。

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
| `4` | `ThursDay` |
| `5` | `FriDay` |
| `6` | `SaturDay` |

## 関連ドキュメント

- [api/README.md](../README.md) — API 共通基盤（FindFilter がこの型を使用）
- [usecase/README.md](../../usecase/README.md) — ユースケース層（`GetKyous` で使用）
