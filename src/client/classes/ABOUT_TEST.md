# classes テスト仕様

## 概要

フロントエンドのユーティリティクラス群のテスト。汎用ヘルパー関数と、サブディレクトリ（api, datas, dnote, kftl）の各モジュールテストを含む。

## テストフレームワーク

Vitest

## `use-*.ts` にテストを書く／書かない基準

`classes/` には `use-*.ts` が300本以上あるが、**全部にテストは書かない**。
基準は「壊れたときに、コンパイラでも型でも他のテストでも気付けないか」の1点。

**書く**:

- 純関数と、それに近い導出（`kftl-tabs.ts` / `mi-board-names.ts` / `kyou-local-insert.ts`）
- **集約先**。同じ処理が何箇所にも手書きされていたのを1つへ寄せたもの（`abort-error.ts` は20箇所、
  `web-push-key.ts` は6箇所）。壊れると全箇所へ同時に波及するのに、
  規約のソース走査は「手書きが残っていないこと」しか見ていない
- 順序・タイミングが本質のもの（`use-registered-tag-column-filter.ts`、
  `use-rykv-view.ts` / `use-mi-view.ts` の列×検索）
- 相手が居るやりとり（`use-plugin-config-dialog.ts` の iframe との postMessage、
  `use-browse-zip-contents-dialog.ts` の階層導出）

**書かない**:

- `useFloatingDialog` を包んで `show()` / `hide()` / `defineExpose` するだけの薄いラッパ。
  ダイアログ本体のロジックを `.vue` から `.ts` へ移した棚卸しで69本増えたが、
  そのうち49本は40行未満のこれで、確認できるのは「型が通ること」だけになる
  （`src/ABOUT_TEST.md` の「型やコンパイラが保証済みのものは書かない」）
- 抽出そのものが維持されていることは `convention-source-scan.test.ts` が
  「ダイアログの `<script setup>` にロジックを残していない」で見張る

## テストファイル一覧（ユーティリティクラス）

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/unit/classes/deep-equals.test.ts` | オブジェクトの深い等価比較 |
| `src/client/__tests__/unit/classes/format-date-time.test.ts` | 日付・時刻のフォーマット処理 |
| `src/client/__tests__/unit/classes/looks-like-url.test.ts` | URL 判定ユーティリティ |
| `src/client/__tests__/unit/classes/linkify-text.test.ts` | 本文中 URL のセグメント分割（リンク化用） |
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
| `src/client/__tests__/unit/classes/use-device-kind.test.ts` | 端末種別の判定（PC / タブレット / スマートフォン）とシングルトン性・リアクティブ性 |
| `src/client/__tests__/unit/classes/kyou-reload.test.ts` | Kyou の引き直し手順（キャッシュ削除→reload→型付きデータ再取得の順、添付タグの強制再取得、同一更新由来の合流、失敗時のリトライ、引き直し中フラグ） |
| `src/client/__tests__/unit/classes/kyou-local-insert.test.ts` | 追加した記録を再検索せず列へ差し込む判定と整列（判定できる/できないフィルタの切り分け、タグ・rep・カレンダー・時間帯・mi の一致判定、並び順、冪等な差し込み） |
| `src/client/__tests__/unit/classes/kyou-local-insert-mi-parity.test.ts` | 上の mi 部分がサーバ側 `find_filter_mi_test.go` と同じ答えを出すこと（対で維持する） |
| `src/client/__tests__/unit/classes/foldable-struct-check.test.ts` | チェック状態をツリーへ単一走査で適用すること（旧実装との等価性） |
| `src/client/__tests__/unit/classes/use-context-menu-position.test.ts` | コンテキストメニューの表示位置（Vuetify の実測配置へ委ねるための状態管理） |
| `src/client/__tests__/unit/classes/dialog-autofocus.test.ts` | ダイアログを開いたときにカーソルを載せる入力欄の選定規則（チェックボックス・読み取り専用・非表示を除く） |
| `src/client/__tests__/unit/classes/use-application-config-view.test.ts` | 設定画面。子ダイアログの適用で props を書き換えないこと、キャンセルで言語・テーマ・期間が開いた時点へ戻ること |
| `src/client/__tests__/unit/classes/use-kyou-list-view-dialog.test.ts` | 一覧ダイアログ。20件のイベント中継と、自分で記録詳細ダイアログをホストする経路 |
| `src/client/__tests__/unit/classes/mi-board-struct.test.ts` | 板ツリーへの板の存在判定と追加（冪等、空文字スキップ、子配列の初期化） |
| `src/client/__tests__/unit/classes/mi-board-column-layout.test.ts` | 板の列見出しの高さ定数が CSS と一致すること（二重管理のズレ検出） |

### 走査型テスト（型では検出できない書き間違いをソース走査で検出する）

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/unit/classes/relay-bundle-source-scan.test.ts` | イベント中継束を `v-on` で渡した要素に同じイベントの `@` を併記していないこと（両方登録されて二重に発火する） |
| `src/client/__tests__/unit/classes/kyou-view-height-source-scan.test.ts` | 記録ビューの高さにパーセント指定を渡していないこと |
| `src/client/__tests__/unit/classes/application-config-update-fields-scan.test.ts` | 設定保存の詰め替え漏れ（書き忘れたフィールドが保存のたびに初期値へ巻き戻る）。永続化フィールド一覧がサーバ実装のキーと一致することも検査する |
| `src/client/__tests__/unit/classes/convention-source-scan.test.ts` | **保守棚卸し全体の安全網。** `autofocus` を view に撒いていない／`:draggable` が `is_pc` 由来である／`.reload(true)` を手書きしていない／中継束を `@` で展開していない／ダイアログの `<script setup>` にロジックを残していない／中断判定を手書きしていない、をソース走査で検出する。検出用の正規表現自体をインラインの見本で突いてあるので、走査が空振りしたまま緑になることがない |
| `src/client/__tests__/unit/classes/column-view-init-source-scan.test.ts` | 列ビュー（rykv / mi）の初期化順序と、新しいタグの既知判定を `emit` より前に行っていることをソース走査で固定する |
| `src/client/__tests__/unit/classes/check-auth-login-page.test.ts` | ログイン画面ではセッション無効の飛ばしを止めること。ログイン失敗も `check_auth` と同じエラーコード帯を通るので、飛ばすと出したばかりのエラー表示がページごと作り直されて消える。ガードが `location.replace` より手前にあることもソース走査で見る |
| `src/client/__tests__/unit/classes/abort-error.test.ts` | 中断（`AbortController.abort()`）の判定。20箇所の手書きを集約した先なので、壊れると全箇所へ同時に波及する。Chrome / Firefox で文言が違うため**メッセージで見るしかない** |
| `src/client/__tests__/unit/classes/web-push-key.test.ts` | VAPID公開鍵（URL-safe base64）→ バイト列。6ページ分の重複を集約した先。パディング補完と `-_`→`+/` 置換 |
| `src/client/__tests__/unit/classes/kyou-change-bus.test.ts` | 画面間の変更伝播バス。自分が出した通知は受けないこと、seq 付きの追記ログで同一 tick の複数件を落とさないこと |
| `src/client/__tests__/unit/classes/kyou-tags.test.ts` | Kyou へのタグ付け（重複の落とし方、削除マークの扱い） |
| `src/client/__tests__/unit/classes/kftl-tabs.test.ts` | メモ帳のタブの純関数（追加・削除・並び・保存マーカーの行数え） |
| `src/client/__tests__/unit/classes/mi-board-names.test.ts` | 板名の並び順を設定の板ツリー順に揃えること・ツリーのルート行のクリックで板を開かないこと |
| `src/client/__tests__/unit/classes/share-target-dedup.test.ts` | Android共有の重複台帳。再配送と意図的な再共有は内容から区別できないので、内容の完全一致と24時間で照会する |
| `src/client/__tests__/unit/classes/floating-dialog-z-order.test.ts` | フローティングダイアログの前面化。z-index は開いている枚数から出す（単調増加にすると Vuetify の overlay を追い越してダイアログ内のメニューが下へ潜る） |
| `src/client/__tests__/unit/classes/dnote-correlation-matrix-layout.test.ts` | 相関マトリクスのレイアウト計算 |
| `src/client/__tests__/unit/classes/kyou-attached-tags-nowrap-source-scan.test.ts` | 付随タグの折り返し指定をソース走査で固定 |
| `src/client/__tests__/unit/classes/kyou-detail-pane-attached-source-scan.test.ts` | 詳細ペインが付随データを出す配線をソース走査で固定 |

## テスト内容

- **deep-equals**: ネストされたオブジェクト、配列、プリミティブ値の等価比較
- **format-date-time**: 日付文字列のフォーマット変換、ロケール対応
- **looks-like-url**: URL 形式判定（http/https、相対パス等）
- **linkify-text**: 本文テキストの URL / 非 URL セグメント分割（末尾約物のトリム、括弧対応、和文区切り）
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
- **use-device-kind**: 実機UAを使った端末種別の判定（タッチ搭載Windowsノートは PC、素の iPad はタブレット、iPad + トラックパッドは PC、スタイラス対応スマートフォンはスマートフォン、Android WebView のスマホ/タブレット、UA-CH 優先、短辺600pxのフォールバック）、`matchMedia` が無い環境で PC に倒れること、`useDeviceKind()` を何度呼んでも購読が増えず同一参照を返すこと、メディアクエリ変化が取得済みの ref に伝わること

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
