# pages テスト仕様

## 概要

ページ・ビュー・ダイアログのテスト。Playwright による E2E テスト（13ルート全網羅 + CRUD/設定/回帰テスト、42 specファイル230テスト宣言）、Vue Composable のユニットテスト、ルーターのテストで構成される。

## E2E の書き方の規約

E2Eには「静かに成功するテスト」を作りやすい落とし穴が2つある。
`eslint.config.js` の `e2e/playwright` ブロックで機械的に防いでいる。

### 条件で本体を包まない（`playwright/no-conditional-in-test`）

```ts
// NG: レコードが見つからないと何も検証せずパスする
const record = findKyouByText(page, label)
if (await record.count() > 0) {
  ...
}

// OK: 見つからなければ失敗する
const record = await waitForKyouByText(page, label)
```

`expect(app).toBeVisible()` や `content.length > 0` で締めるのも同様に避ける。
常に真になるので、テスト名が主張している内容を確認できていない。

### 固定時間のsleepを使わない（`playwright/no-wait-for-timeout`）

`page.waitForTimeout(2000)` は遅いマシンで足りずにフレークし、
速いマシンでは無駄に待つ。`expect(...).toBeVisible()` /
`toBeHidden()` / `toHaveValue()` など、自動リトライする待機に置き換える。
ページ遷移後の読み込み待ちは `crud-helpers.ts` の `navigateToRykv` 等が
読み込み中オーバーレイの消滅を待つ形になっているのでそれを使う。

### 移行状況

中核のCRUDフロー（`add-dialog-crud` / `edit-dialog-crud` / `delete-crud` /
`notification-crud` / `mi-operations`）と共通ヘルパ（`crud-helpers.ts`）は移行済みで、
新規ファイルには最初から `error` が効く。未移行のファイルは
`eslint.config.js` の `e2e/playwright-not-migrated` ブロックに列挙してあり、
そこだけ `warn` に落としてある。直したらリストから消すこと。

### 硬くしたときに引っかかった落とし穴

条件ガードを外して初めて分かったこと。同じ罠を踏まないよう記録しておく。

| 落とし穴 | 対処 |
|---|---|
| `.v-list-item` でメニュー項目を探すと、rykv サイドバーの記録分類ツリー（「気分」「支出」「タスク」…）に当たる | `.v-menu` 配下に限定（`CONTEXT_MENU_ITEM`）。v-menu は `.v-menu > .v-overlay__content` へ teleport される |
| タグ名・テキスト・通知は一覧とサイドバーの両方に出る | `.tag_wrap` / `.text_content` / `.notification_content` で掴む（`waitForAttachedTag` 等） |
| 「タグ追加」を部分一致で探すと先に並ぶ「タグ履歴から追加」に当たる | `MENU` 定数は前後の空白だけ許した完全一致にしてある |
| 各Editビューは「値が変わっていない」と `*_IS_NO_UPDATE_MESSAGE` を出して保存しない（リクエストも飛ばない） | 開いてそのまま保存するテストは書けない。必ず1項目変えてから保存する |
| 新しいタグを含む保存は確認ダイアログを挟む（KFTL送信・タグ追加ダイアログの両方） | `confirmUnknownTagIfShown` が `submitKftlText` と `clickDialogButton` から呼ばれる |
| 通知の時刻欄は readonly で、v-time-picker から選ばないと保存が通らない | `pickNotificationTime` |
| リポストはコンテキストメニューを押しただけでは作られず、確認ダイアログで確定が要る | `clickDialogButton(page, MENU.rekyou)` |
| TimeIs の終了ボタンは Plaing 画面にしか出ない（rykv は `show_timeis_plaing_end_button="false"`）。終了ダイアログの確定ボタンも「保存」ではなく「終了」 | Plaing へ遷移してから操作する |
| KFTL の `、、` は以降の行をすべてテキスト本文として取り込む | 記録＋テキストを作りたい場合はコンテキストメニューでテキストを付ける |
| Mi のコンテキストメニューに「共有」は無い。共有はサイドバーのフッタから。しかもフッタは `application_config.is_show_share_footer` が既定 false で出ない | 設定画面の「共有フッターを表示」を有効にしてから操作する |
| ダイアログは保存しても閉じるとは限らない（タグ追加など） | 完了の合図は「書き込みAPIの応答」。`clickDialogButton` が待ち、errors が空かも見る |
| 記録に付いたテキストと通知は**一覧に出ない**（`kyou-list-view.vue` が `:show_attached_texts="false"` `:show_attached_notifications="false"` を渡している）。タグだけは出る | `openKyouDetailPane` で Kyou詳細ペインを開き、その配下で確認する |
| 板名は `v-select` で自由入力できない。新しい板は隣の ＋ ボタンが開く「板名追加」ダイアログで作る | `createAndSelectMiBoard` |
| リポストは元の記録と同じ本文で表示されるので、本文だけで掴むと元記録のほうを消してしまう | `waitForKyouRowByRepName(page, 'ReKyou', label)` でリポジトリ名と併せて特定する |
| `.v-card` の `.first()` は外側のコンテナに当たる | 最も内側を取る `.last()` を使う |
| 一覧は仮想スクロールで数件しか描画しない。並列に走る他テストの記録に押し出されて、作ったはずの記録が見つからなくなる | `searchByKeyword` で本文を絞ってから件数や有無を見る。キーワード入力欄は「キーワード」チェックボックスがONのときだけ `v-show` される |
| E2E は **PATH 上の `gkill_server`** を起動する。サーバ側（Go）を直しても、ビルドし直さないと古いバイナリで走る | ビルド先を PATH の先頭に置いて実行する。`npm run install_server` は本番サービスの実体（`go/bin/gkill_server.exe`）を上書きするので、検証だけなら別ディレクトリへ `go build -o` する |
| リポストは元の記録を**入れ子で**描画する（`re-kyou-view.vue` が `<KyouView :show_rep_name="true">` で元Kyouを中に出す）。`.kyou_rep_name` の総数は 元1 + リポスト1 + リポスト内の元1 = 3 になる | 総数で数えずリポジトリ名で絞って数える |
| タグ名を変更すると、rykv のタグ絞り込みに**未チェック**で入り、その記録が一覧から消える（チェック状態は `application_config.tag_struct` に保存される。新規追加したタグは自動でチェックされるので非対称） | 絞り込みの影響を受けない Kyou詳細ペインで確認する |

## テストフレームワーク

- **E2E**: Playwright
- **ユニット**: Vitest

## テストファイル一覧

### E2E テスト（42 specファイル, 230テスト宣言）

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
| `src/client/__tests__/e2e/share-page.spec.ts` | `/shared_page` — 不正な共有IDでエラーが表示され、読み込み中のまま止まらないこと（返る内容の網羅はGo側 handle_get_shared_kyous_test.go） |
| `src/client/__tests__/e2e/shared-mi.spec.ts` | `/shared_mi` — 共有タスク |
| `src/client/__tests__/e2e/register-first-account.spec.ts` | `/register_first_account` — 初回アカウント登録 |
| `src/client/__tests__/e2e/set-new-password.spec.ts` | `/set_new_password` — パスワード再設定 |

#### CRUD 操作フロー系（9ファイル）

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/e2e/kftl-crud.spec.ts` | KFTL経由で全データ型記録 + 画面表示確認 |
| `src/client/__tests__/e2e/kftl-timeis-end.spec.ts` | TimeIs終了の全4バリエーション（タイトル/タグ × 指定/存在すれば） |
| `src/client/__tests__/e2e/add-dialog-crud.spec.ts` | FAB→追加ダイアログ (Mi/Lantana/Nlog/TimeIs/URLog/KC) + コンテキストメニューからのTag/Text追加 |
| `src/client/__tests__/e2e/edit-dialog-crud.spec.ts` | 右クリック→編集 (Kmemo/Mi/TimeIs/Nlog/URLog/Tag/Text) + Plaing画面でのTimeIs終了、リポスト |
| `src/client/__tests__/e2e/delete-crud.spec.ts` | 右クリック→削除 (Kmemo/Mi/Nlog/URLog/TimeIs/Tag/Text/ReKyou)。作成→一覧に出る→削除→一覧から消える、を通しで確認 |
| `src/client/__tests__/e2e/view-browse.spec.ts` | 履歴ダイアログ、混合データ型、Mi ボード/Plaing 表示、ZIPブラウズ |
| `src/client/__tests__/e2e/view-history.spec.ts` | Lantana/Mi/Nlog/URLog/ReKyou/Tag/Text の履歴+リポスト+NoImage確認 |
| `src/client/__tests__/e2e/notification-crud.spec.ts` | Notification の追加/編集/削除/閲覧/履歴 |
| `src/client/__tests__/e2e/mi-re-kyou.spec.ts` | MiReKyou（既存Kyouのタスク化）: rykvのコンテキストメニュー「タスクにする」→Mi画面に出る。API面はGo側の TestHandleAddMiReKyou_* 系へ移管 |

#### 認証・ユースケース・設定系（16ファイル）

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/e2e/auth-flow.spec.ts` | ログアウト、パスワード未設定ログイン不可、Rep全チェック確認 |
| `src/client/__tests__/e2e/search-and-summary.spec.ts` | RYKV/Mi キーワード検索、D-note サマリトグル |
| `src/client/__tests__/e2e/mi-operations.spec.ts` | タスク板間移動、完了状態切替、サイドバーからの共有管理ダイアログ |
| `src/client/__tests__/e2e/settings-crud.spec.ts` | 設定ページ各セクション表示確認 |
| `src/client/__tests__/e2e/server-config-crud.spec.ts` | サーバ設定機能テスト（プロファイル/TLS/アカウント/Rep管理） |
| `src/client/__tests__/e2e/user-config-crud.spec.ts` | ユーザ設定機能テスト（API/画像/板名/構造CRUD） |
| `src/client/__tests__/e2e/regression-fixes.spec.ts` | 修正済みバグ回帰テスト（7件） |
| `src/client/__tests__/e2e/misc-operations.spec.ts` | ブックマークレット、GPS、共有リンク、再起動 |
| `src/client/__tests__/e2e/clipboard-save.spec.ts` | Ctrl+V でクリップボード保存ダイアログ表示（RYKV/Mi）、テキスト入力中は抑制、ダイアログ閉じ、クリップボードテキストのプレビュー |
| `src/client/__tests__/e2e/dashboard.spec.ts` | ダッシュボード画面（ナビゲーション、描画、JSエラーなし確認） |
| `src/client/__tests__/e2e/dialog-history.spec.ts` | ダイアログ履歴不変条件（×/Escape/ブラウザバックで閉じてもバックスタックに使用済みエントリが残らない、複数ダイアログを開いたまま画面遷移可能） |
| `src/client/__tests__/e2e/edit-readonly-loading.spec.ts` | Edit系ダイアログの Loading 中 readonly 化とロード完了後の編集可能復帰（API 遅延注入で検証） |
| `src/client/__tests__/e2e/re-kyou.spec.ts` | ReKyou（リポスト）の行を右クリックしたとき、入れ子の参照先ではなくリポスト自身のコンテキストメニューが出ること |
| `src/client/__tests__/e2e/context-menu-viewport.spec.ts` | コンテキストメニューの外接矩形がビューポートに収まること（縦に狭い画面で上向きに反転する経路）。「メニューが出た」だけでは自前クランプ時代でも通るので矩形を測る |
| `src/client/__tests__/e2e/rykv-columns.spec.ts` | rykv の複数列×検索: 別列で検索した結果が検索した列だけに反映される、列リロード・列削除で他列の絞り込みが壊れない |
| `src/client/__tests__/e2e/mi-board-columns.spec.ts` | mi の板列×検索: 各板の列に自板のタスクだけが出る、板クリック後に別列で検索しても板名表示と検索条件が汚染されない |
| `src/client/__tests__/e2e/saved-find-query.spec.ts` | 保存済み検索条件: 設定画面で登録→設定適用→ライフログビューのサイドバーFABから呼び出してサイドバーへ反映、タスク側は未登録なのでFAB非表示 |

### Composable ユニットテスト（49ファイル）

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/unit/composables/add-views.test.ts` | Mi, Tag, Nlog, URLog, Lantana, TimeIs, KC の追加ビュー Composable |
| `src/client/__tests__/unit/composables/edit-views.test.ts` | Kmemo, Mi, Nlog, URLog, TimeIs, Lantana, KC の編集ビュー Composable |
| `src/client/__tests__/unit/composables/context-menus.test.ts` | 10エンティティ型のコンテキストメニュー Composable（ZIPブラウズ項目含む） |
| `src/client/__tests__/unit/composables/confirm-delete.test.ts` | 削除確認ダイアログ Composable |
| `src/client/__tests__/unit/composables/page-composables.test.ts` | ページレベル Composable |
| `src/client/__tests__/unit/composables/query-composables.test.ts` | クエリ操作 Composable パターン |
| `src/client/__tests__/unit/composables/save-clipboard-to-file-dialog.test.ts` | クリップボード保存ダイアログ Composable（初期状態、MIME判定、ファイルサイズ表示、load_clipboard エラーパス、save_or_confirm、useScopedCtrlVForClipboard キー処理） |
| `src/client/__tests__/unit/composables/idf-kyou-view.test.ts` | IDFKyou ビュー Composable（Markdown リッチ表示、拡張子判定、相対リンクの対象記録解決） |
| `src/client/__tests__/unit/composables/re-kyou-view.test.ts` | ReKyou ビュー Composable（元の記録の解決、リポスト自身のメニュー表示） |
| `src/client/__tests__/unit/composables/mi-re-kyou-view.test.ts` | MiReKyou ビュー Composable（既存記録のタスク化。タイトルを持たず target_id 先の内容を表示する） |
| `src/client/__tests__/unit/composables/gps-log-map.test.ts` | GPSログ地図ビュー Composable |
| `src/client/__tests__/unit/composables/kyou-count-calendar.test.ts` | Kyou件数カレンダー Composable |
| `src/client/__tests__/unit/composables/overlay-and-ur-log-view.test.ts` | オーバーレイ表示と URLog ビュー Composable |
| `src/client/__tests__/unit/composables/kyou-view.test.ts` | Kyou ビュー Composable（ReKyou / MiReKyou の参照先を読み込み中はゼロ値の日時を出さず、読み込み中として扱う） |
| `src/client/__tests__/unit/composables/rykv-view-search-routing.test.ts` | useRykvView の列×検索ルーティング（get_kyous を deferred 化し、検索結果が検索した列に届くこと・同一列は最後の検索条件が勝つこと・列削除中の応答が破棄されること・reload_kyou が別列を潰さないことをレース再現で固定） |
| `src/client/__tests__/unit/composables/mi-view-search-routing.test.ts` | useMiView の列(板)×検索ルーティング（rykv 版と対。mi 固有のカレンダー汚染防止と open_or_focus_board の板一致判定を含む） |
| `src/client/__tests__/unit/composables/mi-board-query.test.ts` | mi サイドバーの板選択がフォーカス列の検索条件に追随すること（追随しないと最後にクリックした板名が別列の条件に混入する） |
| `src/client/__tests__/unit/composables/rykv-sidebar-saved-query-apply.test.ts` | rykv サイドバーの保存済み検索条件適用（query_id は列側維持・updated_query は1回・emit は保存側の clone・0件時は FAB 非表示条件の computed が空） |
| `src/client/__tests__/unit/composables/mi-sidebar-saved-query-apply.test.ts` | mi サイドバーの保存済み検索条件適用（rykv 側と対称＋板名は全クリアと違い保存された条件が勝つこと） |
| `src/client/__tests__/unit/composables/kyou-list-view-loading.test.ts` | KyouListView のローディング表示（set_loading(true) が has_loaded を倒し、再検索中に「該当なし」と誤表示しない） |
| `src/client/__tests__/unit/composables/plugin-html-view.test.ts` | プラグイン本文の iframe 受け渡し（ローダーが `gkill_plugin_loader_ready` を名乗ってから注入すること・ready のたびに送り直すこと・`@load` では送り直さない＝注入ループにしないこと・`gkill_iframe_dblclick` を本物の bubbling する dblclick として撃ち直すこと・一覧は srcdoc 直書きのまま） |
| `src/client/__tests__/unit/composables/ryuu-relay-chain.test.ts` | Ryuu の中継チェーン（`kyou_view_relay_event_names` の18件が RyuuItemView から親まで届くこと・`requested_open_rykv_dialog` は kind と payload ごと通ること・フォーカス2件は通さないこと・二重発火しないこと） |

### ルーターテスト

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/unit/router.test.ts` | 全13ルートの定義と遷移 |

## E2E テストヘルパー

- `src/client/__tests__/e2e/helpers.ts` — 初回登録（reset_token取得→register_first_account）+ テストユーザログイン
- `src/client/__tests__/e2e/crud-helpers.ts` — KFTL送信、ページナビゲーション、コンテキストメニュー、FABクリック、フローティングダイアログ自動閉じ。固定sleepは持たず、`waitForKyouByText`（対象が見つかるまで待つ）/ `clickContextMenuItem`（項目が表示されるまで待つ）/ `clickDialogButton`（書き込みAPIの応答を待ち、errors が空かも見る）のようにweb-firstな待機で組んである
- `src/client/__tests__/e2e/check-server.ts` — サーバヘルスチェック
- `src/client/__tests__/e2e/run-e2e.mjs` — テストランナー（gkill_server自動起動・停止）
- `src/client/__tests__/e2e/free-port.mjs` — OS から空きポートを採番（本番 gkill_server の :9999 と衝突させないため）
- `src/client/__tests__/e2e/auth.setup.ts` — Playwright の setup プロジェクト。ログイン状態を作って保存する
- `src/client/__tests__/e2e/global-setup.ts` / `global-teardown.ts` — テスト全体の前後処理

## 実行方法

```bash
# E2E テスト
npm run test_client_e2e

# ユニットテスト（Composable + Router）
npm run test_client_unit
```
