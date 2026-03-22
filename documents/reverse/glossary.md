# gkill 用語集

gkill で使われる独自用語・略称・概念の定義集。コードベース・ロケールファイル・開発者からの情報に基づく。

## 1. アプリケーション名・基本概念

| 用語 | 説明 | 由来 |
|------|------|------|
| **gkill** | ライフログアプリケーション。人生の出来事（メモ、ブックマーク、タスク、気分、支出、打刻、ファイル、Gitコミット等）を記録・閲覧する | lifelogging application😖 |
| **Kyou** | 全データ型の集約ビュー表現。`DataType` フィールドで具体的な型を識別する。API レスポンスで統一的に返却される基底エンティティ | — |
| **Append-Only** | gkill のデータ更新方式。レコードの UPDATE/DELETE を行わず、同一 ID で新しいレコードを INSERT する。`UPDATE_TIME` が最新のレコードが有効データとなり、過去のバージョンは履歴として保持される | — |

## 2. データ型（Kyou サブタイプ）

### 主要データ型

| 用語 | UI表示名 | 説明 | 由来 |
|------|---------|------|------|
| **Kmemo** | メモ帳 | テキストメモ。内容（`Content`）を自由記述で記録する | — |
| **KC** | 数値記録 | タイトル付き実数値（`NumValue`）を記録する汎用数値型。体重・回数・温度など任意の数値を追跡可能 | Super Kintore Counter（筋トレカウンター）の汎化 |
| **Lantana** | 気分 | 気分値（`Mood`）を 0〜10 のスケールで記録する | 日本語名「七変化」。花の色の変化が気分の変化を連想させることから。旧気分記録アプリ Lantana に由来 |
| **Mi** | タスク | タスク管理。ボード名（`BoardName`）でグルーピングし、期限（`LimitTime`）・開始/終了予定・チェック状態（`IsChecked`）を持つ | — |
| **Nlog** | 支出 | 支出記録。店名（`Shop`）・タイトル（`Title`）・金額（`Amount`）を記録する | yeNlog（日本円を記録する型） |
| **URLog** | ブックマーク | URL 記録。URL・タイトル・説明・ファビコン画像・サムネイル画像を保持する | URL + Log |
| **TimeIs** | 打刻帳 | 時間記録（打刻）。タイトル・開始時刻（`StartTime`）・終了時刻（`EndTime`）で状況を記録する。`EndTime` が null の場合は「実行中」状態 | — |
| **IDFKyou** | ファイル | ファイル参照。対象リポジトリ名・ファイルパスを持ち、画像/動画/音声のフラグを保持する | gkill ID allocate to File |
| **ReKyou** | リポスト | 既存 Kyou の再投稿。`TargetID` で参照先の Kyou を指定する | Re（再）+ Kyou |

### 外部データ取り込み型

| 用語 | 説明 |
|------|------|
| **GitCommitLog** | ローカル Git リポジトリからコミットログを読み取ってキャッシュする型。コミットメッセージ・追加行数（`Addition`）・削除行数（`Deletion`）を含む |
| **GPSLog** | GPX ファイルから GPS 位置情報を読み取る型 |

### メタデータ型

| 用語 | UI表示名 | 説明 |
|------|---------|------|
| **Tag** | タグ | 任意の Kyou に紐づくタグ。`TargetID` で対象 Kyou を参照する |
| **Text** | テキスト | 任意の Kyou に紐づくテキスト注釈。`TargetID` で対象 Kyou を参照する |
| **Notification** | 通知 | 任意の Kyou に紐づく通知情報。通知日時（`NotificationTime`）・内容（`Content`）・通知済みフラグ（`IsNotificated`）を持つ |

## 3. 画面・ページ名

| 用語 | UI表示名 | ルート | 説明 | 由来 |
|------|---------|-------|------|------|
| **KFTL** | メモ帳 | `/kftl` | テキストベース記録入力画面。KFTL 構文（後述）を使って複数データ型を一括入力できる | Key Fairy Textbase Lifelogger。記録導線を「あける」鍵開けの妖精 |
| **Rykv** | ライフログビュー | `/rykv` | メインの閲覧・検索画面。タイムライン形式でKyouを表示し、DNote（集計）や Ryuu（関連情報）も統合 | RYuuKyouViewer。流れるように見える（タイムライン無限スクロール）、留める、龍のように（歴史と知を連想） |
| **Mi** | タスク | `/mi` | タスク管理画面。ボード形式で Mi を表示し、チェック状態のフィルタ・ソート・共有機能を提供 | — |
| **Kyou** | 記録 | `/kyou` | 記録一覧表示画面 | — |
| **Mkfl** | 打刻メモ帳 | `/mkfl` | 打刻（TimeIs）とメモ（Kmemo）を組み合わせた複合入力ビュー | Master Key Fairy Lifelogger。記録導線を「あける」鍵開けの妖精 |
| **Plaing** | 実行中 | `/plaing` | 稼働中の TimeIs（`EndTime` が null）を一覧表示する画面。状況キーワードやタグで絞り込み可能 | Playing と Plain のミックス造語 |
| **Saihate** | さいはて | `/saihate` | 記録だけに特化した画面。他画面への遷移ができない。「さいはてでも記録ができる」がコンセプト | 「最果て」。主にスマホのロック画面からアクセスする用途 |

### Rykv 内の機能ビュー

| 用語 | UI表示名 | 説明 | 由来 |
|------|---------|------|------|
| **DNote** | 集計ビュー | データ集計・分析機能。Predicate（条件）でフィルタし、KeyGetter（集計キー）でグルーピングし、AggregateTarget（集計対象）で集計する | — |
| **Ryuu** | 関連情報 | 表示中の Kyou と時刻が近い関連 Kyou を表示する機能。検索範囲（n時間）と抽出条件を指定可能 | — |

### 認証不要ページ

| ルート | 説明 |
|-------|------|
| `/shared_page` | 共有リンクから Kyou を閲覧する画面（認証不要） |
| `/shared_mi` | 共有リンクからタスクを閲覧する画面（認証不要） |
| `/set_new_password` | パスワードリセットリンクから新パスワードを設定する画面 |
| `/regist_first_account` | 初回起動時のアカウント登録画面 |

## 4. KFTL 構文

KFTL（Key Fairy Textbase Lifelogger）は、テキストで複数のデータ型を一括入力するための独自フォーマット。行頭のプレフィックスでデータ型を判別する。

### プレフィックス一覧

| プレフィックス | データ型 | 説明 |
|---------------|---------|------|
| （なし） | Kmemo | デフォルト。プレフィックスなしの行はテキストメモとして扱われる |
| `。` | Tag | タグを追加する。`。タグ名` の形式 |
| `ーー` | Text | テキスト注釈の開始/終了。`ーー` で囲まれた範囲がテキスト |
| `？` | RelatedTime | 関連時刻を指定する。`？時刻` の形式 |
| `、` | Split | 区切り。現在のステートメントを終了し、次のステートメントを開始する |
| `、、` | SplitNextSecond | 区切り＋時刻の `AddSecond` を +1 する |
| `ーか` | KC | 数値記録の開始。後続行でタイトル → 数値を入力 |
| `ーみ` | Mi | タスクの開始。後続行でタイトル → [ボード名] → [期限] → [開始予定] → [終了予定] を入力 |
| `ーら` | Lantana | 気分値の開始。後続行で気分値（0〜10）を入力 |
| `ーん` | Nlog | 支出の開始。後続行でタイトル → 店名 → 金額を入力 |
| `ーう` | URLog | ブックマークの開始。後続行でタイトル → URL を入力 |
| `ーた` | TimeIs Start | TimeIs 開始のみ（`StartTime` を設定、`EndTime` = null） |
| `ーえ` | TimeIs End | タイトル指定で実行中の TimeIs を終了する |
| `ーいえ` | TimeIs End If Exist | 存在する場合のみ TimeIs を終了する |
| `ーたえ` | TimeIs End By Tag | タグ名指定で実行中の TimeIs を終了する |
| `ーいたえ` | TimeIs End By Tag If Exist | タグ名指定で、存在する場合のみ TimeIs を終了する |
| `ーち` | TimeIs | 開始と終了を同時に設定する（`StartTime` + `EndTime` 両方を記録） |
| `！` | Save | 保存実行。パースを終了してリクエストを実行する |

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
| **GkillServerAPI** | HTTP API ハンドラ。gorilla/mux で全エンドポイント（76+）を提供する。`gkill_server_api.go`（約557KB）に集約 |
| **TempReps** | KFTL パース時のトランザクション用一時リポジトリ。`CommitTX` で本リポジトリに反映、`DiscardTX` で破棄する |
| **Rep / 記録保管場所** | データ保存先の SQLite3 ファイル。ユーザ・デバイス・データ型ごとに割り当てられる |
| **RepType / 記録タイプ** | リポジトリの分類。メモ帳、打刻帳、支出、数値記録、タスク、気分、ブックマーク、リポスト等 |
| **LatestDataRepositoryAddress** | 各 ID の最新レコードがどのリポジトリに格納されているかを示すキャッシュ。検索時の `OnlyLatestData` フィルタで使用 |
| **DVNF** | Data Versioning and Notification Framework。ファイルのバージョン管理とコピー・移動を行う CLI ユーティリティ。`gkill dvnf copy/get/move` コマンドで呼び出す |

## 6. 設定用語

| 用語 | UI表示名 | 説明 |
|------|---------|------|
| **Device / プロファイル** | プロファイル | デバイスや利用環境の設定単位。PC・スマートフォン・Wear OS 等のプロファイルを使い分けられる |
| **ApplicationConfig** | 設定 | ユーザ別のアプリケーション設定。UI テーマ・デフォルト表示日数・テンプレート・構造定義等を含む |
| **ServerConfig** | サーバ設定 | サーバ全体の設定。TLS・ポート番号・データディレクトリ・コマンドパス等 |
| **TagStruct** | タグ構造 | タグの階層構造定義。フォルダでグルーピングし、初期化時チェック・非表示優先等のオプションを持つ |
| **RepStruct** | 記録保管場所構造 | リポジトリの構造定義 |
| **RepTypeStruct** | 記録タイプ構造 | リポジトリ型の分類定義 |
| **KFTLTemplate** | テンプレート | KFTL 入力のテンプレート定義。テンプレート名と内容を持ち、Wear OS タイルからの入力にも使用される |
| **DeviceStruct** | プロファイル構造 | デバイスプロファイルの定義 |
| **MiBoardStruct** | — | タスクボードの構造定義 |

## 7. DNote 集計システム用語

DNote はデータ集計・分析機能。Predicate → KeyGetter → AggregateTarget の3段階で処理する。

### 処理コンポーネント

| 用語 | UI表示名 | 説明 |
|------|---------|------|
| **Predicate / 条件** | 条件 | フィルタリング条件。AND/OR/NOT の論理演算で組み合わせ可能。30+ 種類の具象クラスがある |
| **KeyGetter / 集計キー** | 集計キー | グルーピング基準。関連日・関連年月・関連曜日・タグ・データタイプ・タイトル・店名・気分値等から選択 |
| **AggregateTarget / 集計対象** | 集計対象 | 集計関数。件数・合計・平均・最大値・最小値を、支出額・気分値・作業時間・コード行数等に適用可能 |

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
| **RelatedTime** | time.Time | 関連日時。時系列表示のソート基準として使用される |
| **CreateTime** | time.Time | レコード作成日時 |
| **UpdateTime** | time.Time | レコード更新日時。Append-Only のバージョン識別子として機能する |
| **CreateApp** | string | 作成元アプリケーション名（例: "gkill", "wear_os_app"） |
| **CreateDevice** | string | 作成元デバイス（プロファイル）名 |
| **CreateUser** | string | 作成ユーザ名 |
| **UpdateApp / UpdateDevice / UpdateUser** | string | 更新元のアプリ・デバイス・ユーザ |
| **IsDeleted** | bool | 論理削除フラグ。`true` のレコードが INSERT されることで論理削除を実現する |
| **TargetID** | string | Tag / Text / Notification / ReKyou が参照する対象 Kyou の ID |

## 9. その他の用語

| 用語 | 説明 |
|------|------|
| **板（Board）** | Mi（タスク）のグルーピング単位。UI 上では `板名` と表記される |
| **ブックマークレット** | ブラウザから URLog を直接追加するための JavaScript。ログイン時に専用セッションが自動作成される |
| **共有ページ** | 認証不要で Kyou やタスクを閲覧できる公開リンク。`ShareKyouListInfo` で検索条件を含む共有リンクを生成する |
| **オレオレTLS** | 自己署名 TLS 証明書の生成機能。サーバ設定画面から生成できる |
| **ホットリロード** | データの自動再読み込み機能 |
| **FindQuery** | 検索クエリ。キーワード・日時範囲・タグ・データ型・デバイス等の複合条件で Kyou を検索する |
| **OnlyLatestData** | 検索フィルタ。同一 ID のレコードのうち `UpdateTime` が最新のもののみを返す |
| **セッション** | UUID ベースの認証トークン。有効期限は30日。Cookie に `session_id` を保持する |
| **MCP サーバ** | gkill-read-server.mjs。AI 統合用の読み取り専用 MCP サーバ（6ツール提供: gkill_get_kyous, gkill_get_mi_board_list, gkill_get_all_tag_names, gkill_get_all_rep_names, gkill_get_gps_log, gkill_get_application_config） |

## 10. 主要ファイルパス相互参照

用語・概念と実装コードの対応表。

### バックエンド（Go）

| 概念 | ファイルパス | 説明 |
|------|-----------|------|
| APIエンドポイント定義 | `src/server/gkill/api/gkill_server_api_address.go` | 全77エンドポイントのパス・メソッド定義 |
| APIハンドラ（メイン） | `src/server/gkill/api/gkill_server_api.go` | ルーティング登録・ハンドラ実装（~12,300行） |
| APIハンドラ（個別） | `src/server/gkill/api/handle_*.go` | 個別エンドポイントのハンドラ |
| リクエスト/レスポンス型 | `src/server/gkill/api/req_res/` | 全エンドポイントの入出力構造体（150+ファイル） |
| エラーコード定義 | `src/server/gkill/api/message/error_codes.go` | ERR000001〜ERR000372 の定数定義 |
| GkillError / GkillMessage | `src/server/gkill/api/message/` | エラー・メッセージ構造体 |
| KFTLパーサー | `src/server/gkill/api/kftl/` | KFTL テキストパース・リクエスト生成 |
| Embed（SPA埋め込み） | `src/server/gkill/api/embed.go` | `//go:embed embed` ディレクティブ |
| GkillDAOManager | `src/server/gkill/dao/gkill_dao_manager.go` | 全DAOの中央管理 |
| Repository インタフェース | `src/server/gkill/dao/reps/*_repository.go` | 各データ型のリポジトリインタフェース |
| SQLite3実装 | `src/server/gkill/dao/reps/*_repository_sqlite3_impl.go` | SQLite3直接アクセス層 |
| キャッシュ実装 | `src/server/gkill/dao/reps/*_repository_cached_sqlite3_impl.go` | インメモリキャッシュ層 |
| Temp実装 | `src/server/gkill/dao/reps/*_repository_temp_sqlite3_impl.go` | トランザクション用一時層 |
| CLIコマンド・初期化 | `src/server/gkill/main/common/common.go` | Cobra サブコマンド・サーバー起動処理 |
| CLIフラグ定義 | `src/server/gkill/main/common/gkill_options/option.go` | --gkill_home_dir 等のフラグとディレクトリ構成 |
| gkill_server エントリ | `src/server/gkill/main/gkill_server/main.go` | サーバーモード main() |
| gkill デスクトップ エントリ | `src/server/gkill/main/gkill/main.go` | デスクトップアプリ main() |
| Go モジュール定義 | `src/server/go.mod` | Go 1.26.0、モジュールパス |

### フロントエンド（TypeScript / Vue 3）

| 概念 | ファイルパス | 説明 |
|------|-----------|------|
| エントリポイント | `src/client/main.ts` | アプリ初期化（Vuetify, Router, i18n, v-long-press） |
| ルートコンポーネント | `src/client/App.vue` | テーマ管理・オーバーレイ・グローバルスタイル |
| ルート定義 | `src/client/router/index.ts` | 12ルートの定義 |
| GkillAPI シングルトン | `src/client/classes/api/gkill-api.ts` | バックエンド通信クライアント（~3,400行） |
| リクエスト/レスポンス型 | `src/client/classes/api/req_res/` | TypeScript 版入出力型（150+ファイル） |
| データモデル | `src/client/classes/datas/` | Go構造体のTypeScriptミラー |
| KFTLパーサー（フロント） | `src/client/classes/kftl/` | フロントエンド版KFTLパーサー（44ステートメント型） |
| DNote ユーティリティ | `src/client/classes/dnote/` | 集計機能ユーティリティ |
| Service Worker | `src/client/serviceWorker.ts` | PWA・キャッシュ・Push通知・Web Share Target |
| Vuetify 設定 | `src/client/plugins/vuetify.ts` | テーマカラー定義 |
| i18n 設定 | `src/client/i18n.ts` | 7言語の設定・読み込み |
| ロケールファイル | `src/locales/*.json` | ja, en, zh, ko, es, fr, de（~743キー/言語） |

### その他

| 概念 | ファイルパス | 説明 |
|------|-----------|------|
| MCP サーバー | `src/mcp/gkill-read-server.mjs` | 読み取り専用MCPサーバー（6ツール、stdio/HTTP） |
| Android APK | `src/android/` | WebView ラッパー + gkill_server バイナリ同梱 |
| Wear OS | `src/wear_os/` | phone_companion + watch_app（Gradle マルチモジュール） |
| ビルド設定 | `package.json` | npm scripts、依存関係、バージョン (1.1.0-dev) |
| Vite 設定 | `vite.config.ts` | ビルド設定・PWA・エイリアス |
| TypeScript 設定 | `tsconfig.app.json` | フロントエンド TypeScript 設定 |
| ESLint 設定 | `.eslintrc.cjs` | リンター設定 |
| リバースドキュメント | `documents/reverse/` | 設計・仕様のリバースエンジニアリング文書群 |
| UML モデル | `documents/gkill_model.asta` | Astah UML モデル（ユースケース・クラス・ER等） |
| ユーザードキュメント | `documents/gkill_user_document.pdf` | エンドユーザー向けマニュアル |
