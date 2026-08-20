# pages テスト仕様

## 概要

ページ・ビュー・ダイアログのテスト。Playwright による E2E テスト（13ルート全網羅 + CRUD/設定/回帰テスト、44 specファイル250テスト宣言）、Vue Composable のユニットテスト、ルーターのテストで構成される。

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

**移行は完了している。** 免除リスト（かつての `eslint.config.js` の
`e2e/playwright-not-migrated` ブロック）は削除済みで、いまは全 spec に `error` が効く。
CI も `npx eslint --max-warnings 0` で回すので、警告のまま溜めることもできない。

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
| **ログインは IP ごと15分に10回まで**（`gkill_server_api_rate_limit.go`。判定は資格情報を見る**前**なので失敗も成功も1回と数える）。スイート全体のログインは現在5回で、増やすと上限に当たって「ログインしても画面が変わらない」が**別のテストで**出る | 実際にフォームからログインするテストを増やさない。共有 storageState（`auth.setup.ts`）を使う。`--repeat-each` でログイン系 spec を回すのも同じ理由で不可 |
| ログイン失敗も `check_auth` のセッション無効判定と**同じエラーコード**（存在しないユーザIDは ERR000002）を通る。素直に飛ばすと `location.replace("/")` でページごと作り直され、出したばかりのエラー表示が消える | 製品側でログイン画面（`pathname === '/'`）のときは飛ばさないようにしてある（`is_on_login_page`）。`check-auth-login-page.test.ts` が配線を固定する |

## テストフレームワーク

- **E2E**: Playwright
- **ユニット**: Vitest

## テストファイル一覧

### E2E テスト（44 specファイル, 250テスト宣言）

#### ページ表示・ナビゲーション系（14ファイル）

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
| `src/client/__tests__/e2e/shared-mi.spec.ts` | `/shared_mi` — 旧URL。ルータの redirect で `/shared_page` へ引き継がれること（コンポーネントは無い。setup から `router.replace` すると初回ナビゲーションが完了しなくなる） |
| `src/client/__tests__/e2e/register-first-account.spec.ts` | `/register_first_account` — 初回アカウント登録 |
| `src/client/__tests__/e2e/set-new-password.spec.ts` | `/set_new_password` — パスワード再設定 |
| `src/client/__tests__/e2e/rudbeckia.spec.ts` | `/rudbeckia` — ポート。4画面をウィンドウとして開き、ホストしたビューのアプリバーとサイドバーがウィンドウの中に収まること、集計ビューの一覧が高さ0に潰れないこと、画面をまたぐ変更の伝播 |
| `src/client/__tests__/e2e/dashboard.spec.ts` | `/dashboard` — ダッシュボード（ナビゲーション、描画、JSエラーなし） |

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

#### 認証・ユースケース・設定系（21ファイル）

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/e2e/auth-flow.spec.ts` | ログアウト、パスワード未設定ログイン不可、Rep全チェック確認 |
| `src/client/__tests__/e2e/search-and-summary.spec.ts` | RYKV/Mi キーワード検索、D-note サマリトグル |
| `src/client/__tests__/e2e/mi-operations.spec.ts` | タスク板間移動、完了状態切替、サイドバーからの共有管理ダイアログ |
| `src/client/__tests__/e2e/settings-crud.spec.ts` | 設定ページ各セクション表示確認 |
| `src/client/__tests__/e2e/server-config-crud.spec.ts` | サーバ設定機能テスト（プロファイル/TLS/アカウント/Rep管理） |
| `src/client/__tests__/e2e/user-config-crud.spec.ts` | ユーザ設定機能テスト（API/画像/板名/構造CRUD） |
| `src/client/__tests__/e2e/regression-fixes.spec.ts` | 修正済みバグ回帰テスト（8件）。新規タグを付けて追加した記録が**画面遷移せずに**一覧へ残ること（遷移すると既定クエリを作り直すので不具合をすり抜ける）を含む |
| `src/client/__tests__/e2e/misc-operations.spec.ts` | ブックマークレット、GPS、共有リンク、再起動 |
| `src/client/__tests__/e2e/clipboard-save.spec.ts` | Ctrl+V でクリップボード保存ダイアログ表示（RYKV/Mi）、テキスト入力中は抑制、ダイアログ閉じ、クリップボードテキストのプレビュー |
| `src/client/__tests__/e2e/dialog-history.spec.ts` | ダイアログ履歴不変条件（×/Escape/ブラウザバックで閉じてもバックスタックに使用済みエントリが残らない、複数ダイアログを開いたまま画面遷移可能） |
| `src/client/__tests__/e2e/edit-readonly-loading.spec.ts` | Edit系ダイアログの Loading 中 readonly 化とロード完了後の編集可能復帰（API 遅延注入で検証） |
| `src/client/__tests__/e2e/re-kyou.spec.ts` | ReKyou（リポスト）の行を右クリックしたとき、入れ子の参照先ではなくリポスト自身のコンテキストメニューが出ること |
| `src/client/__tests__/e2e/context-menu-viewport.spec.ts` | コンテキストメニューの外接矩形がビューポートに収まること（縦に狭い画面で上向きに反転する経路）。「メニューが出た」だけでは自前クランプ時代でも通るので矩形を測る |
| `src/client/__tests__/e2e/rykv-columns.spec.ts` | rykv の複数列×検索: 別列で検索した結果が検索した列だけに反映される、列リロード・列削除で他列の絞り込みが壊れない |
| `src/client/__tests__/e2e/mi-board-columns.spec.ts` | mi の板列×検索: 各板の列に自板のタスクだけが出る、板クリック後に別列で検索しても板名表示と検索条件が汚染されない |
| `src/client/__tests__/e2e/saved-find-query.spec.ts` | 保存済み検索条件: 設定画面で登録→設定適用→ライフログビューのサイドバーFABから呼び出してサイドバーへ反映、タスク側は未登録なのでFAB非表示 |
| `src/client/__tests__/e2e/rykv-sidebar-defaults.spec.ts` | rykv の既定検索条件と「プロファイル×記録分類→記録先詳細」の算出。列を足すと設定由来の既定条件で検索が飛ぶこと、記録分類のチェック変更が記録先詳細と検索条件へ反映されること |
| `src/client/__tests__/e2e/column-view-initial-load.spec.ts` | 初期検索の完了を待たずに画面を見せること。準備完了の合図はルート要素の `data-gkill-view-ready`（全画面オーバーレイを待つセレクタでは、出る前に `toBeHidden` が通る窓ができる） |
| `src/client/__tests__/e2e/kftl-tabs.spec.ts` | メモ帳のタブ（追加・切替・閉じる・localStorage への永続化）と、IMEで確定してから改行したときの保存マーカー。**IME は CDP の `Input.imeSetComposition` でしか再現できない**（`pressSequentially` は中間の本文を必ず観測してしまい常に緑になる） |
| `src/client/__tests__/e2e/kftl-multi-dialog.spec.ts` | メモ帳ウィンドウを複数枚開く。タブの一覧と中身は共有シングルトン、「いま映しているタブ」だけがウィンドウごと |
| `src/client/__tests__/e2e/dialog-autofocus.spec.ts` | ダイアログを開いたら最初のテキスト入力欄にカーソルが載ること（選び方の判定そのものは `unit/classes/dialog-autofocus.test.ts`） |

### Composable ユニットテスト（59ファイル）

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
| `src/client/__tests__/unit/composables/registered-tag-column-filter.test.ts` | 利用者がその場で作ったタグを列の検索条件へ足す判定（`tags_and` の列は積が必ず空になるので触らないこと・既知タグでは列も引き直しも動かないこと・既知判定は emit 前のその場で行うこと） |
| `src/client/__tests__/unit/composables/new-tag-column-search.test.ts` | 同じ表を useRykvView / useMiView の**両方**へ流す配線テスト（引き直しは列あたり1本・`query_id` 不変・条件が `set_saved_*` に落ちること・親への `registered_tag` 中継を止めないこと） |
| `src/client/__tests__/unit/composables/browse-zip-contents-dialog.test.ts` | ZIPの中を辿るダイアログの純ロジック（フォルダのエントリを持たないZIPでも階層を導出する・画像/テキスト/メディアの巡回が両端で止まる・サイズ表示）。**E2Eには zip の spec が無いのでここが唯一のカバレッジ** |
| `src/client/__tests__/unit/composables/plugin-config-dialog.test.ts` | プラグイン設定ダイアログ。iframe は `allow-same-origin` を持たないので保存を親が肩代わりする経路（`e.source` を緩めると無関係なウィンドウから設定を書き換えられる・保存成功後は設定HTMLを取り直す） |
| `src/client/__tests__/unit/composables/config-struct-sync.test.ts` | 板ツリー・タグツリーのセッション中追随。どちらもサーバに実体が無く、一覧APIから起動時に組み立てられる |
| `src/client/__tests__/unit/composables/confirm-unknown-mi-board.test.ts` | 「新しい板です」確認ゲート。板はサーバ側に実体が無いので、打ち間違いが無言で新しい板を生やす |
| `src/client/__tests__/unit/composables/dashboard-page-reload.test.ts` | useDashboardPage（設定の取得・テーマ・メッセージ・ツリー追随・ログアウトだけの薄いページ）。設定取得の失敗が永久スピナーにならないこと |
| `src/client/__tests__/unit/composables/dashboard-view-reload.test.ts` | useDashboardView の再読込（`registered_kyou` のデバウンス、日付変更と初回ロードの共通化） |
| `src/client/__tests__/unit/composables/dnote-correlation-graph-crud.test.ts` | 相関グラフの CRUD 導線がトレンドグラフと同じ形で端まで繋がっていること |
| `src/client/__tests__/unit/composables/dnote-relay-chain.test.ts` | Dnote の中継チェーンが `requested_reload_kyou` を親まで通すこと（タグ/テキスト/通知の変更はこれしか信号を出さない） |
| `src/client/__tests__/unit/composables/edit-kyou-tags-view.test.ts` | 追加/編集画面に埋め込むタグ欄。値を集めるだけで登録は親の `save()` が行う |
| `src/client/__tests__/unit/composables/edit-mi-board-struct-view.test.ts` | Mi の板構造の編集（削除の walk が「子で true が返ったら親が splice」の形になっていること） |
| `src/client/__tests__/unit/composables/edit-plaing-time-is-dialog.test.ts` | 実行中検索のカスタム条件ダイアログ（「カスタマイズする」チェックの3状態の意味論） |
| `src/client/__tests__/unit/composables/edit-saved-find-query-list-dialog.test.ts` | 保存済み検索条件の一覧編集。受け取ったリストのクローンを編集し、適用まで元を書き換えないこと |
| `src/client/__tests__/unit/composables/find-query-editor-view.test.ts` | 検索条件エディタ（Dnote/Ryuu が使う）。TimeIsのタグツリーへ流すのは `timeis_tags` であって `tags` ではないこと |
| `src/client/__tests__/unit/composables/find-query-editor-dialog-default-signal.test.ts` | 検索条件エディタダイアログの初期値規則（`query_id` が空＝未セットの印を潰さないこと） |
| `src/client/__tests__/unit/composables/find-time-is-query-editor.test.ts` | 実行中検索のカスタム条件エディタが書き込むフィールドの対応 |
| `src/client/__tests__/unit/composables/foldable-struct-device-gates.test.ts` | ツリーの端末種別ゲート（D&Dの可否は `is_pc`、ロングプレス補完は `has_touch`。兼用するとタッチ対応PCでD&Dが死ぬ） |
| `src/client/__tests__/unit/composables/foldable-struct-selected-items.test.ts` | `get_selected_items()` が「入れ物」（フォルダ・ルート）を返さないこと。返すとAND検索が必ず0件になる |
| `src/client/__tests__/unit/composables/kftl-dialog-host.test.ts` | メモ帳ウィンドウの一覧を持つホスト（＋メニューのたびに1枚増える・slot 番号の採番と返却） |
| `src/client/__tests__/unit/composables/kftl-tab-store.test.ts` | メモ帳のタブを持つ共有ストア（モジュールシングルトン。独立した `effectScope` の中で作らないと最初のコンポーネントの unmount で永続化ごと止まる） |
| `src/client/__tests__/unit/composables/kyou-change-propagation.test.ts` | ポートで並べた画面のあいだの変更伝播（自分が出した通知は受けない・`reload_list` は1ドレイン1回に畳む） |
| `src/client/__tests__/unit/composables/kyou-histories-view.test.ts` | 履歴一覧（`related_time` の付け替えが読み直した値を見ること） |
| `src/client/__tests__/unit/composables/kyou-list-view-scroll-to.test.ts` | `scroll_to` の永久リトライ打ち切り（世代カウンタと2秒上限。0件の列で50ms周期の強制レイアウトが増殖していた） |
| `src/client/__tests__/unit/composables/mi-kyou-count-calendar.test.ts` | mi版の件数カレンダー（`is_active` ゲートと日付キー。rykv版と対称） |
| `src/client/__tests__/unit/composables/mi-sidebar-inited.test.ts` | mi サイドバーの節ごとの `inited` フラグ（子が「初回同期か再同期か」を判定する。消すと props 同期のたびにチェックが列をまたいで累積する） |
| `src/client/__tests__/unit/composables/mi-view-initial-load.test.ts` | useMiView の初期化（画面を見せるまで）。`rykv-view-initial-load` と対で、修正が片側にしか入っていないと落ちる |
| `src/client/__tests__/unit/composables/rykv-view-initial-load.test.ts` | useRykvView の初期化（列の骨組みを確定 → 可視化 → 検索の順。`inited` は初期検索の完了に依存しない） |
| `src/client/__tests__/unit/composables/password-reset-link-view.test.ts` | パスワードリセットリンク表示（72時間の期限表示と期限切れ判定・リンク再発行） |
| `src/client/__tests__/unit/composables/registered-kyou-local-insert.test.ts` | 追加された記録を再検索せず列へ差し込む経路（rykv / mi の両方へ同じ表を流す） |
| `src/client/__tests__/unit/composables/rep-query-summary-detail.test.ts` | 「プロファイル×記録分類→記録先詳細」の算出（例外のあとも loading が復帰して算出が生き残ること） |
| `src/client/__tests__/unit/composables/rykv-sidebar-mechanical-emission.test.ts` | サイドバーの機械的な `updated_query` の遮断（値比較ガード。破れると検索中の列をクリックしただけで飛行中の検索が abort される） |
| `src/client/__tests__/unit/composables/ryuu-item-view.test.ts` | 流の1項目（中継束の override。手書きで3件だけ並べていた頃は残り15件を落としていた） |
| `src/client/__tests__/unit/composables/ryuu-view-apply.test.ts` | 流の定義編集（`apply()` は編集結果を親へ渡すだけにする） |
| `src/client/__tests__/unit/composables/server-config-view.test.ts` | サーバ設定画面（props の ServerConfig を複製してから編集する） |
| `src/client/__tests__/unit/composables/sidebar-child-query-sync-emission.test.ts` | サイドバー子クエリビューの「props同期では emit しない」原則（TimeIs / Map / Calendar） |
| `src/client/__tests__/unit/composables/tutorial-on-startup.test.ts` | 起動時チュートリアルは起動時に1回だけ（`application_config` の ref 差し替えで再発火しないこと） |

### ルーターテスト

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/unit/router.test.ts` | コンポーネントを持つ13ルートの定義と遷移、および旧パスを吸収する redirect 専用2ルート（`/regist_first_account` → `/register_first_account`、`/shared_mi` → `/shared_page`。どちらもクエリを引き継ぐ） |

## E2E テストヘルパー

- `src/client/__tests__/e2e/helpers.ts` — 初回登録（reset_token取得→register_first_account）+ テストユーザログイン
- `src/client/__tests__/e2e/crud-helpers.ts` — KFTL送信、ページナビゲーション、コンテキストメニュー、FABクリック、フローティングダイアログ自動閉じ。固定sleepは持たず、`waitForKyouByText`（対象が見つかるまで待つ）/ `clickContextMenuItem`（項目が表示されるまで待つ）/ `clickDialogButton`（書き込みAPIの応答を待ち、errors が空かも見る）のようにweb-firstな待機で組んである
- `src/client/__tests__/e2e/check-server.ts` — サーバヘルスチェック
- `src/client/__tests__/e2e/run-e2e.mjs` — テストランナー（gkill_server自動起動・停止）
- `src/client/__tests__/e2e/free-port.mjs` — OS から空きポートを採番（本番 gkill_server の :9999 と衝突させないため）
- `src/client/__tests__/e2e/auth.setup.ts` — Playwright の setup プロジェクト。ログイン状態を作って保存する
- `src/client/__tests__/e2e/e2e-credentials.ts` — テストユーザのIDとパスワードハッシュ（各 spec へ手書きで散らさない）
- `src/client/__tests__/e2e/global-setup.ts` / `global-teardown.ts` — テスト全体の前後処理

## 実行方法

```bash
# E2E テスト
npm run test_client_e2e

# ユニットテスト（Composable + Router）
npm run test_client_unit
```
