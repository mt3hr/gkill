# main テスト仕様

## 概要

CLI エントリポイントとユーティリティのテスト。共有 CLI ロジック、オプションフラグ、ログルーティング、ゴルーチンプール、各バイナリのエントリポイント、バッチ処理を検証する。

## テストフレームワーク

Go `testing` パッケージ

## テストファイル一覧

### 共有ユーティリティ (common/)

| ファイル | テスト内容 |
|---------|-----------|
| `common/common_test.go` | 共有 CLI ロジック（15テスト）。サブコマンド登録（idf / dvnf / version / generate_thumb_cache / generate_video_cache / clear_cache / generate_plugin_cache / optimize / update_cache / reset_password / add_tag / mcp が全部 RunE + SilenceUsage/SilenceErrors であること）の確認と、`clear_cache` の挙動固定 — `all` で全キャッシュディレクトリを消す / 単一モードで他を残す / `plugin` モードは plugin_cache だけ消す / `ClearPluginCache` が対象ユーザ以外を消さない / 危険な user_id を弾く |
| `common/generate_plugin_cache_test.go` | `generate_plugin_cache` サブコマンド（10テスト + 偽プラグイン用の `TestMain`）。子プロセスの結果分類（stdout 全体が `built` / `no_cache` に完全一致するときだけ成功。exit 0 で空 stdout・余計な出力・exit 2・exit 1 は失敗）、テストバイナリ自身を偽プラグインとして起動して `--gkill-build-cache` の受け渡し・stdout の捕捉・stdin nil（フラグを無視する旧バイナリでもハングしない）・終了コードの分類が繋がること、対象の選択（`all` は発見順に全部・名前指定は完全一致1本・無い名前は利用可能名を列挙してエラー・0本は成功）、存在しない user_id と危険な user_id をプラグインの走査より前に弾くこと（`plugins/<typo>/` を作らない）、1本失敗しても残りを続けて `errors.Join` で返すこと |
| `common/add_tag_test.go` | `add_tag` サブコマンド（33テスト）。ルール JSON の厳格復号（BOM・旧 `use_*` 形式・未知キー・末尾の余分な内容）、拒否する指定（絞り込み無し / 常に0件の空配列 / `keywords` / `update_cache` / 揃っていない地図3値 など）と通す指定の網羅表、クライアントの `FindKyouQuery` の全キーを受けること（TS ソース走査）、`rep_types_in_sidebar` の展開（dvnf 名の3分割・端末絞り・重複除去）、付与予定の重複排除、タグ行が呼ぶたびに違うランダム UUID で出所が `gkill_add_tag` であること（決定的 ID へ戻すと消したタグが付け直されない）、付与済み照会のクエリが `tags_and` を立ててタグ以外の条件を残すこと、httptest モックでの HTTP 応答判定（本文優先・409+ERR000056 も失敗・上限超過・警告メッセージの印字） |
| `common/fix_timezone_test.go` | Android で libc（SQLite の `'localtime'`）へ端末のゾーンを教える経路（5テスト）。packed tzdata（AOSP の索引形式）から名前完全一致で TZif を切り出し、壊れたヘッダ・範囲外・TZif でない中身を拒むこと、TZif が取れたら `$GKILL_HOME/tz/localtime` に置いて `TZ=:<パス>` を入れ同じ中身なら書き直さないこと、取れなければ POSIX 固定オフセット（`JST-9` / `IST-5:30` / `UTC0` / `<-05>+5`）へ落ちること、未展開の `$VAR/gkill`（既定の `--gkill_home_dir` の形）や相対パスを渡しても展開済みの絶対パスが `TZ` に入り CWD に文字どおり `$VAR` というディレクトリを掘らないこと（空に展開されるなら POSIX へ落ちる。Termux で0件が続いた穴）、Android 以外では `TZ` を触らないこと。Android そのものは CI に無いので合成データで固定する |
| `common/gkill_options/option_test.go` | CLI フラグのデフォルト値（`--gkill_home_dir`, `--cache_in_memory`, `--goroutine_pool` 等） |
| `common/mcp_test.go` | `mcp` サブコマンドの配線（4テスト）。`--kind` / `--transport` / `--config` と `schema-budget --update` の存在、`--kind` の必須と検証、設定ファイルが初回に生成されること、親の `--log` は**明示されたときだけ** `MCP_LOG` / 設定より優先すること、`--gkill_home_dir` 未指定時だけ `GKILL_HOME` を採ること。サーバ本体の振る舞いは `gkill/mcp` のテストが持つ |
| `common/gkill_log/child_env_test.go` | プラグインの子プロセスへ継ぐログ設定（5テスト）。レベル語彙 `ParseLevel`、ディレクトリを作れないときに panic せず error を返し既定ロガーも router も差し替えない `InitNamedWith`、`ExportEnvForChildProcesses` → `ChildSettingsFromEnv` の往復、未設定は既定・壊れた値は Warnings で既定へ倒す、`Init()` が3変数を書き出すこと（ADR-0313） |
| `common/gkill_log/gkill_log_named_test.go` | `InitNamed`（2テスト）。接頭辞で統合 + レベル別 7 本の全部を開き静的フィールド `app` / 追加欄が全行に付くこと（本体の `gkill*.log` を1つも作らない = MCP の行が本体の error ログへ混ざらない）、`Init` 前の `Close` が無害なこと |
| `common/gkill_log/gkill_log_test.go` | ログレベル別ルーティング（error, warn, info, debug, trace, trace_sql）、統合ログとレベル別ログへの同時出力、静的 `app` フィールド、サイズ超過時の世代回転、回転無効化。Windowsで開いたファイルをrenameしない順序も実ファイルで固定する |
| `common/threads/threads_test.go` | ゴルーチンプールの生成・タスク実行・プールサイズ管理 |

### エントリポイント

| ファイル | テスト内容 |
|---------|-----------|
| `gkill/main_test.go` | デスクトップアプリ（go-astilectron）エントリポイント |
| `gkill_server/main_test.go` | HTTP サーバエントリポイント |

### バッチ処理

| ファイル | テスト内容 |
|---------|-----------|

## テスト内容

- **CLI オプション**: 各フラグのデフォルト値検証、フラグ解析
- **ログ**: レベル別（none/error/warn/info/debug/trace/trace_sql）のファイルルーティング、統合+分割出力、JSON静的フィールド、サイズ/世代数による回転
- **ゴルーチンプール**: プール生成、並行タスク実行、`runtime.NumCPU()` 準拠のプールサイズ
- **エントリポイント**: cobra コマンド登録、サブコマンド（version, idf, dvnf 等）の存在確認
- **タグ付け**: `add_tag` のルール JSON の復号と検証・`rep_types_in_sidebar` の展開・差分計算・タグ行の組み立て（ランダム UUID）。稼働中サーバへの HTTP 部分は httptest のモックで応答判定（本文の `errors` 優先・非2xx の扱い・上限超過）を固定する
- **プラグインキャッシュ構築**: `generate_plugin_cache` の対象選択・利用者の実在確認の順序・子プロセスの結果分類。プラグインバイナリの代わりにテストバイナリ自身を再exec して（`dao/reps` のプラグインテストと同じ方式）、フラグの受け渡しと stdout / 終了コードの扱いを実プロセスで固定する

## 実行方法

```bash
cd src/server && go test ./gkill/main/...
```

または:

```bash
npm run test_server
```
