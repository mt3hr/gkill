# pages - Vue ページ・ビュー・ダイアログ

## 概要

Vue Router のルートページ、データ操作用 View コンポーネント、ダイアログコンポーネントを格納するディレクトリ。
全コンポーネントは Vuetify 4 を使用し、ロジックは `classes/` の Composable に委譲する設計。

## ディレクトリ構造

```
pages/
├── (ルートページ 14 .vue + 2 .ts)  # Vue Router ルートページ
├── views/                          # View コンポーネント（175 .vue）
└── dialogs/                        # Dialog コンポーネント（93 .vue）
```

## 設計パターン

### Page → View → Dialog 階層

```
Page（ルートページ）
  └── View（メインコンテンツ）
       └── Dialog（モーダル操作）
```

### コンポーネント構成パターン

多くのコンポーネントは以下の3ファイルで構成:
```
xxx-view.vue          # テンプレート + スタイル
xxx-view-props.ts     # Props 定義
xxx-view-emits.ts     # Emits 定義
```

ロジックは `classes/use-xxx-view.ts` Composable に分離。

## ルートページ（14 .vue + 2 .ts）

| ファイル | ルートパス | 説明 |
|---------|----------|------|
| `login-page.vue` | `/` | ログインページ |
| `kftl-page.vue` | `/kftl` | KFTL テキストエディタ |
| `mi-page.vue` | `/mi` | Mi（タスク）管理ページ |
| `rykv-page.vue` | `/rykv` | Rykv（レビュー）ページ |
| `kyou-page.vue` | `/kyou` | Kyou（記録）一覧ページ |
| `mkfl-page.vue` | `/mkfl` | Mkfl（ファイル管理）ページ |
| `plaing-timeis-page.vue` | `/plaing` | 稼働中 TimeIs ページ |
| `saihate-page.vue` | `/saihate` | Saihate（特殊ビュー）ページ |
| `set-new-password-page.vue` | `/set_new_password` | 新パスワード設定ページ |
| `regist-first-account-page.vue` | `/regist_first_account` | 初回アカウント登録 |
| `shared-page.vue` | `/shared_page` | 共有ページ |
| `old-shared-mi-page.vue` | `/shared_mi` | 旧共有 Mi ページ |
| `shared-mi-page.vue` | — | 共有 Mi ページ（新） |
| `shared-rykv-page.vue` | — | 共有 Rykv ページ |
| `shared-mi-page-props.ts` | — | 共有 Mi Props |
| `shared-rykv-page-props.ts` | — | 共有 Rykv Props |

## views/ — 全 View コンポーネント一覧（175 .vue）

### データ追加系（20コンポーネント）

| コンポーネント | 説明 |
|--------------|------|
| `add-kc-view.vue` | KC 追加 |
| `add-lantana-view.vue` | Lantana 追加 |
| `add-mi-view.vue` | Mi 追加 |
| `add-nlog-view.vue` | Nlog 追加 |
| `add-timeis-view.vue` | TimeIs 追加 |
| `add-urlog-view.vue` | URLog 追加 |
| `add-tag-view.vue` | Tag 追加 |
| `add-text-view.vue` | Text 追加 |
| `add-notification-view.vue` | Notification 追加 |
| `add-notification-for-add-mi-view.vue` | Mi 追加時の Notification |
| `add-rep-view.vue` | リポジトリ追加 |
| `add-ryuu-item-view.vue` | Ryuu アイテム追加 |
| `add-dnote-item-view.vue` | DNote アイテム追加 |
| `add-dnote-list-view.vue` | DNote リスト追加 |
| `add-new-device-struct-element-view.vue` | デバイス構造要素追加 |
| `add-new-foloder-view.vue` | フォルダ追加 |
| `add-new-kftl_template-struct-element-view.vue` | KFTL テンプレート要素追加 |
| `add-new-rep-struct-element-view.vue` | リポジトリ構造要素追加 |
| `add-new-rep-type-struct-element-view.vue` | リポジトリ型要素追加 |
| `add-new-tag-struct-element-view.vue` | タグ構造要素追加 |

### データ編集系（27コンポーネント）

| コンポーネント | 説明 |
|--------------|------|
| `edit-kc-view.vue` | KC 編集 |
| `edit-kmemo-view.vue` | Kmemo 編集 |
| `edit-lantana-view.vue` | Lantana 編集 |
| `edit-mi-view.vue` | Mi 編集 |
| `edit-nlog-view.vue` | Nlog 編集 |
| `edit-time-is-view.vue` | TimeIs 編集 |
| `edit-ur-log-view.vue` | URLog 編集 |
| `edit-idf-kyou-view.vue` | IDFKyou 編集 |
| `edit-re-kyou-view.vue` | ReKyou 編集 |
| `edit-tag-view.vue` | Tag 編集 |
| `edit-text-view.vue` | Text 編集 |
| `edit-notification-view.vue` | Notification 編集 |
| `edit-ryuu-item-view.vue` | Ryuu アイテム編集 |
| `edit-folder-view.vue` | フォルダ編集 |
| `edit-dnote-card.vue` / `edit-dnote-item-view.vue` / `edit-dnote-list-view.vue` | DNote 編集 |
| `edit-dnote-predicate-group.vue` | DNote 述語グループ編集 |
| `edit-device-struct-view.vue` / `edit-device-struct-element-view.vue` | デバイス構造編集 |
| `edit-kftl-template-struct-view.vue` / `edit-kftl-template-struct-element-view.vue` | KFTL テンプレート編集 |
| `edit-kftl-template-view.vue` | KFTL テンプレートビュー |
| `edit-rep-struct-view.vue` / `edit-rep-struct-element-view.vue` | リポジトリ構造編集 |
| `edit-rep-type-struct-view.vue` / `edit-rep-type-struct-element-view.vue` | リポジトリ型構造編集 |
| `edit-tag-struct-view.vue` / `edit-tag-struct-element-view.vue` | タグ構造編集 |

### データ表示系（26コンポーネント）

| コンポーネント | 説明 |
|--------------|------|
| `kyou-view.vue` / `kyou-list-view.vue` | Kyou 表示 / 一覧 |
| `kc-view.vue` / `kmemo-view.vue` | KC / Kmemo 表示 |
| `lantana-view.vue` / `mi-view.vue` | Lantana / Mi 表示 |
| `nlog-view.vue` / `ur-log-view.vue` | Nlog / URLog 表示 |
| `time-is-view.vue` / `idf-kyou-view.vue` | TimeIs / IDFKyou 表示 |
| `re-kyou-view.vue` / `git-commit-log-view.vue` | ReKyou / GitCommitLog 表示 |
| `tag-view.vue` / `text-view.vue` / `notification-view.vue` | Tag / Text / Notification 表示 |
| `mi-kyou-view.vue` | Mi の Kyou 表示 |
| `lantana-flower.vue` / `lantana-flowers-view.vue` | Lantana 花表示 |
| `kyou-histories-view.vue` | Kyou 変更履歴 |
| `tag-histories-view.vue` / `text-histories-view.vue` / `notification-histories-view.vue` | 履歴表示 |
| `gps-log-map.vue` | GPS ログ地図表示 |
| `kyou-count-calendar.vue` | Kyou カウントカレンダー |

### 削除確認系（13コンポーネント）

| コンポーネント | 説明 |
|--------------|------|
| `confirm-delete-kyou-view.vue` | Kyou 削除確認 |
| `confirm-delete-tag-view.vue` / `confirm-delete-text-view.vue` | Tag / Text 削除確認 |
| `confirm-delete-notification-view.vue` | Notification 削除確認 |
| `confirm-delete-share-task-list-view.vue` | 共有タスクリスト削除確認 |
| `confirm-delete-device-struct-view.vue` | デバイス構造削除確認 |
| `confirm-delete-kftl-template-struct-view.vue` | KFTL テンプレート削除確認 |
| `confirm-delete-rep-struct-view.vue` / `confirm-delete-rep-type-struct-view.vue` | リポジトリ構造削除確認 |
| `confirm-delete-rep-view.vue` | リポジトリ削除確認 |
| `confirm-delete-tag-struct-view.vue` | タグ構造削除確認 |
| `confirm-delete-dnote-item-list-view.vue` / `confirm-delete-dnote-list-query-view.vue` | DNote 削除確認 |
| `confirm-delete-related-kyou-query-view.vue` | 関連 Kyou クエリ削除確認 |

### 確認・操作系

| コンポーネント | 説明 |
|--------------|------|
| `confirm-generate-tls-files-view.vue` | TLS ファイル生成確認 |
| `confirm-logout-view.vue` | ログアウト確認 |
| `confirm-re-kyou-view.vue` | ReKyou 確認 |
| `confirm-reset-password-view.vue` | パスワードリセット確認 |

### 設定系

| コンポーネント | 説明 |
|--------------|------|
| `application-config-view.vue` | アプリケーション設定 |
| `server-config-view.vue` | サーバ設定 |
| `allocate-rep-view.vue` | リポジトリ割り当て |
| `login-view.vue` / `create-account-view.vue` / `manage-account-view.vue` | 認証・アカウント管理 |
| `set-new-password-view.vue` / `show-password-reset-link-view.vue` | パスワード管理 |
| `regist-first-account-view.vue` | 初回アカウント登録 |

### DNote 系

| コンポーネント | 説明 |
|--------------|------|
| `dnote-view.vue` | DNote メインビュー |
| `dnote-item-view.vue` / `dnote-item-list-view.vue` | DNote アイテム |
| `dnote-item-table-view.vue` / `dnote-list-table-view.vue` | DNote テーブル |
| `dnote-list-view.vue` | DNote リスト |
| `dnote-item-list-context-menu.vue` / `dnote-list-query-context-menu.vue` | DNote コンテキストメニュー |
| `aggregated-list-item.vue` | 集計アイテム |

### コンテキストメニュー系

| コンポーネント | 説明 |
|--------------|------|
| `kc-context-menu.vue` / `kmemo-context-menu.vue` | KC / Kmemo |
| `lantana-context-menu.vue` / `mi-context-menu.vue` | Lantana / Mi |
| `nlog-context-menu.vue` / `ur-log-context-menu.vue` | Nlog / URLog |
| `time-is-context-menu.vue` / `idf-kyou-context-menu.vue` | TimeIs / IDFKyou |
| `re-kyou-context-menu.vue` / `git-commit-log-context-menu.vue` | ReKyou / GitCommitLog |
| `attached-tag-context-menu.vue` / `attached-text-context-menu.vue` | 添付タグ/テキスト |
| `attached-notification-context-menu.vue` / `attached-timeis-plaing-context-menu.vue` | 添付通知/稼働TimeIs |
| `device-struct-context-menu.vue` / `kftl-template-struct-context-menu.vue` | 構造コンテキストメニュー |
| `rep-struct-context-menu.vue` / `rep-type-struct-context-menu.vue` | リポジトリ構造 |
| `tag-struct-context-menu.vue` / `ryuu-item-context-menu.vue` | タグ構造/Ryuu |

### クエリ系

| コンポーネント | 説明 |
|--------------|------|
| `find-query-editor-view.vue` | 検索クエリエディタ |
| `keyword-query.vue` / `period-of-time-query.vue` | キーワード/期間 |
| `tag-query.vue` / `rep-query.vue` | タグ/リポジトリ |
| `time-is-query.vue` / `calendar-query.vue` / `map-query.vue` | TimeIs/カレンダー/地図 |
| `mi-board-query.vue` / `mi-sort-type-query.vue` / `mi-extruct-check-state-query.vue` | Mi クエリ |
| `mi-query-editor-sidebar.vue` / `rykv-query-editor-side-bar.vue` | サイドバー |
| `search-button.vue` / `clear-query-button.vue` | 検索/クリアボタン |

### その他

| コンポーネント | 説明 |
|--------------|------|
| `kftl-view.vue` / `kftl-line-label.vue` / `kftl-template-view.vue` | KFTL エディタ |
| `mkfl-view.vue` | Mkfl ビュー |
| `plaing-timeis-view.vue` / `end-time-is-plaing-view.vue` | 稼働中 TimeIs |
| `attached-tag.vue` / `attached-text.vue` / `attached-notification.vue` | 添付情報 |
| `attached-time-is-plaing.vue` | 添付稼働 TimeIs |
| `upload-file-view.vue` / `decide-related-time-uploaded-file-view.vue` | ファイルアップロード |
| `share-kyou-view.vue` / `share-kyou-link-view.vue` / `share-kyou-footer.vue` | 共有 |
| `shared-mi-view.vue` / `manage-share-task-list-view.vue` | 共有 Mi 管理 |
| `manage-share-button.vue` / `share-button.vue` | 共有ボタン |
| `rykv-view.vue` / `rykv-dialog-host.vue` / `rykv-dialog-host-item.vue` | Rykv |
| `ryuu-view.vue` / `ryuu-item-view.vue` | Ryuu |
| `sidebar-header.vue` | サイドバーヘッダー |
| `foldable-struct.vue` / `rep-struct-element.vue` / `rep-type-element.vue` / `tag-struct-element.vue` | 構造要素 |
| `saihate-stars-overlay.vue` / `snow-fall-overlay.vue` | オーバーレイ効果 |

## dialogs/ — 全 Dialog コンポーネント一覧（93 .vue）

### データ追加ダイアログ（19コンポーネント）

`add-kc-dialog.vue`, `add-lantana-dialog.vue`, `add-mi-dialog.vue`, `add-nlog-dialog.vue`, `add-timeis-dialog.vue`, `add-urlog-dialog.vue`, `add-tag-dialog.vue`, `add-text-dialog.vue`, `add-notification-dialog.vue`, `add-rep-dialog.vue`, `add-ryuu-item-dialog.vue`, `add-dnote-item-dialog.vue`, `add-dnote-list-dialog.vue`, `add-new-device-struct-element-dialog.vue`, `add-new-foloder-dialog.vue`, `add-new-kftl-template-struct-element-dialog.vue`, `add-new-rep-struct-element-dialog.vue`, `add-new-rep-type-struct-element-dialog.vue`, `add-new-tag-struct-element-dialog.vue`

### データ編集ダイアログ（27コンポーネント）

`edit-kc-dialog.vue`, `edit-kmemo-dialog.vue`, `edit-lantana-dialog.vue`, `edit-mi-dialog.vue`, `edit-nlog-dialog.vue`, `edit-time-is-dialog.vue`, `edit-ur-log-dialog.vue`, `edit-idf-kyou-dialog.vue`, `edit-re-kyou-dialog.vue`, `edit-tag-dialog.vue`, `edit-text-dialog.vue`, `edit-notification-dialog.vue`, `edit-ryuu-dialog.vue`, `edit-ryuu-item-dialog.vue`, `edit-folder-dialog.vue`, `edit-dnote-dialog.vue`, `edit-dnote-item-dialog.vue`, `edit-dnote-list-dialog.vue`, `edit-device-struct-dialog.vue`, `edit-device-struct-element-dialog.vue`, `edit-kftl-template-struct-dialog.vue`, `edit-kftl-template-struct-element-dialog.vue`, `edit-rep-struct-dialog.vue`, `edit-rep-struct-element-dialog.vue`, `edit-rep-type-struct-dialog.vue`, `edit-rep-type-struct-element-dialog.vue`, `edit-tag-struct-dialog.vue`, `edit-tag-struct-element-dialog.vue`

### 削除確認ダイアログ（13コンポーネント）

`confirm-delete-idf-kyou-dialog.vue`, `confirm-delete-tag-dialog.vue`, `confirm-delete-text-dialog.vue`, `confirm-delete-notification-dialog.vue`, `confirm-delete-share-kyou-list-dialog.vue`, `confirm-delete-device-struct-dialog.vue`, `confirm-delete-kftl-template-struct-dialog.vue`, `confirm-delete-rep-dialog.vue`, `confirm-delete-rep-struct-dialog.vue`, `confirm-delete-rep-type-struct-dialog.vue`, `confirm-delete-tag-struct-dialog.vue`, `confirm-delete-dnote-item-list-dialog.vue`, `confirm-delete-dnote-list-query-dialog.vue`, `confirm-delete-ryuu-item-dialog.vue`

### その他ダイアログ

| コンポーネント | 説明 |
|--------------|------|
| `confirm-generate-tls-files-dialog.vue` | TLS 生成確認 |
| `confirm-logout-dialog.vue` | ログアウト確認 |
| `confirm-re-kyou-dialog.vue` | ReKyou 確認 |
| `confirm-reset-password-dialog.vue` | パスワードリセット確認 |
| `end-time-is-plaing-dialog.vue` | TimeIs 終了 |
| `find-query-editor-dialog.vue` | 検索クエリエディタ |
| `kftl-dialog.vue` / `kftl-template-dialog.vue` | KFTL |
| `kyou-dialog.vue` / `kyou-histories-dialog.vue` / `kyou-list-view-dialog.vue` | Kyou |
| `lantana-dialog.vue` | Lantana |
| `manage-account-dialog.vue` | アカウント管理 |
| `manage-share-task-list-dialog.vue` | 共有タスクリスト管理 |
| `mkfl-dialog.vue` | Mkfl |
| `new-board-name-dialog.vue` / `new-device-name-dialog.vue` | 新規名前入力 |
| `notification-histories-dialog.vue` / `tag-histories-dialog.vue` / `text-histories-dialog.vue` | 履歴 |
| `server-config-dialog.vue` / `application-config-dialog.vue` | 設定 |
| `allocate-rep-dialog.vue` | リポジトリ割り当て |
| `create-account-dialog.vue` | アカウント作成 |
| `decide-related-time-uploaded-file-dialog.vue` | アップロードファイル時刻設定 |
| `show-password-reset-link-dialog.vue` | パスワードリセットリンク |
| `share-kyou-list-dialog.vue` / `share-kyou-list-link-dialog.vue` | 共有リスト |
| `upload-file-dialog.vue` | ファイルアップロード |

## 開発ガイドライン

### 新しいコンポーネントの追加

1. View: `views/xxx-view.vue` を作成、ロジックは `classes/use-xxx-view.ts` に分離
2. Dialog: `dialogs/xxx-dialog.vue` を作成、ロジックは `classes/use-xxx-dialog.ts` に分離
3. 必要に応じて `xxx-view-props.ts`, `xxx-view-emits.ts` を作成
4. Vuetify 4 のコンポーネントを使用してUI構築
