# gkill_server_api テスト仕様

## 概要

`gkill/api/gkill_server_api/` パッケージのテスト。`gkill/api/` から移動された HTTP API ハンドラ層（handle_*.go 実装89ファイル（+ テスト3ファイル））に対する統合テストを含む。

## テストフレームワーク

Go `testing` パッケージ

## テストファイル一覧

| ファイル | テスト内容 |
|---------|-----------|
| `gkill_server_api_test.go` | API ハンドラ統合テスト（全エンドポイント） |
| `handle_get_shared_kyous_test.go` | 共有ページ（認証なしの公開エンドポイント）の共有スコープ検証 |
| `gkill_server_api_rate_limit_test.go` | ログインレート制限テスト（IP別カウント、ウィンドウ期限、IP抽出） |
| `handle_get_idf_file_path_test.go` | IDF ファイル絶対パス解決ハンドラ（localhost 限定応答、ERR000389、存在確認） |
| `handle_get_idf_kyou_by_relative_path_test.go` | Markdown 相対リンクの IDFKyou 解決ハンドラ（同一 Rep 内解決、パストラバーサル防止） |
| `utils_ssrf_test.go` | `httpGetBase64Data` の SSRF 対策（スキーム制限、内部アドレス拒否、サイズ上限、タイムアウト） |

## テスト内容

### `gkill_server_api_test.go`（統合テスト）

- **データ型別 CRUD**: 全12データ型（Kmemo, Mi, TimeIs, URLog, Nlog, Lantana, KC, Tag, Text, Notification, ReKyou, MiReKyou）の Add / Update / Delete / Get
- **MiReKyou（既存Kyouのタスク化）**: Add / Get / Update に加えて、
  ターゲットのKyouを論理削除するとMi画面の検索から落ちること
  (`TestHandleGetKyous_MiReKyouResolvesTarget`)、
  MiReKyouだけのボードもボード一覧に出ること
  (`TestHandleGetMiBoardList_IncludesMiReKyouOnlyBoard`)。
  MiReKyouの更新だけは他11型と違い、存在しないIDに対して
  `NotFoundMiReKyouError` を返す設計になっている
  (`TestHandleUpdateMiReKyou_Nonexistent_ReturnsError` でその差分を固定)
- **通知の更新・削除**: `TestHandleUpdateNotification_ChangesContent` /
  `TestHandleUpdateNotification_MarksDeleted` は `--cache_in_memory` の
  **true / false 両方**で回す（`useCacheInMemory(t)`）。
  本番の既定は true でRepsがキャッシュ実装に差し替わるため、
  片方だけ通る不具合が実際にあった（キャッシュ版 `GetNotification` の
  バインドずれで `ERR000280`、および集約マップのキー誤りで更新前の版が残る）。
  データ層を触るハンドラテストは両モードで回すこと。
- **セッション管理**: ログイン、セッション検証、アカウント管理、セッション有効期限切れ検出（ERR000373）
- **認証ミドルウェア**: `TestAuthMiddleware_RejectsInvalidSession` が
  セッションを要求する全51エンドポイント × 空セッション / 不正セッション の
  102サブテストを1つのフィクスチャで回す。
  内訳は `wrapAuth` / `wrapAuthRepos` のミドルウェア経由が48、
  `wrapNoAuth` だがハンドラ自身が `getAccountFromSessionID` する
  3エンドポイント（UploadFiles / UploadGPSLogFiles / BrowseZipContents）。
  いずれも `AccountSessionNotFoundError` を返すことまで確認する。

  > 以前はエンドポイントごとに `Test*_InvalidSession` / `Test*_RequiresSession` を
  > 48本持っていたが、1本ごとにサーバとDAO一式をグローバルmutex下で起動していたため
  > このパッケージのテスト時間の約1/3（実測40秒弱）を占めていた。
  > 検証対象は全て同じ認証経路なので、フィクスチャを1回だけ作る形に集約している。
  > エンドポイント単位の粒度は `t.Run` のサブテスト名で維持している。
- **トランザクション**: 複数操作の一括処理
- **GetKyous 複合クエリ**: ワード検索、タグフィルタ、リポジトリフィルタ、カレンダー範囲、Mi チェック状態、複合条件
- **特殊エンドポイント**: GetKyousMCP, SubmitKFTLText, UpdateCache, BrowseZipContents
- **ZIPブラウズ**: BrowseZipContents エンドポイントのセキュリティテスト（パストラバーサル防止、Shift_JISエントリ名デコード、アトミック展開、zip_cache ファイルサーブ）
- **名前リスト**: ボード名一覧、タグ名一覧、リポジトリ名一覧
- **履歴**: タグ履歴、テキスト履歴、通知履歴
- **設定**: サーバ設定読み書き、アプリケーション設定更新、ユーザリポジトリ更新

### `handle_get_shared_kyous_test.go`（共有ページ）

`/api/get_shared_kyous` は `wrapNoAuth` で登録された **セッション不要の公開エンドポイント**。
共有IDさえ知っていれば誰でも叩けるので、「共有対象として保存した検索条件に一致する
Kyou だけが返る」ことが唯一の防壁になる。ここが崩れると共有していないライフログが
第三者に見えるため、漏洩側を重点的に確認している。

- **共有スコープ**: 共有条件に一致するKyouだけが返り、一致しないKyouが混ざらないこと
- **未知の共有ID**: 空文字 / でたらめな文字列 / 未登録UUID でエラーになり、Kyouが0件であること
- **共有の取り消し**: 共有情報を削除したあと、同じ共有IDで取得できなくなること

### `gkill_server_api_rate_limit_test.go`（レート制限テスト）

- **レート制限**: 10回/15分のログイン試行制限、IP別独立カウント、ウィンドウ期限経過後のリセット
- **IP抽出**: IPv4/IPv6アドレスからのポート番号除去

## 実行方法

```bash
cd src/server && go test ./gkill/api/gkill_server_api/...
```

または:

```bash
npm run test_server
```
