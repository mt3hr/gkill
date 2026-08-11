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
| `common/auto_tag_test.go` | `auto_tag` サブコマンド。ルール文字列の解釈（`<prefix>=<tag>` / rep 種別）、付与予定の重複排除、タグ ID が (対象ID, タグ名) で一意かつ**過去に発行した値と一致し続ける**こと、付与済み照会のクエリが `tags_and` を立てて呼び出し元のクエリを壊さないこと |
| `common/gkill_options/option_test.go` | CLI フラグのデフォルト値（`--gkill_home_dir`, `--cache_in_memory`, `--goroutine_pool` 等） |
| `common/gkill_log/gkill_log_test.go` | ログレベル別ルーティング（error, warn, info, debug, trace, trace_sql） |
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
- **ログ**: レベル別（none/error/warn/info/debug/trace/trace_sql）のファイルルーティング
- **ゴルーチンプール**: プール生成、並行タスク実行、`runtime.NumCPU()` 準拠のプールサイズ
- **エントリポイント**: cobra コマンド登録、サブコマンド（version, idf, dvnf 等）の存在確認
- **自動タグ付け**: `auto_tag` のルール解釈・差分計算・タグ ID の再現性（稼働中サーバへの HTTP 部分はテストしない）

## 実行方法

```bash
cd src/server && go test ./gkill/main/...
```

または:

```bash
npm run test_server
```
