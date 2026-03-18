# classes - TypeScript ユーティリティクラス

## 概要

フロントエンドのビジネスロジック、Vue Composable 関数、データモデル、API クライアントを集約するディレクトリ。
Vue コンポーネント（`pages/`）から使用されるロジック層。

## ディレクトリ構造

```
classes/
├── (ルートファイル ~180個)       # use-*.ts Composable + ユーティリティ
├── api/                        # GkillAPI クライアント → api/README.md
├── datas/                      # データモデル → datas/README.md
├── dnote/                      # Dynamic Note システム → dnote/README.md
├── kftl/                       # KFTL パーサ → kftl/README.md
├── dto/                        # DTO（3ファイル）
└── lantana/                    # Lantana ユーティリティ（3ファイル）
```

## ルートファイル: Composable 関数群

Vue 3 の Composable パターン（`use-*.ts`）でコンポーネントのロジックを分離。
各 `use-*.ts` ファイルは対応する Vue コンポーネントのロジック（リアクティブ状態、メソッド、ライフサイクル）を提供する。

### データ追加系 Composable

| ファイル | 対応コンポーネント |
|---------|------------------|
| `use-add-kc-view.ts` / `use-add-kc-dialog.ts` | KC 追加 |
| `use-add-lantana-view.ts` / `use-add-lantana-dialog.ts` | Lantana 追加 |
| `use-add-mi-view.ts` / `use-add-mi-dialog.ts` | Mi 追加 |
| `use-add-nlog-view.ts` / `use-add-nlog-dialog.ts` | Nlog 追加 |
| `use-add-timeis-view.ts` / `use-add-timeis-dialog.ts` | TimeIs 追加 |
| `use-add-urlog-view.ts` / `use-add-urlog-dialog.ts` | URLog 追加 |
| `use-add-tag-view.ts` / `use-add-tag-dialog.ts` | Tag 追加 |
| `use-add-text-view.ts` / `use-add-text-dialog.ts` | Text 追加 |
| `use-add-notification-view.ts` / `use-add-notification-dialog.ts` | Notification 追加 |
| `use-add-notification-for-add-mi-view.ts` | Mi 追加時の Notification |
| `use-add-rep-view.ts` | リポジトリ追加 |
| `use-add-ryuu-item-view.ts` | Ryuu アイテム追加 |

### データ編集系 Composable

| ファイル | 対応コンポーネント |
|---------|------------------|
| `use-edit-kc-view.ts` / `use-edit-kc-dialog.ts` | KC 編集 |
| `use-edit-kmemo-view.ts` / `use-edit-kmemo-dialog.ts` | Kmemo 編集 |
| `use-edit-lantana-view.ts` / `use-edit-lantana-dialog.ts` | Lantana 編集 |
| `use-edit-mi-view.ts` / `use-edit-mi-dialog.ts` | Mi 編集 |
| `use-edit-nlog-view.ts` / `use-edit-nlog-dialog.ts` | Nlog 編集 |
| `use-edit-time-is-view.ts` / `use-edit-time-is-dialog.ts` | TimeIs 編集 |
| `use-edit-ur-log-view.ts` / `use-edit-ur-log-dialog.ts` | URLog 編集 |
| `use-edit-idf-kyou-view.ts` / `use-edit-idf-kyou-dialog.ts` | IDFKyou 編集 |
| `use-edit-re-kyou-view.ts` / `use-edit-re-kyou-dialog.ts` | ReKyou 編集 |
| `use-edit-tag-view.ts` / `use-edit-tag-dialog.ts` | Tag 編集 |
| `use-edit-text-view.ts` / `use-edit-text-dialog.ts` | Text 編集 |
| `use-edit-notification-view.ts` / `use-edit-notification-dialog.ts` | Notification 編集 |
| `use-edit-ryuu-item-view.ts` / `use-edit-ryuu-dialog.ts` | Ryuu 編集 |

### 削除確認系 Composable

| ファイル | 対応コンポーネント |
|---------|------------------|
| `use-confirm-delete-kyou-view.ts` | Kyou 削除確認 |
| `use-confirm-delete-idf-kyou-dialog.ts` | IDFKyou 削除確認 |
| `use-confirm-delete-tag-view.ts` / `use-confirm-delete-tag-dialog.ts` | Tag 削除確認 |
| `use-confirm-delete-text-view.ts` / `use-confirm-delete-text-dialog.ts` | Text 削除確認 |
| `use-confirm-delete-notification-view.ts` / `use-confirm-delete-notification-dialog.ts` | Notification 削除確認 |
| `use-confirm-delete-share-task-list-view.ts` | 共有タスクリスト削除確認 |
| `use-confirm-re-kyou-view.ts` / `use-confirm-re-kyou-dialog.ts` | ReKyou 確認 |

### 表示系 Composable

| ファイル | 対応コンポーネント |
|---------|------------------|
| `use-kyou-view.ts` / `use-kyou-list-view.ts` | Kyou 表示 / 一覧 |
| `use-kc-view.ts` / `use-kmemo-view.ts` | KC / Kmemo 表示 |
| `use-lantana-view.ts` / `use-mi-view.ts` | Lantana / Mi 表示 |
| `use-nlog-view.ts` / `use-ur-log-view.ts` | Nlog / URLog 表示 |
| `use-time-is-view.ts` / `use-idf-kyou-view.ts` | TimeIs / IDFKyou 表示 |
| `use-re-kyou-view.ts` / `use-git-commit-log-view.ts` | ReKyou / GitCommitLog 表示 |
| `use-tag-view.ts` / `use-text-view.ts` / `use-notification-view.ts` | Tag / Text / Notification 表示 |
| `use-mi-kyou-view.ts` | Mi の Kyou 表示 |

### ページ系 Composable

| ファイル | 対応ページ |
|---------|-----------|
| `use-login-page.ts` / `use-login-view.ts` | ログインページ |
| `use-kftl-page.ts` / `use-kftl-view.ts` | KFTL エディタ |
| `use-mi-page.ts` | Mi ページ |
| `use-kyou-page.ts` | Kyou ページ |
| `use-rykv-page.ts` / `use-rykv-view.ts` | Rykv ページ |
| `use-mkfl-page.ts` / `use-mkfl-view.ts` | Mkfl ページ |
| `use-plaing-timeis-page.ts` / `use-plaing-timeis-view.ts` | 稼働中 TimeIs |
| `use-saihate-page.ts` | Saihate ページ |
| `use-shared-page.ts` / `use-shared-mi-page.ts` | 共有ページ |
| `use-set-new-password-page.ts` / `use-set-new-password-view.ts` | パスワード設定 |
| `use-regist-first-account-page.ts` / `use-regist-first-account-view.ts` | 初回アカウント登録 |

### DNote 系 Composable

| ファイル | 説明 |
|---------|------|
| `use-dnote-view.ts` | DNote メインビュー |
| `use-dnote-item-view.ts` / `use-dnote-item-list-view.ts` | DNote アイテム |
| `use-dnote-item-table-view.ts` / `use-dnote-list-table-view.ts` | DNote テーブル |
| `use-dnote-list-view.ts` | DNote リスト |
| `use-add-dnote-item-view.ts` / `use-add-dnote-list-view.ts` | DNote 追加 |
| `use-edit-dnote-item-view.ts` / `use-edit-dnote-list-view.ts` | DNote 編集 |
| `use-edit-dnote-card.ts` / `use-edit-dnote-dialog.ts` | DNote カード/ダイアログ |
| `use-edit-dnote-predicate-group.ts` | DNote 述語グループ |
| `use-aggregated-list-item.ts` | 集計リストアイテム |

### 設定系 Composable

| ファイル | 説明 |
|---------|------|
| `use-application-config-view.ts` | アプリケーション設定 |
| `use-server-config-view.ts` / `use-server-config-dialog.ts` | サーバ設定 |
| `use-edit-device-struct-*.ts` | デバイス構造編集 |
| `use-edit-kftl-template-struct-*.ts` | KFTL テンプレート編集 |
| `use-edit-rep-struct-*.ts` / `use-edit-rep-type-struct-*.ts` | リポジトリ構造編集 |
| `use-edit-tag-struct-*.ts` | タグ構造編集 |
| `use-manage-account-view.ts` | アカウント管理 |

### コンテキストメニュー系 Composable

| ファイル | 対象 |
|---------|------|
| `use-kc-context-menu.ts` | KC |
| `use-kmemo-context-menu.ts` | Kmemo |
| `use-lantana-context-menu.ts` | Lantana |
| `use-mi-context-menu.ts` | Mi |
| `use-nlog-context-menu.ts` | Nlog |
| `use-ur-log-context-menu.ts` | URLog |
| `use-time-is-context-menu.ts` | TimeIs |
| `use-idf-kyou-context-menu.ts` | IDFKyou |
| `use-re-kyou-context-menu.ts` | ReKyou |
| `use-git-commit-log-context-menu.ts` | GitCommitLog |
| `use-attached-tag-context-menu.ts` | 添付タグ |
| `use-attached-text-context-menu.ts` | 添付テキスト |
| `use-attached-notification-context-menu.ts` | 添付通知 |

### クエリ系 Composable

| ファイル | 説明 |
|---------|------|
| `use-find-query-editor-view.ts` / `use-find-query-editor-dialog.ts` | 検索クエリエディタ |
| `use-keyword-query.ts` | キーワードクエリ |
| `use-period-of-time-query.ts` | 期間クエリ |
| `use-tag-query.ts` | タグクエリ |
| `use-rep-query.ts` | リポジトリクエリ |
| `use-time-is-query.ts` | TimeIs クエリ |
| `use-calendar-query.ts` / `use-map-query.ts` | カレンダー/地図クエリ |
| `use-mi-board-query.ts` / `use-mi-sort-type-query.ts` | Mi ボード/ソートクエリ |
| `use-mi-extruct-check-state-query.ts` | Mi チェック状態抽出 |

### ユーティリティファイル

| ファイル | 説明 |
|---------|------|
| `deep-equals.ts` | ディープイコール比較 |
| `delete-gkill-cache.ts` | gkill キャッシュ削除 |
| `format-date-time.ts` | 日時フォーマット |
| `long-press.ts` | ロングプレスディレクティブ |
| `looks-like-url.ts` | URL 判定ユーティリティ |
| `save-as.ts` | ファイル保存ユーティリティ |
| `cookie-store.d.ts` | Cookie Store API 型定義 |

## `dto/` サブディレクトリ（3ファイル）

データエクスポート機能の DTO 定義。

| ファイル | 説明 |
|---------|------|
| `export_dto.ts` | エクスポート DTO 型 |
| `export_dto_types.ts` | エクスポート DTO 型の列挙 |
| `export_prune.ts` | エクスポートデータの剪定 |

## `lantana/` サブディレクトリ（3ファイル）

Lantana（気分値）表示用のユーティリティ。

| ファイル | 説明 |
|---------|------|
| `lantana-flower-state.ts` | 花の状態定義（気分値に応じた表示） |
| `lantana-text-data.ts` | テキストデータ（気分値テキスト） |
| `lantana-text-type.ts` | テキスト型 enum |

## 開発ガイドライン

### Composable パターン

- 各 Vue コンポーネントのロジックは対応する `use-*.ts` に分離
- View 用: `use-xxx-view.ts`、Dialog 用: `use-xxx-dialog.ts`
- コンポーネントの `.vue` ファイルはテンプレートとスタイルのみ、ロジックは Composable に委譲

### 命名規則

- Composable: `use-` プレフィックス + kebab-case
- 対応するコンポーネント名と一致させる（`use-add-kc-view.ts` → `add-kc-view.vue`）
