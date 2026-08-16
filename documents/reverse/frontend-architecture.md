# フロントエンド設計ガイド

## 1. 技術スタック

| カテゴリ | 技術 | バージョン |
|---|---|---|
| フレームワーク | Vue 3 (Composition API) | ^3.5.40 |
| UIライブラリ | Vuetify 4 (Material Design) | ^4.1.7 |
| ルーティング | Vue Router 5 | ^5.2.0 |
| 国際化 | vue-i18n 11 | ^11.4.8 |
| ビルドツール | Vite 8 | ^8.2.0 |
| PWA | vite-plugin-pwa + Workbox | ^1.2.0 |
| TypeScript | TypeScript 6 | ~6.0.0 |
| アイコン | @mdi/font (Material Design Icons) | ^7.4.47 |
| 地図 | vue3-google-map + @googlemaps/js-api-loader | — |
| 日時 | moment | ^2.30.1 |
| Markdown描画 | marked + dompurify | ^18.0.7 / ^3.4.12 |
| 図表描画 | mermaid | ^11.16.0 |
| 型チェック | vue-tsc 3 | ^3.3.9 |
| リンター | ESLint 10 + eslint-plugin-vue 10 (flat config) | ^10.8.0 / ^10.10.0 |
| ユニットテスト | Vitest + jsdom | ^4.1.10 / ^30.0.1 |
| E2Eテスト | @playwright/test | ^1.62.1 |

> バージョンは `package.json` の値。表を更新するときは `package.json` を正とすること。

## 2. ディレクトリ構成

```
src/client/
├── App.vue                          # ルートコンポーネント
├── main.ts                          # エントリポイント
├── i18n.ts                          # i18n設定
├── serviceWorker.ts                 # PWA Service Worker
├── env.d.ts                         # TypeScript環境型定義
├── classes/
│   ├── api/
│   │   ├── gkill-api.ts            # APIクライアント シングルトン (~3,300行)
│   │   ├── gkill-api-response.ts   # レスポンス型
│   │   ├── find_query/             # 検索クエリビルダー
│   │   └── req_res/                # リクエスト/レスポンス型 (172ファイル、サーバー側は186ファイル)
│   ├── datas/                       # TypeScriptデータモデル（Go構造体のミラー）
│   ├── dto/                         # データ転送オブジェクト
│   ├── kftl/                        # KFTLパーサー (48 ステートメント型、日本語/ASCII両プレフィックス対応。ASCII定数とヘルパーは kftl-prefixes.ts)
│   ├── dnote/                       # Dnote集計ユーティリティ（トレンドグラフ集計 dnote-trend-aggregator.ts・dnote-trend/ 含む）
│   ├── lantana/                     # 気分値関連クラス
│   ├── cascade-delete-kyou.ts       # Kyou削除の連鎖削除（Tag/Text/Notification/ReKyou/MiReKyou）
│   ├── component-ref.ts             # ComponentRef 型（any をここに封じ込める）
│   ├── kyou-content-text.ts         # Kyou の内容/IDのクリップボードコピー
│   ├── kyou-view-relay.ts           # Kyou系CRUDイベントの中継ハンドラ束（crudRelayHandlers）
│   ├── markdown-to-html.ts          # marked + dompurify による Markdown 描画
│   ├── mermaid-render.ts            # Mermaid 図の描画
│   ├── tag-struct.ts                # タグ階層構造ユーティリティ
│   ├── long-press.ts                # v-long-press カスタムディレクティブ
│   ├── looks-like-url.ts            # URL判定ユーティリティ
│   └── use-*.ts                     # Composition関数群（コンテキストメニュー等、255ファイル）
├── assets/                          # 画像等の静的アセット
├── __tests__/                       # Vitest ユニットテスト + Playwright E2E
│   ├── e2e/                        # E2E spec（run-e2e.mjs / free-port.mjs / auth.setup.ts 等を含む）
│   ├── helpers/                    # テストヘルパー
│   └── unit/                       # ユニットテスト（api / classes / composables / datas / dnote / kftl。直下に i18n-completeness / router / service-worker）
├── pages/                           # ルートページコンポーネント (15ファイル)
│   ├── login-page.vue
│   ├── kftl-page.vue
│   ├── kyou-page.vue
│   ├── mi-page.vue
│   ├── rykv-page.vue
│   ├── saihate-page.vue
│   ├── mkfl-page.vue
│   ├── plaing-time-is-page.vue
│   ├── dashboard-page.vue
│   ├── set-new-password-page.vue
│   ├── register-first-account-page.vue
│   ├── shared-page.vue
│   ├── old-shared-mi-page.vue
│   ├── shared-mi-page.vue
│   ├── shared-rykv-page.vue
│   ├── views/                       # Viewコンポーネント (197)
│   └── dialogs/                     # ダイアログコンポーネント (111, Esc閉じ対応)
├── plugins/
│   └── vuetify.ts                   # Vuetify設定・テーマ定義
└── router/
    └── index.ts                     # ルート定義
```

## 3. コンポーネント階層

### 3層構造: Page → View → Dialog

```
Page（ルートページ）
  └── View（表示・操作コンポーネント）
       └── Dialog（モーダルダイアログ）
```

| 層 | 配置 | 件数 | 責務 |
|---|---|---|---|
| **Page** | `pages/*.vue` | 15 | ルーティング先。ページ全体のレイアウト（13ルート＋共有用2ページ） |
| **View** | `pages/views/*.vue` | 197 | データ型ごとの追加/編集/一覧表示 |
| **Dialog** | `pages/dialogs/*.vue` | 111 | モーダル操作（確認、詳細編集等） |

### 命名規則

- Page: `{機能名}-page.vue`（例: `kftl-page.vue`, `mi-page.vue`）
- View: データ型 + 操作で命名（例: `add-dnote-item-view.vue`, `kyou-list-view.vue`）
- Dialog: 操作内容で命名（例: `confirm-delete-ryuu-item-dialog.vue`）

### プラグインビューコンポーネント

プラグインKyou（`typed_plugin` フィールドが非 null のKyou）の表示には専用コンポーネントを使用する:

| コンポーネント | 説明 |
|---|---|
| `plugin-html-view.vue` | `GetPluginContentHTMLResponse.html` を iframe srcdoc に展開して表示。postMessage でダークテーマ通知・高さ動的調整を行う |
| `plugin-html-context-menu.vue` | プラグインKyou用コンテキストメニュー（タグ追加・テキスト追加・リポスト等）|
| `plugin-config-dialog.vue` | `GetPluginConfigHTMLResponse.html` を iframe srcdoc に展開してプラグイン設定フォームを表示。`plugin-html-view.vue` が保持し、コンテキストメニューの「プラグイン設定」から開く。保存は iframe からの postMessage を受けて親が `post_plugin_config` を呼ぶ |

**postMessage 通信パターン:**
- 親 → iframe: `{ gkill_theme: 'dark' | 'light' }` — テーマ変更通知（CSS変数切替用）
- iframe → 親: `{ gkill_iframe_size: { width, height } }` — コンテンツサイズ通知（iframe高さ自動調整用）

### MiReKyou コンポーネント

既存の Kyou をタスク化する MiReKyou（`typed_mirekyou`）の追加・編集・表示を担う。
タイトルを持たず、`target_id` で指す対象 Kyou を併せて描画するのが Mi との違い。

| コンポーネント | 説明 |
|---|---|
| `mi-re-kyou-view.vue` | MiReKyou 本体の表示。対象 Kyou を `attached_kyou` として読み込んで描画する |
| `mi-re-kyou-context-menu.vue` | MiReKyou 用コンテキストメニュー |
| `add-mi-re-kyou-view.vue` / `add-mi-re-kyou-dialog.vue` | タスク化（追加） |
| `edit-mi-re-kyou-view.vue` / `edit-mi-re-kyou-dialog.vue` | 編集 |

スケジュール項目（期限・見積開始・見積終了）の入力は `use-mi-re-kyou-schedule-fields.ts` に共通化されている。
データモデルは `classes/datas/mi-re-kyou.ts`、時刻フィールドはいずれも `Date | null`。

### Dnoteトレンドグラフ コンポーネント

Dnote（集計ビュー）の時系列トレンドグラフ機能を構成するコンポーネント群。バックエンドAPIを持たず、取得済みKyouをクライアント側で `DnoteTrendAggregator`（`classes/dnote/dnote-trend-aggregator.ts`）が日/週/月粒度で集計し、スパークライン（折れ線/棒）として描画する:

| コンポーネント | 説明 |
|---|---|
| `dnote-trend-graph-view.vue` | トレンドグラフ本体。スパークライン描画・ツールチップ表示 |
| `dnote-trend-graph-table-view.vue` | 集計値のテーブル表示 |
| `dnote-trend-graph-context-menu.vue` | 右クリックメニュー（編集/削除） |
| `add-dnote-trend-graph-view.vue` / `add-dnote-trend-graph-dialog.vue` | トレンドグラフ追加 |
| `edit-dnote-trend-graph-view.vue` / `edit-dnote-trend-graph-dialog.vue` | トレンドグラフ編集 |
| `confirm-delete-dnote-trend-graph-view.vue` / `confirm-delete-dnote-trend-graph-dialog.vue` | 削除確認 |

### Dnote相関グラフ コンポーネント

取得済みKyouと既存の `DnoteTrendAggregator` を再利用し、2～10指標の相関ヒートマップと選択セルの散布図をクライアント側で表示する。追加・編集・削除は共通の相関グラフダイアログを使い、設定は `dnote_json_data` 内へ保存する。

**iframe セキュリティ:** `sandbox="allow-scripts allow-forms"`（`allow-same-origin` なし）でセッションCookieを隔離する。

### ダイアログ アクセシビリティ

115ダイアログ中90件が `useFloatingDialog()` Composition関数（`src/client/classes/use-floating-dialog.ts`）を共有し、以下のアクセシビリティ機能を提供する。残りは別機構（`useDialogHistoryStack` 等）を用いる（例: `plugin-config-dialog.vue`）:

| 機能 | 説明 |
|------|------|
| **Escape キー閉じ** | Escape キーで `onEscape` コールバックを呼び出しダイアログを閉じる |
| **ARIA属性** | `role="dialog"`, `aria-modal="true"`, `aria-labelledby`（`.gkill-floating-dialog__title` 要素を参照、見つからない場合は `aria-label` にフォールバック） |
| **自動フォーカス** | 開いたときに本文の最初のテキスト入力欄へフォーカスする（`opts.autofocus`、既定 true） |

**自動フォーカスの選び方**（`src/client/classes/dialog-autofocus.ts`）: 探索は `.gkill-floating-dialog__body` の中だけ。ヘッダには透過トグルの `v-checkbox` と×ボタンが常に先頭にあるため、ルートから探すと必ずそれを掴む。既に `autofocus` を書いた要素があるダイアログでは何もせず Vuetify に任せる。`readonly`（日付ピッカーの見せかけ入力）・`disabled`・非表示・`v-selection-control` 配下のチェックボックス・`inputmode="none"` の `v-select` は候補から外す。入力欄は内側の `v-if` でデータ待ちのことが多いので、`MutationObserver` で生えてくるのを2秒だけ見張って一度だけ当てる。自前でフォーカス先を決めているダイアログ（`save-clipboard-to-file-dialog.vue` は保存ボタン）は `autofocus: false` で切る。

※ フォーカストラップ（Tab循環）とフォーカス復帰は v1.1.0 で削除されたまま。自動フォーカスのみ復活させた。

### ダイアログの重なり順

同じ種類のダイアログを複数枚同時に開ける（`rykv-dialog-host` 系、メモ帳ダイアログ）。
重なり順は `use-floating-dialog.ts` のモジュールレベルの `z_order`（末尾が最前面）から出す:

- z-index は `1100 + z_order 内の位置`。**単調増加のカウンタにしてはいけない** ―― Vuetify の
  overlay（メニュー / ツールチップ）が 2400 なので、上へ伸ばし続けるとダイアログの中の
  メニューがダイアログの下へ潜る。この方式なら伸びるのは同時に開いている枚数ぶんだけ
- 触った（`pointerdown` / `focusin`）ダイアログが最前面へ出る。**そのとき自分から開いた
  子孫のダイアログも一緒に上げる** ―― 確認ダイアログは `Teleport to="body"` で親の兄弟に
  なるので、連れていかないと親をクリックしただけで確認が後ろへ隠れる。親子は
  `provide`/`inject` で持つ（コンポーネント木は Teleport をまたいでも保たれる）
- ブラウザバックと Escape が閉じるのは**見た目の最前面**。`use-dialog-history-stack.ts` の
  `stack` は積んだ順なので、前面化のたびに `raise_dialog_history_entries()` で同じ順へ
  並べ替える。2つのコンポーザブルは同じコンポーネントの setup で呼ばれるので、
  `getCurrentInstance()` を鍵にして結んでいる
- 同じ種類を複数枚出すときは、`useFloatingDialog` の保存キー（`${key}:pos` / `:size` /
  `:transparent`）をインスタンスごとに分け、`centerOffset` でずらすこと。全ダイアログが
  `centerMode: "always"` なので、分けないとピクセル単位で完全に重なって位置を奪い合う

### アラート アクセシビリティ

各ページの `v-alert` コンポーネントは `message.is_error` に応じてスクリーンリーダーへの通知方式を切り替える:

- **エラーメッセージ**: `role="alert"`（暗黙的に `aria-live="assertive"`）— 即座に読み上げ
- **通常メッセージ**: `role` なし — コンテナの `aria-live="polite"` に従い、現在の操作を中断せずに読み上げ

## 4. ルート一覧

定義: `src/client/router/index.ts`（Web History モード）。各ルートの詳細は [screen-transition.md](screen-transition.md) および [screen-specs.md](screen-specs.md) を参照。

| パス | 名前 | コンポーネント | 説明 |
|---|---|---|---|
| `/` | `login` | login-page.vue | ログイン画面 |
| `/kftl` | `kftl` | kftl-page.vue | KFTLテキスト入力 |
| `/mi` | `mi` | mi-page.vue | タスク管理 |
| `/rykv` | `rykv` | rykv-page.vue | ライフログビュー（Ryuu「関連情報」はこの画面内の機能） |
| `/kyou` | `kyou` | kyou-page.vue | 記録詳細 |
| `/mkfl` | `mkfl` | mkfl-page.vue | 打刻メモ帳（KFTL入力+TimeIs表示） |
| `/plaing` | `plaing` | plaing-time-is-page.vue | アクティブ打刻一覧 |
| `/dashboard` | `dashboard` | dashboard-page.vue | 日次サマリー（Dnote・GPS・MI一覧） |
| `/saihate` | `saihate` | saihate-page.vue | 記録特化画面（他画面への遷移なし） |
| `/set_new_password` | `set_new_password` | set-new-password-page.vue | パスワード設定 |
| `/register_first_account` | `register_first_account` | register-first-account-page.vue | 初回アカウント登録（旧 `/regist_first_account` からリダイレクト） |
| `/shared_page` | `shared_page` | shared-page.vue | 共有ページ |
| `/shared_mi` | `shared_mi` | old-shared-mi-page.vue | 共有タスク |

## 5. 状態管理

### Pinia/Vuex は使用しない

gkill では **Props/Emit パターンのみ** で状態管理を行う。

| 方式 | 用途 |
|---|---|
| `props` / `emit` | 親子コンポーネント間のデータ受け渡し |
| `ref()` / `reactive()` | コンポーネントローカルの状態 |
| `GkillAPI` シングルトン | バックエンド通信（`GkillAPI.get_instance()`） |
| Vuetify `useTheme()` | テーマ状態（ライト/ダーク切替） |
| vue-i18n | ロケール状態 |
| `use-*.ts` Composition関数 | コンテキストメニュー等の共有ロジック（255ファイル） |

### ComponentRef 型

子コンポーネントへの `ref` は `ComponentRef` 型（`src/client/classes/component-ref.ts`）を使う。

```typescript
export type ComponentRef = Record<string, any>
```

`@typescript-eslint/no-explicit-any` を error にしているため、`any` をこの1ファイルに封じ込めるための型。
テンプレート ref 経由で子の公開メソッドを呼ぶ箇所（約50のコンポーザブル）で使われる。

JSON からクラスインスタンスへの詰め替えは `classes/api/hydrate.ts` の `hydrate()` / `hydrate_all()` に集約されており、
`gkill-api.ts` と `datas/kyou.ts` はこれを使うことでファイル全体の `eslint-disable` を解消している。

### Kyou系イベントの中継束（crudRelayHandlers）

Props/Emit のみで状態を持ち回すため、Kyou の CRUD イベントは View → Dialog → Page と手で中継する必要がある。この中継束は `classes/kyou-view-relay.ts` に共通化されている。

| 関数 | 中継件数 | 用途 |
|---|---|---|
| `build_kyou_view_relay(emits, overrides?)` | 18 | ビュー層。`focused_kyou`/`clicked_kyou` は含めない（発火源なので二重発火する） |
| `build_kyou_dialog_relay(emits, overrides?)` | 20 | ダイアログ層。上の18件＋`focused_kyou`/`clicked_kyou` |

返り値はそのまま `v-on="crudRelayHandlers"` に渡せる。`requested_close_dialog` は中継対象外で、ダイアログが `@requested_close_dialog="hide()"` で自分に繋ぐ。中継イベントを増やすときは `KyouViewRelayArgs` と `kyou_view_relay_event_names` の両方に足すこと（片方だけだとコンパイルエラーになる）。

### GkillAPI シングルトン

`src/client/classes/api/gkill-api.ts` に定義。約3,300行。

- `GkillAPI.get_instance()` / `GkillAPI.get_gkill_api()` でインスタンス取得
- 全90登録エンドポイントに対応するメソッドを持つ（`gkill-api.ts` が保持する `/api/` アドレスは88件）
- `GkillAPIForSharedKyou` サブクラス（共有データ用）
- 各メソッドは `fetch()` → JSONパース → エラーチェック → データ返却

## 6. PWA構成

### Service Worker

定義: `src/client/serviceWorker.ts`

**Workbox設定:**
- `registerType: 'autoUpdate'`（自動更新）
- `strategies: 'injectManifest'`（カスタムService Worker）
- 最大キャッシュファイルサイズ: 10MB

**キャッシュ戦略:**

| キャッシュ名 | 対象 | 説明 |
|---|---|---|
| `gkill-post-kyou-cache` | データ系エンドポイント | get_kyou, get_kmemo, get_plugin_content_html 等のPOSTレスポンスをキャッシュ |
| `gkill-post-config-cache` | 設定系エンドポイント | get_application_config, get_all_tag_names 等 |
| `gkill-lazy-chunk-cache` | 遅延ロードされる JS チャンク | `CacheFirst`。動的 import されたチャンクをキャッシュする |

**キャッシュ有効性検証:**
- `_histories` フィールドの存在チェック
- `errors` 配列が空であること
- `force_reget` パラメータでキャッシュバイパス可能

**キャッシュキー識別子:**
- 通常の Kyou データ: `body.id` または `body.target_id` をキャッシュキーの識別子として使用
- プラグインコンテンツ HTML（`/api/get_plugin_content_html`）: `body.kyou_id` を使用（命名が異なる点に注意）
  - キャッシュキー: `/cache/api/plugin_content_html/{kyou_id}`

**SPAフォールバック:**
- `/`、`/api/*`、`/files/*`、`/zip_cache/*`、`/resources/manual/*`、`/share-target` 以外の全パスを `index.html` にフォールバック（`/resources/manual/*` はヘルプHTMLの実体なので除外、`/share-target` は専用ハンドラが `respondWith` する）
- `/zip_cache/.*` パターンは Service Worker の denylist に追加されており、キャッシュされない

### Web Share Target

Service Worker が `/share-target` POSTを処理：
1. 重複台帳（後述）に同じ内容が期間内に残っていれば保存せず `/saihate?share_result=duplicate&share_entry_id=...` にリダイレクト
2. 共有されたテキストからURLを検出（`decide_share_save_target`）
3. URL → URLog（ブックマーク）として保存
4. テキストのみ → Kmemo（メモ）として保存
5. 保存に成功したときだけ台帳へ追記
6. 保存結果に応じて `/saihate?is_saved=true/false` にリダイレクト

**重複台帳（`src/client/classes/share-target-dedup.ts`）**

Android はタスク（アプリ履歴）から復帰すると同じ共有インテントを再配送するため、初回とまったく同じ
multipart POST がもう一度届く。届く内容が同一なので、再配送と「利用者が意図的にもう一度共有した」を
区別する手段は「保存済みの内容を覚えておく」以外に無い。

- 保管先: Cache Storage の `gkill-share-dedup-cache` / キー `/__gkill_share_dedup/ledger`（JSON配列1件）
- 同一判定: `title` / `text` / `url` の3つが完全一致、かつ保存から24時間以内。台帳は直近100件まで
- 台帳へ載せるのは保存が成功したときだけ（HTTP ok かつ `errors` 空を `is_successful_gkill_response` で確認）
- 重複と判定された共有は保存されず、最果てが `confirm-save-duplicated-shared-data-dialog.vue` で
  「それでも保存する」かを訊く。押されるとフォームに `gkill_force` を立てて同じ `/share-target` を叩き直し
  （保存の実装を2つに割らないため）、応答は 303 ではなく `{ "is_saved": boolean }` の JSON で返る
- 最果ては `is_saved` / `share_result` を読んだ直後にクエリを落とす。残すとアプリ履歴からの復帰で
  「保存しました」が再表示され、保存していないのに二重保存に見える

### Push通知

- `is_notification` / `is_updated_data_notify` イベントを受信
- ブラウザ通知を表示（タイムスタンプ、カスタムアクション付き）

## 7. テーマ

定義: `src/client/plugins/vuetify.ts`

### ライトテーマ (`gkill_theme`)

| 色名 | 値 | 用途 |
|---|---|---|
| primary | `#2672ed` | メインカラー（青） |
| secondary | `#999999` | 補助カラー（グレー） |
| background | `#ffffff` | 背景（白） |
| success | `#4CAF50` | 成功 |
| warning | `#FB8C00` | 警告 |
| error | `#B00020` | エラー |
| highlight | `#8cffbe` | ハイライト |

### ダークテーマ (`gkill_dark_theme`)

| 色名 | 値 | 用途 |
|---|---|---|
| primary | `#2672ed` | メインカラー（青、同一） |
| secondary | `#999999` | 補助カラー（グレー、同一） |
| background | `#212121` | 背景（ダークグレー） |
| success | `#218025` | 成功 |
| warning | `#9e5800` | 警告 |
| error | `#7a0117` | エラー |
| highlight | `#60ab80` | ハイライト |

### テーマ切替

- `App.vue` でテーマを管理
- ライトテーマ時: `SaihateStarsOverlay` 非表示
- ダークテーマ時: `SaihateStarsOverlay` 表示（星空効果）
- ライトテーマ時: `SnowFallOverlay` 表示（雪効果）

## 8. 国際化 (i18n)

定義: `src/client/i18n.ts`

| 設定 | 値 |
|---|---|
| デフォルトロケール | `ja`（日本語） |
| フォールバック | `ja` |
| モード | `legacy: true`, `globalInjection: true` |

### 対応言語（7言語）

| コード | 言語 | キー数 |
|---|---|---|
| `ja` | 日本語 | 858 |
| `en` | 英語 | 858 |
| `zh` | 中国語 | 858 |
| `ko` | 韓国語 | 858 |
| `es` | スペイン語 | 858 |
| `fr` | フランス語 | 858 |
| `de` | ドイツ語 | 858 |

> キー数は全ロケールで一致している必要がある（`npm run verify_docs` が検査する）。

### ロケールファイル

- 配置: `src/locales/*.json`
- 形式: フラットなキー・バリュー（ネストなし）
- キー例: `LOGIN_TITLE`, `RESET_PASSWORD_TITLE`, `SAIHATE_PAGE_TITLE`
- **フロント・バックエンド共有:** ビルド時に `src/locales/` → `src/server/gkill/api/embed/i18n/locales/` にコピー

### 読み込み方式（ja のみ静的バンドル）

`src/client/i18n.ts` はメインチャンクに **`ja` だけを静的 import** し、残り6言語は
`locale_loaders` 経由で**動的 import** する。`set_locale()` が呼ばれた時点で該当ロケールを取得する。

メインチャンクを約 352KB 削減するための意図的な最適化なので、
「全ロケールを最初に読む」形に戻さないこと。

> i18n キーを追加したら `src/locales/` を Go の embed 先へコピーする必要がある。
> 忘れるとバックエンドの `MustLocalizeMessage` が panic する。
> 通常は `npm run install_server`（`prepare_install`）が自動で行う。

## 9. ビルド設定

### Vite設定 (`vite.config.ts`)

| 項目 | 値 |
|---|---|
| パスエイリアス | `@/` → `./src/client/` |
| minify | `true`（デフォルト）。デバッグで読めるJSが欲しいときだけ環境変数 `MINIFY=false` で無効化する（既定offにすると `install_server` / `install_app` が未minifyの約5MBのJSをGoバイナリに埋め込んでしまう） |
| Vue devtools | プロダクションでも有効（`__VUE_PROD_DEVTOOLS__: true`） |
| propsDestructure | 有効 |

### TypeScript設定

- `tsconfig.app.json`: フロントエンド用（`src/client/**/*` + `public/sw.js`）
- `tsconfig.node.json`: ビルドツール用（`@tsconfig/node20` 継承）
- lib: `WebWorker`（Service Worker用）
- types: `google.maps`, `vite-plugin-pwa/client`

### ESLint設定 (`eslint.config.js`)

- flat config 形式（ESLint 10+）
- `pluginVue.configs['flat/essential']` + `vueTsEslintConfig()` を使用
- `@typescript-eslint/no-unused-vars`: warn（`_` プレフィックスは無視）
- `@typescript-eslint/no-explicit-any`: error
- `@typescript-eslint/no-empty-object-type`: error
- `@typescript-eslint/naming-convention`: error（対象は `src/client` 本体のみ。`__tests__` / `*.d.ts` は別流儀として対象外）
- 実行: `npm run lint`

## 10. プラグイン HTML ビュー

### コンポーネント構成

| ファイル | 役割 |
|---|---|
| `pages/views/plugin-html-view.vue` | メイン View（iframe srcdoc でプラグイン HTML を表示） |
| `pages/views/plugin-html-view-props.ts` | `PluginHtmlViewProps`（`KyouViewPropsBase` を拡張） |
| `pages/views/plugin-html-context-menu.vue` | コンテキストメニュー（GitCommitLog と同項目） |
| `pages/views/plugin-html-context-menu-props.ts` | `PluginHtmlContextMenuProps`（`= KyouViewPropsBase`） |
| `classes/use-plugin-html-view.ts` | View レベル Composable（コンテキストメニュー表示・crudRelayHandlers） |
| `classes/use-plugin-html-context-menu.ts` | コンテキストメニュー Composable |

### iframe テーマ連携

- `application_config.use_dark_theme` を `watch` し、変更時に iframe へ postMessage で通知
  - メッセージ形式: `{ gkill_theme: 'dark' | 'light' }`
  - 送信タイミング: iframe の `onload` イベント + `use_dark_theme` 変更時
- `sandbox="allow-scripts allow-forms"`（`allow-same-origin` を付けないことでセッション Cookie アクセスを禁止）
- iframe は `scrolling="no"` で自身のスクロールバーを非表示にし、スクロールを親コンポーネントに委譲

### iframe 高さ自動調整

- iframe 内コンテンツが `{ gkill_iframe_size: { width, height } }` を postMessage で送信
- 親は `e.source === iframe_ref.value.contentWindow` を検証してから `iframe_content_height` を更新
- `iframe height = iframe_content_height + 'px'`（コンテンツ高さ確定前は `80px` フォールバック）

---

## 11. ダッシュボード機能

### DashboardConfig クラス

定義: `src/client/classes/datas/config/dashboard-config.ts`

ダッシュボード画面の表示設定を保持する設定クラス。`ApplicationConfig.dashboard_json_data` にJSON文字列として格納され、デバイス非依存（DEVICE='ALL'）で保存される。

| フィールド | 型 | 説明 |
|---|---|---|
| `dashboard_mi_find_kyou_query` | `FindKyouQuery \| null` | ダッシュボード下部のMI一覧に適用する検索クエリ |
| `dashboard_dnote_find_kyou_query` | `FindKyouQuery \| null` | ダッシュボード上半分左のDnoteビューに適用する検索クエリ |

**静的メソッド:**

| メソッド | 説明 |
|---|---|
| `static parse(json: unknown): DashboardConfig` | JSONオブジェクトからDashboardConfigインスタンスを生成。旧フィールド `dashboard_default_find_kyou_query` を `dashboard_mi_find_kyou_query` へ後方互換マイグレーションする |
| `to_json(): Record<string, unknown>` | JSONシリアライズ用オブジェクトを返す |

**後方互換性:** 旧バージョンで保存された `dashboard_default_find_kyou_query` フィールドは、`parse()` 実行時に `dashboard_mi_find_kyou_query` として読み込まれる。

### ダッシュボードページ

定義: `src/client/pages/dashboard-page.vue` / `src/client/classes/use-dashboard-page.ts`

特定日のライフログを俯瞰する日次サマリー画面。`/dashboard` ルートに対応する。

**画面レイアウト:**

```
┌──────────────────────────────────────┐
│  ツールバー（日付ナビ・設定・ヘルプ）            │
├─────────────────┬────────────────────┤
│  DnoteView      │  GPSLogMap         │
│  （30vh、左半分） │  （30vh、右半分）     │
├─────────────────┴────────────────────┤
│  KyouListView（MI一覧、残り全高さ）           │
│  ※ dashboard_mi_find_kyou_query で絞込    │
└──────────────────────────────────────┘
                              [FABメニュー]
```

**主要機能:**
- 表示日の切替（前日・翌日・カレンダーピッカー）
- EditDashboardDialog 経由でMI検索条件・Dnote検索条件を設定
- FABメニューから全データ型の記録追加が可能
- Enter → KFTLダイアログ、Ctrl+V → クリップボード保存ダイアログ

### MiFindQueryEditorView / MiFindQueryEditorDialog

定義:
- `src/client/pages/views/mi-find-query-editor-view.vue` / `mi-find-query-editor-view-props.ts` / `mi-find-query-editor-view-emits.ts`
- `src/client/pages/dialogs/mi-find-query-editor-dialog.vue` / `mi-find-query-editor-dialog-props.ts` / `mi-find-query-editor-dialog-emits.ts`
- `src/client/classes/use-mi-find-query-editor-view.ts`
- `src/client/classes/use-mi-find-query-editor-dialog.ts`

既存の `find-query-editor-view.vue` が汎用Kyou向けであるのに対し、`mi-find-query-editor-view.vue` はMI（タスク）専用の検索条件エディタ。ダッシュボードのEditDashboardDialogから呼び出される。

**対応フィルタ条件:**

| フィルタ | 説明 |
|---|---|
| チェック状態（`mi-extract-check-state-query`） | 完了/未完了/全件での絞り込み |
| ソート順（`mi-sort-type-query`） | 並び順の選択 |
| キーワード | 全文検索 |
| タグ | タグでの絞り込み |

打刻期間（TimeIs）・GPS位置（map-query）・時間帯の3節は、Mi のサイドバー（`mi-query-editor-sidebar.vue`）と揃えて画面から外してある。`generate_query()` は毎回 `new FindKyouQuery()` から組み直すため、該当フィールドは常に既定値（すべてOFF）になる。

**クエリ反映タイミング:** 「保存」ボタン押下時のみ `emits('requested_apply', query)` を発行し、クエリを親コンポーネントに反映する（リアルタイム反映なし）。

### EditDashboardDialog

定義: `src/client/pages/dialogs/edit-dashboard-dialog.vue` / `edit-dashboard-dialog-props.ts` / `edit-dashboard-dialog-emits.ts` / `src/client/classes/use-edit-dashboard-dialog.ts`

ダッシュボードの表示設定を編集するダイアログ。`DashboardConfig` の各フィールドを編集し、保存時に `UpdateApplicationConfig` API を呼び出して永続化する。

| 設定項目 | 内容 |
|---|---|
| MI検索条件 | `MiFindQueryEditorView` でMI一覧の絞り込み条件を設定 |
| Dnote検索条件 | `FindQueryEditorView` でDnoteビューの条件を設定 |

### PlaingTimeIsConfig クラスと実行中検索条件（ダッシュボード設定と同型）

定義:
- `src/client/classes/datas/config/plaing-time-is-config.ts`（設定クラス。`plaing_timeis_find_kyou_query: FindKyouQuery | null` を1本保持し、`parse()` / `to_json()` を持つ。null は「未設定＝従来どおり全リポジトリ対象」）
- `src/client/classes/api/find_query/generate-plaing-timeis-query.ts`（適用の実体 `generate_plaing_timeis_query()`。GkillAPI 非依存の同期純関数）
- `src/client/pages/dialogs/edit-plaing-time-is-dialog.vue` + `src/client/classes/use-edit-plaing-time-is-dialog.ts`（中間ダイアログ。Ryuu の関連情報アイテムと同じ「☑検索条件をカスタマイズする ＋ [検索条件]」形式で、チェックを外すと null＝未設定に戻る。`is_use_custom_find_kyou_query` は `current_query !== null` の computed get/set。dashboard版と違い emit は Save 時のみでキャンセルで破棄される）
- `src/client/pages/views/find-time-is-query-editor-view.vue` + `src/client/pages/dialogs/find-time-is-query-editor-dialog.vue`（+ 各 props/emits / use-*。Mi 版と同型の専用エディタ。編集面はキーワード・タグ絞り込みトグル・タグの3ブロック。記録保管場所と記録タイプは選ばせず、`generate_query()` が `rep_types=["timeis"]` を立てる）

plaing検索（Kyou付随の実行中表示 `info-base.ts` の `load_attached_timeis()`・実行中画面・KFTLの/end系終了候補検索 `generate-get-plaing-timeis-kyous-query.ts`）の検索条件を `ApplicationConfig.plaing_timeis_json_data`（EAVキー `PLAING_TIMEIS_JSON_DATA`、DEVICE='ALL'）でカスタマイズできる。3経路すべてが `generate_plaing_timeis_query()` を通る。

**適用の意味論:** 保存クエリからコピーされるのは明示リストの6フィールドだけ（`keywords` / `words_and` / `words` / `not_words` / `tags` / `tags_and`。エディタの編集面と1:1。nullable値はnullガード付きコピー）。`plaing_time`（非null）と `rep_types=["timeis"]` は呼び出し元が常に強制上書きする（記録タイプはカスタム条件の有無によらずTimeIs固定。サーバのタイプ系フィルタは和集合で、`plaing_time` 非nullが既にTimeIsのrepへ絞っているため結果は変わらない冪等な明示）。**rep名での絞り込みは常に切る**（`reps=null`）―― 記録保管場所はエディタから外したので保存JSONの `reps` は無視するが、`new FindKyouQuery()` の既定が `reps=[]`（有効・0件指定）なので明示的にnullへ倒さないとサーバのrep名絞り込み（`find_filter.go` Step4）で常に0件になる。カスタム条件適用時のみ `apply_hide_tags` で非表示タグを反映する（未設定時は従来どおり適用しない）。カスタム条件で候補を絞ると、条件外の実行中TimeIsはKFTLの/endで終了できない（仕様）。Wear OS とサーバ内 KFTL の plaing 検索は別系統のため、この設定は Web クライアントにのみ効く。

### SavedFindQueryConfig クラスと保存済み検索条件（ダッシュボード設定と同型）

定義:
- `src/client/classes/datas/config/saved-find-query-config.ts`（設定クラス。`saved_rykv_find_kyou_querys` / `saved_mi_find_kyou_querys` の2リストを保持し、`parse()` / `to_json()` / `clone()` / `clone_items()` を持つ。各アイテムは `SavedFindQueryItem { id, title, find_kyou_query }`（Ryuu の関連情報クエリと同形式）。`parse()` は null・非オブジェクト・不正アイテムを空/除外にフォールバックする＝初回起動考慮）
- `src/client/pages/dialogs/edit-saved-find-query-dialog.vue` + `src/client/classes/use-edit-saved-find-query-dialog.ts`（ハブダイアログ。「ライフログ検索条件」「タスク検索条件」の2ボタン。edit-dashboard-dialog と同型）
- `src/client/pages/dialogs/edit-saved-find-query-list-dialog.vue` + `src/client/classes/use-edit-saved-find-query-list-dialog.ts`（一覧管理ダイアログ。**1コンポーネントを `query_type: 'rykv' | 'mi'` prop で2インスタンス化**。`useFloatingDialog` の storage_key に query_type を含めて位置/サイズ保存を分離。行UIは名前 text-field＋「検索条件を編集」＋上へ/下へ/削除、追加は他画面と揃えた右下FAB（`.position-fixed-saved-find-query`＝`position: absolute`。内側に positioned な祖先を作らないので包含ブロックがスクロール箱の外側 `.gkill-floating-dialog` になり、一覧をスクロールしても右下に固定される）で既定クエリ `generate_default_query_for_rykv/for_mi` ＋既定名の行を足す。クエリ編集は既存 `find-query-editor-dialog.vue`（rykv用）/ `mi-find-query-editor-dialog.vue`（mi用）を再利用）

ユーザ定義の検索条件に名前を付けて何個でも保存し、rykv/mi のサイドバーから呼び出せる機能。`ApplicationConfig.saved_find_query_json_data`（EAVキー `SAVED_FIND_QUERY_JSON_DATA`、DEVICE='ALL'＝全端末共有）に保存される。

**適用の入れ子（キャンセルが全階層で効く）:** クエリエディタの適用→一覧ダイアログのローカル行→一覧の適用→ハブのローカル `current_config`→ハブの適用→`requested_apply_saved_find_query_struct` で `use-application-config-view.ts` の `cloned_application_config` へ→設定画面全体の「適用」で `update_application_config()` が一括送信（`location.reload()` で確定）。各段階で `clone()` を挟み、どの段階のキャンセルでも下位の編集が破棄される。

**サイドバーからの呼び出し:** `use-rykv-query-editor-side-bar.ts` / `use-mi-query-editor-sidebar.ts` の computed `saved_find_querys`（0件なら FAB ごと v-if 非表示）と `apply_saved_query(item)`。適用は `emits_default_query()` と同経路で、`item.find_kyou_query.clone()` に**列側の `query_id` を差し込んで** `query.value` を差し替え、`updated_query` を emit する（＝手編集と同じ扱い。ホットリロードONなら親が自動検索、OFFなら検索ボタンで実行）。保存条件由来の query_id を列へ持ち込むと「列×検索」不変条件が崩れるため、必ず列側を維持する。mi 側は全クリアと違い**板名も保存された条件が勝つ**。FAB はサイドバー下端の sticky 検索バー（`.sidebar_header_wrap`、`overflow: visible` 化）に絶対配置で載せている（position:fixed ではないのでドロワー閉時に一緒に隠れる）。

**制約:** ライフログ用エディタ（`find-query-editor-view.vue`）にはカレンダー節が無いため、保存条件はカレンダー未使用（`calendar_start_date`/`calendar_end_date`=null）で保存される（期間指定はエディタにあるので保存可能）。サイドバーへ反映されるのは「サイドバーに表示される条件」がすべてで、保存された条件がそのまま勝つ。タスク用は打刻期間・GPS位置・時間帯の3節がエディタにもサイドバーにも無いので、それらの絞り込みは保存も復元もされない（過去の保存条件に残っていても、サイドバーを次に触った時点の `generate_query()` で既定値へ落ちる）。

---

## 12. UX改善

### オフラインバナー

`App.vue` に `navigator.onLine` と `online`/`offline` イベントを監視するバナーを追加。ネットワーク切断時に `v-banner` でユーザーに通知する。

### ダイアログ＋ブラウザ履歴

`use-dialog-history-stack.ts` でダイアログの開閉をブラウザ履歴と連動：
- ダイアログ表示時に `history.pushState` で depth 付きエントリを追加
- ブラウザバックで最上位ダイアログを閉じる（depth比較でback/forwardを判別）
- ブラウザフォワードではダイアログを閉じない
- プログラマティック閉じ（Escape含む）時は履歴を巻き戻し

公開API:

| 関数 | 用途 |
|---|---|
| `useDialogHistoryStack(show)` | ダイアログの `show` ref を履歴スタックに登録する |
| `close_dialog_via_history()` | **プログラムからダイアログを閉じるときの唯一の正しい手段**。約44のコンポーザブルが使う。`show.value = false` を直接書くと履歴とずれる |
| `close_top_dialog()` | 最上位のダイアログだけを閉じる |
| `reset_dialog_history()` | 履歴スタックを初期化する。ページリダイレクト時に使う（例: `old-shared-mi-page.vue`） |

### 日付・数値のロケール対応

- `format_time()`: `Intl.DateTimeFormat` ベースでロケール別フォーマット + 曜日表示
- `format_number()`: `Intl.NumberFormat` ベースでロケール別桁区切り

### 空状態メッセージ

リスト表示（kyou-list-view, dnote-list-view 等）でデータ0件時に「表示できるデータがありません」メッセージを表示。

### フォームバリデーション

各追加ビュー（add-nlog, add-mi, add-ur-log, add-time-is, add-kc, add-tag, add-text, add-notification 等）の必須フィールドに `:rules` バリデーションを追加。Vuetify の `v-text-field` / `v-textarea` の `:rules` prop で空チェックを実装。

### レスポンシブデザイン

- `rykv-view.vue` / `mi-view.vue`: ナビゲーションドロワーの幅を `$vuetify.display.smAndDown` で画面幅に応じて切替（スマートフォンでは `100vw`）
- `rykv-view.vue`: 詳細ビュー（`.kyou_detail_view`）の `min-width` を `0` に変更し `max-width: 100vw` を追加。600px以下でフルワイド表示
- `kyou-list-view.vue`: `v-virtual-scroll` に `max-width: 100vw` を追加し、画像リスト幅が画面を超えないように制限

### スコープ付きキーボードショートカット Composable

各 View に対してスコープを限定したキーボードイベントリスナーを登録する Composable 群を `src/client/classes/` に定義している。

| Composable | ファイル | 説明 |
|---|---|---|
| `useScopedEnterForKFTL` | `use-scoped-enter-for-kftl.ts` | 対象 View にフォーカスがある状態で Enter キーを押すとメモ帳ダイアログを開く |
| `useScopedCtrlVForClipboard` | `use-scoped-ctrl-v-for-clipboard.ts` | 対象 View 内が最後にクリックされた状態で Ctrl+V を押すとクリップボード保存ダイアログを開く |

これらは `onMounted` / `onBeforeUnmount` でドキュメントレベルのイベントリスナーを管理し、`rykv-view.vue`・`mi-view.vue`・`plaing-time-is-view.vue` の各 composable から呼び出される。

### クリップボード保存ダイアログ（`save-clipboard-to-file-dialog.vue`）

`use-save-clipboard-to-file-dialog.ts` に主要ロジックを実装するフローティングダイアログ。既存の `POST /api/upload_files` を再利用し、ブラウザ Clipboard API で取得したデータを base64 変換してアップロードする。

| 機能 | 説明 |
|---|---|
| Clipboard API 読み取り | `navigator.clipboard.read()` で画像・テキスト・PDF 等を取得。優先度順（PNG > JPEG > ... > text/plain）で型を自動選択 |
| プレビュー | 画像は `URL.createObjectURL` でプレビュー表示、テキストは先頭5行を表示 |
| 保存済み判定 | SHA-256 ハッシュで前回保存と一致する場合は確認を促す。同一 Blob の二重保存を防止 |
| 連続保存 | 保存後もダイアログを閉じず、`save_btn` にフォーカスを戻して Enter で継続保存可能 |
| Ctrl+V ペースト | ダイアログが最後にクリックされた状態なら `paste` イベントを受け取り、クリップボードデータを自動更新して保存 |
| Enter キー保存 | テキスト入力フィールド外で Enter を押すと `save_or_confirm` を実行 |
| ファイル名自動生成 | `clipboard_YYYYMMDD_HHmmss.ext` 形式。折りたたみ UI で変更可能 |
| 保存先記憶 | 最後に選択した IDF リポジトリ名を `localStorage`（キー: `gkill_clipboard_save_last_rep_name`）に保存 |
| 「最後クリック」判定 | `mousedown` キャプチャリスナーで `.gkill-floating-dialog` を追跡し、ペースト・Enter 保存を自ダイアログのみに限定する |

### 未保存データ警告

KFTL テキストエリアに内容がある状態でページ離脱しようとすると `beforeunload` イベントで警告を表示。加えて、各ページ composable（use-rykv-page, use-mi-page, use-mkfl-page, use-plaing-time-is-page, use-kyou-page, use-saihate-page）にも `beforeunload` ガードを追加し、ダイアログ表示中やロード中のページ離脱を防止。

### 二重送信ガード

確認ビュー・編集ビュー・コンテキストメニュー（タグ履歴からのクイック付与など）の送信系ハンドラは `is_requested_submit` ref を持ち、処理中の再入を先頭で弾く。連打やEnter長押しで同じ登録・削除が二重に走るのを防ぐ。`classes/use-*.ts` の45ファイルが実装している。

### 確認ダイアログのクローズは finally で行う

削除・更新の確認ダイアログは、リクエスト送出後に例外が出てもクローズまで必ず到達させる。クローズ（`emits('requested_close_dialog')`）と `is_requested_submit` の解除は `try` の中ではなく `finally` に置く。`try` の末尾に置くと「サーバには届いていて実際は消えているのに、例外でダイアログが閉じない」状態になる。

### コンテキストメニューの位置決め（`use-context-menu-position.ts`）

25個の `*-context-menu.vue` は位置を自前で計算しない。共通 composable が返す `menu_target`（`[clientX, clientY]`）を `<v-menu v-model="is_show" :target="menu_target" location="bottom start">` に渡し、Vuetify の connected location strategy に任せる。メニューの実寸を測ったうえで、はみ出す方向へ flip / shift される。

以前は各 composable が `left: min(innerWidth - 130, x)` / `top: min(max(50, innerHeight - (8 + 48 * 項目数)), y)` を25箇所にコピペしていた。幅130px は実際のリスト幅（実測79px）と無関係で、高さの項目数はテンプレートと手で同期する不文律だったため、構成ツリー系（`*-struct-context-menu`）は実項目5個に対して `48 * 2` のまま下端ではみ出していた。またこのスタイル文字列は `{ }` で囲まれていたため Vue の `parseStringStyle` が `position: absolute` を捨てており、`.v-overlay`（`position: fixed`）の `left` / `top` だけが効いている状態だった。

`.gkill_context_menu_list { max-height: 70vh; overflow-y: scroll }`（`App.vue`）は残す。極端に項目が多いメニューの高さ上限として機能し、Vuetify はその上限込みの実高さに対して配置する。

### 読み込み中表示の遅延（`use-delayed-loading.ts`）

Kyou は種別を問わず表示のたびに API を1回叩くため、一覧では数十行が同時に読み込み中になる。ローカルサーバでは数十msで返るので、`useDelayedLoading(is_loading, delay_ms = 200)` を通し、読み込みが200msを超えたときだけインジケータを出す（`use-kyou-view.ts` の `show_loading_indicator`）。タイマーは `onScopeDispose` で片付けるので、仮想スクロールで行が使い回されても漏れない。

また KyouView は `kyou.id` が空の間（ReKyou / MiReKyou が参照先を取得している最中など）は日時を表示しない。`related_time` の初期値 `new Date(0)` がそのまま出ると 1970/01/01 が一瞬見えるため。参照先が見つからないときだけ終端メッセージ（`NOT_FOUND_REKYOU_TARGET_ERROR_MESSAGE`）を出す。
