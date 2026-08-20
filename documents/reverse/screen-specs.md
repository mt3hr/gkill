# 画面仕様（各画面の項目定義）

## 概要

本資料では、gkillフロントエンドの各画面の目的・構成・項目定義を説明します。既存の[screen-transition.md](screen-transition.md)（遷移図）を補完する形で、画面ごとの仕様を記述しています。

### コンポーネント統計

| カテゴリ | 数 |
|---|---|
| ルート | 13 |
| ページコンポーネント | 15 |
| ビューコンポーネント | 202 |
| ダイアログコンポーネント | 116 |
| **コンポーネント合計** | **333**（ルートを除く。ビュー202 + ダイアログ116 + ページ15） |

## 1. ルート定義

`src/client/router/index.ts` で定義される15ルート（コンポーネント13 + リダイレクト専用2）です。
旧パスの吸収はコンポーネントではなく**ルータの `redirect`** で行います（`<script setup>` に
top-level await のあるページの setup から `router.replace` すると、初回ナビゲーションが完了しなくなるため）。

| パス | コンポーネント | 画面名 | 説明 |
|---|---|---|---|
| `/` | login-page | ログイン | ユーザー認証 |
| `/kftl` | kftl-page | KFTL入力 | テキスト形式の一括記録入力 |
| `/mi` | mi-page | タスクボード | タスク管理（カンバン） |
| `/rykv` | rykv-page | 履歴閲覧 | Kyou一覧・検索 |
| `/kyou` | kyou-page | 記録詳細 | 個別記録の詳細表示 |
| `/mkfl` | mkfl-page | 打刻メモ帳 | KFTL入力 + 稼働中TimeIs表示の複合ビュー |
| `/plaing` | plaing-time-is-page | 打刻一覧 | アクティブな打刻セッション |
| `/dashboard` | dashboard-page | ダッシュボード | 日次サマリー画面（Dnote・GPS・MI一覧を1画面に集約） |
| `/rudbeckia` | rudbeckia-page | ポート | 4画面をウィンドウとして開ける単一画面（開発コード rudbeckia） |
| `/saihate` | saihate-page | さいはて | 記録特化画面（FABから各種記録を素早く追加、他画面への遷移なし） |
| `/set_new_password` | set-new-password-page | パスワード変更 | パスワードリセット |
| `/register_first_account` | register-first-account-page | 初回登録 | 初回アカウント作成（旧 `/regist_first_account` はリダイレクト） |
| `/shared_page` | shared-page | 共有ページ | 共有コンテンツの表示 |
| `/shared_mi` | （リダイレクト専用。コンポーネント無し） | — | 旧URL。`/shared_page?share_id=…` へクエリごと引き継ぐ |

## 2. 各ページの詳細仕様

### 2.1 ログイン画面（`/`）

**コンポーネント:** `login-page.vue` → `login-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| ユーザーID | 入力 | テキストフィールド |
| パスワード | 入力 | パスワードフィールド |
| ログインボタン | 操作 | 認証実行 |
| バージョン表示 | 表示 | アプリバージョンバッジ |
| メッセージ/エラー | 表示 | アラートツールチップ |

### 2.2 KFTL入力画面（`/kftl`）

**コンポーネント:** `kftl-page.vue` → `kftl-view.vue`

gkill独自のテキスト形式（KFTL）で複数種類の記録を一括入力する画面です。

| 項目 | 種別 | 説明 |
|---|---|---|
| KFTLテキストエリア | 入力 | 複数行テキスト入力（KFTL書式） |
| タブ | 操作 | 書きかけの下書きを複数持つ。`+` で追加、`×` で閉じる（内容が残っていれば確認）。保存に成功したタブは閉じる（後述） |
| テンプレートボタン群 | 操作 | 定型文テンプレートの挿入。**上書きではなく新しいタブで開く** |
| 送信ボタン | 操作 | KFTLテキストの送信・実行 |
| 未知タグ確認 | 表示 | 送信前に、既存タグに無いタグが含まれていれば確認を求める（後述） |
| ナビゲーションメニュー | 操作 | rykv/mi/kftl/plaing/mkfl/saihate間の切替 |
| ローディングオーバーレイ | 表示 | 非同期処理中の表示 |

**未知タグ確認:** 送信ボタンを押すと、リクエスト構築後・実行前に `collect_unknown_tags()`
（`classes/use-kftl-view.ts`）が既存タグに無いタグを収集する。1件でもあれば実行を中断して
確認を表示し、承認されたら `do_submit(skip_unknown_tag_check = true)` で再実行する。
タイプミスによるタグの乱立を防ぐための仕組み。

この確認は `Teleport` でインライン描画されるため、対応する `*-dialog.vue` ファイルは存在しない。
同じ仕組みは `add-tag-view.vue` にもある（i18n キー `CONFIRM_UNKNOWN_TAG_MESSAGE`）。

**タブ:** タブの一覧と中身は共有シングルトンの `classes/use-kftl-tabs.ts` が持ち、localStorage の
`kftl_tabs` に永続化される。KFTLView は `/kftl` ページ・メモ帳ダイアログ・打刻メモ帳の3系統から
使われ、メモ帳ダイアログは複数枚開けるため、インスタンスごとに配列を持つと片方が
もう片方のタブを消してしまう。タブは常に1枚以上で、最後の1枚を閉じると空のタブが1枚できる。

**「いま映しているタブ」はウィンドウごと**（`classes/use-kftl-view.ts` が自前で持つ）。
ストアが持つのは `last_active_tab_id`（次に開くウィンドウの初期表示）だけで、
これによりウィンドウを並べて別々の下書きを見比べられる。共有しているのは中身なので、
2枚が同じタブを映していれば片方の入力がもう片方にも即出る。
別のウィンドウがタブを閉じたときは、`tabs` の新旧 id を比べて隣のタブへ移る。

タブ列は `v-card-title` に同居させる。別の行にするとテキストエリアの縦がタブバーのぶん削られる。
`title_height` は `tab_bar_height + 4` で、実寸は `.kftl_title` の CSS で固定する
（`v-card-title` に `height` prop は無いので、渡すだけでは効かない）。

`v-window` は使わず、**アクティブなタブ1枚だけ**を描画する。非表示の textarea は
`clientWidth` が0になり、`kftl-statement-line.ts` の行数計算が `parseInt("Infinity")` = `NaN`
に落ちて行ラベルが消えるため。

送信対象のタブは `do_submit(target_tab_id, ...)` の引数で渡す。未知タグ確認・未知板名確認では
`do_submit()` をいったん抜けて応答を待つが、gkill のフローティングダイアログは非モーダル
（`App.vue` の `.gkill-float-scrim` が `pointer-events: none`）なので、確認中でも背後の
タブバーは押せてしまう。持ち越し用の `submit_target_tab_id` を読むのは確認からの続行だけで、
新しい送信は必ずアクティブなタブを対象にする（確認を Escape / ブラウザバックで閉じられて
古い値が残っても、次の保存が別のタブへ誤配送されない）。

タブ操作のロックは `is_submitting || show_confirm_unknown_tag_dialog`。`is_requested_submit`
は設定の読み込みが終わるまで真なので鍵に使えず、板名確認は閉じ方によっては
`unknown_mi_boards` が空にならないのでロック条件に入れていない。

### 2.3 タスクボード画面（`/mi`）

**コンポーネント:** `mi-page.vue` → `mi-view.vue`

タスク（Mi）をカンバンボード形式で管理する画面です。

| 項目 | 種別 | 説明 |
|---|---|---|
| ボード選択 | 入力 | タスクボードの切替（mi-board-query） |
| ソート順 | 入力 | 並び順の選択（mi-sort-type-query） |
| 完了状態フィルタ | 入力 | チェック状態での絞り込み（mi-extract-check-state-query） |
| タスクリスト | 表示 | タスク一覧（mi-kyou-view） |
| タスク追加ボタン | 操作 | 新規タスク作成ダイアログ表示 |
| クエリエディタサイドバー | 操作 | 高度な検索条件設定 |
| 保存済み検索条件FAB | 操作 | サイドバー右下の呼び出しボタン。設定画面で登録したタスク検索条件を選択してサイドバーへ適用（0件時は非表示） |
| アラート通知 | 表示 | 期限アラート等の通知表示 |
| クリップボードを保存 | 操作 | クリップボード内容をファイルとして保存するダイアログ（`save-clipboard-to-file-dialog.vue`） |

### 2.4 履歴閲覧画面（`/rykv`）

**コンポーネント:** `rykv-page.vue` → `rykv-view.vue`

全種類の記録（Kyou）を時系列で閲覧・検索する画面です。

| 項目 | 種別 | 説明 |
|---|---|---|
| キーワード検索 | 入力 | 全文検索（keyword-query） |
| 期間指定 | 入力 | 日付範囲（calendar-query） |
| タグフィルタ | 入力 | タグでの絞り込み（tag-query） |
| 打刻フィルタ | 入力 | 打刻期間での絞り込み（time-is-query） |
| リポジトリフィルタ | 入力 | リポジトリでの絞り込み（rep-query） |
| 地図フィルタ | 入力 | GPS位置での絞り込み（map-query） |
| Kyouリスト | 表示 | 記録一覧（kyou-list-view） |
| ファイルアップロード | 操作 | ファイル追加ダイアログ |
| クリップボードを保存 | 操作 | クリップボード内容をファイルとして保存するダイアログ（`save-clipboard-to-file-dialog.vue`） |
| クエリエディタサイドバー | 操作 | 高度な検索条件設定 |
| 保存済み検索条件FAB | 操作 | サイドバー右下の呼び出しボタン。設定画面で登録したライフログ検索条件を選択してサイドバーへ適用（0件時は非表示） |

### 2.5 記録詳細画面（`/kyou`）

**コンポーネント:** `kyou-page.vue` → `kyou-view.vue`

個別の記録を詳細表示する多態コンポーネントです。全12データ型（kmemo / kc / urlog / nlog / timeis / mi / lantana / idf_kyou / git_commit_log / rekyou / mirekyou / plugin）に対応します。

| 項目 | 種別 | 説明 |
|---|---|---|
| 記録本体 | 表示 | データ型に応じた表示（テキスト/数値/URL等） |
| 作成日時 | 表示 | MI用の作成・見積開始/終了・期限表示 |
| 経過時間 | 表示 | TimeIs用の経過時間表示 |
| 打刻終了ボタン | 操作 | アクティブなTimeIsの終了 |
| タグ一覧 | 表示 | attached-tag コンポーネント群 |
| テキスト一覧 | 表示 | attached-text コンポーネント群 |
| 通知一覧 | 表示 | attached-notification コンポーネント群 |
| リポジトリ名 | 表示 | 格納先リポジトリの表示 |
| 関連時間情報 | 表示 | 関連する打刻情報 |
| コンテキストメニュー | 操作 | 右クリック操作メニュー |

### 2.6 打刻メモ帳画面（`/mkfl`）

**コンポーネント:** `mkfl-page.vue` → `mkfl-view.vue` → `kftl-view.vue` + `plaing-time-is-view.vue`

打刻（TimeIs）とメモ（KFTL入力）を組み合わせた複合入力画面です。画面を上下に分割し、上半分でKFTLテキスト入力、下半分で稼働中のTimeIsを表示します。

| 項目 | 種別 | 説明 |
|---|---|---|
| KFTLテキストエリア | 入力 | 上半分：KFTL書式のテキスト入力 |
| テンプレートボタン群 | 操作 | 定型文テンプレートの挿入 |
| 送信ボタン | 操作 | KFTLテキストの送信・実行 |
| アクティブ打刻リスト | 表示 | 下半分：稼働中TimeIsの一覧・終了操作 |

### 2.7 打刻一覧画面（`/plaing`）

**コンポーネント:** `plaing-time-is-page.vue` → `plaing-time-is-view.vue`

アクティブな打刻（TimeIs）セッションの一覧と操作を行う画面です。

| 項目 | 種別 | 説明 |
|---|---|---|
| アクティブ打刻リスト | 表示 | 現在進行中の打刻一覧 |
| リアルタイムタイマー | 表示 | 各打刻の経過時間表示 |
| 終了ボタン | 操作 | 打刻セッションの終了 |
| 全データ型のCRUDイベント | 操作 | 打刻関連の記録操作 |
| クリップボードを保存 | 操作 | クリップボード内容をファイルとして保存するダイアログ（`save-clipboard-to-file-dialog.vue`） |

### 2.8 ダッシュボード画面（`/dashboard`）

**コンポーネント:** `dashboard-page.vue` → `use-dashboard-page.ts`

特定日の記録を俯瞰する日次サマリー画面です。DnoteView・GPSLogMap・KyouListView（MI一覧）を1画面に集約して表示します。

#### ツールバー

| 項目 | 種別 | 説明 |
|---|---|---|
| 前日ボタン（`<`） | 操作 | 表示日を1日前に移動 |
| 日付表示 | 表示 | 現在表示中の日付（クリックでカレンダーピッカーを表示） |
| 翌日ボタン（`>`） | 操作 | 表示日を1日後に移動 |
| カレンダーピッカー | 操作 | 任意の日付にジャンプ |
| 設定ボタン（歯車） | 操作 | EditDashboardDialog を開き、MI検索条件・Dnote検索条件を設定 |
| ヘルプボタン | 操作 | ヘルプ情報の表示 |

#### 表示エリア構成

| エリア | コンポーネント | 配置 | 説明 |
|---|---|---|---|
| 上半分左 | `dnote-view.vue` | 左半分（30vh高さ） | 表示日のDnote集計ノート一覧 |
| 上半分右 | `gps-log-map.vue` | 右半分（30vh高さ） | 表示日のGPS移動軌跡地図 |
| 下部 | `kyou-list-view.vue` | 全幅 | MI一覧（DashboardConfigのdashboard_mi_find_kyou_queryで絞り込み） |

#### DnoteView（集計ビュー）の構成要素

DnoteView はダッシュボードのほか rykv 画面からも利用される。定義タブ内に以下4種類の集計要素を配置できる（フローティング「＋」メニューから追加）:

| 要素 | コンポーネント | 説明 |
|---|---|---|
| 集計項目 | `dnote-item-view.vue` | 単一の集計値（件数・合計・平均等）を表示 |
| 集計リスト | `dnote-list-view.vue` | 条件に合致するKyou一覧を表示 |
| トレンドグラフ | `dnote-trend-graph-view.vue` | 時系列の集計値をスパークライン（折れ線/棒）で表示。集計粒度（日/週/月）・グラフ種別を設定可能。ダブルクリックで編集、右クリックでコンテキストメニュー（編集/削除）、ドラッグ&ドロップで並べ替え |
| 相関グラフ | `dnote-correlation-graph-view.vue` | 2～10指標のPearson／Spearman相関をヒートマップと散布図で表示。共通粒度と方向付きlagを設定可能。行列は7指標まで400px幅の列に横スクロールなしで収まる（トラック幅の根拠は `classes/dnote-correlation-matrix-layout.ts`。8指標以上は行列だけが横スクロールする） |

Dnote 関連のコンポーネントは他に以下がある（追加・編集・削除確認の3点セット）。

| 種別 | コンポーネント |
|---|---|
| ビュー | `dnote-item-table-view.vue`, `dnote-list-table-view.vue`, `dnote-item-list-view.vue`, `edit-dnote-card.vue`, `edit-dnote-predicate-group.vue`, `aggregated-list-item.vue`, `dnote-trend-graph-table-view.vue`, `dnote-trend-graph-context-menu.vue`, `dnote-correlation-graph-table-view.vue`, `dnote-correlation-graph-context-menu.vue`, `dnote-correlation-graph-editor-view.vue`, `add-dnote-correlation-graph-view.vue`, `edit-dnote-correlation-graph-view.vue`, `confirm-delete-dnote-correlation-graph-view.vue` |
| ダイアログ | `add-dnote-item-dialog.vue`, `add-dnote-list-dialog.vue`, `edit-dnote-item-dialog.vue`, `edit-dnote-list-dialog.vue`, `edit-dnote-dialog.vue`, `confirm-delete-dnote-item-list-dialog.vue`, `confirm-delete-dnote-list-query-dialog.vue`, `add-dnote-trend-graph-dialog.vue`, `edit-dnote-trend-graph-dialog.vue`, `confirm-delete-dnote-trend-graph-dialog.vue`, `add-dnote-correlation-graph-dialog.vue`, `edit-dnote-correlation-graph-dialog.vue`, `confirm-delete-dnote-correlation-graph-dialog.vue` |

> 相関グラフも他の集計要素と同じ3点セット（追加・編集・削除確認）に揃えてある。
> 追加ダイアログは `dnote-view.vue` が直接持ち、編集・削除ダイアログとコンテキストメニューは
> 各グラフのビューが持つ。以前は1つのダイアログでモードを切り替え、表ビューが抱えていたため、
> 「＋」メニューから2段のテンプレート ref を辿る必要があり、途中が null だと無言で失敗していた。

集計は全てクライアント側で行われ、専用のバックエンドAPIは存在しない
（`classes/dnote/dnote-trend-aggregator.ts` + `classes/dnote/dnote-trend/`、
`classes/dnote/dnote-correlation-aggregator.ts` + `classes/dnote/dnote-correlation/`）。

#### FABメニュー（右下）

| 項目 | 種別 | 説明 |
|---|---|---|
| KFTL入力 | 操作 | KFTLテキスト入力ダイアログを開く |
| MKFL入力 | 操作 | ファイルアップロードダイアログを開く |
| KC入力 | 操作 | 数値記録ダイアログを開く |
| Urlog入力 | 操作 | ブックマーク追加ダイアログを開く |
| TimeIs入力 | 操作 | 打刻開始ダイアログを開く |
| Mi入力 | 操作 | タスク追加ダイアログを開く |
| Nlog入力 | 操作 | 支出記録ダイアログを開く |
| Lantana入力 | 操作 | 気分記録ダイアログを開く |
| ファイルアップロード | 操作 | ファイルアップロードダイアログを開く |

#### キーボードショートカット

| キー | 動作 |
|---|---|
| Enter | KFTLダイアログを開く |
| Ctrl+V | クリップボード保存ダイアログを開く |

### 2.9 さいはて画面（`/saihate`）

**コンポーネント:** `saihate-page.vue`

記録に特化した画面です。他画面への遷移はなく、フローティングアクションボタン（FAB）から各種記録を素早く追加できます。「さいはてでも記録ができる」がコンセプトです。

| 項目 | 種別 | 説明 |
|---|---|---|
| FABメニュー | 操作 | 以下の記録追加ダイアログを開く |
| └ KFTL入力 | 操作 | KFTLテキスト入力ダイアログ |
| └ MKFL入力 | 操作 | ファイル追加ダイアログ |
| └ KC入力 | 操作 | 数値記録ダイアログ |
| └ Urlog入力 | 操作 | ブックマーク追加ダイアログ |
| └ TimeIs入力 | 操作 | 打刻開始ダイアログ |
| └ Mi入力 | 操作 | タスク追加ダイアログ |
| └ Nlog入力 | 操作 | 支出記録ダイアログ |
| └ Lantana入力 | 操作 | 気分記録ダイアログ |
| └ ファイルアップロード | 操作 | ファイルアップロードダイアログ |
| ヘルプボタン | 操作 | ヘルプダイアログ（`help-dialog.vue`） |
| リロードボタン | 操作 | データ再読込 |
| ログアウトボタン | 操作 | ログアウト確認ダイアログ |

### 2.10 パスワード変更画面（`/set_new_password`）

**コンポーネント:** `set-new-password-page.vue` → `set-new-password-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| 新パスワード | 入力 | パスワードフィールド |
| パスワード確認 | 入力 | 確認用パスワードフィールド |
| 変更ボタン | 操作 | パスワード更新実行 |
| バージョン表示 | 表示 | アプリバージョン |

### 2.11 初回アカウント登録画面（`/register_first_account`）

**コンポーネント:** `register-first-account-page.vue` → `register-first-account-view.vue`（入力フォームの実体はビュー側）

初回起動時のアカウント作成画面です。

| 項目 | 種別 | 説明 |
|---|---|---|
| ユーザーID | 入力 | アカウント名 |
| パスワード | 入力 | パスワード設定 |
| 管理用パスワード | 入力 | 管理者パスワード |
| 登録ボタン | 操作 | アカウント作成実行 |

### 2.12 共有ページ（`/shared_page`）

**コンポーネント:** `shared-page.vue`

共有された記録を表示するルーターページです。共有IDパラメータに基づいてMiまたはRYKVの共有ビューを表示します。

### 2.13 ポート画面（`/rudbeckia`）

**コンポーネント:** `rudbeckia-page.vue` → `rudbeckia-page-dialog-host.vue` → `rudbeckia-page-dialog.vue`

背景と「+」ボタンだけの1画面で、ライフログビュー / タスク / 実行中 / ダッシュボードを**フローティングウィンドウ**として開ける画面です。開発コードは rudbeckia（URL・ファイル名・識別子に使う）、ユーザ向けの呼び名は「ポート」。

さいはてとの違いは、閲覧系の画面を出せることと、タイトルメニューからページ遷移できることです。

| 項目 | 種別 | 説明 |
|---|---|---|
| タイトルメニュー | メニュー | 他画面へのページ遷移（さいはてには無い） |
| 設定 / ヘルプ | ボタン | アプリバー右側。再読込とログアウトは置かない（操作はウィンドウとFABに寄せる） |
| 「+」→「画面」 | メニュー | ライフログビュー / タスク / 実行中 / ダッシュボードをウィンドウとして開く。同じ画面を上限まで並べられ、上限では最前面へフォーカスするだけ |
| 「+」→「記録」 | メニュー | メモ帳・打刻メモ帳・数値記録・ブックマーク・打刻帳・タスク・支出・気分・アップロード・クリップボードを保存 |

**ウィンドウの仕組み:** ホストするのはページではなく**ビュー**（`rykv-view` / `mi-view` / `plaing-time-is-view` / `dashboard-view`）です。ビューは自前の `v-app-bar` と `v-navigation-drawer` を持っているので、Vuetify の**入れ子レイアウト**（`<v-layout>` で包む）でウィンドウの中へ収めます。包まないとレイアウト部品が `position: fixed` のまま画面最上部へ飛びます。

**ホスト時のビューの差分:** 自前のFABを出さない（ポートのFABが唯一）、Enter/Ctrl+V のショートカットを登録しない（`window` レベルなので枚数ぶん多重登録される）、画面切替メニューはページ遷移せず `requested_navigate_page` を上げる（`reset_dialog_history()` はモジュール共有なので、呼ぶと並べている他のウィンドウまで一斉に閉じる）。

**ウィンドウの中身が縦に潰れないようにする:** ウィンドウの `v-card` は「中身の高さに合わせる」既定のままだと、
中の `display: flex` なビュー（集計ビュー等）が高さ0に潰れます。ホスト側で
`.v-card.fill_height_mode` の例外を置き、ホストされる側は
`.fill_height_mode :deep(...)` で最低高さを持たせます。

- **セレクタの詳細度を落とさないこと。** ホストの `v-card` に効いている既定が詳細度3なので、
  例外側も同じ詳細度で書かないと勝てない
- **一覧の区画には `:has()` を付けて、中身があるときだけ最低高さを与える。**
  無条件に与えると、一覧を出していないときに空白の帯が残る
- この壊れ方は CSS のカスケードなのでユニットテストでは捕まらない。
  回帰は E2E（`rudbeckia.spec.ts`）が `getComputedStyle` と `getBoundingClientRect` で見る
## 3. データ型別CRUD画面仕様

### CRUD対応マトリクス

| データ型 | 追加View | 追加Dialog | 編集View | 編集Dialog | 表示View | 削除確認 | 履歴 |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| kmemo | — | — | ✅ | ✅ | ✅ | — | — |
| mi | ✅ | ✅ | ✅ | ✅ | ✅ | — | — |
| timeis | ✅ | ✅ | ✅ | ✅ | ✅ | — | — |
| lantana | ✅ | ✅ | ✅ | ✅ | ✅ | — | — |
| nlog | ✅ | ✅ | ✅ | ✅ | ✅ | — | — |
| urlog | ✅ | ✅ | ✅ | ✅ | ✅ | — | — |
| kc | ✅ | ✅ | ✅ | ✅ | ✅ | — | — |
| tag | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| text | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| notification | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| idf_kyou | — | — | ✅ | ✅ | ✅ | ✅ | — |
| re_kyou | — | — | ✅ | ✅ | ✅ | — | — |
| mi_re_kyou | ✅ | ✅ | ✅ | ✅ | ✅ | — | — |
| git_commit_log | — | — | — | — | ✅ | — | — |
| plugin | — | — | — | — | ✅ | — | — |
| kyou (共通) | — | — | — | — | ✅ | ✅ | ✅ |

**備考:**
- **kmemo**: 追加はKFTL経由で行う
- **idf_kyou**: 追加はMKFL（ファイルアップロード）経由で行う
- **re_kyou** / **mi_re_kyou**: 追加は既存記録のコンテキストメニューから行う
- **git_commit_log** / **plugin**: 読み取り専用。gkill 側から追加・編集・削除はできない
- **kyou-view**: 全12データ型を多態的に表示する統合コンポーネント（`kyou.ts` の `typed_*` スロットと1対1）
- **削除確認・履歴**: 個別の「—」は専用コンポーネントが存在しないことを示す。全データ型の削除確認は `confirm-delete-kyou-view` (共通)、履歴表示は `kyou-histories-view` (共通) で対応する

### Kyou削除時の連鎖削除

`confirm-delete-kyou-view`（`classes/use-confirm-delete-kyou-view.ts`）の削除は `classes/cascade-delete-kyou.ts` の `cascade_delete_kyou()` に委譲され、対象 Kyou 単体では終わらない。

1. 探索（read のみ）: 対象 Kyou に付随する Tag / Text / Notification と、その Kyou を参照している ReKyou / MiReKyou を幅優先で収集する。参照元をさらに辿るため入れ子のリポストも対象になる。循環参照は訪問済みIDで止め、深さ上限は32（超過時 `ERR900093 cascade_delete_depth_exceeded`）。
2. 削除（write）: 集めたものを全件 `is_deleted=true` で論理削除する。Kyou 自身は**最後**に消す（先に消すとサーバの FindKyous が参照元を結果から外し、途中で失敗したときに残骸を再発見できなくなるため）。
3. 画面からは消したID分だけ `deleted_kyou` を発行して取り除く（全件再検索はしない）。

TXID / commit_tx は使わないので部分確定しうるが、追記型DAOのため同じダイアログをもう一度開いて再実行すれば収束する。失敗時は `ERR900094 cascade_delete_failed`（i18n キー `FAILED_CASCADE_DELETE_KYOU_MESSAGE`）。共有画面では削除自体を行わない。

### 3.1 kmemo（テキストメモ）画面仕様

**追加:** KFTL入力画面経由

**表示:** `kmemo-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| メモ本文 | 表示 | プレフォーマットテキスト |
| 作成日時 | 表示 | 記録日時 |

**編集:** `edit-kmemo-view.vue` / `edit-kmemo-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| メモ本文 | 入力 | テキストエリア |
| 日時 | 入力 | 日付・時刻ピッカー |
| 関連日付 | 入力 | 関連日付の選択 |

### 3.2 mi（タスク）画面仕様

**追加:** `add-mi-view.vue` / `add-mi-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| タスク名 | 入力 | タスクのタイトル |
| ボード | 入力 | 所属ボードの選択 |
| 通知設定 | 入力 | 期限通知の追加（add-notification-for-add-mi-view） |

**表示:** `mi-kyou-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| タスク名 | 表示 | タスクタイトル |
| チェック状態 | 表示/操作 | 完了/未完了トグル |
| ボード名 | 表示 | 所属ボード |
| 作成日時 | 表示 | MI作成日時 |
| 見積開始/終了 | 表示 | 見積期間 |
| 期限 | 表示 | タスク期限 |

**編集:** `edit-mi-view.vue` / `edit-mi-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| タスク名 | 入力 | タスクタイトル編集 |
| ボード | 入力 | ボード変更 |
| チェック状態 | 入力 | 完了状態の変更 |
| 通知設定 | 入力 | 通知の追加・編集 |

#### リポストタスク（MiReKyou）

既存のKyouをタスク化したもの。mi画面ではMiと同じ板に並び、同じ並び替え・チェック操作・板間ドラッグ移動ができる。
タイトルは持たず、カード内には対象Kyouがそのまま描画される。

**追加:** `add-mi-re-kyou-view.vue` / `add-mi-re-kyou-dialog.vue`
（各Kyouのコンテキストメニュー「タスクにする」から開く）

| 項目 | 種別 | 説明 |
|---|---|---|
| ボード | 入力 | 所属ボードの選択（新規板名の追加も可） |
| 見積開始/終了 | 入力 | 日付・時刻ピッカー（クリア/現在ボタン付き） |
| 期限 | 入力 | 日付・時刻ピッカー（クリア/現在ボタン付き） |
| 通知設定 | 入力 | 期限通知の追加 |
| 対象Kyou | 表示 | タスク化する対象のプレビュー |

**表示:** `mi-re-kyou-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| チェック状態 | 表示/操作 | 完了/未完了トグル |
| ボード名 | 表示 | 所属ボード |
| 対象Kyou | 表示 | リポスト元のKyouをそのまま描画 |
| 見積開始/終了・期限 | 表示 | 設定されている場合のみ表示 |

**編集:** `edit-mi-re-kyou-view.vue` / `edit-mi-re-kyou-dialog.vue`（タイトル欄なし）

### 3.3 timeis（打刻）画面仕様

**追加:** `add-time-is-view.vue` / `add-time-is-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| タイトル | 入力 | 打刻セッション名 |
| 開始日時 | 入力 | 開始日時の指定 |

**表示:** `time-is-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| タイトル | 表示 | セッション名 |
| 開始日時 | 表示 | 開始時刻 |
| 終了日時 | 表示 | 終了時刻（未終了は空） |
| 経過時間 | 表示 | 開始から終了（または現在）までの経過 |

**アクティブ打刻:** `plaing-time-is-view.vue` / `attached-time-is-plaing.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| リアルタイムタイマー | 表示 | 経過時間のリアルタイム更新 |
| 終了ボタン | 操作 | 打刻終了（end-time-is-plaing-view） |

### 3.4 lantana（気分）画面仕様

**追加:** `add-lantana-view.vue` / `add-lantana-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| 気分値 | 入力 | 0〜10のスライダー/数値入力 |

**表示:** `lantana-view.vue` / `lantana-flowers-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| 気分値 | 表示 | 数値（0〜10） |
| 花ビジュアライゼーション | 表示 | 気分を花で表現（lantana-flower） |

### 3.5 nlog（支出）画面仕様

**追加:** `add-nlog-view.vue` / `add-nlog-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| 店名 | 入力 | 店名（Shop） |
| タイトル | 入力 | 支出の内容（Title） |
| 金額 | 入力 | 金額入力（Amount） |
| 日時 | 入力 | 記録日時 |

**表示:** `nlog-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| 店名 | 表示 | 店名 |
| タイトル | 表示 | 支出の内容 |
| 金額 | 表示 | 支出金額 |
| 記録日時 | 表示 | 記録日時 |

### 3.6 urlog（ブックマーク）画面仕様

**追加:** `add-ur-log-view.vue` / `add-ur-log-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| URL | 入力 | URLフィールド |
| タイトル | 入力 | ページタイトル |

**表示:** `ur-log-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| URL | 表示 | リンク付きURL |
| タイトル | 表示 | ブックマークタイトル |

### 3.7 kc（数値）画面仕様

**追加:** `add-kc-view.vue` / `add-kc-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| 数値 | 入力 | 数値フィールド |

**表示:** `kc-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| 数値 | 表示 | 記録された数値 |

### 3.8 tag（タグ）画面仕様

完全なCRUD + 履歴を持つ付帯データ型です。

**追加:** `add-tag-view.vue` / `add-tag-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| タグ名 | 入力 | タグ文字列（未知タグ時は確認ダイアログ表示） |

**表示:** `tag-view.vue` / `attached-tag.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| タグ名 | 表示 | タグ文字列 |

**削除確認:** `confirm-delete-tag-view.vue` / `confirm-delete-tag-dialog.vue`

**履歴:** `tag-histories-view.vue` / `tag-histories-dialog.vue`

**タグ構造管理:**
- `edit-tag-struct-view.vue` — タグ階層構造の編集
- `edit-tag-struct-element-view.vue` — 個別タグ要素の編集
- `add-new-tag-struct-element-view.vue` — 新規タグ要素の追加
- `confirm-delete-tag-struct-view.vue` — タグ構造の削除確認

### 3.9 text（テキスト）画面仕様

記録に付与する補足テキスト。完全なCRUD + 履歴を持ちます。

**追加:** `add-text-view.vue` / `add-text-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| テキスト本文 | 入力 | テキストエリア |

**表示:** `text-view.vue` / `attached-text.vue`

**削除確認:** `confirm-delete-text-view.vue`

**履歴:** `text-histories-view.vue` / `text-histories-dialog.vue`

### 3.10 notification（通知）画面仕様

記録に紐づくプッシュ通知設定。完全なCRUD + 履歴を持ちます。

**追加:** `add-notification-view.vue` / `add-notification-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| 通知日時 | 入力 | 日時ピッカー |

**表示:** `notification-view.vue` / `attached-notification.vue`

**削除確認:** `confirm-delete-notification-view.vue`

**履歴:** `notification-histories-view.vue` / `notification-histories-dialog.vue`

### 3.11 idf_kyou（ファイル）画面仕様

**追加:** MKFL（ファイルアップロード）経由 / `upload-file-dialog.vue`

**表示:** `idf-kyou-view.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| テキスト内容 | 表示 | 拡張子 `.txt` のファイルのみ、内容をインラインの `<pre>` 要素で表示。ファイル名はリンク形式で上部に表示 |
| ファイルリンク | 操作 | テキストファイル以外（画像・動画・音声以外）はファイル名をリンクで表示しクリックで開く |
| 画像表示 | 表示 | 画像ファイルはサムネイルまたは原寸で表示 |
| 動画プレーヤー | 表示 | 動画ファイルはインライン再生 |
| 音声プレーヤー | 表示 | 音声ファイルはインライン再生 |
| ローディング表示 | 表示 | テキストファイル読み込み中は進捗バー（`v-progress-linear`）を表示 |

**テキストファイル表示仕様（v1.1.4 で追加）:**
- 対象：拡張子が `.txt` のファイルのみ（`is_text` フラグで判定）
- 表示形式：`<pre>` 要素（`white-space: pre-wrap`, `word-break: break-all`, 等幅フォント）
- 通常表示時はスクロール可能。サムネイルサイズ（`is_image_request_to_thumb_size`）では `overflow: hidden`
- Composable: `use-idf-kyou-view.ts` の `is_text`, `text_content`, `text_loading` で制御

**編集:** `edit-idf-kyou-view.vue` / `edit-idf-kyou-dialog.vue`

| 項目 | 種別 | 説明 |
|---|---|---|
| ファイル名 | 入力 | テキスト入力 |
| 日時 | 入力 | 日付・時刻ピッカー |

**削除確認:** `confirm-delete-idf-kyou-dialog.vue`（ダイアログのみ。`-view.vue` は存在しない）

**コンテキストメニュー:** `idf-kyou-context-menu.vue`

- `is_zip=true` の場合のみ「ZIP内容を閲覧」メニュー項目が有効化される

## 4. 検索・フィルタコンポーネント

gkillの検索機能は複数のクエリコンポーネントを組み合わせて実現します。

| コンポーネント | 説明 |
|---|---|
| `keyword-query.vue` | キーワード全文検索 |
| `calendar-query.vue` | 日付範囲指定 |
| `period-of-time-query.vue` | 時間範囲指定 |
| `tag-query.vue` | タグフィルタ |
| `time-is-query.vue` | 打刻期間フィルタ |
| `rep-query.vue` | リポジトリフィルタ |
| `map-query.vue` | 地図・GPS位置フィルタ |
| `mi-board-query.vue` | タスクボードフィルタ |
| `mi-sort-type-query.vue` | ソート順選択 |
| `mi-extract-check-state-query.vue` | タスク完了状態フィルタ |
| `find-query-editor-view.vue` | 高度な検索条件ビルダー（汎用Kyou向け） |
| `mi-find-query-editor-view.vue` | MI専用検索条件エディタ（チェック状態フィルタ・ソート順・キーワード・タグ対応） |
| `find-time-is-query-editor-view.vue` | 実行中TimeIs専用検索条件エディタ（キーワード・タグ絞り込みトグル・タグ対応。リポジトリと記録タイプはTimeIs固定で選ばせない。plaing検索カスタム条件用） |
| `rykv-query-editor-side-bar.vue` | RYKV画面用検索サイドバー |
| `mi-query-editor-sidebar.vue` | Mi画面用検索サイドバー |
| `clear-query-button.vue` | フィルタリセットボタン |
| `search-button.vue` | 検索実行ボタン |

## 5. 設定・管理コンポーネント

### アカウント管理

| コンポーネント | 説明 |
|---|---|
| `create-account-view.vue` | アカウント作成 |
| `manage-account-view.vue` | アカウント管理 |
| `show-password-reset-link-view.vue` | パスワードリセットリンク表示 |

### サーバー設定

| コンポーネント | 説明 |
|---|---|
| `application-config-view.vue` | アプリケーション設定 |
| `server-config-view.vue` | サーバー設定 |
| `confirm-generate-tls-files-view.vue` | TLS証明書生成確認 |

### デバイス管理

| コンポーネント | 説明 |
|---|---|
| `edit-device-struct-view.vue` | デバイス構造編集 |
| `edit-device-struct-element-view.vue` | デバイス要素編集 |
| `add-new-device-struct-element-view.vue` | デバイス要素追加 |
| `confirm-delete-device-struct-view.vue` | デバイス削除確認 |

### リポジトリ管理

| コンポーネント | 説明 |
|---|---|
| `add-rep-view.vue` | リポジトリ追加 |
| `allocate-rep-view.vue` | リポジトリ割当 |
| `confirm-delete-rep-view.vue` | リポジトリ削除確認 |
| `edit-rep-struct-view.vue` | リポジトリ構造編集 |
| `edit-rep-type-struct-view.vue` | RepType構造編集 |

### KFTLテンプレート管理

| コンポーネント | 説明 |
|---|---|
| `kftl-template-view.vue` | テンプレート表示 |
| `edit-kftl-template-struct-view.vue` | テンプレート構造編集 |
| `edit-kftl-template-struct-element-view.vue` | テンプレート要素編集 |

## 6. ダイアログシステム

### ダイアログ共通構造

全ダイアログは以下のパターンで構成されています。

```
Teleport to body
├── スクリムオーバーレイ（半透明トグル対応）
├── ドラッグ可能ヘッダー（マウス + タッチ対応）
├── コンテンツ（ビューコンポーネント埋め込み）
├── 閉じるボタン
└── 透明モードチェックボックス
```

**v1.1.0 変更点:**
- ダイアログ表示時の自動フォーカスを削除（ページスクロール位置に影響しないように）
- フォーカストラップ（Tab/Shift+Tab循環）を削除
- フォーカス復帰（ダイアログ閉じ時に元要素へ戻す）を削除
- 透過モード切替時にダイアログの高さが縮小するバグを修正（ResizeObserver更新を300ms抑制）

### ダイアログ一覧（カテゴリ別）

> ダイアログは全部で111件ある。以下は主要なものをカテゴリ別に整理したもので、網羅的な一覧ではない。
> 実体は `src/client/pages/dialogs/*.vue` を参照。

#### データ追加ダイアログ

| ダイアログ | 対象データ型 |
|---|---|
| `add-mi-dialog.vue` | タスク |
| `add-time-is-dialog.vue` | 打刻 |
| `add-lantana-dialog.vue` | 気分 |
| `add-nlog-dialog.vue` | 支出 |
| `add-ur-log-dialog.vue` | ブックマーク |
| `add-kc-dialog.vue` | 数値 |
| `add-tag-dialog.vue` | タグ |
| `add-text-dialog.vue` | テキスト |
| `add-notification-dialog.vue` | 通知 |
| `add-mi-re-kyou-dialog.vue` | リポストタスク（既存Kyouのコンテキストメニュー「タスクにする」から開く） |

#### データ編集ダイアログ

| ダイアログ | 対象データ型 |
|---|---|
| `edit-kmemo-dialog.vue` | テキストメモ |
| `edit-mi-dialog.vue` | タスク |
| `edit-time-is-dialog.vue` | 打刻 |
| `edit-lantana-dialog.vue` | 気分 |
| `edit-nlog-dialog.vue` | 支出 |
| `edit-ur-log-dialog.vue` | ブックマーク |
| `edit-kc-dialog.vue` | 数値 |
| `edit-tag-dialog.vue` | タグ |
| `edit-text-dialog.vue` | テキスト |
| `edit-notification-dialog.vue` | 通知 |
| `edit-idf-kyou-dialog.vue` | ファイル |
| `edit-re-kyou-dialog.vue` | リポスト |
| `edit-mi-re-kyou-dialog.vue` | リポストタスク（板名・開始/終了/期限。タイトル欄なし） |

#### 削除確認ダイアログ

| ダイアログ | 対象 |
|---|---|
| `confirm-delete-tag-dialog.vue` | タグ |
| `confirm-delete-text-dialog.vue` | テキスト |
| `confirm-delete-notification-dialog.vue` | 通知 |
| `confirm-delete-idf-kyou-dialog.vue` | ファイル |
| `confirm-logout-dialog.vue` | ログアウト |

`confirm-re-kyou-dialog.vue` は削除ではなく**リポスト実行の確認**なので、上表には含めない。

#### 履歴ダイアログ

| ダイアログ | 対象 |
|---|---|
| `tag-histories-dialog.vue` | タグ変更履歴 |
| `text-histories-dialog.vue` | テキスト変更履歴 |
| `notification-histories-dialog.vue` | 通知変更履歴 |
| `kyou-histories-dialog.vue` | 記録変更履歴 |

#### 設定ダイアログ

| ダイアログ | 対象 |
|---|---|
| `application-config-dialog.vue` | アプリケーション設定 |
| `server-config-dialog.vue` | サーバー設定 |
| `manage-account-dialog.vue` | アカウント管理 |
| `create-account-dialog.vue` | アカウント作成 |
| `edit-dashboard-dialog.vue` | ダッシュボード設定（MI検索条件・Dnote検索条件の編集） |
| `edit-plaing-time-is-dialog.vue` | 実行中検索条件設定（「検索条件をカスタマイズする」チェックボックス＋条件編集ボタン。チェックOFFで未設定＝デフォルト動作に戻る） |
| `edit-saved-find-query-dialog.vue` | 保存済み検索条件のハブ（ライフログ検索条件・タスク検索条件の2ボタン） |
| `edit-saved-find-query-list-dialog.vue` | 保存済み検索条件の一覧管理（名前・条件編集・並べ替え・削除。追加は右下FAB。`query_type` prop で rykv/mi の2インスタンス） |
| `edit-dnote-dialog.vue` | Dnote（集計ビュー）設定 |
| `edit-ryuu-dialog.vue` | Ryuu（関連情報ビュー）設定 |
| `new-board-name-dialog.vue` | Mi ボード名の新規作成 |
| `new-device-name-dialog.vue` | デバイス名の新規作成 |
| `add-rep-dialog.vue` / `allocate-rep-dialog.vue` | リポジトリ追加・割当 |
| `confirm-generate-tls-files-dialog.vue` | TLS証明書生成確認 |
| `confirm-reset-password-dialog.vue` / `show-password-reset-link-dialog.vue` | パスワードリセット |
| `plugin-config-dialog.vue` | プラグイン設定（プラグイン Kyou のコンテキストメニュー「プラグイン設定」から開く） |

構造編集系（タグ構造・リポジトリ構造・RepType構造・KFTLテンプレート構造・デバイス）は
`add-new-*-struct-element-dialog.vue` / `edit-*-struct[-element]-dialog.vue` /
`confirm-delete-*-struct-dialog.vue` の3点セットが対象ごとに用意されている。

#### プラグイン関連コンポーネント

| コンポーネント | 種別 | 説明 |
|---|---|---|
| `plugin-html-view.vue` | ビュー | プラグインが返す HTML を iframe で描画。`sandbox="allow-scripts allow-forms"`、`scrolling="no"`、高さは postMessage（`gkill_iframe_size`）で親が調整し、未確定時は 80px |
| `plugin-html-context-menu.vue` | ビュー | プラグインKyou用コンテキストメニュー |
| `plugin-config-dialog.vue` | ダイアログ | プラグイン設定フォーム（`plugin-html-view.vue` が保持） |

テーマは親から `gkill_theme` を postMessage して CSS 変数を切り替える。

本文の渡し方は表示先で変わる。一覧（height が数値）は `srcdoc` 直書き、
Ryuu・rykv 詳細ペイン・KyouDialog（height が文字列）は定数ローダーへの postMessage 注入で、
後者はローダーが `gkill_plugin_loader_ready` を名乗ってから送る。
iframe 内のダブルクリックは `gkill_iframe_dblclick` で親へ返し、
親が本物の `dblclick` を撃ち直して KyouDialog を開く。
詳細は [plugin-system.md](plugin-system.md) の7章。

#### Ryuu（関連情報）コンポーネント

`rykv-view.vue` に埋め込まれる関連情報ビュー。

| コンポーネント | 種別 |
|---|---|
| `ryuu-view.vue` / `ryuu-item-view.vue` | ビュー |
| `add-ryuu-item-view.vue` / `edit-ryuu-item-view.vue` | ビュー |
| `ryuu-item-context-menu.vue` | コンテキストメニュー |
| `add-ryuu-item-dialog.vue` / `edit-ryuu-item-dialog.vue` / `edit-ryuu-dialog.vue` / `confirm-delete-ryuu-item-dialog.vue` | ダイアログ |

#### Rykv ダイアログホスト

Rykv 配下のダイアログは個別に配置されるのではなく、`rykv-dialog-host.vue` /
`rykv-dialog-host-item.vue` が一括でホストする。開けるダイアログの種類は
`rykv-dialog-kind.ts` の `RykvDialogKind`（28種）で定義される。

#### ヘルプ・チュートリアル

| ダイアログ | 対象 |
|---|---|
| `help-dialog.vue` | 各ページのツールバーから開くヘルプ |
| `tutorial-dialog.vue` | チュートリアル |

#### 検索条件エディタダイアログ

| ダイアログ | 対象 |
|---|---|
| `find-query-editor-dialog.vue` | 汎用Kyou検索条件エディタ |
| `mi-find-query-editor-dialog.vue` | MI専用検索条件エディタ（チェック状態フィルタ・ソート順・キーワード等） |
| `find-time-is-query-editor-dialog.vue` | 実行中TimeIs専用検索条件エディタ（plaing検索カスタム条件用） |

#### 共有ダイアログ

| ダイアログ | 対象 |
|---|---|
| `share-kyou-list-dialog.vue` | 共有設定一覧 |
| `share-kyou-list-link-dialog.vue` | 共有リンク表示 |
| `confirm-delete-share-kyou-list-dialog.vue` | 共有削除確認 |
| `manage-share-task-list-dialog.vue` | 共有タスク管理 |

#### ファイル操作ダイアログ

| ダイアログ | 対象 |
|---|---|
| `upload-file-dialog.vue` | ファイルアップロード |
| `save-clipboard-to-file-dialog.vue` | クリップボード内容をファイルとして保存 |

#### ZIP閲覧ダイアログ

| ダイアログ | 対象 |
|---|---|
| `browse-zip-contents-dialog.vue` | ZIPファイル内容閲覧 |

### ZIP閲覧ダイアログ仕様

**コンポーネント:** `browse-zip-contents-dialog.vue`

IDFKyouのZIPファイルの内容を階層的に閲覧するフローティングダイアログです。

| 項目 | 種別 | 説明 |
|---|---|---|
| パンくずナビゲーション | 操作 | 現在のディレクトリ階層を表示し、各階層をクリックで直接移動可能 |
| サブディレクトリ一覧 | 操作 | 現在階層の直下のフォルダを表示。クリックで階層を掘り下げる |
| 親ディレクトリへ戻る (`..`) | 操作 | ルート以外で表示。クリックで上の階層に移動 |
| ファイル一覧 | 表示 | 現在階層の直下のファイルのみ表示（ファイル名・サイズ） |
| 画像プレビュー | 表示 | 現在階層のZIP内画像ファイルをサムネイル表示（クリックで拡大、ページ送り対応） |
| テキストビューワー | 表示 | テキスト系ファイルをクリックでオーバーレイ表示（512KBまで、ページ送り対応） |
| メディアプレイヤー | 表示 | 動画・音声ファイルをクリックでオーバーレイ再生（ページ送り対応。行とオーバーレイに「新しいタブで開く」「ダウンロード」の副次導線あり。ブラウザで再生できない形式はメッセージとダウンロードリンクを表示） |
| PDFリンク | 操作 | PDFファイルをクリックで新しいタブに表示（`/zip_cache/` は `.pdf` のみCSP sandboxを付けない） |
| ファイルリンク | 操作 | 上記以外のファイルは `/zip_cache/` 経由でダウンロード |
| 空フォルダ表示 | 表示 | ファイルもサブディレクトリもない場合にメッセージを表示 |

**表示条件:** IDFKyouのコンテキストメニューで「ZIP内容を閲覧」を選択した場合に表示される。`is_zip=true` のIDFKyouに対してのみコンテキストメニュー項目が有効化される。

## 7. イベント伝播パターン

コンポーネント間のイベント伝播は以下のパターンで統一されています。

```
CRUDリレーイベント（ビュー層 18件）:
├── received_messages / received_errors
├── registered_kyou / updated_kyou / deleted_kyou
├── registered_tag / updated_tag / deleted_tag
├── registered_text / updated_text / deleted_text
├── registered_notification / updated_notification / deleted_notification
├── requested_update_check_kyous
├── requested_reload_kyou / requested_reload_list
└── requested_open_rykv_dialog

ダイアログ層だけが追加で中継する 2件:
└── focused_kyou / clicked_kyou
```

> `KyouViewEmits`（`pages/views/kyou-view-emits.ts`）は全21イベント。`requested_close_dialog` は中継しない（ダイアログが `@requested_close_dialog="hide()"` で自分に繋ぐ）。`focused_kyou`/`clicked_kyou` はビュー層が発火源なので中継せず、ダイアログ層だけが中継する（入れ子の KyouView で二重発火するため）。

各ViewコンポーネントはTypeScriptで型定義されたProps/Emitsを持ちます。

| ファイル | 用途 |
|---|---|
| `*-props.ts` | 入力プロパティのインターフェース定義 |
| `*-emits.ts` | イベント発信のシグネチャ定義 |

### 中継束の共通化（`classes/kyou-view-relay.ts`）

中継ハンドラは各コンポーザブルで手書きせず、`build_kyou_view_relay(emits)` / `build_kyou_dialog_relay(emits)` が返すハンドラ束（慣例名 `crudRelayHandlers`）を使う。テンプレート側は `v-on="crudRelayHandlers"` の1行で受け渡す。挙動を変えたいイベントだけ第2引数 `overrides` で差し替える。

中継対象は `kyou_view_relay_event_names`（18件）と `kyou_focus_relay_event_names`（2件）が唯一の情報源で、型（`KyouViewRelayArgs`）と配列の両方に足さないとコンパイルエラーになる（`Exclude` による網羅チェック）。イベントを増やすときは両方を更新すること。

## 8. コンテキストメニュー

右クリック（`contextmenu.prevent`）で表示されるコンテキストメニューが各データ型に用意されています。

#### Kyou 系コンテキストメニューの共通項目

データ型ごとの編集・削除に加え、Kyou 系のメニューはおおむね以下を共通で持ちます。

| 項目 | i18n キー | 説明 |
|---|---|---|
| タスク化 | `ADD_MI_REKYOU_TITLE` | この Kyou を対象にした MiReKyou を追加する |
| タグ追加 | — | 付帯タグを追加。直近使用タグのクイック追加サブメニューを持つ |
| テキスト追加 | — | 付帯テキストを追加 |
| 通知追加 | `ADD_NOTIFICATION_TITLE` | 付帯通知を追加 |
| 内容コピー | `COPY_CONTENT_TITLE` | Kyou の内容をクリップボードへコピー（`classes/kyou-content-text.ts`） |
| IDコピー | `COPY_ID_TITLE` | Kyou の ID をクリップボードへコピー |
| フォルダを開く / ファイルを開く | — | `application_config.session_is_local` が真のときのみ表示 |

| コンテキストメニュー | 対象 |
|---|---|
| `kmemo-context-menu.vue` | テキストメモ |
| `mi-context-menu.vue` | タスク |
| `time-is-context-menu.vue` | 打刻 |
| `lantana-context-menu.vue` | 気分 |
| `nlog-context-menu.vue` | 支出 |
| `ur-log-context-menu.vue` | ブックマーク |
| `kc-context-menu.vue` | 数値 |
| `idf-kyou-context-menu.vue` | ファイル（`is_zip=true` 時に「ZIP内容を閲覧」メニュー追加） |
| `re-kyou-context-menu.vue` | リポスト |
| `mi-re-kyou-context-menu.vue` | リポストタスク |
| `git-commit-log-context-menu.vue` | Gitコミット |
| `plugin-html-context-menu.vue` | プラグインKyou（編集・削除なし） |
| `dnote-trend-graph-context-menu.vue` | Dnoteトレンドグラフ |
| `attached-tag-context-menu.vue` | 付帯タグ |
| `attached-text-context-menu.vue` | 付帯テキスト |
| `attached-notification-context-menu.vue` | 付帯通知 |
| `attached-time-is-plaing-context-menu.vue` | アクティブ打刻 |
| `device-struct-context-menu.vue` | デバイス構造（設定画面） |
| `dnote-item-list-context-menu.vue` | Dnote項目リスト |
| `dnote-list-query-context-menu.vue` | Dnoteクエリ |
| `kftl-template-struct-context-menu.vue` | KFTLテンプレート構造（設定画面） |
| `rep-struct-context-menu.vue` | リポジトリ構造（設定画面） |
| `rep-type-struct-context-menu.vue` | RepType構造（設定画面） |
| `ryuu-item-context-menu.vue` | Ryuu項目 |
| `tag-struct-context-menu.vue` | タグ構造（設定画面） |

## 関連資料

- [screen-transition.md](screen-transition.md) — 画面遷移図
- [frontend-architecture.md](frontend-architecture.md) — フロントエンド設計ガイド
- [glossary.md](glossary.md) — 用語集
- [usecase.md](usecase.md) — ユースケース一覧
