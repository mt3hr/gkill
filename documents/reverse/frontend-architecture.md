# フロントエンド設計ガイド

## 1. 技術スタック

| カテゴリ | 技術 | バージョン |
|---|---|---|
| フレームワーク | Vue 3 (Composition API) | ^3.4.29 |
| UIライブラリ | Vuetify 3 (Material Design) | ^3.11.1 |
| ルーティング | Vue Router | ^4.3.3 |
| 国際化 | vue-i18n | ^9.14.4 |
| ビルドツール | Vite | ^5.3.1 |
| PWA | vite-plugin-pwa + Workbox | ^0.21.1 |
| TypeScript | TypeScript | ~5.4.0 |
| アイコン | @mdi/js (Material Design Icons) | ^7.4.47 |
| 地図 | vue3-google-map + @googlemaps/js-api-loader | — |
| 日時 | moment | ^2.30.1 |
| 型チェック | vue-tsc | ^2.0.21 |
| リンター | ESLint + eslint-plugin-vue | ^8.57.0 |

## 2. ディレクトリ構成

```
src/client/
├── App.vue                          # ルートコンポーネント
├── main.ts                          # エントリポイント
├── i18n.ts                          # i18n設定
├── serviceWorker.ts                 # PWA Service Worker (~279行)
├── env.d.ts                         # TypeScript環境型定義
├── classes/
│   ├── api/
│   │   ├── gkill-api.ts            # APIクライアント シングルトン (~3,400行)
│   │   ├── gkill-api-response.ts   # レスポンス型
│   │   ├── find_query/             # 検索クエリビルダー
│   │   └── req_res/                # リクエスト/レスポンス型 (150ファイル、サーバー側は164ファイル)
│   ├── datas/                       # TypeScriptデータモデル（Go構造体のミラー）
│   ├── dto/                         # データ転送オブジェクト
│   ├── kftl/                        # KFTLパーサー (44 ステートメント型)
│   ├── dnote/                       # DNote集計ユーティリティ
│   ├── lantana/                     # 気分値関連クラス
│   ├── long-press.ts                # v-long-press カスタムディレクティブ
│   ├── looks-like-url.ts            # URL判定ユーティリティ
│   └── use-*.ts                     # Composition関数群（コンテキストメニュー等）
├── pages/                           # ルートページコンポーネント (14ファイル)
│   ├── login-page.vue
│   ├── kftl-page.vue
│   ├── kyou-page.vue
│   ├── mi-page.vue
│   ├── rykv-page.vue
│   ├── saihate-page.vue
│   ├── mkfl-page.vue
│   ├── plaing-timeis-page.vue
│   ├── set-new-password-page.vue
│   ├── regist-first-account-page.vue
│   ├── shared-page.vue
│   ├── old-shared-mi-page.vue
│   ├── views/                       # Viewコンポーネント (175)
│   └── dialogs/                     # ダイアログコンポーネント (92, Esc閉じ対応)
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
| **Page** | `pages/*.vue` | 14 | ルーティング先。ページ全体のレイアウト（12ルート＋共有用2ページ） |
| **View** | `pages/views/*.vue` | 175 | データ型ごとの追加/編集/一覧表示 |
| **Dialog** | `pages/dialogs/*.vue` | 92 | モーダル操作（確認、詳細編集等） |

### 命名規則

- Page: `{機能名}-page.vue`（例: `kftl-page.vue`, `mi-page.vue`）
- View: データ型 + 操作で命名（例: `add-kmemo-view.vue`, `kyou-list-view.vue`）
- Dialog: 操作内容で命名（例: `confirm-delete-dialog.vue`）

### ダイアログ アクセシビリティ

全92ダイアログは `useFloatingDialog()` Composition関数（`src/client/classes/use-floating-dialog.ts`）を共有し、以下のアクセシビリティ機能を提供する:

| 機能 | 説明 |
|------|------|
| **Escape キー閉じ** | Escape キーで `onEscape` コールバックを呼び出しダイアログを閉じる |
| **ARIA属性** | `role="dialog"`, `aria-modal="true"`, `aria-labelledby`（`.gkill-floating-dialog__title` 要素を参照、見つからない場合は `aria-label` にフォールバック） |

※ v1.1.0でフォーカストラップ（Tab循環）、自動フォーカス、フォーカス復帰は削除された。

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
| `/rykv` | `rykv` | rykv-page.vue | 関連情報ビュー |
| `/kyou` | `kyou` | kyou-page.vue | ライフイベント一覧 |
| `/mkfl` | `mkfl` | mkfl-page.vue | 打刻メモ帳（KFTL入力+TimeIs表示） |
| `/plaing` | `plaing` | plaing-timeis-page.vue | アクティブ打刻一覧 |
| `/saihate` | `saihate` | saihate-page.vue | 記録特化画面（他画面への遷移なし） |
| `/set_new_password` | `set_new_password` | set-new-password-page.vue | パスワード設定 |
| `/regist_first_account` | `regist_first_account` | regist-first-account-page.vue | 初回アカウント登録 |
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
| `use-*.ts` Composition関数 | コンテキストメニュー等の共有ロジック（211ファイル） |

### GkillAPI シングルトン

`src/client/classes/api/gkill-api.ts` に定義。約3,400行。

- `GkillAPI.get_instance()` / `GkillAPI.get_gkill_api()` でインスタンス取得
- 全79エンドポイントに対応するメソッドを持つ
- `GkillAPIForSharedKyou` サブクラス（共有データ用）
- 各メソッドは `fetch()` → JSONパース → エラーチェック → データ返却

## 6. PWA構成

### Service Worker

定義: `src/client/serviceWorker.ts`（~279行）

**Workbox設定:**
- `registerType: 'autoUpdate'`（自動更新）
- `strategies: 'injectManifest'`（カスタムService Worker）
- 最大キャッシュファイルサイズ: 10MB

**キャッシュ戦略:**

| キャッシュ名 | 対象 | 説明 |
|---|---|---|
| `gkill-post-kyou-cache` | データ系エンドポイント | get_kyou, get_kmemo 等のPOSTレスポンスをキャッシュ |
| `gkill-post-config-cache` | 設定系エンドポイント | get_application_config, get_all_tag_names 等 |

**キャッシュ有効性検証:**
- `_histories` フィールドの存在チェック
- `errors` 配列が空であること
- `force_reget` パラメータでキャッシュバイパス可能

**SPAフォールバック:**
- `/`、`/api/*`、`/files/*` 以外の全パスを `index.html` にフォールバック

### Web Share Target

Service Worker が `/share-target` POSTを処理：
1. 共有されたテキストからURLを検出
2. URL → URLog（ブックマーク）として保存
3. テキストのみ → Kmemo（メモ）として保存
4. 保存結果に応じて `/saihate?is_saved=true/false` にリダイレクト

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
| `ja` | 日本語 | ~765 |
| `en` | 英語 | ~765 |
| `zh` | 中国語 | ~765 |
| `ko` | 韓国語 | ~765 |
| `es` | スペイン語 | ~765 |
| `fr` | フランス語 | ~765 |
| `de` | ドイツ語 | ~765 |

### ロケールファイル

- 配置: `src/locales/*.json`
- 形式: フラットなキー・バリュー（ネストなし）
- キー例: `LOGIN_TITLE`, `RESET_PASSWORD_TITLE`, `SAIHATE_PAGE_TITLE`
- **フロント・バックエンド共有:** ビルド時に `src/locales/` → `src/server/gkill/api/embed/i18n/locales/` にコピー

## 9. ビルド設定

### Vite設定 (`vite.config.ts`)

| 項目 | 値 |
|---|---|
| パスエイリアス | `@/` → `./src/client/` |
| minify | `false`（デフォルト）、環境変数 `MINIFY=true` で有効化 |
| Vue devtools | プロダクションでも有効（`__VUE_PROD_DEVTOOLS__: true`） |
| propsDestructure | 有効 |

### TypeScript設定

- `tsconfig.app.json`: フロントエンド用（`src/client/**/*` + `public/sw.js`）
- `tsconfig.node.json`: ビルドツール用（`@tsconfig/node20` 継承）
- lib: `WebWorker`（Service Worker用）
- types: `google.maps`, `vite-plugin-pwa/client`

### ESLint設定 (`.eslintrc.cjs`)

- extends: `plugin:vue/vue3-essential`, `eslint:recommended`, `@vue/eslint-config-typescript`
- `@typescript-eslint/no-unused-vars`: warn（`_` プレフィックスは無視）
- 実行: `npm run lint`

## 10. UX改善

### オフラインバナー

`App.vue` に `navigator.onLine` と `online`/`offline` イベントを監視するバナーを追加。ネットワーク切断時に `v-banner` でユーザーに通知する。

### ダイアログ＋ブラウザ履歴

`use-dialog-history-stack.ts` でダイアログの開閉をブラウザ履歴と連動：
- ダイアログ表示時に `history.pushState` で depth 付きエントリを追加
- ブラウザバックで最上位ダイアログを閉じる（depth比較でback/forwardを判別）
- ブラウザフォワードではダイアログを閉じない
- プログラマティック閉じ（Escape含む）時は履歴を巻き戻し

### 日付・数値のロケール対応

- `format_time()`: `Intl.DateTimeFormat` ベースでロケール別フォーマット + 曜日表示
- `format_number()`: `Intl.NumberFormat` ベースでロケール別桁区切り

### 空状態メッセージ

リスト表示（kyou-list-view, dnote-list-view 等）でデータ0件時に「表示できるデータがありません」メッセージを表示。

### フォームバリデーション

各追加ビュー（add-nlog, add-mi, add-urlog, add-timeis, add-kc, add-tag, add-text, add-notification 等）の必須フィールドに `:rules` バリデーションを追加。Vuetify の `v-text-field` / `v-textarea` の `:rules` prop で空チェックを実装。

### レスポンシブデザイン

- `rykv-view.vue` / `mi-view.vue`: ナビゲーションドロワーの幅を `$vuetify.display.smAndDown` で画面幅に応じて切替（スマートフォンでは `100vw`）
- `rykv-view.vue`: 詳細ビュー（`.kyou_detail_view`）の `min-width` を `0` に変更し `max-width: 100vw` を追加。600px以下でフルワイド表示
- `kyou-list-view.vue`: `v-virtual-scroll` に `max-width: 100vw` を追加し、画像リスト幅が画面を超えないように制限

### 未保存データ警告

KFTL テキストエリアに内容がある状態でページ離脱しようとすると `beforeunload` イベントで警告を表示。加えて、各ページ composable（use-rykv-page, use-mi-page, use-mkfl-page, use-plaing-timeis-page, use-kyou-page, use-saihate-page）にも `beforeunload` ガードを追加し、ダイアログ表示中やロード中のページ離脱を防止。
