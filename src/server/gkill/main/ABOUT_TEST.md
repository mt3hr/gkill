# main テスト仕様

## 概要

CLI エントリポイントとユーティリティのテスト。共有 CLI ロジック、オプションフラグ、ログルーティング、ゴルーチンプール、各バイナリのエントリポイント、バッチ処理を検証する。

## テストフレームワーク

Go `testing` パッケージ

## テストファイル一覧

### 共有ユーティリティ (common/)

| ファイル | テスト内容 |
|---------|-----------|
| `common/common_test.go` | 共有 CLI ロジック（サーバ初期化、共通処理） |
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
| `gkill_fitbit_kc_convert_batch/gkill_fitbit_kc_convert_batch_test.go` | Fitbit データ → KC 変換バッチ処理 |
| `old_data_transfomer_for_old_rykv/old_data_transfomer_test.go` | 旧 RYKV データの移行・変換 |

## テスト内容

- **CLI オプション**: 各フラグのデフォルト値検証、フラグ解析
- **ログ**: レベル別（none/error/warn/info/debug/trace/trace_sql）のファイルルーティング
- **ゴルーチンプール**: プール生成、並行タスク実行、`runtime.NumCPU()` 準拠のプールサイズ
- **エントリポイント**: cobra コマンド登録、サブコマンド（version, idf, dvnf 等）の存在確認
- **バッチ**: Fitbit JSON → KC レコード変換、旧形式データのマイグレーション

## 実行方法

```bash
cd src/server && go test ./gkill/main/...
```

または:

```bash
npm run test_server
```
