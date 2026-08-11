# gkill_server_api テスト仕様

## 概要

`gkill/api/gkill_server_api/` パッケージのテスト。`gkill/api/` から移動された HTTP API ハンドラ層（handle_*.go 実装91ファイル（+ テスト5ファイル））に対する統合テストを含む。

## テストフレームワーク

Go `testing` パッケージ

## テストファイル一覧

| ファイル | テスト内容 |
|---------|-----------|
| `gkill_server_api_test.go` | API ハンドラ統合テスト（全エンドポイント） |
| `handle_get_shared_kyous_test.go` | 共有ページ（認証なしの公開エンドポイント）の共有スコープ検証。共有条件に一致するKyouが0件のとき、実体データまで含めて1件も返さないこと（キーワード以外の条件で0件になった場合も含む。ID絞り込みを外すと実体取得は条件を解釈しないため全件返ってしまう） |
| `get_kyous_regressions_test.go` | 記録取得の回帰。プラグイン検索失敗が警告として返ること（エラーにしない）、実行中の指定が無いときは実行中で絞らないこと、打刻タグでの絞り込みが記録側のタグ指定なしでも効くこと |
| `gzip_middleware_test.go` | API応答のgzip圧縮。`Accept-Encoding` を見て圧縮すること、対象外パスや非対応クライアントには圧縮をかけないこと、ストリーミング応答を壊さないこと |
| `gkill_server_api_rate_limit_test.go` | ログインレート制限テスト（IP別カウント、ウィンドウ期限、IP抽出） |
| `handle_get_idf_file_path_test.go` | IDF ファイル絶対パス解決ハンドラ（localhost 限定応答、ERR000389、存在確認） |
| `handle_get_idf_kyou_by_relative_path_test.go` | Markdown 相対リンクの IDFKyou 解決ハンドラ（同一 Rep 内解決、パストラバーサル防止） |
| `handle_zip_cache_file_serve_test.go` | `/zip_cache/` の利用者分離（他人のキャッシュを読めないこと、ユーザーごとに分かれていない旧レイアウトを配信しないこと、`../` / `..%2F` で抜けられないこと、セッション無し・不正セッションの拒否）と、利用者ファイル配信のセキュリティヘッダ（後述）、ZIP展開物一覧の種別フラグ（`TestBuildZipEntriesMediaFlags`、後述） |
| `handle_reset_password_test.go` | パスワードリセットのセッション検証（後述） |
| `handle_set_new_password_test.go` | リセットトークンの期限切れを汎用の失敗と区別して返すこと（後述） |
| `handle_update_account_status_test.go` | ログイン中のアカウント自身を無効化させないこと（後述） |
| `get_device_cache_test.go` | デバイス名キャッシュ（`sync.Once` による `GetAllServerConfigs` 呼び出し削減）の検証 |
| `plugin_content_html_cache_test.go` | プラグイン本文HTMLキャッシュ（TTL・件数上限・singleflight による同時要求の集約）の検証 |
| `utils_ssrf_test.go` | `httpGetBase64Data` の SSRF 対策（スキーム制限、内部アドレス拒否、サイズ上限、タイムアウト） |

## テスト内容

### `gkill_server_api_test.go`（統合テスト）

- **データ型別 CRUD**: 全12データ型（Kmemo, Mi, TimeIs, URLog, Nlog, Lantana, KC, Tag, Text, Notification, ReKyou, MiReKyou）の Add / Update / Delete / Get
- **MiReKyou（既存Kyouのタスク化）**: Add / Get / Update に加えて、
  ターゲットのKyouを論理削除するとMi画面の検索から落ちること
  (`TestHandleGetKyous_MiReKyouResolvesTarget`)、
  MiReKyouだけのボードもボード一覧に出ること
  (`TestHandleGetMiBoardList_IncludesMiReKyouOnlyBoard`)
- **存在しないIDへの更新**: 全13型とも `NotFound*Error` を返す
  (`TestHandleUpdate*_Nonexistent_ReturnsError`)。
  以前は Mi / MiReKyou 以外の11型で、ユースケース側の存在チェックが
  書き込みの後ろに置かれていて到達できず、更新のつもりが新規レコードを
  作って成功を返していた。当時は挙動をそのまま固定した
  `TestHandleUpdate*_Nonexistent_Succeeds` 群があったが、
  チェックを書き込みの前へ移したうえでエラー期待へ揃えている
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

加えて、共有情報そのものの **所有者の扱い** を3本で固定している。
閲覧側は保存されたレコードの `user_id` をそのまま使って対象ユーザーのリポジトリを開くため、
「誰の共有として保存されるか」がそのままアクセス範囲になる。
作成・更新・削除の3経路すべてで、セッションの持ち主以外の共有に触れないことを確認する
（`addSecondAccount` で admin とは別の一般アカウントを作って検証する）。

- **作成時の所有者**: リクエスト本文の `user_id` に他人を指定しても、保存される所有者は
  セッション側になり、他人のライフログが共有ページに出ないこと
  (`TestHandleAddShareKyouListInfo_IgnoresRequestUserID`)
- **他人の共有の更新拒否**: 共有IDを知っているだけの別ユーザーが検索条件を広げられないこと
  (`TestHandleUpdateShareKyouListInfo_OtherUsersShareIsRejected`)
- **他人の共有の削除拒否**: 同じく別ユーザーが共有を取り消せないこと
  (`TestHandleDeleteShareKyouListInfos_OtherUsersShareIsRejected`)

更新・削除は「存在しない」と「所有者が違う」を同じエラーコードで返す。
区別すると共有IDの存在有無を問い合わせるオラクルになるため、意図的に揃えている。

### 利用者ファイル配信のセキュリティヘッダ

`/files/` と `/zip_cache/` は、取り込んだファイルやZIPの展開物を
拡張子から決めた Content-Type で同一オリジンから配信する。
展開時に拡張子の許可リストは無いので、受け取った `.cbz` にHTMLやSVGが
入っていればブラウザはそれをHTMLとして解釈する。
セッションクッキーはクライアント側のJSが `document.cookie` で書いており
`HttpOnly` を付けられないため、同一オリジンでスクリプトが動くと読み出せてしまう。

`withUserContentSecurityHeaders`（`utils.go`）がルート側で
`X-Content-Type-Options: nosniff` と `Content-Security-Policy: sandbox` を付ける。
`sandbox` に `allow-scripts` を付けていないので中のスクリプトは実行されない。
`sandbox` はドキュメントとして読み込まれたときにだけ効くため、
`<img>` や `<video>` のサブリソースとしての表示には影響しない。

例外として `.pdf`（大小無視）には `sandbox` を付けない。付けると opaque origin に
なり Chrome の内蔵PDFビューワが無効化されて、新しいタブでの表示がダウンロードに
落ちるため。`nosniff` は `.pdf` にも付き Content-Type は `application/pdf` に
固定されるので、HTMLを `.pdf` にリネームして持ち込んでもHTMLとしては解釈されない。

`TestZipCacheFileServeSetsSecurityHeaders` がこれを固定する
（HTMLは `sandbox` + `nosniff`、PDFは `sandbox` 無し + `nosniff` +
`application/pdf`、拡張子の大小無視、の3サブテスト）。
**ルート登録を `serve.go` と同じ形（ラッパー経由）にしないと素通りする**ので注意。
また `index.html` という名前は `http.FileServer` がディレクトリへ
リダイレクトしてしまうため、テストでは別名を使っている。

`TestBuildZipEntriesMediaFlags` は、ZIP展開物一覧の種別フラグ
（`is_image` / `is_text` / `is_video` / `is_audio` / `is_pdf`）が拡張子どおりに
立つことをテーブルテストで固定する。`.ts` が TypeScript（is_text）と
MPEG-TS（is_video）の両方に該当する既知の重複もここで文書化している
（クライアントはテンプレートの分岐順で is_text を優先する）。

### `handle_reset_password_test.go`（パスワードリセットのセッション検証）

`/api/reset_password` は `wrapNoAuth` で登録されており、認証をハンドラ自身が行う。
管理者権限があれば任意アカウントのリセットトークンを発行できるエンドポイントなので、
セッション検証が緩いと失効済みセッションや別用途のセッションから全アカウントを奪える。

以前は `LoginSessionDAO.GetLoginSession` を直接呼んでおり、
「セッションIDがDBに存在するか」しか見ていなかった。他エンドポイントが
`getAccountFromSessionID` で行っている検証をすべて素通りしていたので、
同じ関数を通す形に統一したうえで以下を固定している。

- **有効期限切れのセッション**: 期限を過ぎたセッションでリセットできないこと
- **ブックマークレット用セッション**: `ApplicationName` が `urlog_bookmarklet` の
  セッションでリセットできないこと。この値はブックマークレットのURLのクエリ文字列に
  載るため、ブラウザ履歴やブックマーク同期から漏れうる
- **無効化済み管理者**: セッションは生きたままアカウントだけ `IsEnable = false` に
  したとき、リセットできないこと
- **通常のセッションは成功する**: 上記3件が「そもそも常に失敗する」だけでないことを担保する

### `handle_set_new_password_test.go`（リセットトークンの期限切れ）

リセットトークンは発行から72時間で切れる。期限切れであることが伝わらないと、
利用者にはリンクが壊れているようにしか見えず、管理者に再発行を頼めばよいことに
気づけない。一方で「期限切れである」と答えてよいのはトークンが一致したときだけで、
総当たりに対してトークンの存在をもらしてはいけない。

- **期限切れ**: 一致するトークンが期限切れなら `ERR000408` を返し、パスワードは設定しないこと
- **不一致**: トークンが一致しないときは期限切れを名乗らないこと（従来どおり `ERR000247`）
- **期限内**: 期限内のトークンならパスワードが設定され、使い終わったトークンが消えること
  （上の2件が「そもそも常に失敗する」だけでないことの担保）

### `handle_update_account_status_test.go`（自分自身の無効化）

管理者アカウントが1つしかない構成が普通なので、自分を無効化すると誰も管理画面に
入れなくなり、復帰手段がサーバと同じマシンでのCLI操作だけになる。
画面側でもチェックボックスを操作できなくしてあるが、APIを直接叩かれても弾く。

- **自分の無効化**: `ERR000409` で弾き、アカウントが有効なままであること
- **自分の有効化**: 禁止しているのが無効化だけで、有効化まで巻き込んでいないこと
- **存在しないユーザ**: `GetAccount` が「見つからない」をnilで返すため、
  nilチェックを落とすとnilポインタ参照になる経路がエラーとして返ること

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
