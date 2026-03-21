# api テスト仕様

## 概要

`GkillAPI` シングルトンクラスの全メソッドをテストする。全11データ型に対する CRUD 操作、設定管理、共有機能、アップロード、トランザクション、通知、エラーハンドリング、セッション管理、エンドポイントアドレス検証をカバーしている。

## テストフレームワーク

Vitest

## テストファイル

| ファイル | 内容 |
|---------|------|
| `src/client/__tests__/unit/api/gkill-api.test.ts` | GkillAPI の全メソッドテスト |

## テスト内容

- **データ型別 CRUD**: Kmemo, Mi, TimeIs, URLog, Nlog, Lantana, KC, Tag, Text, Notification, ReKyou の追加・更新・削除・取得
- **設定操作**: アプリケーション設定、サーバ設定の読み書き
- **共有機能**: Kyou の共有設定 CRUD
- **アップロード**: ファイルアップロード処理
- **トランザクション**: 複数操作のトランザクション処理
- **通知**: プッシュ通知ターゲットの管理
- **エラーハンドリング**: API エラーレスポンスの処理
- **セッション管理**: ログイン・ログアウト・セッション検証
- **エンドポイントアドレス検証**: サーバアドレスの正規化と検証

## テストヘルパー

- `src/client/__tests__/helpers/mock-api.ts` — API モックユーティリティ
- `src/client/__tests__/helpers/factory.ts` — テストデータファクトリ（`makeKmemo`, `makeMi`, `makeTag` 等）

## 実行方法

```bash
npm run test_client_unit
```
