# api - GkillAPI クライアント

## 概要

gkill サーバの全 HTTP API を呼び出す TypeScript クライアント層。
`GkillAPI` シングルトンクラス（約171KB）が全エンドポイントのラッパーメソッドを提供する。
サーバ側の対応実装: `src/server/gkill/api/`

## ディレクトリ構造

```
api/
├── gkill-api.ts                          # メイン API クラス（シングルトン、~171KB）
├── gkill-api-request.ts                  # 基底リクエスト型
├── gkill-api-response.ts                 # 基底レスポンス型
├── gkill-error.ts                        # エラー型
├── gkill-message.ts                      # メッセージ型
├── drop-type-foldable-struct.ts          # ドロップ型折り畳み構造体
├── file-data.ts                          # ファイルデータ型
├── generate-get-plaing-timeis-kyous-query.ts  # Plaing TimeIs クエリ生成
├── upload-file-conflict-behavior-type.ts # アップロード競合動作型
├── find_query/                           # 検索クエリ型
├── message/                              # エラー/メッセージ型
└── req_res/                              # 全 Request/Response 型
```

## ルートファイル（9ファイル）

| ファイル | 役割 |
|---------|------|
| `gkill-api.ts` | **GkillAPI シングルトン**。全エンドポイントの呼び出しメソッドを集約（~171KB） |
| `gkill-api-request.ts` | 基底リクエスト型（session_id 等の共通フィールド） |
| `gkill-api-response.ts` | 基底レスポンス型（errors, messages 等の共通フィールド） |
| `gkill-error.ts` | `GkillError` 型定義 |
| `gkill-message.ts` | `GkillMessage` 型定義 |
| `drop-type-foldable-struct.ts` | ドラッグ&ドロップ時のデータ型折り畳み構造体 |
| `file-data.ts` | `FileData` 型（アップロードファイル情報） |
| `generate-get-plaing-timeis-kyous-query.ts` | 稼働中 TimeIs の Kyou 取得クエリ生成ユーティリティ |
| `upload-file-conflict-behavior-type.ts` | ファイルアップロード競合時の動作 enum |

## サブディレクトリ

### `find_query/`（5ファイル）— 検索クエリ型

サーバ側 `api/find/` と対応。

| ファイル | 説明 |
|---------|------|
| `find-kyou-query.ts` | `FindKyouQuery` — 検索条件（キーワード、日付範囲、タグ等） |
| `mi-check-state.ts` | Mi チェック状態 enum |
| `mi-sort-type.ts` | Mi ソート順 enum |
| `mood-operator.ts` | 気分値比較演算子 enum |
| `week-of-days.ts` | 曜日フィルタ enum |

### `message/`（2ファイル）— エラー/メッセージ型

| ファイル | 説明 |
|---------|------|
| `gkill_error.ts` | エラーコード定義 |
| `gkill_message.ts` | メッセージコード定義 |

### `req_res/`（154ファイル）— Request/Response 型

サーバ側 `api/req_res/` と1対1で対応する TypeScript 型定義。
各エンドポイントに `*-request.ts` + `*-response.ts` のペアが存在。

#### 認証系

- `login-request.ts` / `login-response.ts`
- `logout-request.ts` / `logout-response.ts`
- `reset-password-request.ts` / `reset-password-response.ts`
- `set-new-password-request.ts` / `set-new-password-response.ts`
- `add-account-request.ts` / `add-account-response.ts`

#### データ追加系（add-*）

- `add-kc-request.ts` / `add-kc-response.ts`
- `add-kmemo-request.ts` / `add-kmemo-response.ts`
- `add-kyou-info-request.ts` / `add-kyou-info-response.ts`
- `add-lantana-request.ts` / `add-lantana-response.ts`
- `add-mi-request.ts` / `add-mi-response.ts`
- `add-nlog-request.ts` / `add-nlog-response.ts`
- `add-notification-request.ts` / `add-notification-response.ts`
- `add-re-kyou-request.ts` / `add-re-kyou-response.ts`
- `add-share-kyou-list-info-request.ts` / `add-share-kyou-list-info-response.ts`
- `add-tag-request.ts` / `add-tag-response.ts`
- `add-text-request.ts` / `add-text-response.ts`
- `add-time-is-request.ts` / `add-time-is-response.ts`
- `add-ur-log-request.ts` / `add-ur-log-response.ts`

#### データ更新系（update-*）

- `update-idf-kyou-request.ts` / `update-idf-kyou-response.ts`
- `update-kc-request.ts` / `update-kc-response.ts`
- `update-kmemo-request.ts` / `update-kmemo-response.ts`
- `update-lantana-request.ts` / `update-lantana-response.ts`
- `update-mi-request.ts` / `update-mi-response.ts`
- `update-nlog-request.ts` / `update-nlog-response.ts`
- `update-notification-request.ts` / `update-notification-response.ts`
- `update-re-kyou-request.ts` / `update-re-kyou-response.ts`
- `update-tag-request.ts` / `update-tag-response.ts`
- `update-text-request.ts` / `update-text-response.ts`
- `update-time-is-request.ts` / `update-time-is-response.ts`
- `update-ur-log-request.ts` / `update-ur-log-response.ts`
- `update-account-status-request.ts` / `update-account-status-response.ts`
- `update-application-config-request.ts` / `update-application-config-response.ts`
- `update-server-config-request.ts` / `update-server-config-response.ts`
- `update-share-kyou-list-info-request.ts` / `update-share-kyou-list-info-response.ts`
- `update-user-reps-request.ts` / `update-user-reps-response.ts`
- `update-cache-request.ts` / `update-cache-response.ts`

#### データ取得系（get-*）

- `get-kyou-request.ts` / `get-kyou-response.ts`
- `get-kyous-request.ts` / `get-kyous-response.ts`
- `get-kc-request.ts` / `get-kc-response.ts`
- `get-kmemo-request.ts` / `get-kmemo-response.ts`
- `get-lantana-request.ts` / `get-lantana-response.ts`
- `get-mi-request.ts` / `get-mi-response.ts`
- `get-nlog-request.ts` / `get-nlog-response.ts`
- `get-re-kyou-request.ts` / `get-re-kyou-response.ts`
- `get-time-is-request.ts` / `get-time-is-response.ts`
- `get-ur-log-request.ts` / `get-ur-log-response.ts`
- `get-idf-kyou-request.ts` / `get-idf-kyou-response.ts`
- `get-git-commit-log-request.ts` / `get-git-commit-log-response.ts`
- `get-gps-log-request.ts` / `get-gps-log-response.ts`
- `get-mi-board-request.ts` / `get-mi-board-response.ts`
- `get-all-tag-names-request.ts` / `get-all-tag-names-response.ts`
- `get-all-rep-names-request.ts` / `get-all-rep-names-response.ts`
- `get-tags-by-target-id-request.ts` / `get-tags-by-target-id-response.ts`
- `get-tag-history-by-tag-id-request.ts` / `get-tag-history-by-tag-id-response.ts`
- `get-texts-by-target-id-request.ts` / `get-texts-by-target-id-response.ts`
- `get-text-history-by-text-id-request.ts` / `get-text-history-by-text-id-response.ts`
- `get-notifications-by-target-id-request.ts` / `get-notifications-by-target-id-response.ts`
- `get-notification-history-by-notification-id-request.ts` / `get-notification-history-by-notification-id-response.ts`
- `get-application-config-request.ts` / `get-application-config-response.ts`
- `get-server-configs-request.ts` / `get-server-configs-response.ts`
- `get-repositories-request.ts` / `get-repositories-response.ts`
- `get-shared-kyous-request.ts` / `get-shared-kyous-response.ts`
- `get-share-kyou-list-infos-request.ts` / `get-share-kyou-list-infos-response.ts`
- `get-gkill-notification-public-key-request.ts` / `get-gkill-notification-public-key-response.ts`
- `get-updated-datas-by-time-request.ts` / `get-updated-datas-by-time-response.ts`
- `get-kyous-mcp-request.ts` / `get-kyous-mcp-response.ts`

#### その他操作系

- `commit-tx-request.ts` / `commit-tx-response.ts`
- `discard-tx-request.ts` / `discard-tx-response.ts`
- `delete-share-kyou-list-info-request.ts` / `delete-share-kyou-list-infos-response.ts`
- `generate-tls-file-request.ts` / `generate-tls-file-response.ts`
- `open-directory-request.ts` / `open-directory-response.ts`
- `open-file-request.ts` / `open-file-response.ts`
- `register-gkill-notification-request.ts` / `register-gkill-notification-response.ts`
- `reload-repositories-request.ts` / `reload-repositories-response.ts`
- `submit-kftl-text-request.ts` / `submit-kftl-text-response.ts`
- `upload-files-request.ts` / `upload-files-response.ts`
- `upload-gps-log-files-request.ts` / `upload-gps-log-files-response.ts`
- `urlog-bookmarklet-request.ts`

## 開発ガイドライン

### 新しいエンドポイントを追加する場合

1. `req_res/` に `xxx-request.ts` と `xxx-response.ts` を追加
2. `gkill-api.ts` の `GkillAPI` クラスに呼び出しメソッドを追加
3. サーバ側 `api/req_res/` にも対応する Go 構造体を追加

### 命名規則

- ファイル名: kebab-case（例: `add-kmemo-request.ts`）
- サーバ側の snake_case ファイル名と対応（`add_kmemo_request.go` ↔ `add-kmemo-request.ts`）
