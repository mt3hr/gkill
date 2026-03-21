# pages テスト仕様

## 概要

ページ・ビュー・ダイアログのテスト。Playwright による E2E テスト（12ルート全網羅）、Vue Composable のユニットテスト、ルーターのテストで構成される。

## テストフレームワーク

- **E2E**: Playwright
- **ユニット**: Vitest

## テストファイル一覧

### E2E テスト（12ファイル）

| ファイル | テスト対象ルート |
|---------|----------------|
| `src/client/__tests__/e2e/login.spec.ts` | `/login` — セッション永続化、認証リダイレクト、パスワードマスキング |
| `src/client/__tests__/e2e/kftl-dialog.spec.ts` | `/kftl` — KFTL テキスト入力、マルチライン、テンプレートセクション |
| `src/client/__tests__/e2e/mi-board.spec.ts` | `/mi` — タスクボード、JS エラーチェック、レスポンシブ、FAB |
| `src/client/__tests__/e2e/rykv.spec.ts` | `/rykv` — JS エラーチェック、モバイルビューポート、URL永続化 |
| `src/client/__tests__/e2e/mkfl.spec.ts` | `/mkfl` — ファイル管理 |
| `src/client/__tests__/e2e/plaing.spec.ts` | `/plaing` — 計画ビュー |
| `src/client/__tests__/e2e/settings.spec.ts` | `/settings` — 設定画面コンテンツ、JS エラーチェック、インタラクティブ操作 |
| `src/client/__tests__/e2e/kyou-list.spec.ts` | `/kyou` — レコード一覧 |
| `src/client/__tests__/e2e/share-page.spec.ts` | `/shared` — 共有ページ |
| `src/client/__tests__/e2e/shared-mi.spec.ts` | `/shared_mi` — 共有タスク |
| `src/client/__tests__/e2e/regist-first-account.spec.ts` | `/regist_first_account` — 初回アカウント登録 |
| `src/client/__tests__/e2e/set-new-password.spec.ts` | `/set_new_password` — パスワード再設定 |

### Composable ユニットテスト（6ファイル）

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/unit/composables/add-views.test.ts` | Mi, Tag, Nlog, URLog, Lantana, TimeIs, KC の追加ビュー Composable |
| `src/client/__tests__/unit/composables/edit-views.test.ts` | Kmemo, Mi, Nlog, URLog, TimeIs, Lantana, KC の編集ビュー Composable |
| `src/client/__tests__/unit/composables/context-menus.test.ts` | 10エンティティ型のコンテキストメニュー Composable |
| `src/client/__tests__/unit/composables/confirm-delete.test.ts` | 削除確認ダイアログ Composable |
| `src/client/__tests__/unit/composables/page-composables.test.ts` | ページレベル Composable |
| `src/client/__tests__/unit/composables/query-composables.test.ts` | クエリ操作 Composable パターン |

### ルーターテスト

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/unit/router.test.ts` | 全12ルートの定義と遷移 |

## E2E テストヘルパー

- `src/client/__tests__/e2e/helpers.ts` — E2E テスト共通ユーティリティ
- `src/client/__tests__/e2e/check-server.ts` — サーバヘルスチェック

## 実行方法

```bash
# E2E テスト
npm run test_client_e2e

# ユニットテスト（Composable + Router）
npm run test_client_unit
```
