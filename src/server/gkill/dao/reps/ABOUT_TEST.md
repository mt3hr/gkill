# reps テスト仕様

## 概要

全13データ型のリポジトリ SQLite3 実装テスト。CRUD 操作、キャッシュ層、一時リポジトリ層、キャッシュ更新処理を網羅する。

## テストフレームワーク

Go `testing` パッケージ（インメモリ SQLite3 使用）

## テストファイル一覧

### SQLite3 リポジトリ実装テスト（13ファイル）

| ファイル | テスト対象 |
|---------|-----------|
| `kmemo_repository_sqlite3_impl_test.go` | Kmemo（テキストメモ）リポジトリ |
| `mi_repository_sqlite3_impl_test.go` | Mi（タスク）リポジトリ |
| `lantana_repository_sqlite3_impl_test.go` | Lantana（気分値）リポジトリ |
| `time_is_repository_sqlite3_impl_test.go` | TimeIs（タイムスタンプ）リポジトリ |
| `ur_log_repository_sqlite3_impl_test.go` | URLog（ブックマーク）リポジトリ |
| `nlog_repository_sqlite3_impl_test.go` | Nlog（支出記録）リポジトリ |
| `kc_repository_sqlite3_impl_test.go` | KC（数値記録）リポジトリ |
| `tag_repository_sqlite3_impl_test.go` | Tag（タグ）リポジトリ |
| `text_repository_sqlite3_impl_test.go` | Text（テキスト注釈）リポジトリ |
| `notification_repository_sqlite3_impl_test.go` | Notification（通知）リポジトリ |
| `re_kyou_repository_sqlite3_impl_test.go` | ReKyou（リポスト）リポジトリ |
| `mi_re_kyou_repository_sqlite3_impl_test.go` | MiReKyou（リポストタスク）リポジトリ。5射影・板名フィルタ・履歴・最新データ位置 |
| `idf_kyou_repository_sqlite3_impl_test.go` | IDFKyou（ファイル）リポジトリ |

### 外部ソースを直接読むリポジトリ実装テスト（2ファイル）

SQLite3 を持たず、ローカルの git リポジトリや GPX ファイルをその場で読む実装。4層（interface → sqlite3 → cached → temp）の型に当てはまらない。

| ファイル | テスト対象 |
|---------|-----------|
| `git_commit_log_repository_local_dir_impl_test.go` | git コミットログ（ローカルディレクトリ直読み）。削除フラグ付きなら該当なし、ID 指定の意味論（未指定=全件 / 非nil空=0件 / 実ハッシュ=1件）、期間の両端を含むこと、**git リポジトリでないパスなら `ErrNotGitRepository` を包んだエラーを返すこと**（呼び出し側がその rep だけスキップできるようにするため） |
| `gps_log_repository_gpx_dir_impl_test.go` | GPS ログ（GPX ディレクトリ直読み）。日付ごとのファイル解決、期間の両端を含むこと、開始と終了が逆なら入れ替えること、該当日のファイルが無ければ読み飛ばすこと |

### キャッシュ・一時・ユーティリティテスト

| ファイル | テスト内容 |
|---------|-----------|
| `cached_and_temp_test.go` | キャッシュ層 / 一時リポジトリ層の動作検証（MiReKyou のキャッシュ再構築・TX分離を含む）。各 `TestCached*_AddAndGet` は **Add した直後に Get で取り直せること**を確認する（後述） |
| `re_kyou_granular_cache_test.go` | ReKyou のグラニュラーキャッシュ動作検証 |
| `idf_granular_cache_test.go` | IDFKyou のグラニュラーキャッシュ動作検証（`re_kyou_granular_cache_test.go` の IDFKyou 版） |
| `re_kyou_cached_deadlock_test.go` | 共有RWMutexの再帰RLockによる検索恒久ハングの回帰テスト |
| `git_commit_log_cached_nested_pool_test.go` | gitキャッシュビルド中フォールバックのネスト並列プール枯渇と isCacheBuilding データ競合の回帰テスト |
| `mi_re_kyou_cached_nested_pool_test.go` | MiReKyou キャッシュのネスト並列プール枯渇の回帰テスト（`git_commit_log_cached_nested_pool_test.go` と同種） |
| `gkill_repositories_test.go` | 最新版アドレスキャッシュの排他制御（後述） |
| `target_resolution_memo_test.go` | ReKyou/MiReKyou ワード委譲検索のターゲット解決メモ |
| `db_file_change_detector_test.go` | DBファイル変更検出（キャッシュ無効化トリガ） |
| `derived_cache_path_test.go` | 派生キャッシュ（サムネ/動画/ZIP）のユーザ別パス解決 |
| `idf_kyou_repository_batch_test.go` | IDFKyou のバッチ処理 |
| `ur_log_cache_thumbnail_test.go` | URLog サムネイルキャッシュ |
| `shared_find_query_mutation_test.go` | 共有 FindQuery の変更検証 |
| `plugin_repository_impl_test.go` | プラグインのサブプロセス管理（後述）と、検索失敗をエラーではなく警告として返す結線 |
| `find_warnings_test.go` | 検索中の警告コレクタ（収集、コレクタ未設定時の無害さ、並行追加） |
| `find_word_match_test.go` | キーワード一致判定の共通化。除外語のみの検索、AND/OR、ファイル本文の走査、絶対パスを検索対象に含めないこと |
| `cached_find_only_latest_test.go` | 「最新版のみ」指定がキャッシュ実装でも非キャッシュ実装と同じに効くこと（Nlog / KC / TimeIs） |
| `get_kyou_latest_version_test.go` | 単体取得が最新版を返すこと |
| `repositories_id_chunk_test.go` | IDリストの分割（`findChunkedByIDs`）。SQLite のバインド変数上限を超えると**エラーが立たないまま0件**になるので、上限またぎを実データ寸法で固定する |
| `local_rep_cache_granular_test.go` | ローカルキャッシュのコピー省略判定。**「コピーが要るか」の判定を `os.Remove` より後に置くと常に「要コピー」になり**、毎回1.3GBコピーし直したうえで変更検知が丸ごと無効化される |
| `plugin_typed_adapters_test.go` | プラグインの型別/付随データのアダプタ。読み取りが索引から即答し、**プラグインへ往復しない**こと（1件ずつ聞きに行くと一覧の行数ぶんの直列 stdio 呼び出しになる） |
| `plugin_kyou_rep_name_test.go` | プラグイン Kyou の `RepName`。空なら manifest の `rep_name` で埋め、空でない不一致は上書きせず**組み合わせごとに1回だけ**警告すること |
| `gps_log_repository_plugin_impl_test.go` | GPSLog 専用プラグイン（`emits_kyou: false`）のリポジトリ実装 |
| `cache_find_bench_test.go` / `tag_find_bench_test.go` / `kyou_json_bench_test.go` | ベンチマーク（`go test` の既定では走らない）。タグ絞り込みの2経路の交差点や、応答JSONの組み立てを実測するためのもの |
| `mi_find_kyous_parity_test.go` | Mi のキャッシュ実装と非キャッシュ実装で、大小無視と「最新版のみ」の扱いが一致すること |
| `testhelper_test.go` | テストヘルパーユーティリティ |
| `cache/latest_data_repository_address_dao_sqlite3_impl_test.go` | キャッシュアドレス DAO |
| `rep_cache_updater/rep_cache_updater_test.go` | キャッシュ更新処理 |

## テスト内容

- **CRUD 操作**: 各データ型の Create / Read / Update / Delete
- **検索・フィルタ**: ID 検索、日付範囲検索、全件取得
- **IDFKyou**: 追加・取得・カレンダーフィルタ・論理削除・履歴・OnlyLatestData・IsZip検出・Rep名取得の11テストケース
- **4層リポジトリパターン**: interface → SQLite3 実装 → キャッシュ実装 → 一時実装の各層
- **キャッシュ管理**: キャッシュアドレスの解決、キャッシュの更新とクリア

### キャッシュ実装は「Addした直後にGetで取り直せる」まで見る

`TestCachedNotification_AddAndGet` は以前
「キャッシュ実装は直接の Add+Get では往復しない」と書いて `GetRepName` が
空でないことだけを確認していたが、往復しなかったのは仕様ではなく
`GetNotification` のバインドずれ（キャッシュ側のSELECTは `REP_NAME` を列から読むので
プレースホルダは1つしかないのに `queryArgs` に `repName` を積んでいて、
`WHERE ID` に `"notification"` が入っていた）が原因だった。
本番の既定は `--cache_in_memory=true` なので、この経路が壊れると
通知の編集・削除が `ERR000280` で必ず失敗する。

キャッシュ実装を足す・直すときは、`TestCachedReKyou_AddAndGet` と同じく
**取り直した中身**まで確認すること。往復しないなら実装側を疑う。

### `plugin_repository_impl_test.go`（プラグインのサブプロセス管理）

`plugin_repository_impl.go` はプラグインバイナリをサブプロセスとして起動し、
stdio の改行区切りJSONで会話する。ここが壊れると
「プラグインの記録だけ出ない」「1つのプラグインが固まって画面全体の読み込みが終わらない」
といった形で表面化する。

外部バイナリをビルドせずに検証するため、標準ライブラリ `os/exec` のテストで使われている
**テストバイナリ自身を再exec する方式**をとっている。`TestMain` が環境変数
`GKILL_TEST_PLUGIN_MODE` を見て、偽プラグインとしてプロトコルを喋る。
`pluginRepositoryImpl` は実行ファイルを `filepath.Join(pluginDir, Executable[+".exe"])`
で解決するので、`pluginDir` にテストバイナリのあるディレクトリ、`Executable` に
テストバイナリ名を渡すと自分自身が起動する。

- **遅延起動**: リポジトリを作っただけではプロセスが起動せず、最初のクエリで起動する。
  2回目以降のクエリではプロセスを使い回す
- **往復**: `FindKyous` / `GetKyou` / `GetKyouHistories` / `GetContentHTML` /
  `GetConfigHTML` / `PostConfig` / `IsAlive` が stdio 越しに往復する。
  偽プラグインはクエリの Words をそのまま Kyou のIDに詰めて返すので、
  `FindQuery` → `PluginQuery` の変換が効いているかも同時に見える
- **IDフィルタ**: `findQueryToPluginQuery` は `IDs` をプラグインへ渡さない設計なので、
  gkill側で補完しないとID指定検索にプラグインの全件が混入する
- **自動再起動**: プラグインが落ちても次のリクエストで再起動され、呼び出し側にはエラーが見えない
- **デッドライン**: 応答しないプラグインが Deadline で打ち切られ、かつリトライ対象外
  （起動は1回きり）であること。あわせて Deadline 超過ではプロセスが**回収**され、
  次の呼び出しで起動し直されること
- **呼び出し元のキャンセル**: HTTPクライアントの切断（＝ context のキャンセル）では
  プラグインプロセスを道連れにしないこと。フロントは全リクエストに `AbortController` を
  張っているので、ここを混同すると画面を操作するだけでプラグインが落ちる。
  起動回数だけでは kill を検出できないため、キャンセル後にもう一度呼び出して
  同じプロセスで応答が返ることまで確認する
- **古い応答の読み捨て**: 打ち切った呼び出しの応答が遅れて届いても、レスポンスIDの
  突き合わせで読み捨てられ、後続の呼び出しに混入しないこと
- **直列化**: stdio は1本しかないので、並行リクエストが容量1のチャネル（`callSlot`）で直列化され
  レスポンスが取り違わらないこと（偽プラグインが KyouID をそのまま返すので検出できる）。
  順番待ちが `maxPluginQueueWait`（既定10秒）を超えた呼び出しは `ErrPluginBusy` になるだけで、
  プロセスは回収されないこと

### `gkill_repositories_test.go`（最新版アドレスキャッシュの排他制御）

`GkillRepositories` はユーザ+デバイス単位で共有され、検索（`FindKyous`）と
追加/更新（`usecase`, `handle_commit_tx`）の両方から同時に触られる。
最新版アドレスの map を素のまま公開していたころは、
「rykv の自動更新中に KFTL で投稿する」「タブを2枚開く」だけで
`concurrent map read and map write` に当たり、recover できない fatal error で
サーバプロセスごと落ちていた。

**`go test -race` で検出させることを狙ったテスト**なので、
通常実行で素通りしたことは正常性の証明にならない。

- **1件ずつの並行読み書き**: `SetLatestDataRepositoryAddress` と
  `GetLatestDataRepositoryAddress` が同時に走っても壊れず、書いた分がすべて読めること
- **map全体の差し替えとの並行**: 検索開始時のキャッシュ再取得
  （`SetLatestDataRepositoryAddresses`）は map のヘッダごと入れ替わるため、
  読み手が旧 map を見ている最中の切り替えで特に壊れやすい
- **nil map からの初期化**: 一度も検索を通していない状態で追加操作が来ても
  panic しないこと（「初回検索前に KFTL で投稿する」経路で実際に踏む）

## 実行方法

```bash
cd src/server && go test ./gkill/dao/reps/...
```

または:

```bash
npm run test_server
```
