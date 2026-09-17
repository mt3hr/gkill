# plugin/sdk テスト仕様

## 概要

プラグイン作者向け Go SDK のテスト。**56テスト（6ファイル）**。

`sdk.Run()` の stdin/stdout ループ、`--gkill-build-cache` の単独モード、`sdk.EnsureConfig()` の `config.json` 自動生成、
`sdk.OpenSources()` の ZIP 走査、`sdk.CacheDBPath()` のキャッシュDB配置、`sdk.Query.MatchText()` のワード判定を検証する。
SDK は gkill 本体と別プロセスで動くプラグイン側のライブラリなので、ここでの回帰は
全プラグイン（archived_git_commit_log / chatgpt / claudeai / claudecode / codex / fitbit / locationhistory / example）に同時に波及する。

## テストファイル

| ファイル | テスト数 | 対象 |
|---------|---------|------|
| `sdk_test.go` | 22 | `Run()` のメッセージループ（`TestRunLoop_*`）と型別データの往復 |
| `build_cache_test.go` | 5 | `--gkill-build-cache` の単独モード（`TestRunBuildCache_*`）と、同梱プラグインが `Handler.BuildCache` を配線していることのソース走査（`TestBundledPluginsWireBuildCache`） |
| `config_test.go` | 4 | `EnsureConfig()`（`TestEnsureConfig_*`） |
| `source_test.go` | 18 | `OpenSources()` の ZIP 走査（`TestOpenSources_*` ほか） |
| `cache_path_test.go` | 5 | `CacheDBPath()` / `IsSafePathElement()`（`TestCacheDBPath_*`） |
| `match_words_test.go` | 2 | `Query.MatchText()` / `Query.Matcher()`（`TestQueryMatchText` / `TestQueryMatcherDoesNotMutateQuery`）。gkill 本体と同じ規則（大小無視の部分一致、ID は前方一致、除外語は ID を見ない、空語は無視）で、元の Query を書き換えないこと |

## sdk_test.go — stdio ループ

`Run()` は「1行 = 1 JSON メッセージ」のループ本体。プラグインが gkill 本体と話す唯一の経路なので、
コマンド分岐・未実装時のフォールバック・異常入力での継続を個別に固定している。

| テスト | 検証内容 |
|-------|---------|
| `TestRunLoop_Ping` | `ping` に応答する |
| `TestRunLoop_GetRepName` | `get_rep_name` がマニフェストの `rep_name` を返す。`Handler.RepNames` 未設定なら `rep_names` は出ない |
| `TestRunLoop_GetRepName_OmitsRepNamesWhenNotImplemented` | 未実装のプラグインの応答に `rep_names` 欄そのものが無い（欄があると gkill は「複数 rep 名に対応」と読む） |
| `TestRunLoop_GetRepName_ReturnsRepNames` | `Handler.RepNames` の名前がそのまま `rep_names` に載り、`rep_name` と併存する |
| `TestRunLoop_GetRepName_EmptyRepNamesIsEmptyArrayNotOmitted` | 実装済みで0個（nil / 空スライス）なら `"rep_names": []` を出す。欄を落とすと manifest 名にフォールバックされ「未対応」と読まれる |
| `TestRunLoop_GetRepName_ErrorGoesToErrors` | `RepNames` のエラーは `errors` に載り、名前は返さない |
| `TestRunLoop_FindKyous` | `find_kyous` が Handler の結果を返す |
| `TestRunLoop_FindKyousNotImplemented` | `FindKyous` 未設定時にエラー応答になる |
| `TestRunLoop_FindKyousError` | Handler がエラーを返したとき、エラーとして中継される |
| `TestRunLoop_GetKyouFallsBackToFindKyous` | `GetKyou` 未実装なら `FindKyous` で代替する |
| `TestRunLoop_GetContentHTMLDefault` | `GetContentHTML` 未実装時の既定 HTML |
| `TestRunLoop_PostConfigDefaultSavesForm` | `PostConfig` 未実装時、送られたフォームを既定処理で保存する |
| `TestRunLoop_UnknownCommand` | 未知のコマンドでエラー応答を返す |
| `TestRunLoop_InvalidJSONContinues` | 壊れた JSON を受けてもループを止めない |
| `TestRunLoop_CloseStopsLoop` | `close` でループを終了する |
| `TestRunLoop_StdinCloseReturnsFalse` | stdin が閉じたらループを抜ける |
| `TestRunLoop_PassesUserIDToHandler` | `user_id` が Handler へ渡る |
| `TestRunLoop_PassesConfigToHandler` | `config.json` の内容が Handler へ渡る |
| `TestRunLoop_GetGPSLogs` | `get_gps_logs` が Handler の結果を返す |
| `TestRunLoop_GetGPSLogsNotImplemented` | `GetGPSLogs` 未設定時にエラー応答になる |
| `TestRunLoop_GetGPSLogsNilQuery` | クエリが nil でも落ちない |
| `TestKyouTypedDataRoundTrip` | 型別データ（`PluginTypedData`）が JSON を往復しても壊れないこと |

## build_cache_test.go — 単独モード

`--gkill-build-cache` 付きで起動されたとき `Run()` は stdio ループに入らず `runBuildCache` を1回だけ呼ぶ。
gkill 側（`gkill_server generate_plugin_cache`）は stdout の結果行だけで成否を判定するので、
「何を stdout に書くか」を固定している。

| テスト | 検証内容 |
|-------|---------|
| `TestRunBuildCache_NilHandlerPrintsNoCache` | `BuildCache` が nil なら `no_cache` を出して成功扱い（`all` 指定でキャッシュ無しプラグインを赤くしない） |
| `TestRunBuildCache_SuccessPrintsBuilt` | 1回だけ呼ばれ、`EnsureConfig` の結果と `newCtx` の user id がそのまま渡り、`built` を出す |
| `TestRunBuildCache_ErrorGoesToStderrAndReturnsFalse` | 失敗は stderr（`ERROR: build cache: …`）に出て false。stdout には何も書かない（書くと gkill が成功と読む） |
| `TestRunBuildCache_WritesOnlyResultLineToStdout` | `LogWarn` を呼んでも stdout は結果行1行だけ（gkill 側は完全一致で判定する） |
| `TestBundledPluginsWireBuildCache` | `src/plugins/gkill_plugin_*/main.go` を走査し、全部 `BuildCache:` を配線していること。欠けると `generate_plugin_cache all` でそのプラグインだけ `no_cache` になりエラーも出ない |

## config_test.go — EnsureConfig

初回起動時に `manifest.json` と同じフォルダへ `config.json` を既定値で自動生成する処理。
**既存ファイルを上書きしないこと**がこの機能の要件なので、そこを固定している。

| テスト | 検証内容 |
|-------|---------|
| `TestEnsureConfig_CreatesWhenMissing` | 無ければ既定値で作る |
| `TestEnsureConfig_KeepsExisting` | 既にあれば触らない（ユーザの設定を壊さない） |
| `TestEnsureConfig_NoDefaults` | `DefaultConfig` を持たない Handler では何も作らない |
| `TestEnsureConfig_EmptyPluginDir` | プラグインディレクトリが空文字なら何もしない |

## source_test.go — ZIP 走査

Google Takeout の ZIP を展開せずに読むための走査。fitbit / 位置情報 / archived_git_commit_log / chatgpt / claudeai の5本が使う。
テスト用の ZIP は `archive/zip` で実行時に組み立てる（バイナリをコミットしない）。

| テスト | 検証内容 |
|-------|---------|
| `TestOpenSources_ReadsZipEntries` | ZIP 内のエントリを列挙し、中身を読める。日本語のエントリ名も解決する |
| `TestOpenSources_IgnoresLooseFiles` | 展開済みのファイルは読まず、`extracted_folder` として報告する |
| `TestOpenSources_AcceptFiltersEntries` | `accept` で要らないエントリを取りこぼさず落とす |
| `TestOpenSources_SplitPartsAreOneExport` | 分割された `-001` `-002` が1つの取り込み世代にまとまる |
| `TestOpenSources_ExportsAreSortedNewestFirst` | 世代が新しい順に並ぶ（同着はIDの降順） |
| `TestOpenSources_MixedExportsInOneDirIsReported` | 1フォルダに時期の違う書き出しが混ざっていたら報告する |
| `TestOpenSources_SpannedArchiveIsReportedNotFatal` | `.z01` 形式の分割アーカイブを検出して報告し、他のZIPは読み続ける |
| `TestOpenSources_NestedZipIsReported` | ZIP の中の ZIP には潜らず報告する |
| `TestOpenSources_BrokenArchiveIsReportedNotFatal` | 壊れたZIP 1本で走査全体を止めない |
| `TestExportIDOf_SplitPartsShareOneExport` | 分割パートの世代IDが一致する |
| `TestExportIDOf_DifferentExportsInOneDirAreDistinct` | **同じフォルダの別の書き出しは別の世代になる**（フォルダだけを単位にすると合算されて値が2倍になる） |
| `TestTakeoutStamp` | ZIP名から書き出し時刻を取り出す。Takeout でない名前は空 |
| `TestSourceEntry_ReadHead` | 先頭 n バイトだけを伸長する（形式判定用） |
| `TestSourceEntry_ConcurrentOpen` | 同じ ZIP のエントリを並行に開いても壊れない |
| `TestReadAllLimited` | 上限を超えるエントリはエラーにする |
| `TestSplitZipEntryPath` | `<ZIP>!/<エントリ>` を分解する |
| `TestExpandSourcePatterns` | ワイルドカード・`~`・環境変数を展開する |
| `TestParseSourcePatterns` | 配列でも改行区切りでも読める。空なら既定値 |

## 実行方法

```bash
cd src/server && go test ./gkill/plugin/sdk/...
```

`npm run test_server`（`cd src/server && go test ./...`）にも含まれる。

## 関連ドキュメント

- [../README.md](../README.md) — SDK の使い方
- [../../../../plugins/ABOUT_TEST.md](../../../../plugins/ABOUT_TEST.md) — SDK を使う各プラグインのテスト（独立モジュールのため `npm test` の対象外）
- [../../../../../documents/reverse/plugin-system.md](../../../../../documents/reverse/plugin-system.md) — プラグインシステム仕様
