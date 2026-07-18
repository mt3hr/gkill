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
| `src/client/__tests__/unit/classes/use-dialog-history-stack.test.ts` | ダイアログ＋ブラウザ履歴スタック管理 |
| `src/client/__tests__/unit/classes/markdown-to-html.test.ts` | Markdown → HTML 変換（見出し・表・コード・画像、サニタイズ） |
| `src/client/__tests__/unit/classes/mermaid-render.test.ts` | Markdown 内 Mermaid コードブロックの図描画 |
| `src/client/__tests__/unit/classes/foldable-struct-move.test.ts` | Struct ツリーの移動ロジック（上へ/下へ/フォルダへ移動） |

## テスト内容

- **deep-equals**: ネストされたオブジェクト、配列、プリミティブ値の等価比較
- **format-date-time**: 日付文字列のフォーマット変換、ロケール対応
- **looks-like-url**: URL 形式判定（http/https、相対パス等）
- **long-press**: Vue カスタムディレクティブの登録・発火タイミング
- **save-as**: Blob ダウンロードの処理フロー
- **delete-gkill-cache**: Service Worker キャッシュのクリア処理
- **markdown-to-html**: Markdown ファイル（.md/.markdown）のリッチ HTML 変換とサニタイズ
- **mermaid-render**: ```mermaid コードブロックの SVG 描画
- **use-dialog-history-stack**: ブラウザバックでダイアログ閉じ、フォワードでダイアログ維持、複数ダイアログの順次閉じ、プログラマティック閉じ、Escape閉じ、Branch C/D ロジック
- **foldable-struct-move**: 同一親内の上下入れ替え、フォルダ/ルートへの移動、自分自身・子孫フォルダ・非フォルダへの移動拒否と失敗時のツリー保全、移動先候補列挙

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
| `lantana/` | 独立テストなし。`LantanaFlowerState` 等の enum 型定義のみ。テストは `datas/lantana.test.ts` でカバー |
