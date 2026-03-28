# client テスト仕様

## 概要

Vue 3 フロントエンドのテスト。ユニットテスト（~676テスト）と E2E テスト（187テスト、29ファイル）の合計~863テストで、API クライアント、データモデル、パーサ、Composable、ルーター、i18n、Service Worker、全12ルート + CRUD操作フロー + 設定機能テスト + 回帰テストをカバーする。

## テストフレームワーク

- **ユニットテスト**: Vitest（jsdom 環境）
- **E2E テスト**: Playwright（baseURL: `localhost:5173`, タイムアウト: 60秒）

## ディレクトリ構成

```
src/client/__tests__/
├── unit/                    # ユニットテスト (~676テスト)
│   ├── api/                 #   GkillAPI クライアント
│   ├── classes/             #   ユーティリティクラス (6ファイル)
│   ├── datas/               #   データモデル (22ファイル)
│   ├── dnote/               #   D-note モジュール (5ファイル)
│   ├── kftl/                #   KFTL パーサ (5ファイル)
│   ├── composables/         #   Vue Composable (6ファイル)
│   ├── router.test.ts       #   ルーター (12ルート)
│   ├── i18n-completeness.test.ts  #  i18n 完全性 (7ロケール)
│   └── service-worker.test.ts     #  Service Worker
├── e2e/                     # E2E テスト (187テスト, 29ファイル)
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

### ユニットテスト (~676テスト)

| カテゴリ | テスト数（概算） | 内容 |
|---------|----------------|------|
| GkillAPI | 多数 | 全11データ型の CRUD、設定、共有、セッション管理 |
| データモデル | 22ファイル | 全22データ型の生成・代入・シリアライゼーション |
| D-note | 5ファイル | 述語関数、キー取得、集計ターゲット、アグリゲータ |
| KFTL パーサ | 5ファイル | 構文解析、型判定、リクエスト生成 |
| Composable | 6ファイル | 追加/編集ビュー、コンテキストメニュー、削除確認 |
| ルーター | 1ファイル | 12ルートの定義と遷移 |
| i18n | 1ファイル | 7ロケールのキー完全性 |
| Service Worker | 1ファイル | Workbox プリキャッシュ、POST キャッシュ、プッシュ通知 |
| ユーティリティ | 6ファイル | deep-equals, format-date-time, looks-like-url, long-press, save-as, delete-gkill-cache |

### E2E テスト (187テスト, 29ファイル)

全12ルートを Playwright で検証。CRUD操作フロー（KFTL記録→追加→編集→削除→閲覧）、認証フロー（ログイン/ログアウト/初回登録）、KFTL TimeIs終了全バリエーション、履歴・リポスト表示、Mi操作（板間移動/完了状態/共有）、サーバ設定/ユーザ設定の機能テスト、修正済みバグの回帰テストをカバー。

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
