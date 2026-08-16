# gkill 用語集

gkill で使われる独自用語・略称・概念の定義集。コードベース・ロケールファイル・開発者からの情報に基づく。

## 1. アプリケーション名・基本概念

| 用語 | 説明 | 由来 |
|------|------|------|
| **gkill** | ライフログアプリケーション。人生の出来事（メモ、ブックマーク、タスク、気分、支出、打刻、ファイル、Gitコミット等）を記録・閲覧する | "gkill is not a lifelogging application" / gyou to kyou infinity lifelogger |
| **Kyou** | 全データ型の集約ビュー表現。`DataType` フィールドで具体的な型を識別する。API レスポンスで統一的に返却される基底エンティティ | 今日、共、況、狂 |
| **Append-Only** | gkill のデータ更新方式。レコードの UPDATE/DELETE を行わず、同一 ID で新しいレコードを INSERT する。`UPDATE_TIME` が最新のレコードが有効データとなり、過去のバージョンは履歴として保持される | — |

## 2. データ型（Kyou サブタイプ）

### 主要データ型

| 用語 | UI表示名 | 説明 | 由来 |
|------|---------|------|------|
| **Kmemo** | メモ帳 | テキストメモ。内容（`Content`）を自由記述で記録する | — |
| **KC** | 数値記録 | タイトル付き実数値（`NumValue`）を記録する汎用数値型。体重・回数・温度など任意の数値を追跡可能 | Super Kintore Counter（筋トレカウンター）の汎化 |
| **Lantana** | 気分 | 気分値（`Mood`）を 0〜10 のスケールで記録する | 日本語名「七変化」。花の色の変化が気分の変化を連想させることから。旧気分記録アプリ Lantana に由来 |
| **Mi** | タスク | タスク管理。ボード名（`BoardName`）でグルーピングし、期限（`LimitTime`）・開始/終了予定・チェック状態（`IsChecked`）を持つ | 未、みー |
| **Nlog** | 支出 | 支出記録。店名（`Shop`）・タイトル（`Title`）・金額（`Amount`）を記録する | yeNlog（日本円を記録する型） |
| **URLog** | ブックマーク | URL 記録。URL・タイトル・説明・ファビコン画像・サムネイル画像を保持する | URL + Log |
| **TimeIs** | 打刻帳 | 時間記録（打刻）。タイトル・開始時刻（`StartTime`）・終了時刻（`EndTime`）で状況を記録する。`EndTime` が null の場合は「実行中」状態 | Time is ??? |
| **IDFKyou** | ファイル | ファイル参照。対象リポジトリ名・ファイルパスを持ち、画像/動画/音声のフラグを保持する。`IsZip` フラグでZIPファイルかどうかを識別し、ZIPの場合はコンテキストメニューから内容閲覧が可能 | gkill ID allocate to File |
| **ReKyou** | リポスト | 既存 Kyou の再投稿。`TargetID` で参照先の Kyou を指定する | Re（再）+ Kyou |
| **MiReKyou** | リポストタスク | 既存 Kyou をタスク化したもの。`TargetID` で参照先の Kyou を指定し、Mi と同じボード名・期限・開始/終了予定・チェック状態を持つ。タイトルは持たず、表示は参照先 Kyou をそのまま描画する | Mi + ReKyou |

### 外部データ取り込み型

| 用語 | 説明 |
|------|------|
| **GitCommitLog** | ローカル Git リポジトリからコミットログを読み取ってキャッシュする型。コミットメッセージ・追加行数（`Addition`）・削除行数（`Deletion`）を含む |
| **GPSLog** | GPX ファイルから GPS 位置情報を読み取る型 |
| **PluginKyou** | 外部プラグインバイナリが提供する Kyou。`DataType` はプラグイン定義による（例: `chatgpt_conversation`, `claude_conversation`, `claude_code_turn`, `codex_turn`）。コンテンツ表示は `GetContentHTML` が返す HTML を iframe (srcdoc) で描画する |
| **PluginRepository** | プラグインバイナリをサブプロセスとして起動し stdio 改行区切り JSON で通信するリポジトリ実装（`src/server/gkill/dao/reps/plugin_repository_impl.go`）。`RepType` を持たず4層パターンにも属さない |
| **PluginManager** | ユーザごとにプラグインディレクトリを走査し、`manifest.json` を持つものを `PluginRepository` として登録する（`src/server/gkill/dao/plugin_manager.go`） |
| **PluginManifest** | プラグインのメタデータ（`protocol_version`, `name`, `version`, `description`, `data_type`, `rep_name`, `executable`, `min_gkill_version` の8フィールド）。`name` はディレクトリ名と一致させる。同梱プラグイン6本（chatgpt / claudeai / claudecode / codex / fitbit / google_locationhistory）はバイナリに `//go:embed` しており `--gkill-print-manifest` で出力できる（`--gkill-print-config` で既定の `config.json` も出せる）。`gkill_example` は埋め込みもフラグも持たない |
| **プラグインディレクトリ** | `$GKILL_HOME/plugins/{userID}/{pluginName}/` — manifest.json・実行ファイル・`config.json` を格納するディレクトリ |
| **plugin_cache** | プラグインの SQLite3 キャッシュ置き場。`$GKILL_HOME/caches/plugin_cache/{userID}/{pluginName}/cache.db`。プラグインディレクトリではなく gkill のキャッシュディレクトリ配下にあるため `clear_cache plugin` で削除できる |
| **source_dirs** | プラグインの `config.json` で取り込み元フォルダを指定するキー。グロブ・`~`・環境変数を展開し、検索のたびに読み直される |

### メタデータ型

| 用語 | UI表示名 | 説明 |
|------|---------|------|
| **Tag** | タグ | 任意の Kyou に紐づくタグ。`TargetID` で対象 Kyou を参照する |
| **Text** | テキスト | 任意の Kyou に紐づくテキスト注釈。`TargetID` で対象 Kyou を参照する |
| **Notification** | 通知 | 任意の Kyou に紐づく通知情報。通知日時（`NotificationTime`）・内容（`Content`）・通知済みフラグ（`IsNotificated`）を持つ |

## 3. 画面・ページ名

| 用語 | UI表示名 | ルート | 説明 | 由来 |
|------|---------|-------|------|------|
| **Login** | ログイン | `/` | ログイン画面。ルート直下に割り当てられている | — |
| **KFTL** | メモ帳 | `/kftl` | テキストベース記録入力画面。KFTL 構文（後述）を使って複数データ型を一括入力できる | Key Fairy Textbase Lifelogger。記録導線を「あける」鍵開けの妖精 |
| **Rykv** | ライフログビュー | `/rykv` | メインの閲覧・検索画面。タイムライン形式でKyouを表示し、Dnote（集計）や Ryuu（関連情報）も統合 | RYuuKyouViewer。流れるように見える（タイムライン無限スクロール）、留める、龍のように（歴史と知を連想） |
| **Mi** | タスク | `/mi` | タスク管理画面。ボード形式で Mi を表示し、チェック状態のフィルタ・ソート・共有機能を提供 | — |
| **Kyou** | 記録 | `/kyou` | 記録一覧表示画面 | — |
| **Mkfl** | 打刻メモ帳 | `/mkfl` | 打刻（TimeIs）とメモ（Kmemo）を組み合わせた複合入力ビュー | Master Key Fairy Lifelogger。記録導線を「あける」鍵開けの妖精 |
| **Plaing** | 実行中 | `/plaing` | 稼働中の TimeIs（`EndTime` が null）を一覧表示する画面。状況キーワードやタグで絞り込み可能。**`plaing` は "playing" の誤綴りではなく製品綴りとして凍結**（下記参照） | Playing と Plain のミックス造語 |
| **Dashboard** | ダッシュボード | `/dashboard` | 特定日の記録を俯瞰する日次サマリー画面。DnoteView・GPSLogMap・KyouListView（MI一覧）を1画面に集約し、日付ナビゲーションで過去日のデータも確認できる | — |
| **Saihate** | さいはて | `/saihate` | 記録だけに特化した画面。他画面への遷移ができない。「さいはてでも記録ができる」がコンセプト | 「最果て」。主にスマホのロック画面からアクセスする用途 |

### Rykv 内の機能ビュー

| 用語 | UI表示名 | 説明 | 由来 |
|------|---------|------|------|
| **Dnote** | 集計ビュー | データ集計・分析機能。Predicate（条件）でフィルタし、KeyGetter（集計キー）でグルーピングし、AggregateTarget（集計対象）で集計する | Daily Record summary Notebook |
| **Ryuu** | 関連情報 | 表示中の Kyou と時刻が近い関連 Kyou を表示する機能。検索範囲（n時間）と抽出条件を指定可能 | 流、龍 |

### 認証不要ページ

| ルート | 説明 |
|-------|------|
| `/shared_page` | 共有リンクから Kyou / タスクを閲覧する画面（認証不要）。`view_type` に応じて内部で振り分ける |
| `/shared_mi` | 旧・共有タスク閲覧URL。実体は `/shared_page?share_id=…` へのリダイレクタ |
| `/set_new_password` | パスワードリセットリンクから新パスワードを設定する画面 |
| `/register_first_account` | 初回起動時のアカウント登録画面（旧パス `/regist_first_account` は vue-router がリダイレクト） |

> `/shared_rykv` はサーバ側（`serve.go`）では SPA が配信されるが、
> `src/client/router/index.ts` に対応するルート定義が無い。

## 4. KFTL 構文

KFTL（Key Fairy Textbase Lifelogger）は、テキストで複数のデータ型を一括入力するための独自フォーマット。行頭のプレフィックスでデータ型を判別する。

### プレフィックス一覧

プレフィックスは**日本語と ASCII の2系統**があり、サーバ側パーサ（`kftl_factory.go:8-45`）と
クライアント側パーサ（`kftl-prefixes.ts:6-24`）のどちらも両方を受理する。

| 日本語 | ASCII | データ型 | 説明 |
|---|---|---------|------|
| （なし） | （なし） | Kmemo | デフォルト。プレフィックスなしの行はテキストメモとして扱われる |
| `。` | `#` | Tag | タグを追加する。`。タグ名` の形式 |
| `ーー` | `--` | Text | テキスト注釈の開始/終了。`ーー` で囲まれた範囲がテキスト |
| `？` | `?` | RelatedTime | 関連時刻を指定する。`？時刻` の形式 |
| `、` | `,` | Split | 区切り。現在のステートメントを終了し、次のステートメントを開始する |
| `、、` | `,,` | SplitNextSecond | 区切り＋時刻の `AddSecond` を +1 する |
| `ーか` | `/num` | KC | 数値記録の開始。後続行でタイトル → 数値を入力 |
| `ーみ` | `/mi` | Mi | タスクの開始。後続行でタイトル → [ボード名] → [期限] → [開始予定] → [終了予定] を入力 |
| `ーら` | `/mood` | Lantana | 気分値の開始。後続行で気分値（0〜10）を入力 |
| `ーん` | `/expense` | Nlog | 支出の開始。後続行で店名 → タイトル → 金額を入力 |
| `ーう` | `/url` | URLog | ブックマークの開始。後続行でタイトル → URL を入力 |
| `ーた` | `/start` | TimeIs Start | TimeIs 開始のみ（`StartTime` を設定、`EndTime` = null） |
| `ーえ` | `/end` | TimeIs End | タイトル指定で実行中の TimeIs を終了する |
| `ーいえ` | `/end?` | TimeIs End If Exist | 存在する場合のみ TimeIs を終了する |
| `ーたえ` | `/endt` | TimeIs End By Tag | タグ名指定で実行中の TimeIs を終了する |
| `ーいたえ` | `/endt?` | TimeIs End By Tag If Exist | タグ名指定で、存在する場合のみ TimeIs を終了する |
| `ーち` | `/timeis` | TimeIs | 開始と終了を同時に設定する（`StartTime` + `EndTime` 両方を記録） |
| `！` | `!` | Save | 保存実行。パースを終了してリクエストを実行する |

> クライアント側では日本語プレフィックスは**固定リテラルではなく i18n キー経由**で解決される
> （`kftl-prefixes.ts:26-29` の `matches_exact(line, i18n_key, ascii_prefix)`）。
> つまりロケールによって変わりうる。ASCII 側は固定。

### KFTL パーサの主要コンポーネント

| コンポーネント | 説明 |
|---------------|------|
| **kftlFactory** | 行頭プレフィックスから `KFTLStatementLine` の種別を決定するファクトリ。`prevLineIsMetaInfo` フラグで行の文脈を管理する |
| **KFTLStatement** | KFTL テキスト全体をパースし、リクエストを生成・実行する |
| **KFTLStatementLine** | 各行の処理を担当するインタフェース。データ型ごとに実装がある |
| **KFTLRequest / KFTLRequestMap** | パース結果のリクエスト。ID ベースでグルーピングされ、`DoRequest()` でリポジトリに保存する |

## 5. アーキテクチャ用語

| 用語 | 説明 |
|------|------|
| **Repository 4層パターン** | 各データ型のデータアクセスを4層で実装するパターン: (1) `*_repository.go`（インタフェース定義） → (2) `*_repository_sqlite3_impl.go`（SQLite3 直接アクセス） → (3) `*_repository_cached_sqlite3_impl.go`（キャッシュ付きラッパー） → (4) `*_repository_temp_sqlite3_impl.go`（トランザクション用一時リポジトリ） |
| **GkillRepositories** | ユーザ別の全リポジトリ集約構造体。読み取り用（`XxxReps` = 複数リポジトリの集約）と書き込み用（`WriteXxxRep` = 単一リポジトリ）を保持する |
| **GkillDAOManager** | 全 DAO の中央管理。`GetRepositories()` でユーザ別リポジトリを取得し、`GetTempReps()` でトランザクション用一時リポジトリを管理する |
| **GkillServerAPI** | HTTP API ハンドラ。gorilla/mux で全エンドポイント（92定義・90登録）を提供する。`gkill_server_api/` パッケージ（handle_*.go 99ファイル）に分割実装 |
| **TempReps** | KFTL パース時のトランザクション用一時リポジトリ。`CommitTX` で本リポジトリに反映、`DiscardTX` で破棄する |
| **Rep / 記録保管場所** | データ保存先の SQLite3 ファイル。ユーザ・デバイス・データ型ごとに割り当てられる |
| **RepType / 記録タイプ** | リポジトリの分類。メモ帳、打刻帳、支出、数値記録、タスク、気分、ブックマーク、リポスト等 |
| **LatestDataRepositoryAddress** | 各 ID の最新レコードがどのリポジトリに格納されているかを示すキャッシュ。検索時の `OnlyLatestData` フィルタで使用 |
| **DVNF** | DeVice Name Folder Naming Framework。ファイルのバージョン管理とコピー・移動を行う CLI ユーティリティ。`gkill dvnf copy/get/move` コマンドで呼び出す |

## 6. 設定用語

| 用語 | UI表示名 | 説明 |
|------|---------|------|
| **Device / プロファイル** | プロファイル | デバイスや利用環境の設定単位。PC・スマートフォン・Wear OS 等のプロファイルを使い分けられる |
| **ApplicationConfig** | 設定 | ユーザ別のアプリケーション設定。UI テーマ・デフォルト表示日数・テンプレート・構造定義等を含む。`dashboard_json_data` フィールドでダッシュボード設定、`plaing_timeis_json_data` フィールドで実行中検索条件、`saved_find_query_json_data` フィールドで保存済み検索条件も保持する |
| **DashboardConfig** | ダッシュボード設定 | ダッシュボード画面の表示設定クラス（`src/client/classes/datas/config/dashboard-config.ts`）。`dashboard_mi_find_kyou_query`（MI一覧の検索条件）と `dashboard_dnote_find_kyou_query`（Dnoteビューの検索条件）の2つの `FindKyouQuery` を保持する。`ApplicationConfig.dashboard_json_data` にJSON文字列として格納される |
| **dashboard_json_data** | — | `APPLICATION_CONFIG` テーブルの KEY として使用されるキー名。`DEVICE='ALL'`（デバイス非依存）で保存され、DashboardConfig の JSON 文字列を VALUE に格納する。`ignoreDeviceNameConfigKey` リストに含まれるため、デバイス固有設定の上書きを受けない |
| **PlaingTimeIsConfig** | 実行中検索条件 | plaing検索（Kyou付随の実行中表示・実行中画面・KFTLの/end系終了候補検索）のカスタム検索条件クラス（`src/client/classes/datas/config/plaing-time-is-config.ts`）。`plaing_timeis_find_kyou_query`（`FindKyouQuery \| null`）を1本保持し、null は「未設定＝従来どおり全リポジトリ対象」を表す。適用の実体は `generate_plaing_timeis_query()`（`src/client/classes/api/find_query/generate-plaing-timeis-query.ts`）。Wear OS の `buildPlaingFindQuery`（`GkillApiClient.kt`）とサーバ内 KFTL（`kftl_timeis.go`）の plaing 検索は別系統のため、この設定は Web クライアントにのみ効く |
| **plaing_timeis_json_data** | — | `APPLICATION_CONFIG` テーブルの KEY（`PLAING_TIMEIS_JSON_DATA`）として使用されるキー名。`DEVICE='ALL'`（デバイス非依存）で保存され、PlaingTimeIsConfig の JSON 文字列を VALUE に格納する。`ignoreDeviceNameConfigKey` リストに含まれる |
| **SavedFindQueryConfig** | 検索条件 | 保存済み検索条件クラス（`src/client/classes/datas/config/saved-find-query-config.ts`）。`saved_rykv_find_kyou_querys`（ライフログ用）と `saved_mi_find_kyou_querys`（タスク用）の2つの名前付き検索条件リストを保持する。各アイテムは `{id, title, find_kyou_query}`（Ryuu の関連情報クエリと同形式）。設定画面の「検索条件」ボタン → ハブダイアログ（`edit-saved-find-query-dialog.vue`）→ 種別別の一覧管理ダイアログ（`edit-saved-find-query-list-dialog.vue`、1コンポーネント2インスタンス）で登録・更新・削除・並べ替えする。rykv/mi サイドバーの呼び出しFAB（0件時非表示）から選択すると `apply_saved_query()` がサイドバーへ適用する（query_id は列側を維持。ホットリロードONなら自動検索、OFFなら検索ボタンで実行） |
| **saved_find_query_json_data** | — | `APPLICATION_CONFIG` テーブルの KEY（`SAVED_FIND_QUERY_JSON_DATA`）として使用されるキー名。`DEVICE='ALL'`（デバイス非依存）で保存され、SavedFindQueryConfig の JSON 文字列を VALUE に格納する。`ignoreDeviceNameConfigKey` リストに含まれる |
| **ServerConfig** | サーバ設定 | サーバ全体の設定。TLS・ポート番号・データディレクトリ・コマンドパス等 |
| **TagStruct** | タグ構造 | タグの階層構造定義。フォルダでグルーピングし、初期化時チェック・非表示優先等のオプションを持つ |
| **RepStruct** | 記録保管場所構造 | リポジトリの構造定義 |
| **RepTypeStruct** | 記録タイプ構造 | リポジトリ型の分類定義 |
| **KFTLTemplate** | テンプレート | KFTL 入力のテンプレート定義。テンプレート名と内容を持ち、Wear OS タイルからの入力にも使用される |
| **DeviceStruct** | プロファイル構造 | デバイスプロファイルの定義 |
| **MiBoardStruct** | — | タスクボードの構造定義。設定画面の「板構造」から並び順の変更と削除ができる（板名は実データ由来なので編集不可。フォルダ分けや表示名の変更もしない）。ApplicationConfig の `MI_BOARD_STRUCT` として全端末で共有される |

## 7. Dnote 集計システム用語

Dnote はデータ集計・分析機能。Predicate → KeyGetter → AggregateTarget の3段階で処理する。集計要素として集計項目・集計リスト・トレンドグラフ・相関グラフの4種類を定義タブ内に配置できる。

### 処理コンポーネント

| 用語 | UI表示名 | 説明 |
|------|---------|------|
| **Predicate / 条件** | 条件 | フィルタリング条件。AND/OR/NOT の論理演算で組み合わせ可能。30+ 種類の具象クラスがある |
| **KeyGetter / 集計キー** | 集計キー | グルーピング基準。関連日・関連年月・関連曜日・タグ・データタイプ・タイトル・店名・気分値等から選択 |
| **AggregateTarget / 集計対象** | 集計対象 | 集計関数。件数・合計・平均・最大値・最小値を、支出額・気分値・作業時間・コード行数等に適用可能 |
| **DnoteTrendGraph / トレンドグラフ** | トレンドグラフ | 時系列集計グラフ。取得済み Kyou を `DnoteTrendAggregator`（`src/client/classes/dnote/dnote-trend-aggregator.ts`）が集計粒度（日/週/月）で時系列集計し、スパークライン（折れ線/棒）で表示する。集計項目・集計リストと並ぶ第3の集計要素。サーバーAPIを持たずクライアント側のみで完結する |
| **DnoteCorrelationGraph / 相関グラフ** | 相関グラフ | 2～10個の独立した指標を同じ粒度で集計し、Pearson／Spearmanの方向付き相関行列と散布図を表示する第4の集計要素。サーバーAPIを持たずクライアント側のみで完結する |

### Predicate の主なカテゴリ

| カテゴリ | 例 |
|---------|---|
| 論理演算 | AND, OR, NOT |
| データ型 | データタイプ一致 |
| テキスト検索 | メモ内容含む、タスクタイトル含む、支出タイトル含む、店名含む、打刻タイトル含む |
| 数値比較 | 気分値以上/以下、金額以上/以下、コード行数以上/以下 |
| タグ・時刻 | タグ名一致、指定曜日に関連する |

## 8. 共通フィールド用語

全 Kyou サブタイプが持つ共通フィールド。

| 用語 | 型 | 説明 |
|------|---|------|
| **ID** | string (UUID) | レコード識別子。Append-Only 方式のため**主キーではない**（同一 ID が複数行存在する） |
| **RepName** | string | 所属リポジトリ名 |
| **RelatedTime** | time.Time | 関連日時。時系列表示のソート基準として使用される。一部のデータ型（Mi, TimeIs, Notification）ではDBカラムとして存在せず、SQLクエリ内で他カラムから動的に導出される |
| **CreateTime** | time.Time | レコード作成日時 |
| **UpdateTime** | time.Time | レコード更新日時。Append-Only のバージョン識別子として機能する |
| **CreateApp** | string | 作成元アプリケーション名（例: "gkill", "wear_os_app"） |
| **CreateDevice** | string | 作成元デバイス（プロファイル）名 |
| **CreateUser** | string | 作成ユーザ名 |
| **UpdateApp / UpdateDevice / UpdateUser** | string | 更新元のアプリ・デバイス・ユーザ |
| **IsDeleted** | bool | 論理削除フラグ。`true` のレコードが INSERT されることで論理削除を実現する |
| **TargetID** | string | Tag / Text / Notification / ReKyou / MiReKyou が参照する対象 Kyou の ID |

## 9. その他の用語

| 用語 | 説明 |
|------|------|
| **板（Board）** | Mi（タスク）のグルーピング単位。UI 上では `板名` と表記される |
| **ブックマークレット** | ブラウザから URLog を直接追加するための JavaScript。ログイン時に専用セッションが自動作成される |
| **共有ページ** | 認証不要で Kyou やタスクを閲覧できる公開リンク。`ShareKyouListInfo` で検索条件を含む共有リンクを生成する |
| **オレオレTLS** | 自己署名 TLS 証明書の生成機能。サーバ設定画面から生成できる |
| **ホットリロード** | データの自動再読み込み機能 |
| **FindQuery** | 検索クエリ。キーワード・日時範囲・タグ・データ型・デバイス等の複合条件で Kyou を検索する |
| **サムネイルキャッシュ / 互換動画キャッシュ** | IDFリポジトリ配下の画像・動画から作る派生ファイル。`$HOME/gkill/caches/{thumb,video}_cache/{user_id}/{rep_name}/` に `{sha1(相対パス)}_{ファイルサイズ}_{W}x{H}.jpg`（動画は `_compat_720p.mp4`）の名前で保存される。専用の配信ルートは無く、`/files/{rep_name}/...?thumb=` 経由でのみ生成・配信される。rep名は利用者間で一意でないため、利用者IDを1階層挟んで混ざらないようにしている |
| **ZIPキャッシュ** | IDFKyouのZIPファイルを展開したキャッシュ。`$HOME/gkill/caches/zip_cache/{user_id}/{rep_name}/{sha1}/` に保存される。利用者ID・リポジトリ名・**ZIPファイルの絶対パス文字列**のSHA1をキーとし、同一ファイルの再展開を回避する。配信時は利用者のディレクトリを起点に固定するので、他人のキャッシュは読めない |
| **ZipEntry** | ZIP内のファイルエントリ情報。ファイル名・サイズ・パス等を含む。`/api/browse_zip_contents` のレスポンスとして返却される |
| **OnlyLatestData** | 検索フィルタ。同一 ID のレコードのうち `UpdateTime` が最新のもののみを返す。Kyou 本体だけでなく**付随するタグ・テキストにも同じ規則が適用される**ため、タグ名を書き換えたあとに旧タグ名で検索してもヒットしない（`find_filter.go` 各所と `find_kyou_context.go` の `isLatestData`） |
| **連鎖削除 (cascade delete)** | Kyou を削除したとき、付随する Tag / Text / Notification と、その Kyou を `TargetID` で参照している ReKyou / MiReKyou を幅優先で辿って一緒に論理削除する処理（`src/client/classes/cascade-delete-kyou.ts`）。探索（read）と削除（write）を完全に分け、Kyou 自身は**最後**に消す（先に消すとサーバの FindKyous が参照元を結果から外し、途中失敗時に残骸を再発見できなくなるため）。TXID / commit_tx は使わないので部分確定しうるが、追記型DAOなので再実行で収束する。深さ上限32（超過時 `ERR900093 cascade_delete_depth_exceeded`）、失敗時は `ERR900094 cascade_delete_failed` |
| **セッション** | UUID ベースの認証トークン。有効期限は30日。Cookie に `session_id` を保持する。期限以外に、**パスワードを設定しなおすとそのユーザの全セッションが失効する**（他端末のログインも落ちる） |
| **Argon2id** | パスワードの保存に使うメモリハードな鍵導出関数。gkill は `m=65536 KiB, t=3, p=4`、ソルト16バイト、鍵長32バイトで用いる。総当たりに必要な計算資源を引き上げ、`account.db` が流出しても資格情報にならないようにするのが目的 |
| **PHC文字列** | Argon2id の保存形式。`$argon2id$v=19$m=65536,t=3,p=4$<ソルト>$<ハッシュ>` のようにアルゴリズム・パラメータ・ソルトを値自身に含む。パラメータが保存値側にあるので、後からコストを変えても既存の値をそのまま照合できる |
| **パスワードリセットトークン** | パスワードを設定しなおすための単回使用の秘密（UUIDv4）。有効期限は72時間で `ACCOUNT.PASSWORD_RESET_TOKEN_EXPIRATION` に持つ。照合は constant-time。管理者の `/api/reset_password` か CLI の `reset_password` で発行する |
| **MCP サーバ** | AI 統合用 MCP サーバ。3バリアントが存在する。**Read専用**（`gkill-read-server.mjs`、9ツール）・**Write専用**（`gkill-write-server.mjs`、24ツール）・**ReadWrite統合**（`gkill-readwrite-server.mjs`、29ツール）。いずれも共通のプラグインツール1つ（`lib/plugin-tools.mjs` の `PLUGIN_TOOLS`）を含む。各バリアントは stdio（ローカル）/ HTTP（OAuth 2.1付きリモート）の2モードをサポート |

### 凍結された綴り

以下は一見タイポに見えるが**製品綴りとして凍結**しており、修正しない。
綴り修正の提案が繰り返されるのを防ぐためここに記録する。

| 綴り | 一般的な綴り | 凍結理由 |
|---|---|---|
| `plaing` / `Plaing` | playing | SPAルート `/plaing`・`FindQuery` の JSONキー（`plaing_time`）・MCP ツール入力スキーマ・Wear OS のデータレイヤーパス・7言語マニュアルのページ名・`default_page` の保存値と、互換面が5系統に及ぶ。造語としての由来もある（上表参照） |

## 10. 主要ファイルパス相互参照

用語・概念と実装コードの対応表。

### バックエンド（Go）

| 概念 | ファイルパス | 説明 |
|------|-----------|------|
| APIエンドポイント定義 | `src/server/gkill/api/gkill_server_api/gkill_server_api_address.go` | 全92エンドポイントのパス・メソッド定義（91 POST + 1 GET。うち90登録） |
| APIハンドラ（個別） | `src/server/gkill/api/gkill_server_api/handle_*.go` | 個別エンドポイントのハンドラ（handle_*.go 99ファイル、1ハンドラ1ファイル） |
| アクセスログミドルウェア | `src/server/gkill/api/gkill_server_api/gkill_server_api_access_log.go` | gorilla/mux ミドルウェア。全HTTPリクエストのアクセスログを `ACCESS` レベルで記録 |
| リクエスト/レスポンス型 | `src/server/gkill/api/req_res/` | 全エンドポイントの入出力構造体（186ファイル） |
| エラーコード定義 | `src/server/gkill/api/message/error_codes.go` | ERR000001〜ERR000409 の定数定義（計408件。ERR000243は欠番） |
| GkillError / GkillMessage | `src/server/gkill/api/message/` | エラー・メッセージ構造体 |
| KFTLパーサー | `src/server/gkill/api/kftl/` | KFTL テキストパース・リクエスト生成 |
| Embed（SPA埋め込み） | `src/server/gkill/api/embed.go` | `//go:embed embed` ディレクティブ |
| GkillDAOManager | `src/server/gkill/dao/gkill_dao_manager.go` | 全DAOの中央管理 |
| Repository インタフェース | `src/server/gkill/dao/reps/*_repository.go` | 各データ型のリポジトリインタフェース |
| SQLite3実装 | `src/server/gkill/dao/reps/*_repository_sqlite3_impl.go` | SQLite3直接アクセス層 |
| キャッシュ実装 | `src/server/gkill/dao/reps/*_repository_cached_sqlite3_impl.go` | インメモリキャッシュ層 |
| Temp実装 | `src/server/gkill/dao/reps/*_repository_temp_sqlite3_impl.go` | トランザクション用一時層 |
| パスワードのハッシュ化・検証 | `src/server/gkill/dao/account/password_hash.go` | Argon2id、資格情報とユーザIDの形式検証 |
| CLIコマンド・初期化 | `src/server/gkill/main/common/common.go` | Cobra サブコマンド・サーバー起動処理 |
| パスワード管理CLI | `src/server/gkill/main/common/password_admin.go` | `reset_password` サブコマンド、ローカル管理者セッションの発行 |
| CLIフラグ定義 | `src/server/gkill/main/common/gkill_options/option.go` | --gkill_home_dir 等のフラグとディレクトリ構成 |
| gkill_server エントリ | `src/server/gkill/main/gkill_server/main.go` | サーバーモード main() |
| gkill デスクトップ エントリ | `src/server/gkill/main/gkill/main.go` | デスクトップアプリ main() |
| Go モジュール定義 | `src/server/go.mod` | Go 1.26.4、モジュールパス |

### フロントエンド（TypeScript / Vue 3）

| 概念 | ファイルパス | 説明 |
|------|-----------|------|
| エントリポイント | `src/client/main.ts` | アプリ初期化（Vuetify, Router, i18n, v-long-press） |
| ルートコンポーネント | `src/client/App.vue` | テーマ管理・オーバーレイ・グローバルスタイル |
| ルート定義 | `src/client/router/index.ts` | 13ルートの定義 |
| GkillAPI シングルトン | `src/client/classes/api/gkill-api.ts` | バックエンド通信クライアント（約3,300行） |
| リクエスト/レスポンス型 | `src/client/classes/api/req_res/` | TypeScript 版入出力型（172ファイル） |
| データモデル | `src/client/classes/datas/` | Go構造体のTypeScriptミラー |
| DashboardConfig | `src/client/classes/datas/config/dashboard-config.ts` | ダッシュボード設定クラス（MI検索条件・Dnote検索条件） |
| PlaingTimeIsConfig | `src/client/classes/datas/config/plaing-time-is-config.ts` | 実行中検索条件クラス（plaing検索のカスタム条件） |
| SavedFindQueryConfig | `src/client/classes/datas/config/saved-find-query-config.ts` | 保存済み検索条件クラス（ライフログ用・タスク用の名前付き検索条件リスト） |
| ダッシュボードページ | `src/client/pages/dashboard-page.vue` | `/dashboard` ルートのページコンポーネント |
| ダッシュボードComposable | `src/client/classes/use-dashboard-page.ts` | ダッシュボードページのComposable |
| EditDashboardDialog | `src/client/pages/dialogs/edit-dashboard-dialog.vue` | ダッシュボード設定編集ダイアログ |
| EditSavedFindQueryDialog | `src/client/pages/dialogs/edit-saved-find-query-dialog.vue` | 保存済み検索条件のハブダイアログ（ライフログ/タスクの2ボタン） |
| EditSavedFindQueryListDialog | `src/client/pages/dialogs/edit-saved-find-query-list-dialog.vue` | 保存済み検索条件の一覧管理ダイアログ（`query_type` prop で rykv/mi の2インスタンス） |
| MiFindQueryEditorView | `src/client/pages/views/mi-find-query-editor-view.vue` | MI専用検索条件エディタビュー |
| MiFindQueryEditorDialog | `src/client/pages/dialogs/mi-find-query-editor-dialog.vue` | MI専用検索条件エディタダイアログ |
| KFTLパーサー（フロント） | `src/client/classes/kftl/` | フロントエンド版KFTLパーサー（41ステートメント型。`kftl_*/` 配下の具象クラス数） |
| Dnote ユーティリティ | `src/client/classes/dnote/` | 集計機能ユーティリティ |
| Service Worker | `src/client/serviceWorker.ts` | PWA・キャッシュ・Push通知・Web Share Target |
| Vuetify 設定 | `src/client/plugins/vuetify.ts` | テーマカラー定義 |
| i18n 設定 | `src/client/i18n.ts` | 7言語の設定・読み込み |
| ロケールファイル | `src/locales/*.json` | ja, en, zh, ko, es, fr, de（906キー/言語） |

### その他

| 概念 | ファイルパス | 説明 |
|------|-----------|------|
| MCP サーバー（Read） | `src/mcp/gkill-read-server.mjs` | 読み取り専用MCPサーバー（9ツール = 固有8 + プラグイン1、stdio/HTTP） |
| MCP サーバー（Write） | `src/mcp/gkill-write-server.mjs` | 書き込み専用MCPサーバー（24ツール = 固有23 + プラグイン1、stdio/HTTP） |
| MCP サーバー（ReadWrite） | `src/mcp/gkill-readwrite-server.mjs` | 読み書き統合MCPサーバー（29ツール = 固有28 + プラグイン1、stdio/HTTP） |
| MCP プラグインツール | `src/mcp/lib/plugin-tools.mjs` | 3サーバ共通の `gkill_get_plugin_list` と、`gkill_get_kyous` へプラグイン本文を埋める `inlinePluginContents`（読み取りのみ。`post_plugin_config` は公開しない） |
| MCP アクセスログ | `src/mcp/lib/access-log.mjs` | MCPサーバのアクセスログモジュール。`MCP_LOG` 環境変数で制御 |
| Android APK | `src/android/` | WebView ラッパー + gkill_server バイナリ同梱 |
| Wear OS | `src/wear_os/` | phone_companion + watch_app（Gradle マルチモジュール） |
| ビルド設定 | `package.json` | npm scripts、依存関係、バージョン |
| Vite 設定 | `vite.config.ts` | ビルド設定・PWA・エイリアス |
| TypeScript 設定 | `tsconfig.app.json` | フロントエンド TypeScript 設定 |
| ESLint 設定 | `eslint.config.js` | リンター設定（flat config） |
| リバースドキュメント | `documents/reverse/` | 設計・仕様のリバースエンジニアリング文書群 |
| UML モデル | `documents/gkill_model.asta` | Astah UML モデル（ユースケース・クラス・ER等） |
| ユーザードキュメント | `documents/gkill_user_document.pdf` | エンドユーザー向けマニュアル |
