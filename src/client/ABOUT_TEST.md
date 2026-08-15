# client テスト仕様

## 概要

Vue 3 フロントエンドのテスト。ユニットテスト（1408テスト宣言、133ファイル）と E2E テスト（215テスト宣言、40ファイル）の合計1623テスト宣言で、API クライアント、データモデル、パーサ、Composable、ルーター、i18n、Service Worker、全13ルート + CRUD操作フロー + 設定機能テスト + 回帰テスト + クリップボード保存機能テスト + ダッシュボード機能テスト + Markdown/Mermaid表示 + トレンドグラフ集計 + 相関グラフの統計計算をカバーする。

## テストフレームワーク

- **ユニットテスト**: Vitest（jsdom 環境）
- **E2E テスト**: Playwright（baseURL: `localhost:5173`, タイムアウト: 60秒）

## ディレクトリ構成

```
src/client/__tests__/
├── unit/                    # ユニットテスト (1184テスト宣言, 111ファイル)
│   ├── api/                 #   GkillAPI クライアント
│   ├── classes/             #   ユーティリティクラス (31ファイル, kyou-view-relay / cascade-delete-kyou / confirm-dialog-close / edit-view-no-update-check / markdown-to-html / mermaid-render / use-dialog-history-stack / delayed-loading / foldable-struct-move / foldable-struct-check / kyou-content-text / use-context-menu-position / use-device-kind / linkify-text / application-config-update-fields-scan / dialog-autofocus / kyou-view-height-source-scan / mi-board-column-layout / use-application-config-view 含む)
│   ├── datas/               #   データモデル (33ファイル, dashboard-config / saved-find-query-config / plaing-time-is-config / mi-re-kyou / append-not-found-tags 含む)
│   ├── dnote/               #   D-note モジュール (7ファイル, trend-aggregator.test.ts 含む)
│   ├── kftl/                #   KFTL パーサ (5ファイル)
│   ├── composables/         #   Vue Composable (45ファイル, kyou-view / idf-kyou-view / re-kyou-view / mi-re-kyou-view / save-clipboard-to-file-dialog / rykv-view-search-routing / mi-view-search-routing / rykv-sidebar-saved-query-apply / mi-sidebar-saved-query-apply / mi-board-query / kyou-list-view-loading / sidebar-child-query-sync-emission / kyou-list-view-scroll-to / rep-query-summary-detail / find-query-editor-dialog-default-signal / plugin-html-view / ryuu-relay-chain 含む)
│   ├── router.test.ts       #   ルーター (13ルート, dashboard 含む)
│   ├── i18n-completeness.test.ts  #  i18n 完全性 (7ロケール)
│   └── service-worker.test.ts     #  Service Worker
├── e2e/                     # E2E テスト (218テスト宣言, 41 specファイル + auth.setup.ts / global-setup.ts / global-teardown.ts / free-port.mjs)
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

### ユニットテスト（1408テスト宣言、133ファイル）

| カテゴリ | テスト数（概算） | 内容 |
|---------|----------------|------|
| GkillAPI | 多数 | 全11データ型の CRUD、設定、共有、セッション管理 |
| データモデル | 29ファイル | 各データ型の生成・代入・シリアライゼーション（DashboardConfig / MiReKyou 含む）+ タグ付与の横断検証 |
| D-note | 8ファイル | 述語関数、キー取得、集計ターゲット、アグリゲータ、トレンドグラフ集計（trend-aggregator）、相関グラフの統計・ペア対応付け（correlation-aggregator） |
| KFTL パーサ | 5ファイル | 構文解析、型判定、リクエスト生成 |
| Composable | 45ファイル | 追加/編集ビュー、コンテキストメニュー、削除確認、クリップボード保存、ページ/クエリ Composable、Kyouビュー（未取得時の日時非表示・読み込み中表示）、IDFKyouビュー（Markdown表示）、ReKyou / MiReKyou ビュー（参照先なしの終端状態、参照先の更新通知での引き直し）、Dnote 中継チェーン（`requested_reload_kyou` が親まで届くこと）、Ryuu 中継チェーン（中継18件が親まで届き、フォーカス2件は通さないこと）、プラグイン本文の iframe 受け渡し（ローダーが名乗ってから注入すること・iframe 内のダブルクリックを親へ撃ち直すこと）、rykv/mi の列×検索ルーティング（検索結果が検索した列に届き最後の条件が勝つこと、フォーカス切替のflush中に届く機械的updated_queryを検索にしないこと）、mi 板選択のフォーカス列追随、KyouListView のローディング表示と `scroll_to` リトライの世代破棄・上限打ち切り、サイドバー子クエリビュー（TimeIs/Map/Calendar）の「props同期ではemitしない」原則、RepQuery のプロファイル×記録分類→記録先詳細算出（例外後も loading が復帰して算出が生き残ること）、検索条件エディタダイアログの初期値規則（query_id空=未セットの印を潰さないこと）、mi サイドバー/検索条件エディタの `inited` 集約（画面から外した節のフラグを残すとスピナーのまま固まるため、フラグ集合を固定する）、パスワードリセットリンク表示（URLにuser_idを載せること・有効期限の表示と期限切れ判定・リンク再発行）と自分自身のアカウント判定 |
| ルーター | 1ファイル | 13ルートの定義と遷移（dashboard 含む） |
| i18n | 1ファイル | 7ロケールのキー完全性 |
| Service Worker | 1ファイル | Workbox プリキャッシュ、POST キャッシュ、プッシュ通知、`/zip_cache/.*` denylist |
| ユーティリティ | 31ファイル | deep-equals, format-date-time, looks-like-url, long-press, save-as, delete-gkill-cache, markdown-to-html, mermaid-render, use-dialog-history-stack, delayed-loading, foldable-struct-move, foldable-struct-check（チェックツリーへの単一パス適用が旧実装＝項目ごと全走査と等価であること。O(項目数×ノード数)化するとrep数百の環境で列フォーカス切替が数秒固まる）, kyou-content-text, kyou-view-relay（イベント中継束の網羅）, cascade-delete-kyou（Kyou削除の連鎖削除）, use-confirm-delete-kyou-view, confirm-dialog-close（確認ダイアログが例外時も閉じる）, edit-view-no-update-check（更新なし判定に関連日時を含める）, use-context-menu-position（コンテキストメニューの座標ターゲット）, kyou-reload（引き直し手順と同時リクエストの合流）, use-kyou-list-view-dialog（DNote から開く一覧ダイアログの伝播）, relay-bundle-source-scan（`v-on` の中継束と `@` の二重配線をソース走査で検出）, use-device-kind（PC / タブレット / スマートフォンの判定とシングルトン性）, linkify-text（本文中URLのセグメント分割）, application-config-update-fields-scan（設定保存の詰め替え網羅をソース走査で検出。漏れると保存のたびにゼロ値へ巻き戻る）, dialog-autofocus（ダイアログを開いたときのフォーカス先選び。ヘッダのチェックボックスや readonly の日付欄を掴まないこと）, kyou-view-height-source-scan（行ではない場所にパーセントの高さを渡していないこと。渡すと MiReKyou の参照先が消える）, mi-board-column-layout（Mi板の列見出しの高さが定数とCSSで一致していること）, use-application-config-view（子ダイアログの適用が props を書き換えないこと・ロケール/テーマがキャンセルで戻ること） |

### E2E テスト (212テスト宣言, 39 specファイル)

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
