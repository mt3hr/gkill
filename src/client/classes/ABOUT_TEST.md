# classes テスト仕様

## 概要

フロントエンドのユーティリティクラス群のテスト。汎用ヘルパー関数と、サブディレクトリ（api, datas, dnote, kftl）の各モジュールテストを含む。

## テストフレームワーク

Vitest

## テストファイル一覧（ユーティリティクラス）

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/unit/classes/deep-equals.test.ts` | オブジェクトの深い等価比較 |
| `src/client/__tests__/unit/classes/format-date-time.test.ts` | 日付・時刻のフォーマット処理 |
| `src/client/__tests__/unit/classes/looks-like-url.test.ts` | URL 判定ユーティリティ |
| `src/client/__tests__/unit/classes/long-press.test.ts` | `v-long-press` ディレクティブ |
| `src/client/__tests__/unit/classes/save-as.test.ts` | ファイル保存ユーティリティ |
| `src/client/__tests__/unit/classes/delete-gkill-cache.test.ts` | gkill キャッシュ削除処理 |

## テスト内容

- **deep-equals**: ネストされたオブジェクト、配列、プリミティブ値の等価比較
- **format-date-time**: 日付文字列のフォーマット変換、ロケール対応
- **looks-like-url**: URL 形式判定（http/https、相対パス等）
- **long-press**: Vue カスタムディレクティブの登録・発火タイミング
- **save-as**: Blob ダウンロードの処理フロー
- **delete-gkill-cache**: Service Worker キャッシュのクリア処理

## 実行方法

```bash
npm run test_client_unit
```

## 関連ドキュメント

| サブディレクトリ | テスト仕様 |
|----------------|-----------|
| `api/` | [api/ABOUT_TEST.md](api/ABOUT_TEST.md) |
| `datas/` | [datas/ABOUT_TEST.md](datas/ABOUT_TEST.md) |
| `dnote/` | [dnote/ABOUT_TEST.md](dnote/ABOUT_TEST.md) |
| `kftl/` | [kftl/ABOUT_TEST.md](kftl/ABOUT_TEST.md) |
