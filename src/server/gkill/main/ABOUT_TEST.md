# main テスト仕様

## 概要

CLI エントリポイントとユーティリティのテスト。共有 CLI ロジック、オプションフラグ、ログルーティング、ゴルーチンプール、各バイナリのエントリポイント、バッチ処理を検証する。

## テストフレームワーク

Go `testing` パッケージ

## テストファイル一覧

### 共有ユーティリティ (common/)

| ファイル | テスト内容 |
|---------|-----------|
| `common/common_test.go` | 共有 CLI ロジック（14テスト）。サブコマンド登録（idf / dvnf / version / generate_thumb_cache / generate_video_cache / optimize / update_cache）の確認と、`clear_cache` の挙動固定 — `all` で全キャッシュディレクトリを消す / 単一モードで他を残す / `plugin` モードは plugin_cache だけ消す / `ClearPluginCache` が対象ユーザ以外を消さない / 危険な user_id を弾く |
| `common/add_tag_test.go` | `add_tag` サブコマンド（33テスト）。ルール JSON の厳格復号（BOM・旧 `use_*` 形式・未知キー・末尾の余分な内容）、拒否する指定（絞り込み無し / 常に0件の空配列 / `keywords` / `update_cache` / 揃っていない地図3値 など）と通す指定の網羅表、クライアントの `FindKyouQuery` の全キーを受けること（TS ソース走査）、`rep_types_in_sidebar` の展開（dvnf 名の3分割・端末絞り・重複除去）、付与予定の重複排除、タグ ID が (対象ID, タグ名) で一意かつ**過去に発行した値と一致し続ける**こと（改名後も `gkill_auto_tag` 名前空間）、付与済み照会のクエリが `tags_and` を立ててタグ以外の条件を残すこと、httptest モックでの HTTP 応答判定（本文優先・409+ERR000056 のスキップ・上限超過・警告メッセージの印字） |
| `common/gkill_options/option_test.go` | CLI フラグのデフォルト値（`--gkill_home_dir`, `--cache_in_memory`, `--goroutine_pool` 等） |
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
- **タグ付け**: `add_tag` のルール JSON の復号と検証・`rep_types_in_sidebar` の展開・差分計算・タグ ID の再現性。稼働中サーバへの HTTP 部分は httptest のモックで応答判定（本文の `errors` 優先・非2xx の扱い・上限超過）を固定する

## 実行方法

```bash
cd src/server && go test ./gkill/main/...
```

または:

```bash
npm run test_server
```
