# reps テスト仕様

## 概要

全12データ型のリポジトリ SQLite3 実装テスト。CRUD 操作、キャッシュ層、一時リポジトリ層、キャッシュ更新処理を網羅する。

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

### キャッシュ・一時・ユーティリティテスト

| ファイル | テスト内容 |
|---------|-----------|
| `cached_and_temp_test.go` | キャッシュ層 / 一時リポジトリ層の動作検証（MiReKyou のキャッシュ再構築・TX分離を含む）。各 `TestCached*_AddAndGet` は **Add した直後に Get で取り直せること**を確認する（後述） |
| `re_kyou_granular_cache_test.go` | ReKyou のグラニュラーキャッシュ動作検証 |
| `plugin_repository_impl_test.go` | プラグインのサブプロセス管理（後述） |
| `testhelper_test.go` | テストヘルパーユーティリティ |
| `cache/latest_data_repository_address_dao_sqlite3_impl_test.go` | キャッシュアドレス DAO |
| `cache/rep_cache_updater/rep_cache_updater_test.go` | キャッシュ更新処理 |
| `rep_cache_updater/rep_cache_updater_test.go` | キャッシュ更新処理（代替パス） |

## テスト内容

- **CRUD 操作**: 各データ型の Create / Read / Update / Delete
- **検索・フィルタ**: ID 検索、日付範囲検索、全件取得
- **IDFKyou**: 追加・取得・カレンダーフィルタ・論理削除・履歴・OnlyLatestData・IsZip検出・Rep名取得の9テストケース
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
- **IDフィルタ**: `findQueryToPluginQuery` は `UseIDs` をプラグインへ渡さない設計なので、
  gkill側で補完しないとID指定検索にプラグインの全件が混入する
- **自動再起動**: プラグインが落ちても次のリクエストで再起動され、呼び出し側にはエラーが見えない
- **タイムアウト**: 応答しないプラグインに対して context のキャンセルでリクエストが打ち切られ、
  かつタイムアウトはリトライ対象外（起動は1回きり）であること
- **直列化**: stdio は1本しかないので、並行リクエストが mutex で直列化され
  レスポンスが取り違わらないこと（偽プラグインが KyouID をそのまま返すので検出できる）

## 実行方法

```bash
cd src/server && go test ./gkill/dao/reps/...
```

または:

```bash
npm run test_server
```
