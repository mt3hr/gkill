# 画面仕様（各画面の項目定義）

## 概要

本資料では、gkillフロントエンドの各画面の目的・構成・項目定義を説明します。既存の[screen-transition.md](screen-transition.md)（遷移図）を補完する形で、画面ごとの仕様を記述しています。

### コンポーネント統計

| カテゴリ | 数 |
|---|---|
| ルート | 12 |
| ページコンポーネント | 14 |
| ビューコンポーネント | 175 |
| ダイアログコンポーネント | 92 |
| **コンポーネント合計** | **281**（ルートを除く） |

## 1. ルート定義

`src/client/router/index.ts` で定義される12ルートです。

| パス | コンポーネント | 画面名 | 説明 |
|---|---|---|---|
| `/` | login-page | ログイン | ユーザー認証 |
| `/kftl` | kftl-page | KFTL入力 | テキスト形式の一括記録入力 |
| `/mi` | mi-page | タスクボード | タスク管理（カンバン） |
| `/rykv` | rykv-page | 履歴閲覧 | Kyou一覧・検索 |
| `/kyou` | kyou-page | 記録詳細 | 個別記録の詳細表示 |
| `/mkfl` | mkfl-page | 打刻メモ帳 | KFTL入力 + 稼働中TimeIs表示の複合ビュー |
| `/plaing` | plaing-timeis-page | 打刻一覧 | アクティブな打刻セッション |
| `/saihate` | saihate-page | さいはて | 記録特化画面（FABから各種記録を素早く追加、他画面への遷移なし） |
| `/set_new_password` | set-new-password-page | パスワード変更 | パスワードリセット |
| `/regist_first_account` | regist-first-account-page | 初回登録 | 初回アカウント作成 |
| `/shared_page` | shared-page | 共有ページ | 共有コンテンツの表示 |
| `/shared_mi` | old-shared-mi-page | 共有タスク | 旧形式の共有タスク（shared_pageへリダイレクト） |

## 2. 各ページの詳細仕様

### 2.1 ログイン画面（`/`）

**コンポーネント:** `login-page.vue` → `login-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| ユーザーID | 入力 | テキストフィールド |
| パスワード | 入力 | パスワードフィールド |
| ログインボタン | 操作 | 認証実行 |
| バージョン表示 | 表示 | アプリバージョンバッジ |
| メッセージ/エラー | 表示 | アラートツールチップ |

### 2.2 KFTL入力画面（`/kftl`）

**コンポーネント:** `kftl-page.vue` → `kftl-view.vue`

gkill独自のテキスト形式（KFTL）で複数種類の記録を一括入力する画面です。

| 項目 | 種別 | 説明 |
|---|---|---|
| KFTLテキストエリア | 入力 | 複数行テキスト入力（KFTL書式） |
| テンプレートボタン群 | 操作 | 定型文テンプレートの挿入 |
| 送信ボタン | 操作 | KFTLテキストの送信・実行 |
| ナビゲーションメニュー | 操作 | rykv/mi/kftl/plaing/mkfl/saihate間の切替 |
| ローディングオーバーレイ | 表示 | 非同期処理中の表示 |

### 2.3 タスクボード画面（`/mi`）

**コンポーネント:** `mi-page.vue` → `mi-view.vue`

タスク（Mi）をカンバンボード形式で管理する画面です。

| 項目 | 種別 | 説明 |
|---|---|---|
| ボード選択 | 入力 | タスクボードの切替（mi-board-query） |
| ソート順 | 入力 | 並び順の選択（mi-sort-type-query） |
| 完了状態フィルタ | 入力 | チェック状態での絞り込み（mi-extruct-check-state-query） |
| タスクリスト | 表示 | タスク一覧（mi-kyou-view） |
| タスク追加ボタン | 操作 | 新規タスク作成ダイアログ表示 |
| クエリエディタサイドバー | 操作 | 高度な検索条件設定 |
| アラート通知 | 表示 | 期限アラート等の通知表示 |

### 2.4 履歴閲覧画面（`/rykv`）

**コンポーネント:** `rykv-page.vue` → `rykv-view.vue`

全種類の記録（Kyou）を時系列で閲覧・検索する画面です。

| 項目 | 種別 | 説明 |
|---|---|---|
| キーワード検索 | 入力 | 全文検索（keyword-query） |
| 期間指定 | 入力 | 日付範囲（calendar-query） |
| タグフィルタ | 入力 | タグでの絞り込み（tag-query） |
| 打刻フィルタ | 入力 | 打刻期間での絞り込み（time-is-query） |
| リポジトリフィルタ | 入力 | リポジトリでの絞り込み（rep-query） |
| 地図フィルタ | 入力 | GPS位置での絞り込み（map-query） |
| Kyouリスト | 表示 | 記録一覧（kyou-list-view） |
| ファイルアップロード | 操作 | ファイル追加ダイアログ |
| クエリエディタサイドバー | 操作 | 高度な検索条件設定 |

### 2.5 記録詳細画面（`/kyou`）

**コンポーネント:** `kyou-page.vue` → `kyou-view.vue`

個別の記録を詳細表示する多態コンポーネントです。全14データ型に対応します。

| 項目 | 種別 | 説明 |
|---|---|---|
| 記録本体 | 表示 | データ型に応じた表示（テキスト/数値/URL等） |
| 作成日時 | 表示 | MI用の作成・見積開始/終了・期限表示 |
| 経過時間 | 表示 | TimeIs用の経過時間表示 |
| 打刻終了ボタン | 操作 | アクティブなTimeIsの終了 |
| タグ一覧 | 表示 | attached-tag コンポーネント群 |
| テキスト一覧 | 表示 | attached-text コンポーネント群 |
| 通知一覧 | 表示 | attached-notification コンポーネント群 |
| リポジトリ名 | 表示 | 格納先リポジトリの表示 |
| 関連時間情報 | 表示 | 関連する打刻情報 |
| コンテキストメニュー | 操作 | 右クリック操作メニュー |

### 2.6 打刻メモ帳画面（`/mkfl`）

**コンポーネント:** `mkfl-page.vue` → `kftl-view.vue` + `plaing-timeis-view.vue`

打刻（TimeIs）とメモ（KFTL入力）を組み合わせた複合入力画面です。画面を上下に分割し、上半分でKFTLテキスト入力、下半分で稼働中のTimeIsを表示します。

| 項目 | 種別 | 説明 |
|---|---|---|
| KFTLテキストエリア | 入力 | 上半分：KFTL書式のテキスト入力 |
| テンプレートボタン群 | 操作 | 定型文テンプレートの挿入 |
| 送信ボタン | 操作 | KFTLテキストの送信・実行 |
| アクティブ打刻リスト | 表示 | 下半分：稼働中TimeIsの一覧・終了操作 |

### 2.7 打刻一覧画面（`/plaing`）

**コンポーネント:** `plaing-timeis-page.vue` → `plaing-timeis-view.vue`

アクティブな打刻（TimeIs）セッションの一覧と操作を行う画面です。

| 項目 | 種別 | 説明 |
|---|---|---|
| アクティブ打刻リスト | 表示 | 現在進行中の打刻一覧 |
| リアルタイムタイマー | 表示 | 各打刻の経過時間表示 |
| 終了ボタン | 操作 | 打刻セッションの終了 |
| 全データ型のCRUDイベント | 操作 | 打刻関連の記録操作 |

### 2.8 さいはて画面（`/saihate`）

**コンポーネント:** `saihate-page.vue`

記録に特化した画面です。他画面への遷移はなく、フローティングアクションボタン（FAB）から各種記録を素早く追加できます。「さいはてでも記録ができる」がコンセプトです。

| 項目 | 種別 | 説明 |
|---|---|---|
| FABメニュー | 操作 | 以下の記録追加ダイアログを開く |
| └ KFTL入力 | 操作 | KFTLテキスト入力ダイアログ |
| └ MKFL入力 | 操作 | ファイル追加ダイアログ |
| └ KC入力 | 操作 | 数値記録ダイアログ |
| └ Urlog入力 | 操作 | ブックマーク追加ダイアログ |
| └ TimeIs入力 | 操作 | 打刻開始ダイアログ |
| └ Mi入力 | 操作 | タスク追加ダイアログ |
| └ Nlog入力 | 操作 | 支出記録ダイアログ |
| └ Lantana入力 | 操作 | 気分記録ダイアログ |
| リロードボタン | 操作 | データ再読込 |
| ログアウトボタン | 操作 | ログアウト確認ダイアログ |

### 2.9 パスワード変更画面（`/set_new_password`）

**コンポーネント:** `set-new-password-page.vue` → `set-new-password-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| 新パスワード | 入力 | パスワードフィールド |
| パスワード確認 | 入力 | 確認用パスワードフィールド |
| 変更ボタン | 操作 | パスワード更新実行 |
| バージョン表示 | 表示 | アプリバージョン |

### 2.10 初回アカウント登録画面（`/regist_first_account`）

**コンポーネント:** `regist-first-account-page.vue`

初回起動時のアカウント作成画面です。

| 項目 | 種別 | 説明 |
|---|---|---|
| ユーザーID | 入力 | アカウント名 |
| パスワード | 入力 | パスワード設定 |
| 管理用パスワード | 入力 | 管理者パスワード |
| 登録ボタン | 操作 | アカウント作成実行 |

### 2.11 共有ページ（`/shared_page`）

**コンポーネント:** `shared-page.vue`

共有された記録を表示するルーターページです。共有IDパラメータに基づいてMiまたはRYKVの共有ビューを表示します。

## 3. データ型別CRUD画面仕様

### CRUD対応マトリクス

| データ型 | 追加View | 追加Dialog | 編集View | 編集Dialog | 表示View | 削除確認 | 履歴 |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| kmemo | — | — | ✅ | ✅ | ✅ | — | — |
| mi | ✅ | ✅ | ✅ | ✅ | ✅ | — | — |
| timeis | ✅ | ✅ | ✅ | ✅ | ✅ | — | — |
| lantana | ✅ | ✅ | ✅ | ✅ | ✅ | — | — |
| nlog | ✅ | ✅ | ✅ | ✅ | ✅ | — | — |
| urlog | ✅ | ✅ | ✅ | ✅ | ✅ | — | — |
| kc | ✅ | ✅ | ✅ | ✅ | ✅ | — | — |
| tag | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| text | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| notification | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| idf_kyou | — | — | ✅ | ✅ | ✅ | ✅ | — |
| re_kyou | — | — | ✅ | ✅ | ✅ | ✅ | — |
| kyou (共通) | — | — | — | — | ✅ | ✅ | ✅ |

**備考:**
- **kmemo**: 追加はKFTL経由で行う
- **idf_kyou**: 追加はMKFL（ファイルアップロード）経由で行う
- **re_kyou**: 追加は既存記録のコンテキストメニューから行う
- **kyou-view**: 全14データ型を多態的に表示する統合コンポーネント
- **削除確認・履歴**: 個別の「—」は専用コンポーネントが存在しないことを示す。全データ型の削除確認は `confirm-delete-kyou-view` (共通)、履歴表示は `kyou-histories-view` (共通) で対応する

### 3.1 kmemo（テキストメモ）画面仕様

**追加:** KFTL入力画面経由

**表示:** `kmemo-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| メモ本文 | 表示 | プレフォーマットテキスト |
| 作成日時 | 表示 | 記録日時 |

**編集:** `edit-kmemo-view.vue` / `edit-kmemo-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| メモ本文 | 入力 | テキストエリア |
| 日時 | 入力 | 日付・時刻ピッカー |
| 関連日付 | 入力 | 関連日付の選択 |

### 3.2 mi（タスク）画面仕様

**追加:** `add-mi-view.vue` / `add-mi-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| タスク名 | 入力 | タスクのタイトル |
| ボード | 入力 | 所属ボードの選択 |
| 通知設定 | 入力 | 期限通知の追加（add-notification-for-add-mi-view） |

**表示:** `mi-kyou-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| タスク名 | 表示 | タスクタイトル |
| チェック状態 | 表示/操作 | 完了/未完了トグル |
| ボード名 | 表示 | 所属ボード |
| 作成日時 | 表示 | MI作成日時 |
| 見積開始/終了 | 表示 | 見積期間 |
| 期限 | 表示 | タスク期限 |

**編集:** `edit-mi-view.vue` / `edit-mi-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| タスク名 | 入力 | タスクタイトル編集 |
| ボード | 入力 | ボード変更 |
| チェック状態 | 入力 | 完了状態の変更 |
| 通知設定 | 入力 | 通知の追加・編集 |

### 3.3 timeis（打刻）画面仕様

**追加:** `add-timeis-view.vue` / `add-timeis-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| タイトル | 入力 | 打刻セッション名 |
| 開始日時 | 入力 | 開始日時の指定 |

**表示:** `time-is-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| タイトル | 表示 | セッション名 |
| 開始日時 | 表示 | 開始時刻 |
| 終了日時 | 表示 | 終了時刻（未終了は空） |
| 経過時間 | 表示 | 開始から終了（または現在）までの経過 |

**アクティブ打刻:** `plaing-timeis-view.vue` / `attached-time-is-plaing.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| リアルタイムタイマー | 表示 | 経過時間のリアルタイム更新 |
| 終了ボタン | 操作 | 打刻終了（end-time-is-plaing-view） |

### 3.4 lantana（気分）画面仕様

**追加:** `add-lantana-view.vue` / `add-lantana-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| 気分値 | 入力 | 0〜10のスライダー/数値入力 |

**表示:** `lantana-view.vue` / `lantana-flowers-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| 気分値 | 表示 | 数値（0〜10） |
| 花ビジュアライゼーション | 表示 | 気分を花で表現（lantana-flower） |

### 3.5 nlog（支出）画面仕様

**追加:** `add-nlog-view.vue` / `add-nlog-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| 店名 | 入力 | 店名（Shop） |
| タイトル | 入力 | 支出の内容（Title） |
| 金額 | 入力 | 金額入力（Amount） |
| 日時 | 入力 | 記録日時 |

**表示:** `nlog-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| 店名 | 表示 | 店名 |
| タイトル | 表示 | 支出の内容 |
| 金額 | 表示 | 支出金額 |
| 記録日時 | 表示 | 記録日時 |

### 3.6 urlog（ブックマーク）画面仕様

**追加:** `add-urlog-view.vue` / `add-urlog-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| URL | 入力 | URLフィールド |
| タイトル | 入力 | ページタイトル |

**表示:** `ur-log-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| URL | 表示 | リンク付きURL |
| タイトル | 表示 | ブックマークタイトル |

### 3.7 kc（数値）画面仕様

**追加:** `add-kc-view.vue` / `add-kc-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| 数値 | 入力 | 数値フィールド |

**表示:** `kc-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| 数値 | 表示 | 記録された数値 |

### 3.8 tag（タグ）画面仕様

完全なCRUD + 履歴を持つ付帯データ型です。

**追加:** `add-tag-view.vue` / `add-tag-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| タグ名 | 入力 | タグ文字列（未知タグ時は確認ダイアログ表示） |

**表示:** `tag-view.vue` / `attached-tag.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| タグ名 | 表示 | タグ文字列 |

**削除確認:** `confirm-delete-tag-view.vue` / `confirm-delete-tag-dialog.vue`

**履歴:** `tag-histories-view.vue` / `tag-histories-dialog.vue`

**タグ構造管理:**
- `edit-tag-struct-view.vue` — タグ階層構造の編集
- `edit-tag-struct-element-view.vue` — 個別タグ要素の編集
- `add-new-tag-struct-element-view.vue` — 新規タグ要素の追加
- `confirm-delete-tag-struct-view.vue` — タグ構造の削除確認

### 3.9 text（テキスト）画面仕様

記録に付与する補足テキスト。完全なCRUD + 履歴を持ちます。

**追加:** `add-text-view.vue` / `add-text-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| テキスト本文 | 入力 | テキストエリア |

**表示:** `text-view.vue` / `attached-text.vue`

**削除確認:** `confirm-delete-text-view.vue`

**履歴:** `text-histories-view.vue` / `text-histories-dialog.vue`

### 3.10 notification（通知）画面仕様

記録に紐づくプッシュ通知設定。完全なCRUD + 履歴を持ちます。

**追加:** `add-notification-view.vue` / `add-notification-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| 通知日時 | 入力 | 日時ピッカー |

**表示:** `notification-view.vue` / `attached-notification.vue`

**削除確認:** `confirm-delete-notification-view.vue`

**履歴:** `notification-histories-view.vue` / `notification-histories-dialog.vue`

## 4. 検索・フィルタコンポーネント

gkillの検索機能は複数のクエリコンポーネントを組み合わせて実現します。

| コンポーネント | 説明 |
|---|---|
| `keyword-query.vue` | キーワード全文検索 |
| `calendar-query.vue` | 日付範囲指定 |
| `period-of-time-query.vue` | 時間範囲指定 |
| `tag-query.vue` | タグフィルタ |
| `time-is-query.vue` | 打刻期間フィルタ |
| `rep-query.vue` | リポジトリフィルタ |
| `map-query.vue` | 地図・GPS位置フィルタ |
| `mi-board-query.vue` | タスクボードフィルタ |
| `mi-sort-type-query.vue` | ソート順選択 |
| `mi-extruct-check-state-query.vue` | タスク完了状態フィルタ |
| `find-query-editor-view.vue` | 高度な検索条件ビルダー |
| `rykv-query-editor-sidebar.vue` | RYKV画面用検索サイドバー |
| `mi-query-editor-sidebar.vue` | Mi画面用検索サイドバー |
| `clear-query-button.vue` | フィルタリセットボタン |
| `search-button.vue` | 検索実行ボタン |

## 5. 設定・管理コンポーネント

### アカウント管理

| コンポーネント | 説明 |
|---|---|
| `create-account-view.vue` | アカウント作成 |
| `manage-account-view.vue` | アカウント管理 |
| `show-password-reset-link-view.vue` | パスワードリセットリンク表示 |

### サーバー設定

| コンポーネント | 説明 |
|---|---|
| `application-config-view.vue` | アプリケーション設定 |
| `server-config-view.vue` | サーバー設定 |
| `confirm-generate-tls-files-view.vue` | TLS証明書生成確認 |

### デバイス管理

| コンポーネント | 説明 |
|---|---|
| `edit-device-struct-view.vue` | デバイス構造編集 |
| `edit-device-struct-element-view.vue` | デバイス要素編集 |
| `add-new-device-struct-element-view.vue` | デバイス要素追加 |
| `confirm-delete-device-struct-view.vue` | デバイス削除確認 |

### リポジトリ管理

| コンポーネント | 説明 |
|---|---|
| `add-rep-view.vue` | リポジトリ追加 |
| `allocate-rep-view.vue` | リポジトリ割当 |
| `confirm-delete-rep-view.vue` | リポジトリ削除確認 |
| `edit-rep-struct-view.vue` | リポジトリ構造編集 |
| `edit-rep-type-struct-view.vue` | RepType構造編集 |

### KFTLテンプレート管理

| コンポーネント | 説明 |
|---|---|
| `kftl-template-view.vue` | テンプレート表示 |
| `edit-kftl-template-view.vue` | テンプレート編集 |
| `edit-kftl-template-struct-view.vue` | テンプレート構造編集 |
| `edit-kftl-template-struct-element-view.vue` | テンプレート要素編集 |

## 6. ダイアログシステム

### ダイアログ共通構造

全ダイアログは以下のパターンで構成されています。

```
Teleport to body
├── スクリムオーバーレイ（半透明トグル対応）
├── ドラッグ可能ヘッダー（マウス + タッチ対応）
├── コンテンツ（ビューコンポーネント埋め込み）
├── 閉じるボタン
└── 透明モードチェックボックス
```

**v1.1.0 変更点:**
- ダイアログ表示時の自動フォーカスを削除（ページスクロール位置に影響しないように）
- フォーカストラップ（Tab/Shift+Tab循環）を削除
- フォーカス復帰（ダイアログ閉じ時に元要素へ戻す）を削除
- 透過モード切替時にダイアログの高さが縮小するバグを修正（ResizeObserver更新を300ms抑制）

### ダイアログ一覧（カテゴリ別）

#### データ追加ダイアログ

| ダイアログ | 対象データ型 |
|---|---|
| `add-mi-dialog.vue` | タスク |
| `add-timeis-dialog.vue` | 打刻 |
| `add-lantana-dialog.vue` | 気分 |
| `add-nlog-dialog.vue` | 支出 |
| `add-urlog-dialog.vue` | ブックマーク |
| `add-kc-dialog.vue` | 数値 |
| `add-tag-dialog.vue` | タグ |
| `add-text-dialog.vue` | テキスト |
| `add-notification-dialog.vue` | 通知 |

#### データ編集ダイアログ

| ダイアログ | 対象データ型 |
|---|---|
| `edit-kmemo-dialog.vue` | テキストメモ |
| `edit-mi-dialog.vue` | タスク |
| `edit-time-is-dialog.vue` | 打刻 |
| `edit-lantana-dialog.vue` | 気分 |
| `edit-nlog-dialog.vue` | 支出 |
| `edit-ur-log-dialog.vue` | ブックマーク |
| `edit-kc-dialog.vue` | 数値 |
| `edit-tag-dialog.vue` | タグ |
| `edit-text-dialog.vue` | テキスト |
| `edit-notification-dialog.vue` | 通知 |
| `edit-idf-kyou-dialog.vue` | ファイル |
| `edit-re-kyou-dialog.vue` | リポスト |

#### 削除確認ダイアログ

| ダイアログ | 対象 |
|---|---|
| `confirm-delete-tag-dialog.vue` | タグ |
| `confirm-delete-text-dialog.vue` | テキスト |
| `confirm-delete-notification-dialog.vue` | 通知 |
| `confirm-delete-idf-kyou-dialog.vue` | ファイル |
| `confirm-re-kyou-dialog.vue` | リポスト |
| `confirm-logout-dialog.vue` | ログアウト |

#### 履歴ダイアログ

| ダイアログ | 対象 |
|---|---|
| `tag-histories-dialog.vue` | タグ変更履歴 |
| `text-histories-dialog.vue` | テキスト変更履歴 |
| `notification-histories-dialog.vue` | 通知変更履歴 |
| `kyou-histories-dialog.vue` | 記録変更履歴 |

#### 設定ダイアログ

| ダイアログ | 対象 |
|---|---|
| `application-config-dialog.vue` | アプリケーション設定 |
| `server-config-dialog.vue` | サーバー設定 |
| `manage-account-dialog.vue` | アカウント管理 |
| `create-account-dialog.vue` | アカウント作成 |
| `edit-folder-dialog.vue` | フォルダ編集 |

#### 共有ダイアログ

| ダイアログ | 対象 |
|---|---|
| `share-kyou-list-dialog.vue` | 共有設定一覧 |
| `share-kyou-list-link-dialog.vue` | 共有リンク表示 |
| `confirm-delete-share-kyou-list-dialog.vue` | 共有削除確認 |
| `manage-share-task-list-dialog.vue` | 共有タスク管理 |

## 7. イベント伝播パターン

コンポーネント間のイベント伝播は以下のパターンで統一されています。

```
CRUDリレーイベント:
├── deleted_kyou / deleted_tag / deleted_text / deleted_notification
├── registered_kyou / registered_tag / registered_text / registered_notification
├── updated_kyou / updated_tag / updated_text / updated_notification
├── received_errors / received_messages
├── focused_kyou / clicked_kyou
└── requested_reload_kyou / requested_reload_list
```

各ViewコンポーネントはTypeScriptで型定義されたProps/Emitsを持ちます。

| ファイル | 用途 |
|---|---|
| `*-props.ts` | 入力プロパティのインターフェース定義 |
| `*-emits.ts` | イベント発信のシグネチャ定義 |

## 8. コンテキストメニュー

右クリック（`contextmenu.prevent`）で表示されるコンテキストメニューが各データ型に用意されています。

| コンテキストメニュー | 対象 |
|---|---|
| `kmemo-context-menu.vue` | テキストメモ |
| `mi-context-menu.vue` | タスク |
| `time-is-context-menu.vue` | 打刻 |
| `lantana-context-menu.vue` | 気分 |
| `nlog-context-menu.vue` | 支出 |
| `ur-log-context-menu.vue` | ブックマーク |
| `kc-context-menu.vue` | 数値 |
| `idf-kyou-context-menu.vue` | ファイル |
| `re-kyou-context-menu.vue` | リポスト |
| `git-commit-log-context-menu.vue` | Gitコミット |
| `attached-tag-context-menu.vue` | 付帯タグ |
| `attached-text-context-menu.vue` | 付帯テキスト |
| `attached-notification-context-menu.vue` | 付帯通知 |
| `attached-timeis-plaing-context-menu.vue` | アクティブ打刻 |
| `device-struct-context-menu.vue` | デバイス構造（設定画面） |
| `dnote-item-list-context-menu.vue` | DNote項目リスト |
| `dnote-list-query-context-menu.vue` | DNoteクエリ |
| `kftl-template-struct-context-menu.vue` | KFTLテンプレート構造（設定画面） |
| `rep-struct-context-menu.vue` | リポジトリ構造（設定画面） |
| `rep-type-struct-context-menu.vue` | RepType構造（設定画面） |
| `ryuu-list-item-context-menu.vue` | Ryuuリスト項目 |
| `tag-struct-context-menu.vue` | タグ構造（設定画面） |

## 関連資料

- [screen-transition.md](screen-transition.md) — 画面遷移図
- [frontend-architecture.md](frontend-architecture.md) — フロントエンド設計ガイド
- [glossary.md](glossary.md) — 用語集
- [usecase.md](usecase.md) — ユースケース一覧
