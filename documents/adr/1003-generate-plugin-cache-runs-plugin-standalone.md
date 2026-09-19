# ADR-1003: generate_plugin_cache はプラグインを単独モードで直接起動して同期構築する。稼働中サーバにも stdio プロトコルにも足さない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-16 |
| Sources | `.claude/skills/gkill-cli-ops/SKILL.md`「CLI サブコマンドと運用の約束」 / `src/server/gkill/main/common/generate_plugin_cache.go` / `src/server/gkill/plugin/sdk/sdk.go` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/main/common/generate_plugin_cache.go` |

## Context

**プラグインのキャッシュを「同期の直後に、完了まで待って」作る手段が無かった。**

同梱プラグイン（fitbit / codex / chatgpt / claudeai / claudecode / archived_git_commit_log / google_locationhistory）の
SQLite キャッシュ（`caches/plugin_cache/{user}/{plugin}/cache.db`）は、プラグインプロセスが stdio で最初に呼ばれたあと
常駐ビルダが非同期に作る（[ADR-0305](0305-plugin-background-builder-wal.md)）。新しいデータ
（Google Takeout の zip・ChatGPT のエクスポート等）が同期スクリプトで降りてきても、誰かが画面を開いて検索するまで
取り込みは始まらず、`update_cache` が作るサーバ側の型別索引（fitbit の KC 等）も古い cache.db を読む。
fitbit の初回構築は実測 155 秒なので、最初に開いた画面は数分のあいだ結果が痩せる。

サムネイル（`generate_thumb_cache`）と互換動画（`generate_video_cache`）には「同期の後に CLI で作っておく」
道具があり、運用スクリプトはそれを `update_cache` と並べて回している。
プラグインのキャッシュにも同列の道具が要る。

## Decision

`gkill_server generate_plugin_cache <plugin_name|all> <user_id...>` は、**プラグインバイナリを `--gkill-build-cache` 付きで
直接起動し、終わるまで待つ**。SDK は `Handler.BuildCache` を足し、このフラグを見たら stdio ループに入らず
それを同期で1回呼んで終了する（結果行 `built` / `no_cache` を stdout に1行、診断は stderr、失敗は exit 1）。
稼働中サーバには頼らず、HTTP API も stdio プロトコルのコマンドも足さない。

成否は子プロセスの stdout 全体が結果行に完全一致するときだけ成功とし、exit 0 で stdout が空・exit 2 は
「`--gkill-build-cache` を知らない旧バイナリ」として失敗に数える。利用者の実在確認はプラグインの走査より前に置く。
プラグインは逐次に起動し、タイムアウトは設けず、1本失敗しても残りを続けて `errors.Join` で返す。

## Rejected alternatives

- **稼働中サーバへの HTTP API を足し、サーバのプラグインプロセスに作らせる（`update_cache` と同じ型）** —
  書き手が1本で済むのは魅力だが、サーバが止まっていると使えない。プラグインの stdio ハンドラは
  「数十ミリ秒で返れ」（IsAlive 5 秒で kill、順番待ち 10 秒で `ErrPluginBusy`）なので、同期の `build_cache`
  コマンドは必ず殺され、非同期の kick + `get_build_status` のポーリングにするしかない。すると
  プロトコルに2コマンド、SDK に2フィールド、7プラグインに状態応答（locationhistory は `build_state` すら持たない）、
  HTTP ルート（ルート表・golden・TS の対・http_status）の追随が要る。`--gkill-print-manifest` と同じ
  「stdio ループに入らない起動」の型が既にあり、そちらは SDK 1箇所と7プラグイン各1フィールドで済む。
- **stdio プロトコルに `build_cache` コマンドを足して常駐プロセスに同期で作らせる** — 上と同じ理由で
  ハンドラ期限に殺される。期限をこのコマンドだけ延ばすと、構築中はスロットが塞がって同じプラグインの
  `find_kyous` が `ErrPluginBusy` になり、常駐ビルダが「構築中も現キャッシュを即答する」ために
  存在している意味が消える。
- **常駐ビルダを kick して `cache_meta.build_state` をポーリングする** — `build_state` の語彙も有無も
  プラグインごとに違い（fitbit は `error` あり、chatgpt は `source_signature` 一致で即 `idle`、
  locationhistory は `last_scan_unix` のみ）、判定を gkill 側に持つとプラグインを足すたびに壊れる。
- **gkill 本体が cache.db を直接作る** — スキーマはプラグインごとに別物で、SDK を使わない
  第三者プラグイン（`gkill_plugin_uguisu`）には手が届かない。取り込みの知識をプラグインの外へ出さない。
- **プラグインを並列に起動する** — stderr の進捗が混ざって読めなくなり、fitbit の並列パーサ（`ScanWorkers`）が
  数 GB の Takeout zip を同時に開く。実測でも fitbit の 155 秒が支配的で、他は数秒〜数十秒なので並列化の
  利得は小さい。
- **子プロセスにタイムアウトを張る** — fitbit の初回は実測 155 秒だが、環境（Termux / HDD）で数倍ぶれる。
  切ると「途中まで取り込んだ状態で exit 1」になり、完了を待つという目的に反する。
  Ctrl+C（ctx のキャンセル）で殺せれば十分。
- **exit 0 で結果行が無いバイナリを成功扱いにする** — SDK を使わない独自実装や旧 SDK のバイナリは
  フラグを無視して stdio ループに入り、stdin を繋がない CLI からは EOF で即 exit 0 する。これを成功に
  数えると「キャッシュができていないのに緑」になる。エラー1行の煩さより、静かに緑になる方が高くつく。
- **`Handler.BuildCache` が nil のプラグインを失敗扱いにする** — 雛形の `gkill_example` や
  キャッシュを持たないプラグインが `all` 指定のたびに赤くなり、本当の失敗が埋もれる。
  「キャッシュを持たない」は `no_cache` として区別して成功に数える。
- **SDK の単独モードで `signal.NotifyContext` を張る** — 張ると SIGINT の既定動作（即終了）が抑止され、
  ctx を見ない構築関数（fitbit / codex / chatgpt 系の `build`）が Ctrl+C の後も走り続ける。
  ctx は `newCtx(userID)`（Background 派生）だけにする。
- **利用者の実在確認を省く** — `PluginManager.DiscoverPlugins` は `plugins/{user}/` を MkdirAll するので、
  user_id を打ち間違えるたびにディレクトリが増える。account.db を引いてから走査する。

## Consequences

- **サーバ稼働中に同じプラグインの常駐ビルダが走っていれば、同じ WAL の cache.db に2プロセスが書く。**
  各プラグインは `busy_timeout(6000)` の自前 DSN と短いバッチ tx で開くので破損はしないが、相手の tx が
  6 秒を超えると `database is locked` で、衝突した側はその周回だけ失敗する（CLI なら exit 1 → 再実行で回復。
  常駐側は次の kick か 5 分ティックでやり直す）。運用スクリプトは新しいデータを置いた直後にまとめて流すので、
  常駐ビルダと重なるのはその瞬間に画面を開いている場合だけ。
- **サーバ側の型別索引は次の `update_cache`（か TTL 5 分の自動再構築）まで新しい cache.db を読まない。**
  同期スクリプトでは `generate_plugin_cache all <user>` を `update_cache` の**前**に置く。
- **SDK を使わないプラグインは自前で `--gkill-build-cache` に結果行を返す必要がある。** 返さないと
  `all` に含まれるたびに「結果行が無い」エラー行と exit 1 になる（静かではないので意図どおり）。
  `gkill_plugin_uguisu` は同日に対応し、キャッシュを持たないので `no_cache` を1行返す。
  対応前のバイナリが配置されている間はその1本だけ失敗行が出る。
- **旧 SDK でビルドしたプラグインバイナリは exit 2 で失敗に数える。** 配布で7本を同時に入れ替えるまで、運用スクリプトの新しい行は失敗を印字する
  （exit code は見ていないので同期は止まらない）。
- `BuildCache` の配線が欠けたプラグインは `all` でも `no_cache` として通り、キャッシュが作られないのに
  エラーが出ない。同梱7本は `plugin/sdk/build_cache_test.go` のソース走査が固定する。第三者プラグインは守れない。
- `PluginRepository` に `GetPluginDir()`、`reps` に `PluginExecutablePath` を足した。CLI で `.exe` の付与を
  再実装しないための共有点で、常駐起動（`ensureStarted`）も同じ関数を通る。

## Evidence

- fitbit の初回構築は実データ（271MB zip）で約 155 秒、差分は 1 秒未満（[ADR-0305](0305-plugin-background-builder-wal.md) の実測）。
  archived_git_commit_log は 88 zip / 3,447 コミットで約 60 秒。他は数秒。
- 旧 SDK のバイナリに `--gkill-build-cache` を渡すと Go の `flag`（`ExitOnError`）が usage を stderr に出して exit 2 する。
  stdin を繋がずに起動した SDK バイナリは `runLoop` が `scanner.Scan()` の EOF で false を返し、`Run` は何も出さず exit 0 する
  （`plugin/sdk/sdk_test.go` の `TestRunLoop_StdinCloseReturnsFalse`）。
- 並列化の利得は未計測 — fitbit 1本が支配的なので、並列にしても壁時計は fitbit の所要時間に張り付く。

## Related tests

- `src/server/gkill/main/common/generate_plugin_cache_test.go`
- `src/server/gkill/main/common/common_test.go`
- `src/server/gkill/plugin/sdk/build_cache_test.go`
- `src/server/gkill/dao/reps/plugin_repository_impl_test.go`
