# client テスト仕様

## 概要

Vue 3 フロントエンドのテスト。ユニットテスト（839テスト宣言、70ファイル）と E2E テスト（195テスト宣言、34 specファイル）の合計1034テスト宣言で、API クライアント、データモデル、パーサ、Composable、ルーター、i18n、Service Worker、全13ルート + CRUD操作フロー + 設定機能テスト + 回帰テスト + クリップボード保存機能テスト + ダッシュボード機能テスト + Markdown/Mermaid表示 + トレンドグラフ集計をカバーする。

## テストフレームワーク

- **ユニットテスト**: Vitest（jsdom 環境）
- **E2E テスト**: Playwright（baseURL: `localhost:5173`, タイムアウト: 60秒）

## ディレクトリ構成

```
src/client/__tests__/
├── unit/                    # ユニットテスト (839テスト宣言, 70ファイル)
│   ├── api/                 #   GkillAPI クライアント
│   ├── classes/             #   ユーティリティクラス (11ファイル, markdown-to-html / mermaid-render / use-dialog-history-stack / foldable-struct-move / kyou-content-text 含む)
│   ├── datas/               #   データモデル (28ファイル, dashboard-config / mi-re-kyou / append-not-found-tags 含む)
│   ├── dnote/               #   D-note モジュール (7ファイル, trend-aggregator.test.ts 含む)
│   ├── kftl/                #   KFTL パーサ (5ファイル)
│   ├── composables/         #   Vue Composable (13ファイル, idf-kyou-view / re-kyou-view / mi-re-kyou-view / save-clipboard-to-file-dialog 含む)
│   ├── router.test.ts       #   ルーター (13ルート, dashboard 含む)
│   ├── i18n-completeness.test.ts  #  i18n 完全性 (7ロケール)
│   └── service-worker.test.ts     #  Service Worker
├── e2e/                     # E2E テスト (195テスト宣言, 34 specファイル + auth.setup.ts / global-setup.ts / global-teardown.ts / free-port.mjs)
│   ├── helpers.ts           #   初回登録 + ログイン
│   ├── crud-helpers.ts      #   KFTL送信、ナビゲーション、コンテキストメニュー、FAB
│   ├── check-server.ts      #   サーバヘルスチェック
│   └── run-e2e.mjs          #   テストランナー（gkill_server 自動起動・停止）
└── helpers/                 # テストヘルパー
    ├── factory.ts           #   テストデータファクトリ
    ├── mock-api.ts          #   API モック
    └── setup-i18n.ts        #   i18n セットアップ
```

## テスト内容

### ユニットテスト（839テスト宣言、70ファイル）

| カテゴリ | テスト数（概算） | 内容 |
|---------|----------------|------|
| GkillAPI | 多数 | 全11データ型の CRUD、設定、共有、セッション管理 |
| データモデル | 28ファイル | 各データ型の生成・代入・シリアライゼーション（DashboardConfig / MiReKyou 含む）+ タグ付与の横断検証 |
| D-note | 7ファイル | 述語関数、キー取得、集計ターゲット、アグリゲータ、トレンドグラフ集計（trend-aggregator） |
| KFTL パーサ | 5ファイル | 構文解析、型判定、リクエスト生成 |
| Composable | 13ファイル | 追加/編集ビュー、コンテキストメニュー、削除確認、クリップボード保存、ページ/クエリ Composable、IDFKyouビュー（Markdown表示）、ReKyou / MiReKyou ビュー |
| ルーター | 1ファイル | 13ルートの定義と遷移（dashboard 含む） |
| i18n | 1ファイル | 7ロケールのキー完全性 |
| Service Worker | 1ファイル | Workbox プリキャッシュ、POST キャッシュ、プッシュ通知、`/zip_cache/.*` denylist |
| ユーティリティ | 11ファイル | deep-equals, format-date-time, looks-like-url, long-press, save-as, delete-gkill-cache, markdown-to-html, mermaid-render, use-dialog-history-stack, foldable-struct-move, kyou-content-text |

### E2E テスト (195テスト宣言, 34 specファイル)

全13ルートを Playwright で検証。CRUD操作フロー（KFTL記録→追加→編集→削除→閲覧）、認証フロー（ログイン/ログアウト/初回登録）、KFTL TimeIs終了全バリエーション、履歴・リポスト表示、Mi操作（板間移動/完了状態/共有）、サーバ設定/ユーザ設定の機能テスト、ZIPファイルブラウズダイアログ（コンテキストメニューからの起動、ZIP内コンテンツ表示）、修正済みバグの回帰テスト、ダッシュボード画面（日付ナビゲーション、FAB、設定ボタン）をカバー。

| 追加ファイル | 内容 |
|---|---|
| `dashboard.spec.ts` | ダッシュボード画面の表示・日付ナビゲーション・FAB・設定ボタン動作確認（9テスト） |

## 設定ファイル

- `vitest.config.ts` — ユニットテスト設定（jsdom, Vue 3 対応）
- `playwright.config.ts` — E2E テスト設定

## 実行方法

```bash
# 全フロントエンドテスト
npm run test_client

# ユニットテストのみ
npm run test_client_unit

# E2E テストのみ
npm run test_client_e2e
```

## 関連ドキュメント

| サブディレクトリ | テスト仕様 |
|----------------|-----------|
| `classes/` | [classes/ABOUT_TEST.md](classes/ABOUT_TEST.md) |
| `classes/api/` | [classes/api/ABOUT_TEST.md](classes/api/ABOUT_TEST.md) |
| `classes/datas/` | [classes/datas/ABOUT_TEST.md](classes/datas/ABOUT_TEST.md) |
| `classes/dnote/` | [classes/dnote/ABOUT_TEST.md](classes/dnote/ABOUT_TEST.md) |
| `classes/kftl/` | [classes/kftl/ABOUT_TEST.md](classes/kftl/ABOUT_TEST.md) |
| `pages/` | [pages/ABOUT_TEST.md](pages/ABOUT_TEST.md) |
