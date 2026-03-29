# pages テスト仕様

## 概要

ページ・ビュー・ダイアログのテスト。Playwright による E2E テスト（12ルート全網羅 + CRUD/設定/回帰テスト、29ファイル187テスト）、Vue Composable のユニットテスト、ルーターのテストで構成される。

## テストフレームワーク

- **E2E**: Playwright
- **ユニット**: Vitest

## テストファイル一覧

### E2E テスト（29ファイル, 187テスト）

#### ページ表示・ナビゲーション系（12ファイル）

| ファイル | テスト対象ルート |
|---------|----------------|
| `src/client/__tests__/e2e/login.spec.ts` | `/` — セッション永続化、認証リダイレクト、パスワードマスキング |
| `src/client/__tests__/e2e/kftl-dialog.spec.ts` | `/kftl` — KFTL テキスト入力、マルチライン、テンプレートセクション |
| `src/client/__tests__/e2e/mi-board.spec.ts` | `/mi` — タスクボード、JS エラーチェック、レスポンシブ、FAB |
| `src/client/__tests__/e2e/rykv.spec.ts` | `/rykv` — JS エラーチェック、モバイルビューポート、URL永続化 |
| `src/client/__tests__/e2e/mkfl.spec.ts` | `/mkfl` — ファイル管理 |
| `src/client/__tests__/e2e/plaing.spec.ts` | `/plaing` — 計画ビュー |
| `src/client/__tests__/e2e/settings.spec.ts` | `/saihate` — 設定画面コンテンツ、JS エラーチェック、インタラクティブ操作 |
| `src/client/__tests__/e2e/kyou-list.spec.ts` | `/kyou` — レコード一覧 |
| `src/client/__tests__/e2e/share-page.spec.ts` | `/shared_page` — 共有ページ |
| `src/client/__tests__/e2e/shared-mi.spec.ts` | `/shared_mi` — 共有タスク |
| `src/client/__tests__/e2e/regist-first-account.spec.ts` | `/regist_first_account` — 初回アカウント登録 |
| `src/client/__tests__/e2e/set-new-password.spec.ts` | `/set_new_password` — パスワード再設定 |

#### CRUD 操作フロー系（8ファイル）

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/e2e/kftl-crud.spec.ts` | KFTL経由で全データ型記録 + 画面表示確認 |
| `src/client/__tests__/e2e/kftl-timeis-end.spec.ts` | TimeIs終了の全4バリエーション（タイトル/タグ × 指定/存在すれば） |
| `src/client/__tests__/e2e/add-dialog-crud.spec.ts` | FAB→追加ダイアログ (Mi/Lantana/Nlog/TimeIs/URLog/KC/Tag/Text + 最小入力/全項目入力) |
| `src/client/__tests__/e2e/edit-dialog-crud.spec.ts` | 右クリック→編集 (Kmemo/Mi/Nlog/URLog/TimeIs/Lantana/Tag + TimeIs終了/ReKyou/Text編集) |
| `src/client/__tests__/e2e/delete-crud.spec.ts` | 右クリック→削除 (Kmemo/Mi/Lantana/Nlog/URLog/TimeIs/Tag/Text/ReKyou) |
| `src/client/__tests__/e2e/view-browse.spec.ts` | 履歴ダイアログ、混合データ型、Mi ボード/Plaing 表示、ZIPブラウズ |
| `src/client/__tests__/e2e/view-history.spec.ts` | Lantana/Mi/Nlog/URLog/ReKyou/Tag/Text の履歴+リポスト+NoImage確認 |
| `src/client/__tests__/e2e/notification-crud.spec.ts` | Notification の追加/編集/削除/閲覧/履歴 |

#### 認証・ユースケース・設定系（9ファイル）

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/e2e/auth-flow.spec.ts` | ログアウト、パスワード未設定ログイン不可、Rep全チェック確認 |
| `src/client/__tests__/e2e/search-and-summary.spec.ts` | RYKV/Mi キーワード検索、D-note サマリトグル |
| `src/client/__tests__/e2e/mi-operations.spec.ts` | タスク板間移動、完了状態、共有閲覧/停止 |
| `src/client/__tests__/e2e/settings-crud.spec.ts` | 設定ページ各セクション表示確認 |
| `src/client/__tests__/e2e/server-config-crud.spec.ts` | サーバ設定機能テスト（プロファイル/TLS/アカウント/Rep管理） |
| `src/client/__tests__/e2e/user-config-crud.spec.ts` | ユーザ設定機能テスト（API/画像/板名/構造CRUD） |
| `src/client/__tests__/e2e/regression-fixes.spec.ts` | 修正済みバグ回帰テスト（7件） |
| `src/client/__tests__/e2e/misc-features.spec.ts` / `misc-operations.spec.ts` | 見た目区別、ブックマークレット、GPS、共有リンク、再起動 |

### Composable ユニットテスト（6ファイル）

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/unit/composables/add-views.test.ts` | Mi, Tag, Nlog, URLog, Lantana, TimeIs, KC の追加ビュー Composable |
| `src/client/__tests__/unit/composables/edit-views.test.ts` | Kmemo, Mi, Nlog, URLog, TimeIs, Lantana, KC の編集ビュー Composable |
| `src/client/__tests__/unit/composables/context-menus.test.ts` | 10エンティティ型のコンテキストメニュー Composable（ZIPブラウズ項目含む） |
| `src/client/__tests__/unit/composables/confirm-delete.test.ts` | 削除確認ダイアログ Composable |
| `src/client/__tests__/unit/composables/page-composables.test.ts` | ページレベル Composable |
| `src/client/__tests__/unit/composables/query-composables.test.ts` | クエリ操作 Composable パターン |

### ルーターテスト

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/unit/router.test.ts` | 全12ルートの定義と遷移 |

## E2E テストヘルパー

- `src/client/__tests__/e2e/helpers.ts` — 初回登録（reset_token取得→regist_first_account）+ テストユーザログイン
- `src/client/__tests__/e2e/crud-helpers.ts` — KFTL送信、ページナビゲーション、コンテキストメニュー、FABクリック、フローティングダイアログ自動閉じ
- `src/client/__tests__/e2e/check-server.ts` — サーバヘルスチェック
- `src/client/__tests__/e2e/run-e2e.mjs` — テストランナー（gkill_server自動起動・停止）

## 実行方法

```bash
# E2E テスト
npm run test_client_e2e

# ユニットテスト（Composable + Router）
npm run test_client_unit
```
