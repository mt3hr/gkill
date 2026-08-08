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
| `src/client/__tests__/unit/classes/kyou-content-text.test.ts` | Kyou の内容 / ID のクリップボードコピー用テキスト生成 |
| `src/client/__tests__/unit/classes/kyou-view-relay.test.ts` | Kyou 系イベント中継束（ビュー18件／ダイアログ20件を漏れなく張ること、`requested_close_dialog`・フォーカス系を張らないこと） |
| `src/client/__tests__/unit/classes/cascade-delete-kyou.test.ts` | Kyou 削除の連鎖削除（Tag/Text/Notification と参照元 ReKyou/MiReKyou、削除順、深さ上限） |
| `src/client/__tests__/unit/classes/use-confirm-delete-kyou-view.test.ts` | Kyou 削除確認ビュー（連鎖削除の呼び出しとクローズ） |
| `src/client/__tests__/unit/classes/confirm-dialog-close.test.ts` | 確認ダイアログ（Tag/Text/Notification 削除・リポスト作成）が例外時も必ず閉じること、連打で多重リクエストにならないこと |
| `src/client/__tests__/unit/classes/edit-view-no-update-check.test.ts` | 「更新がありません」判定に `related_time` を含めること（日時だけ変更しても保存される） |
| `src/client/__tests__/unit/classes/delayed-loading.test.ts` | 読み込み中表示の遅延（速く終わった読み込みで明滅させない） |

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
- **kyou-view-relay**: `build_kyou_view_relay` が18イベントを漏れなく張ること、`build_kyou_dialog_relay` がそこへフォーカス系2件を足した20イベントを張ること、`requested_close_dialog` を張らないこと、引数の素通し、`overrides` で指定したイベントだけ差し替わること
- **cascade-delete-kyou**: 付随する Tag / Text / Notification と参照元 ReKyou / MiReKyou の論理削除、多段参照の再帰探索、循環参照・自己参照での停止、逆引きを全部終えてから削除し Kyou 自身は最後に消すこと、1本失敗しても投げきってエラーを集約すること、`force_reget` の付与、削除済み ReKyou を辿らないこと、同一 id は `update_time` 最新の1件だけ消すこと、深さ上限での打ち切り、`errors: null` で throw しないこと、共有画面では何もしないこと
- **use-confirm-delete-kyou-view**: 削除成功時に `deleted_kyou` と `requested_close_dialog` を emit すること、例外時もエラーを出したうえで閉じること、削除中の二重押しでリクエストを重ねないこと
- **confirm-dialog-close**: Tag / Text / Notification の削除確認とリポスト作成の各ダイアログが、成功時も例外時も必ず閉じること、連打してもリクエストが1回だけであること
- **edit-view-no-update-check**: 本文を変えずに関連日時だけ変更しても更新リクエストが飛ぶこと（kmemo / kc）、本文も日時も変えなければ「更新なし」で閉じないこと
- **delayed-loading**: しきい値未満で終わった読み込みではインジケータを立てないこと、しきい値を超えたら立てて完了で下ろすこと、待ち時間を指定できること

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
