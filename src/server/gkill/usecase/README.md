# usecase - ユースケース層

## 概要

HTTP 非依存のビジネスロジック層。API ハンドラ（`gkill_server_api/`）から抽出した共通処理を `UsecaseContext` 構造体のメソッドとして集約し、
HTTP コンテキストに依存しない再利用可能なロジックを提供する。

API ハンドラ、MCP サーバ、Wear OS 連携など複数のエントリポイントから共通して利用される。

## ディレクトリ構造

```
usecase/
├── usecase.go           # UsecaseContext 構造体・コンストラクタ
├── kyou.go              # Kyou 検索・履歴取得
├── kmemo.go             # Kmemo 追加・更新・履歴取得
├── timeis.go            # TimeIs 追加・更新・履歴取得
├── lantana.go           # Lantana 追加・更新・履歴取得
├── kc.go                # KC 追加・更新・履歴取得
├── nlog.go              # Nlog 追加・更新・履歴取得
├── urlog.go             # URLog 追加・更新・履歴取得
├── mi.go                # Mi 追加・更新・履歴取得・ボード一覧
├── tag.go               # Tag 追加・更新・対象ID検索・履歴取得
├── text.go              # Text 追加・更新・対象ID検索・履歴取得
├── notification.go      # Notification 追加・更新・対象ID検索・履歴取得
├── idf_kyou.go          # IDFKyou 更新・履歴取得
├── rekyou.go            # ReKyou 追加・更新・履歴取得
├── git_commit_log.go    # GitCommitLog 取得
└── rep_names.go         # 全タグ名・全リポジトリ名取得
```

## 設計思想

### UsecaseContext

`UsecaseContext` は本層の依存関係をまとめる構造体。`GkillDAOManager` と `FindFilter` を保持する。

```go
type UsecaseContext struct {
    DAOManager *dao.GkillDAOManager
    FindFilter *api.FindFilter
}
```

### HTTP 非依存

全メソッドは `context.Context`、`*reps.GkillRepositories`、ユーザ情報（userID, device, localeName）、エンティティ、トランザクションID を引数に取る。
`net/http` パッケージへの依存はなく、HTTP 層からもバッチ処理からも呼び出し可能。

### 一貫したエラーハンドリング

全メソッドは `([]*message.GkillError, error)` または `(T, []*message.GkillError, error)` を返す。
`GkillError` は i18n 対応のローカライズ済みメッセージを含む。

### キャッシュ対応

書き込み系メソッドはメインリポジトリへの書き込み後、`gkill_options.CacheXxxReps` フラグに応じてキャッシュリポジトリへも同期する。
また、`LatestDataRepositoryAddresses` を更新して最新データの所在を記録する。

### トランザクション対応

書き込み系メソッドは `txID *string` パラメータを受け取る。
- `txID == nil`: 通常書き込み（WriteXxxRep に直接追加）
- `txID != nil`: トランザクション書き込み（TempReps に追加、後で CommitTX でコミット）

## ファイル一覧（16ファイル）

| ファイル | 説明 | 関数数 |
|---------|------|--------|
| `usecase.go` | `UsecaseContext` 構造体定義、`NewUsecaseContext()` コンストラクタ | 1 |
| `kyou.go` | Kyou 一覧検索（FindFilter 経由）、履歴取得 | 2 |
| `kmemo.go` | Kmemo（テキストメモ）の追加・更新・履歴取得 | 3 |
| `timeis.go` | TimeIs（タイムスタンプ）の追加・更新・履歴取得 | 3 |
| `lantana.go` | Lantana（気分値 0-10）の追加・更新・履歴取得 | 3 |
| `kc.go` | KC（数値記録）の追加・更新・履歴取得 | 3 |
| `nlog.go` | Nlog（支出記録）の追加・更新・履歴取得 | 3 |
| `urlog.go` | URLog（ブックマーク）の追加・更新・履歴取得 | 3 |
| `mi.go` | Mi（タスク）の追加・更新・履歴取得・ボード一覧取得 | 4 |
| `tag.go` | Tag の追加・更新・対象 ID 検索・履歴取得 | 4 |
| `text.go` | Text の追加・更新・対象 ID 検索・履歴取得 | 4 |
| `notification.go` | Notification の追加・更新・対象 ID 検索・履歴取得 | 4 |
| `idf_kyou.go` | IDFKyou（ファイル）の更新・履歴取得 | 2 |
| `rekyou.go` | ReKyou（リポスト）の追加・更新・履歴取得 | 3 |
| `git_commit_log.go` | GitCommitLog の取得 | 1 |
| `rep_names.go` | 全タグ名一覧・全リポジトリ名一覧の取得 | 2 |

**合計: 45 関数**（コンストラクタ 1 + Add 系 10 + Update 系 11 + Get 系 23）

## エクスポート関数一覧

### 構造体・コンストラクタ

| 関数 | ファイル | 説明 |
|------|---------|------|
| `NewUsecaseContext` | `usecase.go` | UsecaseContext を生成 |

### データ追加系（10関数）

| 関数 | ファイル | 戻り値の特徴 |
|------|---------|-------------|
| `AddKmemo` | `kmemo.go` | エラーのみ |
| `AddTimeIs` | `timeis.go` | エラーのみ |
| `AddLantana` | `lantana.go` | エラーのみ |
| `AddKC` | `kc.go` | エラーのみ |
| `AddNlog` | `nlog.go` | エラーのみ |
| `AddURLog` | `urlog.go` | エラーのみ |
| `AddMi` | `mi.go` | エラーのみ |
| `AddReKyou` | `rekyou.go` | エラーのみ |
| `AddTag` | `tag.go` | `*reps.Tag` を返す |
| `AddText` | `text.go` | `*reps.Text` を返す |
| `AddNotification` | `notification.go` | `*reps.Notification` を返す |

### データ更新系（11関数）

| 関数 | ファイル | 戻り値の特徴 |
|------|---------|-------------|
| `UpdateKmemo` | `kmemo.go` | エラーのみ |
| `UpdateTimeIs` | `timeis.go` | エラーのみ |
| `UpdateLantana` | `lantana.go` | エラーのみ |
| `UpdateKC` | `kc.go` | エラーのみ |
| `UpdateNlog` | `nlog.go` | エラーのみ |
| `UpdateURLog` | `urlog.go` | エラーのみ |
| `UpdateMi` | `mi.go` | エラーのみ |
| `UpdateReKyou` | `rekyou.go` | エラーのみ |
| `UpdateIDFKyou` | `idf_kyou.go` | エラーのみ |
| `UpdateTag` | `tag.go` | `*reps.Tag` を返す |
| `UpdateText` | `text.go` | `*reps.Text` を返す |
| `UpdateNotification` | `notification.go` | `*reps.Notification` を返す |

### データ取得系（23関数）

| 関数 | ファイル | 説明 |
|------|---------|------|
| `GetKyous` | `kyou.go` | FindFilter 経由で Kyou 一覧検索 |
| `GetKyouHistories` | `kyou.go` | Kyou 履歴取得（UpdateTime 指定可） |
| `GetKmemoHistories` | `kmemo.go` | Kmemo 履歴取得 |
| `GetTimeIsHistories` | `timeis.go` | TimeIs 履歴取得 |
| `GetLantanaHistories` | `lantana.go` | Lantana 履歴取得 |
| `GetKCHistories` | `kc.go` | KC 履歴取得 |
| `GetNlogHistories` | `nlog.go` | Nlog 履歴取得 |
| `GetURLogHistories` | `urlog.go` | URLog 履歴取得 |
| `GetMiHistories` | `mi.go` | Mi 履歴取得 |
| `GetMiBoardList` | `mi.go` | Mi ボード一覧取得 |
| `GetReKyouHistories` | `rekyou.go` | ReKyou 履歴取得 |
| `GetIDFKyouHistories` | `idf_kyou.go` | IDFKyou 履歴取得 |
| `GetGitCommitLog` | `git_commit_log.go` | GitCommitLog 取得 |
| `GetTagsByTargetID` | `tag.go` | 対象 ID に紐づく Tag 一覧 |
| `GetTagHistoriesByTagID` | `tag.go` | Tag 変更履歴 |
| `GetTextsByTargetID` | `text.go` | 対象 ID に紐づく Text 一覧 |
| `GetTextHistoriesByTextID` | `text.go` | Text 変更履歴 |
| `GetNotificationsByTargetID` | `notification.go` | 対象 ID に紐づく Notification 一覧 |
| `GetNotificationHistoriesByNotificationID` | `notification.go` | Notification 変更履歴 |
| `GetAllTagNames` | `rep_names.go` | 全タグ名一覧 |
| `GetAllRepNames` | `rep_names.go` | 全リポジトリ名一覧 |

## アーキテクチャ上の位置

```
HTTP API ハンドラ (api/gkill_server_api/)
MCP サーバ (mcp/)
Wear OS 連携 (wear_os/)
         ↓
   ユースケース層 (gkill/usecase/)  ← HTTP 非依存ビジネスロジック
         ↓
  DAO 層 (gkill/dao/reps/)          ← SQLite3 リポジトリ実装
         ↓
    SQLite3 データベース
```

## 典型的な呼び出しパターン

### ハンドラからの呼び出し例（AddKmemo）

```
1. ハンドラが JSON リクエストをデコード → AddKmemoRequest
2. ハンドラが usecaseContext.AddKmemo(ctx, repositories, userID, device, localeName, kmemo, txID) を呼び出し
3. ユースケース層が重複チェック → WriteKmemoRep に書き込み → キャッシュ同期 → LatestDataRepositoryAddress 更新
4. ハンドラが結果を JSON レスポンスとして返却
```

### Add 系メソッドの共通処理フロー

1. 対象 ID で既存データの存在チェック（重複防止）
2. `txID == nil` の場合: `WriteXxxRep` に追加 → キャッシュフラグに応じてキャッシュ Rep にも追加
3. `txID != nil` の場合: `TempReps.XxxTempRep` に追加
4. `WriteXxxRep.GetRepName()` でリポジトリ名を取得
5. `LatestDataRepositoryAddresses` を更新

## 開発ガイドライン

### 新しいユースケースの追加方法

1. 対応するデータ型のファイル（例: `xxx.go`）に `UsecaseContext` のメソッドとして実装
2. 引数: `ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, ...`
3. 戻り値: `([]*message.GkillError, error)` または `(T, []*message.GkillError, error)`
4. エラーメッセージは `api.GetLocalizer(localeName).MustLocalizeMessage()` で i18n 対応
5. ログは `slog.Log(ctx, gkill_log.Debug, ...)` で出力

### 注意事項

- `net/http` パッケージをインポートしないこと（HTTP 非依存の維持）
- 書き込み系は必ずキャッシュ同期と `LatestDataRepositoryAddresses` 更新を行うこと
- トランザクション対応（`txID` パラメータ）を忘れないこと

## 関連ドキュメント

- [api/gkill_server_api/](../api/gkill_server_api/) — ユースケースを呼び出す HTTP ハンドラ層
- [dao/reps/README.md](../dao/reps/README.md) — ユースケースから利用するリポジトリ層
- [ABOUT_TEST.md](ABOUT_TEST.md) — テスト仕様
- [documents/reverse/usecase.md](../../../../documents/reverse/usecase.md) — ユースケース一覧（機能仕様）
