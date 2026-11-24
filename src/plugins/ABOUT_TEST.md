# テスト仕様 — src/plugins/

## 概要

`src/plugins/` 配下のプラグインのうち、examples/gkill_example には独自のテストファイルは存在しない。

プラグイン機能のテストは以下でカバーされている：

| テスト場所 | テスト内容 |
|---|---|
| `src/server/gkill/api/gkill_server_api/` | プラグインのロード・subprocess起動・stdio通信・クラッシュ後自動再起動の統合テスト |
| `src/server/gkill/dao/reps/` | `plugin_repository_impl.go` のリポジトリ層テスト |
| `src/plugins/gkill_plugin_claudecode/loader_test.go` | ターン分割・ファイル種別判定・サブエージェント紐付け・ツール要約のユニットテスト（`testdata/` の合成トランスクリプトを使用）。CLI 形式（`promptSource` が `typed` / `queued`）とデスクトップアプリ / Agent SDK 形式（`sdk` + `origin.kind=human`）の両方で人間の発言が分かれること、`origin` 無しの `sdk`・`system`・AskUserQuestion の回答が応答の注記に畳まれること、タイトルが `ai-title` 優先で `custom-title` にフォールバックすること、assistant レコードの `message.model` が返事の Kyou と検索テキストに載ること（`TestBuildMessagesCarriesModel`）も見る |
| `src/plugins/gkill_plugin_claudecode/render_test.go` | 詳細 HTML。ラベルが「あなた」/「Claude Code」、最後のテキストが本文で途中のテキストとツール実行が `class="process"` の中、チップが project / branch / model / 所要時間の順に空白区切り、エスケープ（`TestRenderMessageHTMLEscapesAndFolds`）。本文と経過の分け方 `splitReply`（最後のテキスト1つが本文、テキストが無ければ本文は空） |
| `src/plugins/gkill_plugin_claudecode/cache_test.go` | トランスクリプトからのキャッシュ構築。空からの構築、バッチ commit の永続性（途中のバッチが残る）、読み取りが構築のロックを待たないこと（構築中の並行読み取りを含む）、セッション単位の差分更新、消えたファイルの記録を消すこと、スキーマの版を上げたら作り直すこと、統計（`Stats`）が状態を返すこと |
| `src/plugins/gkill_plugin_codex/{reader,loader,fold,cache,render,config}_test.go` | 巨大行リーダ・`session_meta` の identity/environment 分離・IDE とアプリ内ブラウザの前置きの剥がし・0.151 以降の新レーン（`item_completed`）の読み取りと旧レーンとの重複防止・質問と回答・最終回答を本文にして経過を畳む描画（描画の共通部分は SDK の `chatview` に移り、`render_test.go` が見るのは Codex の写し方と本文 / 経過の分け方）・Kyou ID の安定性・差分再構築・構築中の並行読み取りのユニットテスト（`testdata/` の合成ロールアウトを使用。新レーンは `testdata/item_lane/`） |
| `src/plugins/gkill_plugin_antigravity/{scan,loader,cache,config,render,html,query_ids,find_kyous,builder}_test.go` | Antigravity のチャットログの取り込み（`testdata/` の合成の会話2件を使用）。読むのは `transcript_full.jsonl` だけで、省略版の `transcript.jsonl` と `chunks/` の写しを拾わないこと（拾うと会話が二重になる）、会話IDを生の置き場（`brain/<会話ID>/.system_generated/logs/`）と同期先（`<会話ID>/`）の両方で祖先フォルダ名から採れること、UUID の祖先が無い transcript を読まずに警告へ出すこと、同じ会話が2か所にあれば大きいほうだけを読むこと。人間の発言1件と応答1件に分かれること、`content` が空の `PLANNER_RESPONSE` ばかりの応答で「空でない最後の content」が本文になること、ツールの実行結果の `content` を保持しないこと（`error` だけ注記にする）、同じ本文の注記を1件に畳むこと、最初の `USER_INPUT` より前のステップを落とさないこと、`USER_INPUT` が無い会話でも0件にならないこと、追記しても既存の Kyou ID が動かないこと（`TestKyouIDIsStableAcrossAppend`）、`conversation_summaries.db` だけ変わったときにタイトルを差し替えて transcript を読み直さないこと、manifest と定数の一致（`TestManifestMatchesConstants`）、設定画面 HTML 描画。人間の発言から `<USER_REQUEST>` の中身だけを本文にすること（`TestSplitUserRequest` の表駆動と `TestBuildMessagesStripsUserRequestWrapper`）—— 現在時刻だけの `<ADDITIONAL_METADATA>` は捨て、`<USER_SETTINGS_CHANGE>` は折りたたみの前置きへ回して検索テキストに入れないこと（`TestSearchTextOfExcludesPreamble`）、**知らない形（タグ無し・閉じていない・ブロックの外に文字・中身が空・知らないブロック）は何も剥がさずそのまま本文にすること**、人間の吹き出しに包みの跡が出ないこと（`TestRenderMessageHTMLHumanShowsRequestOnly`） |
| `src/plugins/gkill_plugin_{chatgpt,claudeai}/render_test.go` | 詳細 HTML。会話タイトルの見出し、ラベルが「あなた」/「ChatGPT」・「Claude」、インラインコードが `<code>`、本文のエスケープ、head の印（`gkill_iframe_size` / `gkill_theme`）。ChatGPT はモデル名のチップが出ること・無いメッセージにはチップの枠が無いこと |
| `src/plugins/gkill_plugin_archived_git_commit_log/{cache,scan,find_kyous,config,render}_test.go` | zip の中の Git リポジトリの取り込み。testdata に `.git` は置けないので、テストのたびに go-git でリポジトリを作って `archive/zip` で固める（`testutil_test.go`）。native の git rep と同じ列（ID=ハッシュ・rep 名=ディレクトリ名・コミッタ日時とゾーン・author・行数）で入ること、同じリポジトリを2つの zip に入れても1件で zip を外すと他に無いコミットだけ消えること、コミット0件の `git init` 直後は0件で正常、`.git` の無い zip は素通り、1 zip に複数（入れ子・ルート直下）、packfile、指紋による増分、`.git` の上限超過は理由を残して飛ばす、構築中の並行読み取り |
| `src/plugins/gkill_plugin_vrchat/{parse,records}_test.go` | VRChat の出力ログの読み取りと記録の組み立て（`testdata/fixtures/` の合成ログを使用）。拾う行だけを出来事にし、時刻の無い継続行は拾わないこと、長すぎる行は読み捨てて数え CRLF の行も読めること、時刻付きの行が無いファイルは記録を作らないこと。正常終了の起動でワールドの滞在・URLog・同じ部屋の人・アバターの切り替え（Kmemo）・動画（URLog。続けて同じ URL は1件）がそろい終了の行で全部閉じること、落ちた起動は最後の時刻付きの行で開いている TimeIs を全部閉じること、ログが伸びても両方にある記録の ID が変わらないこと・ID が種類をまたいで重ならないこと（ADR-0317）・ID がファイル名や置き場所に依らないこと |
| `src/plugins/gkill_plugin_vrchat/{cache,config,find_kyous,html,main}_test.go` | 1から構築・同じ起動のログが2か所にあれば一番大きいファイルから1回ぶんだけ作ること・追記されたファイルだけを読み直す差分構築・消えたファイルの記録が消えること・スキーマ版の更新とログを読むゾーンの変更で作り直すこと（ゾーンは time.Local を呼ぶたびに読む。ADR-0319）・LIMIT と期間・構築中の並行読み取り・マッチしない指定の報告。`config.json` の既定と再読み込み・既定の指定へのフォールバック・設定画面の入力の分割・設定画面 HTML 描画・manifest と定数の一致。種類ごとに `data_type` と型別データ（TimeIs / ワールドと動画の URLog / Kmemo）が合っていて manifest の `data_type` で上書きしないこと（`TestKyouOfSetsTypedDataPerKind`）。`main_test.go` はログの時刻を UTC で読むようにしてから走らせる `TestMain` |
| `src/plugins/gkill_plugin_{antigravity,chatgpt,claudeai,claudecode,codex,archived_git_commit_log,gemini,vrchat}/builder_test.go` | `buildOnce`（常駐ビルダと単独モードの共通経路）が構築の失敗を `cache_meta` の `build_state=error` / `build_error` に残し、stderr に `ERROR:` 行を出してエラーを返すこと。成功時は `build_state` を触らない。fitbit は `runOnce`（`gkill_plugin_fitbit/builder_test.go`）で ERROR 行と完了 Info の出し分けを見る |
| `src/plugins/gkill_plugin_{chatgpt,claudeai}/loader_test.go` | 要素ごとに読む `decodeConversationArray` が、配列でない JSON・途中で切れた JSON・配列の後ろに続きがある JSON をエラーにすること（黙って0件や途中までにしない） |
| `src/plugins/gkill_plugin_{chatgpt,claudeai}/source_test.go` | 設定画面に出す走査の問題の絞り込み `relevantProblems`（ZIP の中の ZIP・混在・マッチしない指定は出さず、展開済みフォルダはプラグイン自身のフォルダなら出さず他は文言差し替え）と、取り込み元の署名 `sourceSignature` が列挙順に依存せず CRC32 / Size の変化で変わること |
| `src/plugins/gkill_plugin_{chatgpt,claudeai}/cache_test.go` | エクスポート ZIP の直読み。テストのたびに `archive/zip` で ZIP を組む（バイナリは commit しない）。ZIP を作り直しても中身が同じなら作り直さない（署名は Path:CRC32:Size で更新時刻を見ない。Claude.ai の実物は 1980 年固定）、展開済みの `conversations.json` を直置き・直接指定しても読まず既存キャッシュを残して `source_problems` に出る、会話ファイルを含まない ZIP（`projects-000.zip` 等）は素通り、同じ ZIP に分割形式と旧形式があれば分割形式だけ（優先はアーカイブ単位）、同じ会話 ID が2つの ZIP にあれば update_time が新しい版だけ、プラグインフォルダ自身の `manifest.json` / `config.json` は「展開済みフォルダ」と警告しない、バッチ commit の永続性、読み取りが `buildMu` を取らないこと、gen 掃除。ChatGPT は `message.metadata.model_slug` が `model` 列に載り、無い返事は会話の `default_model_slug` で補い、人間の発言には付けないこと（`TestChatGPTModelSlugIsCached`） |
| `src/plugins/gkill_plugin_gemini/{cache,loader,htmltext,source,render,html,manifest,find_kyous,find_kyous_ids,builder}_test.go` | Google Takeout の「マイ アクティビティ」→「Gemini アプリ」の取り込み（テストのたびに `archive/zip` で ZIP を組む。バイナリは commit しない）。**日本語アカウントの配置（`マイ アクティビティ/Gemini アプリ/マイアクティビティ.json`）と英語アカウントの配置（`My Activity/Gemini Apps/MyActivity.json`）の両方が読めること**（`TestGeminiReadsLocalizedAndEnglishLayouts`。名前で選ぶと日本語環境で黙って0件になる）、別製品のマイアクティビティと Takeout の `Gemini/` にある Gem / 予定アクションの空 HTML を警告も出さずに無視すること、添付の ZIP（ZIP の中の ZIP）で設定画面が埋まらないこと、HTML 形式のアクティビティは `source_problems` に出ること（黙って0件にしない）、プロンプトも返答も無い記録（「アシスタント機能を使用」）から Kyou を作らないこと、`safeHtmlItem` が複数あれば連結すること、会話の見出しが最も古いプロンプトになること、同じ往復が新旧2つの書き出しにあれば新しいほうだけを採り `mixed_exports` は警告に残すこと、署名（Path:CRC32:Size）が変われば作り直し mtime だけの違いでは作り直さないこと、バッチ commit の永続性、読み取りが `buildMu` を取らないこと、展開済みフォルダとプラグインフォルダ自身の扱い、同一時刻複数往復の ID 一意性、返答の関連時刻をプロンプトの1秒後に置き gkill 本体の並べ方（秒の降順 → ID）で往復ごとに「返答 → プロンプト」に並ぶこと（ID は元の時刻のまま。`TestGeminiReplyIsOrderedAfterPrompt`）、`GetMsgByID` のヒット/ミス、設定画面 HTML 描画。`title` の接頭辞の落とし方（コロン形式 / `Prompted ` / どちらでもない操作の記録）、会話 ID を `details[].url` から採ること、返答の HTML を markdown 風の平文へ写すこと（タグ網羅・実体参照・リンク・上限）、Kyou ID が (会話 ID, 時刻, 役割, 連番) から決定的に出ること（`TestKyouIDOfIsStableAndDistinct`）、manifest と定数の一致（`TestManifestMatchesCode`） |
| `src/plugins/gkill_plugin_fitbit/{loader,cache,data_source,metrics,timeparse}_test.go` | Takeout の CSV の日別集計。部分集計は (ファイル, 日, データソース) 単位で、同じ日に時計（Pixel Watch 2）とスマホ（Phone Health Connect）の行が並んでも足さないこと（時計があれば時計だけ、無い日は `secondary_data_sources` の順で1つ、時計を替えた日は時計どうしを足す、補助に同じ名前を重ねて書いても先頭の順位が効く。`TestCache_TrackerWinsOverPhoneWithinADay` / `TestChooseDataSources`）、設定を変えると取り込み直さずに全日を畳み直し、同じ規則なら `dirty_day` を積まないこと（`TestCache_SecondaryDataSourcesChangeRefolds` / `TestCache_RefoldOnlyWhenFoldRuleChanges`）、旧スキーマ版のキャッシュを作り直すこと、心拍が UTC の2ファイルにまたがる1日を畳めること、差分判定が CRC32 で mtime に頼らないこと、分割 ZIP は合算し別の書き出しは新しいほうだけを採ること（ADR-0303 / ADR-0312）、指標の登録表と時刻の解釈 |
| `src/plugins/gkill_plugin_fitbit/{config,main,html}_test.go` | `configOf` が `secondary_data_sources` の「キー無し＝既定」と「空配列＝全部合算」を区別すること、`foldRule` が大小・空白の違いで変わらず並びの違いで変わること、`PostConfig` が空欄を空配列（全部合算の明示）で保存すること、単独モードの `BuildCache` が同期で1周して戻った時点で日次の値が引けること、設定画面が現在値を出し保存スクリプトが同じキーで postMessage に載せること |
| `src/plugins/gkill_plugin_google_locationhistory/{cache,parsers,manifest,main}_test.go` | Takeout の位置情報の履歴（タイムライン JSON と GPS CSV）の点の重複除去・精度フィルタ・訪問点の除外・並びとページング・期間の両端・増分の走査・読めない形式の報告・上限、manifest の宣言、単独モードの `BuildCache` が同期で1周すること、走査の失敗が `ERROR:` 行に残ること |
| `src/plugins/gkill_plugin_{antigravity,chatgpt,claudeai,claudecode,codex,fitbit,archived_git_commit_log,gemini,vrchat}/find_kyous_test.go` | FindKyous のワード判定。SDK の `Query.MatchText`（gkill 本体と同じ規則）で肯定語・除外語・AND/OR・ID 前方一致（7文字以上の語で。しきい値の境界そのものは下の `match_words_test.go` が守る）が効くこと、chatgpt / claudeai は会話タイトル、codex はスレッド名、archived_git_commit_log はリポジトリ名と author 名にも当たること、vrchat は同じ部屋にいた人の記録もワールド名で当たること、fitbit は数値でも当たり空文字の語で全件が消えないこと、LIMIT が絞った後に掛かること。gkill 本体はプラグインが返した Kyou のワードを再判定しないので、ここが唯一の判定 |
| `src/plugins/gkill_plugin_{chatgpt,claudeai,claudecode,gemini}/find_kyous_ids_test.go`・`gkill_plugin_{antigravity,codex,fitbit,archived_git_commit_log}/query_ids_test.go` | gkill の検索条件の全体（`find_query`）の IDs で読む量を減らすこと。IDs が nil なら今までどおり全件、指定ならその記録だけ、`[]` なら0件。archived_git_commit_log は Reps（リポジトリ名）でも同じ。gkill 本体も ID で絞り直すので結果は変わらず、読む量だけが減る（ADR-0318） |
| `src/plugins/gkill_plugin_vrchat/end_timeis_test.go` | IncludeEndTimeIs のときだけ TimeIs の行から終了の Kyou（`timeis_end`、時刻は終了時刻、ID・型別データ・タグ・版は開始と同じ）も返すこと、開始は開始時刻・終了は終了時刻で期間を判定すること、LIMIT が両方を合わせて新しい順に効くこと、キャッシュの SQL が終了時刻だけ期間に入る TimeIs も拾い IDs で絞れること、同じ部屋の人のタグが `VRChat_room` であること（ADR-0318） |
| `src/server/gkill/plugin/sdk/match_words_test.go` | `sdk.Query.MatchText` / `Matcher` の判定規則（ID の前方一致を見るのは7文字以上の語だけ。7文字未満は ID を見ず、7文字ちょうど・かな7文字は rune 数で数える）と、元の Query を書き換えないこと |
| `src/server/gkill/plugin/sdk/cache_path_test.go` | キャッシュDBの置き場所の解決（`sdk.CacheDBPath`）。`GKILL_HOME` あり／なし（pluginDirから推定）／想定外の構成（プラグインフォルダにフォールバック）／pluginDirが空、の4パターンとパス要素の検証。6プラグインが1文字違わず同じものを持っていたのでSDKへ移した |

`clear_cache plugin` でキャッシュを消せることは
`src/server/gkill/main/common/common_test.go` でテストしている。

## 実行方法

プラグインは独立した Go モジュールなので `cd src/server && go test ./...`（`npm run test_server`）では実行されない。
モジュールを1つずつ回す `npm run test_plugins`（`src/tools/test_plugins.mjs`）を用意しており、`npm test` からも呼ばれる。

```bash
# 全プラグインまとめて（go.mod を持つディレクトリを自動で探して回る）
npm run test_plugins

# 個別に実行する場合
cd src/plugins/gkill_plugin_antigravity && go test ./...
cd src/plugins/gkill_plugin_claudecode && go test ./...
cd src/plugins/gkill_plugin_codex      && go test ./...
cd src/plugins/gkill_plugin_chatgpt    && go test ./...
cd src/plugins/gkill_plugin_claudeai   && go test ./...
cd src/plugins/gkill_plugin_gemini     && go test ./...
cd src/plugins/gkill_plugin_fitbit     && go test ./...
cd src/plugins/gkill_plugin_google_locationhistory && go test ./...
cd src/plugins/gkill_plugin_archived_git_commit_log && go test ./...
cd src/plugins/gkill_plugin_vrchat    && go test ./...
```

`gkill_plugin_codex` でとくに落としてはいけないテスト:

- `TestParseRolloutTakesIdentityFromFirstMetaOnly` — サブエージェントのファイルには2つ目として
  「親の」`session_meta` が入っている。identity をマージすると自分が親にすり替わり、
  親子の Kyou ID が衝突する
- `TestKyouIDIsStableAcrossAppend` — 追記しても既存の Kyou ID が動かないこと
  （動くとユーザが付けたタグやテキストが迷子になる）
- `TestClassifyHeadUnknownPayloadIsKept` — 判定できない行を捨てないこと
  （捨てるとログの形式が変わった日に会話が静かに消える）
- `TestConcurrentReadDuringBuild` — 構築中に検索が詰まらないこと（WAL + ロック分割の回帰）
- `TestParseRolloutReadsItemCompletedLane` — 0.151 以降の形（会話が `item_completed` にだけ載る）を読めること
  （読めないと人間の発言が見つからず、セッション全体がツール呼び出しだけの応答1件に溜まる。エラーは出ない）
- `TestParseRolloutDropsItemLaneInLegacyTurns` — 両方のレーンが同じターンに並んでも会話が二重にならないこと

## プラグイン SDK

SDK 自体のテストは `src/server/gkill/plugin/sdk/` にあり、`src/server` のテストなので
`npm run test_server` で走る。詳細は [server/gkill/plugin/sdk/ABOUT_TEST.md](../server/gkill/plugin/sdk/ABOUT_TEST.md) を参照。

- `plugin_log_test.go`（8テスト）— `Run()` が開く `$GKILL_HOME/logs/gkill_plugin_<name>*.log`。`sdk.LogWarn` / `LogError` は
  stderr の接頭辞行（`last_error` 向け）とファイルの両方へ、`LogInfo` / `LogDebug` はファイルだけへ、`closeLogging` は何度呼んでも安全。
  home が分からなければ stderr だけで続行し、stdout には1バイトも書かない（ADR-0313）。
- `build_cache_test.go`（5テスト）— `--gkill-build-cache` の単独モード。結果行 `built` / `no_cache` 以外を stdout に書かないこと、
  失敗は stderr だけに出して結果行を出さないこと、同梱プラグインが全部 `Handler.BuildCache` を配線していることのソース走査。
- `source_test.go`（18テスト）— ZIP の走査 `OpenSources`（フォルダ・ZIP 直指定・glob、分割 ZIP の束ね、世代の判定、
  展開済みフォルダ・入れ子 ZIP・壊れた ZIP の問題報告、CRC32 / Size の取り出し）。
- `config_test.go`（4テスト）— `EnsureConfig`（config.json の自動生成）。生成される／既存ファイルを
  上書きしない／`DefaultConfig` が nil なら作らない／`pluginDir` が空ならカレントディレクトリを汚さない。
- `sdk_test.go`（22テスト）— `Run()` の stdin/stdout ループ本体（`TestRunLoop_*`）。コマンド分岐、
  未実装時のフォールバック、壊れた JSON でも止まらないこと、`close` / stdin クローズでの終了、
  `get_rep_name` の `rep_names`（未実装なら欄なし・実装済みで0個なら `[]`・エラーは `errors`）を固定している。
- `cache_path_test.go`（5テスト）・`match_words_test.go`（3テスト）— 上の表のとおり。
- `find_query_test.go`（5テスト）— 検索条件の全体 `find_query` の受け取りと `MatchID` / `IDMatcher` / `MatchRep`（ADR-0318）。
- `timezone_test.go`（2テスト）— Android で `time.Local` を端末のゾーンへ直すこと（ADR-0319）。
- `chatview/chatview_test.go`（12テスト）・`chatview/text_test.go`（3テスト）— AI チャット系6本の本文 HTML を描く `chatview`。
  head の契約（postMessage のキー集合が `gkill_iframe_size` / `gkill_theme` だけ。dblclick は書かない）、全欄のエスケープ、
  チップの空白区切り・重複除去、Body → 作業の経過の順と要約行、各ブロックの markup、4MiB の打ち切り、見つからないページ、
  `FormatDuration` / `SummaryLine`、コードフェンスとインラインコードだけを描く `RenderText`（それ以外は `html.EscapeString` と一致）。
  ここでの回帰は6本の表示に同時に出る（ADR-0315）。
- `configview/configview_test.go`（12テスト）・`configview/form_test.go`（5テスト）・`configview/text_test.go`（7テスト）— 同梱10本の設定画面 HTML を描く `configview`（ADR-0316）。head の契約と CSS 変数の定義、postMessage のキー集合、全欄のエスケープ、進捗バー、フォームと保存スクリプト、Prose 記法。

## 新しいプラグインのテスト方針

新しいプラグインを作成した場合は以下を推奨：

1. `main.go` の `FindKyous` ロジックをユニットテスト可能な関数に分離する
2. ローカルにエクスポート ZIP（または相当データファイル）を用意し、手動動作確認を行う
3. プラグインを gkill に組み込み、E2E で Kyou がタイムラインに表示されることを確認する
