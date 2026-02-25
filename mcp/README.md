## MCP連携（Read専用）
gkill の Read API を MCP サーバとして公開できます。  
この実装は `R` のみ対応で、`Add/Update/Delete` は公開しません。

### 起動
1. gkill_server を起動（例: `http://127.0.0.1:9999`）
2. 環境変数を設定
3. MCP サーバを起動

```powershell
$env:GKILL_BASE_URL="http://127.0.0.1:9999"
$env:GKILL_USER="admin"
$env:GKILL_PASSWORD_SHA256="<sha256 hex>"  # または GKILL_PASSWORD
npm run mcp:gkill-read
```

### 主な環境変数
- `GKILL_BASE_URL` (default: `http://127.0.0.1:9999`)
- `GKILL_USER`
- `GKILL_PASSWORD_SHA256` または `GKILL_PASSWORD`
- `GKILL_SESSION_ID` (任意。指定時はログインをスキップ)
- `GKILL_LOCALE` (default: `ja`)

### 提供ツール
- `gkill.get_kyous`
- `gkill.get_kyou`
- `gkill.get_kmemo`
- `gkill.get_kc`
- `gkill.get_urlog`
- `gkill.get_nlog`
- `gkill.get_timeis`
- `gkill.get_mi`
- `gkill.get_lantana`
- `gkill.get_rekyou`
- `gkill.get_git_commit_log`
- `gkill.get_idf_kyou`
- `gkill.get_mi_board_list`
- `gkill.get_all_tag_names`
- `gkill.get_all_rep_names`
- `gkill.get_tags_by_target_id`
- `gkill.get_texts_by_target_id`
- `gkill.get_notification_histories_by_notification_id`
- `gkill.get_gps_log`

### AI用運用ガイド（MCP）
AIが安定して呼び出せるよう、以下のルールを推奨します。

#### 1) ドメイン用語の簡易辞書
- `kyou`: 記録の親データ（一覧検索の主対象）
- `kmemo`: テキストメモ
- `kc`: 数値記録
- `urlog`: URL/ブックマーク記録
- `nlog`: 支出/収入記録
- `timeis`: 打刻（開始/終了）
- `mi`: タスク記録
- `lantana`: 気分値記録
- `rekyou`: リポスト情報
- `idf_kyou`: ファイル関連記録

#### 2) ツール選択フロー（推奨）
1. まず `gkill.get_kyous` で候補一覧を取得  
2. 必要に応じて `id` を使い、`gkill.get_kyou` や各種 `get_*` で詳細履歴を取得  
3. 関連情報が必要なら `target_id` を使い、`gkill.get_tags_by_target_id` / `gkill.get_texts_by_target_id` を追加取得  
4. 通知履歴は `gkill.get_notification_histories_by_notification_id`、地図系は `gkill.get_gps_log` を使う

#### 3) ID受け渡しルール
- `id`: 個別データの履歴取得に使うキー（例: `get_kyou`, `get_kmemo`）
- `target_id`: タグ/テキストのぶら下がり取得に使うキー（例: `get_tags_by_target_id`）
- 推奨: 一覧系の結果から `id` を取り出し、次ツールへ引き渡す

#### 4) 日時フォーマット
- 日時は ISO-8601 を推奨  
- 例: `2026-02-25T10:30:00+09:00`
- `gkill.get_gps_log` の `start_date` / `end_date` は必須

#### 5) `gkill.get_kyous` の実用クエリ例

最小（デフォルト条件）:
```json
{
  "query": {}
}
```

タグ絞り込み:
```json
{
  "query": {
    "use_tags": true,
    "tags": ["work", "meeting"],
    "tags_and": false
  }
}
```

期間絞り込み（カレンダ）:
```json
{
  "query": {
    "use_calendar": true,
    "calendar_start_date": "2026-02-01T00:00:00+09:00",
    "calendar_end_date": "2026-02-28T23:59:59+09:00"
  }
}
```

Mi抽出:
```json
{
  "query": {
    "for_mi": true,
    "mi_check_state": "uncheck",
    "mi_sort_type": "limit_time"
  }
}
```

#### 6) 代表的なレスポンス形
全ツール共通で、基本的に以下を含みます。
```json
{
  "messages": [],
  "errors": []
}
```

一覧系（例: `gkill.get_kyous`）:
```json
{
  "messages": [],
  "errors": [],
  "kyous": []
}
```

履歴系（例: `gkill.get_kyou`）:
```json
{
  "messages": [],
  "errors": [],
  "kyou_histories": []
}
```

補助系:
- `gkill.get_all_tag_names` -> `tag_names: string[]`
- `gkill.get_all_rep_names` -> `rep_names: string[]`
- `gkill.get_mi_board_list` -> `mi_boards` 相当の配列
- `gkill.get_gps_log` -> `gps_logs: []`

#### 7) エラー時の推奨リトライ方針
- 認証系（例: `ERR000013`, `ERR000002`, `ERR000238`）:
  - セッション再取得（再ログイン）後に同一リクエストを1回再試行
- 入力不正系:
  - 引数を見直して再実行（特に `id`, `target_id`, 日時）
- データなし:
  - エラーではなく空配列として扱う

#### 8) 大量データ取得時の運用
- 一度に広い期間を投げず、期間を分割して段階取得する
- まず `get_kyous` で候補を絞り、その後個別取得へ進む
- 同一条件の連打を避け、必要最小限のクエリにする

