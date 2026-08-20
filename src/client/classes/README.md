# classes - TypeScript ユーティリティクラス

## 概要

フロントエンドのビジネスロジック、Vue Composable 関数、データモデル、API クライアントを集約するディレクトリ。
Vue コンポーネント（`pages/`）から使用されるロジック層。

## ディレクトリ構造

```
classes/
├── (ルートファイル 365個)        # use-*.ts Composable + ユーティリティ
├── api/                        # GkillAPI クライアント → api/README.md
├── datas/                      # データモデル → datas/README.md
├── dnote/                      # Dynamic Note システム → dnote/README.md
├── kftl/                       # KFTL パーサ → kftl/README.md
├── dto/                        # DTO（2ファイル）
└── lantana/                    # Lantana ユーティリティ（1ファイル）
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
| `use-add-time-is-view.ts` / `use-add-time-is-dialog.ts` | TimeIs 追加 |
| `use-add-ur-log-view.ts` / `use-add-ur-log-dialog.ts` | URLog 追加 |
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
| `use-confirm-delete-kyou-view.ts` | Kyou 削除確認（`cascade-delete-kyou.ts` 経由で連鎖削除） |
| `use-confirm-unknown-mi-board.ts` | まだ実在しない板名を保存する前の確認ゲート（Mi/MiReKyou の追加・編集、KFTL、板間D&D の6経路で共有） |
| `use-confirm-delete-idf-kyou-dialog.ts` | IDFKyou 削除確認 |
| `use-confirm-delete-tag-view.ts` / `use-confirm-delete-tag-dialog.ts` | Tag 削除確認 |
| `use-confirm-delete-text-view.ts` / `use-confirm-delete-text-dialog.ts` | Text 削除確認 |
| `use-confirm-delete-notification-view.ts` / `use-confirm-delete-notification-dialog.ts` | Notification 削除確認 |
| `use-confirm-re-kyou-view.ts` / `use-confirm-re-kyou-dialog.ts` | ReKyou 確認 |

### 表示系 Composable

| ファイル | 対応コンポーネント |
|---------|------------------|
| `use-kyou-view.ts` / `use-kyou-list-view.ts` | Kyou 表示 / 一覧 |
| `use-kyou-list-view-dialog.ts` | Kyou 一覧ダイアログ（DNote から開く）。抱えているリストを自分で引き直し、rykv ダイアログも自分でホストする |
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
| `use-kftl-page.ts` / `use-kftl-view.ts` | KFTL エディタ。`use-kftl-view.ts` はタブのホストも兼ねる（送信対象タブは `do_submit()` の引数で渡す。確認ダイアログの往復中も誤配送しないため）。保存マーカーの自動送信の入口は textarea の `@input` とテンプレート貼り付けの2つで、テンプレートは watch の印に相乗りせず直接判定を呼ぶ |
| `use-kftl-dialog-host.ts` | メモ帳ダイアログの一覧（開いているウィンドウ）。スロット番号は空いている最小のものを払い出す ―― `useFloatingDialog` の保存キーとカスケード量がこれで決まる。上限8枚 |
| `use-kftl-tabs.ts` | KFTL のタブを持つ**共有シングルトン**ストア。`/mkfl` ではインラインの KFTLView と plaing 側のメモ帳ダイアログが同時にマウントされるので、インスタンスごとに配列を持つと片方の古い配列でもう片方のタブが消える。独立した `effectScope(true)` で作る（setup 直下に watch を張ると最初のコンポーネントの unmount で永続化が止まる）。送信中のタブ id の排他（`try_begin_submit` / `end_submit`）もここが持つ ―― 送信中フラグはビューごとなので、ウィンドウをまたいだ二重送信を止められるのはここだけ |
| `use-mi-page.ts` | Mi ページ |
| `use-kyou-page.ts` | Kyou ページ |
| `use-rykv-page.ts` / `use-rykv-view.ts` | Rykv ページ |
| `use-mkfl-page.ts` / `use-mkfl-view.ts` | Mkfl ページ |
| `use-plaing-time-is-page.ts` / `use-plaing-time-is-view.ts` | 稼働中 TimeIs |
| `use-saihate-page.ts` | Saihate ページ |
| `use-shared-page.ts` / `use-shared-mi-page.ts` | 共有ページ |
| `use-set-new-password-page.ts` / `use-set-new-password-view.ts` | パスワード設定 |
| `use-register-first-account-page.ts` / `use-register-first-account-view.ts` | 初回アカウント登録 |

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
| `use-edit-mi-board-struct-view.ts` / `use-mi-board-struct-context-menu.ts` | 板構造編集（Mi の板の並び順の変更と削除） |
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
| `use-find-time-is-query-editor-view.ts` / `use-find-time-is-query-editor-dialog.ts` | 実行中検索条件エディタ（plaing検索カスタム条件用。編集面はキーワードとタグのみで、記録タイプはTimeIs固定） |
| `use-edit-plaing-time-is-dialog.ts` | 実行中検索条件設定の中間ダイアログ（`is_use_custom_find_kyou_query` のON/OFFで未設定へ戻せる） |
| `use-keyword-query.ts` | キーワードクエリ |
| `use-period-of-time-query.ts` | 期間クエリ |
| `use-tag-query.ts` | タグクエリ |
| `use-rep-query.ts` | リポジトリクエリ |
| `use-time-is-query.ts` | TimeIs クエリ |
| `use-calendar-query.ts` / `use-map-query.ts` | カレンダー/地図クエリ |
| `use-mi-board-query.ts` / `use-mi-sort-type-query.ts` | Mi ボード/ソートクエリ |
| `use-mi-extract-check-state-query.ts` | Mi チェック状態抽出 |

### プラグイン系 Composable

| ファイル | 説明 |
|---------|------|
| `use-plugin-html-view.ts` | プラグイン HTML コンテンツ表示ロジック |
| `use-plugin-html-context-menu.ts` | プラグイン HTML コンテキストメニューロジック |

### その他 Composable

| ファイル | 説明 |
|---------|------|
| `use-save-clipboard-to-file-dialog.ts` | クリップボード内容ファイル保存ダイアログロジック |
| `use-scoped-ctrl-v-for-clipboard.ts` | Ctrl+V キーボードショートカットハンドラ（rykv/mi/plaing 画面でクリップボード保存） |
| `use-scoped-enter-for-kftl.ts` | Enter キーショートカットハンドラ（KFTL ダイアログ起動） |
| `use-dialog-history-stack.ts` | ダイアログ履歴スタック管理（バック操作・Escape キー対応） |
| `use-delayed-loading.ts` | 読み込み中表示の遅延（速く終わった読み込みでインジケータを明滅させない） |
| `use-device-kind.ts` | 端末種別（PC / タブレット / スマートフォン）とタッチの有無。ドラッグ&ドロップの可否は `is_pc` で判断する。モジュールレベルのシングルトン |

### ユーティリティファイル

| ファイル | 説明 |
|---------|------|
| `deep-equals.ts` | ディープイコール比較 |
| `delete-gkill-cache.ts` | gkill キャッシュ削除 |
| `format-date-time.ts` | 日時フォーマット |
| `long-press.ts` | ロングプレスディレクティブ |
| `looks-like-url.ts` | URL 判定ユーティリティ |
| `linkify-text.ts` | テキストを URL / 非 URL のセグメント列に分割（`linkified-text.vue` の本文中 URL リンク化用） |
| `save-as.ts` | ファイル保存ユーティリティ |
| `markdown-to-html.ts` | Markdown → HTML 変換（IDFKyou の .md/.markdown リッチ表示用。DOMPurify サニタイズ付き） |
| `mermaid-render.ts` | Markdown 内 ```mermaid コードブロックの図描画 |
| `decode-text.ts` | テキストファイルの文字コード判定・デコード |
| `kftl-tabs.ts` | KFTL のタブの純関数（`derive_kftl_tab_label` / `add_kftl_tab` / `close_kftl_tab` / `load_kftl_tabs` / `save_kftl_tabs`）。タブは常に1枚以上。旧形式の単一キー `kftl_content` からの移行もここ。`parse_kftl_tabs` は壊れたJSONでも throw しない |
| `kyou-view-relay.ts` | Kyou 系イベントの中継ハンドラ束（`build_kyou_view_relay` / `build_kyou_dialog_relay` / ページ最上位の `RykvDialogHost` 用 `build_kyou_dialog_host_handlers`）。`v-on="crudRelayHandlers"` にそのまま渡す |
| `kyou-reload.ts` | Kyou を最新化する唯一の手順（`refresh_kyou` / `refresh_kyou_in_list` / `build_mi_reload_query`）。同じ更新から派生した引き直しは `new_reload_batch()` の値を共有して合流させる。引き直し中は `is_kyou_reloading(id)` が真 |
| `cascade-delete-kyou.ts` | Kyou 削除時の連鎖削除。付随する Tag / Text / Notification と、その Kyou を参照している ReKyou / MiReKyou も論理削除する |
| `cookie-store.d.ts` | Cookie Store API 型定義 |

`KyouViewEmits` の21イベントのうち、ビュー層は18件を中継する。`requested_close_dialog` はダイアログが自分で `hide()` に繋ぐため中継しない。`focused_kyou` / `clicked_kyou` はビュー層が発火源で、入れ子の KyouView で二重発火するためダイアログ層（`build_kyou_dialog_relay`、18+2＝20件）だけが中継する。イベント名はマップ型 + `satisfies` + `Exclude` で網羅を機械検査しており、型に足して配列に足し忘れるとビルドが落ちる。

連鎖削除は Kyou 自身を最後に消す。先に消すとサーバの `FindKyous` が参照元を結果から外してしまい、ReKyou / MiReKyou を辿れなくなるため。TXID / commit_tx は使っていないので途中で失敗すると部分的に確定した状態が残る。その場合は `ERR900094 cascade_delete_failed`（i18n: `FAILED_CASCADE_DELETE_KYOU_MESSAGE`）を返す。参照の連鎖を辿る深さは32段で打ち切り（`ERR900093 cascade_delete_depth_exceeded`）。

## `dto/` サブディレクトリ（2ファイル）

データエクスポート機能の DTO 定義。

| ファイル | 説明 |
|---------|------|
| `export_dto.ts` | エクスポート DTO 型 |
| `export_prune.ts` | エクスポートデータの剪定 |

## `lantana/` サブディレクトリ（1ファイル）

Lantana（気分値）表示用のユーティリティ。

| ファイル | 説明 |
|---------|------|
| `lantana-flower-state.ts` | 花の状態定義（気分値に応じた表示） |

## 開発ガイドライン

### Composable パターン

- 各 Vue コンポーネントのロジックは対応する `use-*.ts` に分離
- View 用: `use-xxx-view.ts`、Dialog 用: `use-xxx-dialog.ts`
- コンポーネントの `.vue` ファイルはテンプレートとスタイルのみ、ロジックは Composable に委譲

### 識別子の命名規約（eslint で機械検査）

- データクラスのプロパティ/メソッド・ローカル変数・通常関数: snake_case（Go 側 JSON タグとの写像）
- コンポーザブル: `useXxx` / イベントコールバック: `onXxx` / CRUD リレーハンドラ束: `xxxHandlers`（camelCase）
- 型・インタフェース: PascalCase（Props/Emits 型も `MiViewProps` のように PascalCase）
- enum メンバー: snake_case

`eslint.config.js` の `app/naming-convention` ブロックが **error** で検査する。
対象は `src/client` 本体のみ（`__tests__` / `*.d.ts` は別流儀として対象外）。

### 命名規則

- Composable: `use-` プレフィックス + kebab-case
- 対応するコンポーネント名と一致させる（`use-add-kc-view.ts` → `add-kc-view.vue`）
