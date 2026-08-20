# client - Vue 3 フロントエンド

## 概要

gkill のフロントエンド。Vue 3 + Vuetify 4 + TypeScript で構築された PWA（Progressive Web App）。
Vite でビルドし、Go バックエンドにバイナリ埋め込みされて配信される。

## 技術スタック

- **フレームワーク**: Vue 3（Composition API）
- **UI ライブラリ**: Vuetify 4
- **言語**: TypeScript
- **ビルドツール**: Vite
- **ルーティング**: Vue Router
- **国際化**: vue-i18n（7言語対応）
- **PWA**: Service Worker によるオフライン対応

## ディレクトリ構造

```
client/
├── App.vue                  # ルートコンポーネント
├── main.ts                  # アプリケーションエントリポイント
├── i18n.ts                  # vue-i18n 初期化
├── serviceWorker.ts         # Service Worker 登録
├── global.d.ts              # グローバル型定義
├── classes/                 # TS ユーティリティ → classes/README.md
│   ├── api/                 # GkillAPI クライアント → classes/api/README.md
│   ├── datas/               # データモデル → classes/datas/README.md
│   ├── dnote/               # Dynamic Note → classes/dnote/README.md
│   ├── kftl/                # KFTL パーサ → classes/kftl/README.md
│   ├── dto/                 # DTO（2ファイル）
│   └── lantana/             # Lantana ユーティリティ（1ファイル）
├── pages/                   # Vue ページ → pages/README.md
│   ├── views/               # View コンポーネント（202 .vue）
│   └── dialogs/             # Dialog コンポーネント（116 .vue）
├── plugins/
│   └── vuetify.ts           # Vuetify プラグイン設定
└── router/
    └── index.ts             # Vue Router 設定
```

## エントリポイント

### `main.ts`

アプリケーションの初期化:
- Vue 3 アプリケーション作成
- Vuetify 4 プラグイン登録
- Vue Router 登録
- vue-i18n 登録
- カスタム `v-long-press` ディレクティブ登録

### `App.vue`

ルートコンポーネント。`<router-view>` でページを切り替え。

### `i18n.ts`

vue-i18n の初期化。`src/locales/` の7言語 JSON ファイルを読み込む。

### `serviceWorker.ts`

Service Worker の登録。PWA オフライン対応。

## ルーティング（13ルート）

`router/index.ts` で定義:

| パス | 名前 | ページ | 説明 |
|-----|------|-------|------|
| `/` | login | `login-page.vue` | ログインページ |
| `/kftl` | kftl | `kftl-page.vue` | KFTL テキストエディタ |
| `/mi` | mi | `mi-page.vue` | Mi（タスク）管理 |
| `/rykv` | rykv | `rykv-page.vue` | Rykv（レビュー）— Kyou の閲覧・検索・編集 |
| `/kyou` | kyou | `kyou-page.vue` | Kyou 記録一覧 |
| `/mkfl` | mkfl | `mkfl-page.vue` | Mkfl（ファイル管理） |
| `/plaing` | plaing | `plaing-time-is-page.vue` | 稼働中 TimeIs 一覧 |
| `/dashboard` | dashboard | `dashboard-page.vue` | ダッシュボード（日次サマリー・集計・GPS・タスク） |
| `/rudbeckia` | rudbeckia | `rudbeckia-page.vue` | ポート。4画面をウィンドウとして並べる単一画面（開発コード rudbeckia） |
| `/saihate` | saihate | `saihate-page.vue` | Saihate（特殊ビュー） |
| `/set_new_password` | set_new_password | `set-new-password-page.vue` | 新パスワード設定 |
| `/register_first_account` | register_first_account | `register-first-account-page.vue` | 初回アカウント登録（旧 `/regist_first_account` からリダイレクト） |
| `/shared_page` | shared_page | `shared-page.vue` | Kyou 共有ページ |

これに加えて、コンポーネントを持たない **リダイレクト専用ルートが2つ** ある:
`/regist_first_account` → `/register_first_account`、`/shared_mi` → `/shared_page`（どちらもクエリを引き継ぐ）。
**旧パスの吸収はルータの `redirect` で行うこと** ―― `<script setup>` に top-level await のあるページの
setup から `router.replace` すると、初回ナビゲーションが完了しなくなる。

## 設計パターン

### Composable パターン

各コンポーネントのロジック（リアクティブ状態、メソッド、計算プロパティ）は
`classes/use-*.ts` の Composable 関数に分離:

```
xxx-view.vue          ← テンプレート + スタイル
  └── use-xxx-view.ts ← ロジック（Composable）
```

### Props / Emits 分離

多くのコンポーネントで Props と Emits を別ファイルに分離:
```
xxx-view.vue
xxx-view-props.ts
xxx-view-emits.ts
```

### Dialog / View 分離

操作は View（インライン表示）と Dialog（モーダル表示）の2つの形態で提供:
```
views/add-kc-view.vue      ← View 版
dialogs/add-kc-dialog.vue  ← Dialog 版（View をラップ）
```

### カスタムディレクティブ

- `v-long-press`: ロングプレスイベント。モバイル対応のコンテキストメニュー表示に使用

## plugins/

### `vuetify.ts`

Vuetify 4 のプラグイン設定:
- テーマ設定
- コンポーネント登録
- アイコン設定

## router/

### `index.ts`

Vue Router の設定。`createWebHistory` を使用した HTML5 History モード。
全13ルートを定義。

## ビルド

```bash
# 開発サーバ起動（フロントエンドのみ）
npm run dev

# 本番ビルド（型チェック + Vite ビルド）
npm run build

# ESLint
npm run lint
```

ビルド結果は `dist/` に出力され、`npm run install_server` で `src/server/gkill/api/embed/html/` にコピーされる。

## 関連ドキュメント

- [classes/README.md](classes/README.md) — TypeScript ユーティリティ
- [classes/api/README.md](classes/api/README.md) — GkillAPI クライアント
- [classes/datas/README.md](classes/datas/README.md) — データモデル
- [classes/dnote/README.md](classes/dnote/README.md) — Dynamic Note システム
- [classes/kftl/README.md](classes/kftl/README.md) — KFTL パーサ
- [pages/README.md](pages/README.md) — ページ・ビュー・ダイアログ
